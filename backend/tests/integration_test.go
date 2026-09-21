// Package tests contains integration tests that run against a real MySQL
// database (resume_platform_test by default) rather than mocks, exercising
// the actual repository/service layers together. This is what the project
// spec's "integration tests: register, login, create company, create job,
// create candidate, upload resume, parse resume, apply to job, calculate
// match" requirement refers to.
//
// Run with: go test ./tests/... -v
// Requires a reachable MySQL instance with the schema already migrated --
// see README.md "Testing" section. Skips automatically if TEST_DATABASE_DSN
// (or the default local DSN) is unreachable, so `go test ./...` doesn't
// hard-fail in environments without MySQL configured.
package tests

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/resume-platform/backend/internal/ai"
	"github.com/resume-platform/backend/internal/auth"
	"github.com/resume-platform/backend/internal/models"
	"github.com/resume-platform/backend/internal/repository"
	"github.com/resume-platform/backend/internal/services"
	"github.com/resume-platform/backend/internal/storage"
	"github.com/rs/zerolog"
)

func testDSN() string {
	if v := os.Getenv("TEST_DATABASE_DSN"); v != "" {
		return v
	}
	return "app_user:app_password_dev123@tcp(127.0.0.1:3306)/resume_platform_test?parseTime=true&charset=utf8mb4&loc=Local"
}

func connectTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("mysql", testDSN())
	if err != nil {
		t.Skipf("skipping integration test: cannot open test db: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Skipf("skipping integration test: test db unreachable (run MySQL + migrations first): %v", err)
	}
	return db
}

func truncateAll(t *testing.T, db *sql.DB) {
	t.Helper()
	tables := []string{
		"interview_evaluations", "interview_answers", "interview_questions", "interviews",
		"match_results", "applications",
		"job_match_weights", "job_certifications", "job_skills", "jobs",
		"resume_certifications", "resume_skills", "resume_projects", "resume_experience", "resume_education", "resumes",
		"companies", "candidates", "refresh_tokens", "users",
	}
	if _, err := db.Exec("SET FOREIGN_KEY_CHECKS=0"); err != nil {
		t.Fatalf("failed to disable fk checks: %v", err)
	}
	for _, tbl := range tables {
		if _, err := db.Exec("TRUNCATE TABLE " + tbl); err != nil {
			t.Fatalf("failed to truncate %s: %v", tbl, err)
		}
	}
	if _, err := db.Exec("SET FOREIGN_KEY_CHECKS=1"); err != nil {
		t.Fatalf("failed to re-enable fk checks: %v", err)
	}
}

// createResumeFromText writes the resume text to real storage and inserts
// the matching DB row, bypassing only the HTTP multipart upload layer --
// this test exercises the storage + parsing + downstream pipeline, not
// multipart file handling (that's covered by internal/resume's own unit
// tests and internal/storage's path-traversal tests).
func createResumeFromText(ctx context.Context, repo *repository.ResumeRepository, store storage.FileStorage, candidateID int64, text string) (int64, error) {
	storageKey, err := store.Save("resumes", "resume.txt", strings.NewReader(text))
	if err != nil {
		return 0, err
	}
	r := &models.Resume{
		CandidateID: candidateID, OriginalFilename: "resume.txt", StoredFilename: storageKey,
		FilePath: store.AbsolutePath(storageKey), FileSize: uint32(len(text)), MimeType: "text/plain",
		Status: models.ResumeStatusUploaded,
	}
	id, err := repo.Create(ctx, r)
	if err != nil {
		return 0, err
	}
	if err := repo.SaveRawText(ctx, id, text); err != nil {
		return 0, err
	}
	return id, nil
}

// TestFullWorkflow_RegisterJobResumeApplyMatch exercises the entire
// candidate + company + job + resume + application + matching pipeline
// against a real database, mirroring exactly the workflow the project spec
// asks to be integration-tested.
func TestFullWorkflow_RegisterJobResumeApplyMatch(t *testing.T) {
	db := connectTestDB(t)
	defer db.Close()
	truncateAll(t, db)

	logger := zerolog.Nop()
	tmpDir := t.TempDir()
	ctx := context.Background()

	userRepo := repository.NewUserRepository(db)
	refreshTokenRepo := repository.NewRefreshTokenRepository(db)
	candidateRepo := repository.NewCandidateRepository(db)
	companyRepo := repository.NewCompanyRepository(db)
	resumeRepo := repository.NewResumeRepository(db)
	jobRepo := repository.NewJobRepository(db)
	applicationRepo := repository.NewApplicationRepository(db)
	matchRepo := repository.NewMatchRepository(db)

	tokenManager := auth.NewTokenManager("test-access-secret", "test-refresh-secret", 900000000000, 604800000000000)
	fileStorage, err := storage.NewLocalStorage(tmpDir)
	if err != nil {
		t.Fatalf("failed to create local storage: %v", err)
	}
	aiProvider := ai.NewStubProvider()

	authService := services.NewAuthService(userRepo, refreshTokenRepo, candidateRepo, companyRepo, tokenManager)
	jobService := services.NewJobService(jobRepo, companyRepo)
	resumeService := services.NewResumeService(resumeRepo, candidateRepo, fileStorage, aiProvider, 10*1024*1024, logger)
	matchingService := services.NewMatchingService(matchRepo, applicationRepo, jobRepo, resumeRepo, jobService, aiProvider, logger)
	applicationService := services.NewApplicationService(applicationRepo, jobRepo, candidateRepo, jobService, matchingService)

	// --- 1. register candidate ---
	candAuth, err := authService.RegisterCandidate(ctx, services.RegisterCandidateInput{
		Email: "integration.candidate@example.com", Password: "SecurePass123", FullName: "Integration Test Candidate",
	})
	if err != nil {
		t.Fatalf("RegisterCandidate failed: %v", err)
	}
	if candAuth.User.Role != models.RoleCandidate {
		t.Errorf("expected candidate role, got %s", candAuth.User.Role)
	}

	// --- 2. register company ---
	compAuth, err := authService.RegisterCompany(ctx, services.RegisterCompanyInput{
		Email: "integration.company@example.com", Password: "SecurePass123", CompanyName: "Integration Test Co",
	})
	if err != nil {
		t.Fatalf("RegisterCompany failed: %v", err)
	}

	// --- 3. login as candidate (verify password auth roundtrip) ---
	loginResult, err := authService.Login(ctx, "integration.candidate@example.com", "SecurePass123")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if loginResult.User.ID != candAuth.User.ID {
		t.Errorf("login returned a different user than registration")
	}

	// --- 4. create + publish job ---
	job, err := jobService.Create(ctx, compAuth.User.ID, services.CreateJobInput{
		Title: "Junior Go Backend Developer", Description: "Build REST APIs in Go with MySQL and Docker.",
		ExperienceMinYears: 1,
		Skills: []services.JobSkillInput{
			{Name: "Go", IsRequired: true, Weight: 2},
			{Name: "MySQL", IsRequired: true, Weight: 1.5},
			{Name: "Docker", IsRequired: false, Weight: 1},
		},
	})
	if err != nil {
		t.Fatalf("Create job failed: %v", err)
	}
	if _, err := jobService.Publish(ctx, compAuth.User.ID, job.ID); err != nil {
		t.Fatalf("Publish job failed: %v", err)
	}

	// --- 5. IDOR check: a different (non-owning) company user must not be
	// able to publish/modify this job ---
	otherCompAuth, err := authService.RegisterCompany(ctx, services.RegisterCompanyInput{
		Email: "integration.other-company@example.com", Password: "SecurePass123", CompanyName: "Other Co",
	})
	if err != nil {
		t.Fatalf("RegisterCompany (other) failed: %v", err)
	}
	if _, err := jobService.Close(ctx, otherCompAuth.User.ID, job.ID); err == nil {
		t.Error("expected an authorization error when a non-owning company tries to close another company's job")
	}

	// --- 6. upload + parse resume ---
	candidate, err := candidateRepo.FindByUserID(ctx, candAuth.User.ID)
	if err != nil {
		t.Fatalf("FindByUserID failed: %v", err)
	}
	resumeText := "Integration Test Candidate\nintegration.candidate@example.com\nSkills: Go, MySQL, Docker, Git"
	resumeID, err := createResumeFromText(ctx, resumeRepo, fileStorage, candidate.ID, resumeText)
	if err != nil {
		t.Fatalf("createResumeFromText failed: %v", err)
	}
	full, err := resumeService.ParseResume(ctx, resumeID)
	if err != nil {
		t.Fatalf("ParseResume failed: %v", err)
	}
	if len(full.Skills) == 0 {
		t.Fatal("expected parsed resume to have extracted skills")
	}

	// --- 7. apply to job ---
	app, err := applicationService.Apply(ctx, candAuth.User.ID, job.ID)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	if app.Status != models.AppStatusApplied {
		t.Errorf("expected status 'applied', got %s", app.Status)
	}

	// --- 8. duplicate application must be rejected ---
	if _, err := applicationService.Apply(ctx, candAuth.User.ID, job.ID); err == nil {
		t.Error("expected an error when applying to the same job twice")
	}

	// --- 9. match score was calculated automatically on apply ---
	match, err := matchingService.GetForApplication(ctx, app.ID)
	if err != nil {
		t.Fatalf("GetForApplication failed (match should have been calculated on apply): %v", err)
	}
	if match.SkillScore <= 0 {
		t.Errorf("expected a positive skill score given the candidate has Go/MySQL/Docker, got %v", match.SkillScore)
	}

	// --- 10. IDOR check: candidate cannot access another candidate's application ---
	otherCandAuth, err := authService.RegisterCandidate(ctx, services.RegisterCandidateInput{
		Email: "integration.other-candidate@example.com", Password: "SecurePass123", FullName: "Other Candidate",
	})
	if err != nil {
		t.Fatalf("RegisterCandidate (other) failed: %v", err)
	}
	if _, err := applicationService.AssertAccess(ctx, otherCandAuth.User.ID, models.RoleCandidate, app.ID); err == nil {
		t.Error("expected an authorization error when a different candidate tries to access this application")
	}

	// --- 11. company shortlists the candidate ---
	updated, err := applicationService.UpdateStatus(ctx, compAuth.User.ID, app.ID, models.AppStatusShortlisted)
	if err != nil {
		t.Fatalf("UpdateStatus (shortlist) failed: %v", err)
	}
	if updated.Status != models.AppStatusShortlisted {
		t.Errorf("expected status 'shortlisted', got %s", updated.Status)
	}
}
