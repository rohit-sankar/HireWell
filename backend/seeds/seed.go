//go:build ignore

// Run with: go run seeds/seed.go
//
// Seeds realistic demo data matching the project spec's example scenario:
// candidate "John Doe" (skills: Go, MySQL, REST APIs, Docker, Git; projects:
// AI Resume Parser, E-commerce API), company "Example Technologies", and a
// published "Junior Go Backend Developer" job requiring exactly those
// skills -- so the full match/interview pipeline has a realistic, coherent
// scenario to demonstrate end to end.
//
// This is intentionally a plain `go run` script (not wired into the server
// binary) so seeding is an explicit, visible, opt-in developer action, never
// something that could accidentally run against a production database.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/resume-platform/backend/internal/ai"
	"github.com/resume-platform/backend/internal/auth"
	"github.com/resume-platform/backend/internal/config"
	"github.com/resume-platform/backend/internal/database"
	"github.com/resume-platform/backend/internal/models"
	"github.com/resume-platform/backend/internal/repository"
	"github.com/resume-platform/backend/internal/services"
	"github.com/resume-platform/backend/internal/storage"
	"github.com/rs/zerolog"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("db connect error: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	logger := zerolog.New(os.Stdout)

	userRepo := repository.NewUserRepository(db)
	refreshTokenRepo := repository.NewRefreshTokenRepository(db)
	candidateRepo := repository.NewCandidateRepository(db)
	companyRepo := repository.NewCompanyRepository(db)
	resumeRepo := repository.NewResumeRepository(db)
	jobRepo := repository.NewJobRepository(db)
	applicationRepo := repository.NewApplicationRepository(db)
	matchRepo := repository.NewMatchRepository(db)

	tokenManager := auth.NewTokenManager(cfg.JWTSecret, cfg.JWTRefreshSecret, cfg.JWTAccessExpiry, cfg.JWTRefreshExpiry)
	fileStorage, err := storage.NewLocalStorage(cfg.UploadDir)
	if err != nil {
		log.Fatalf("storage error: %v", err)
	}
	aiProvider, err := ai.NewProvider(cfg.AIProvider, cfg.AIAPIKey, cfg.AIModel)
	if err != nil {
		log.Fatalf("ai provider error: %v", err)
	}

	authService := services.NewAuthService(userRepo, refreshTokenRepo, candidateRepo, companyRepo, tokenManager)
	jobService := services.NewJobService(jobRepo, companyRepo)
	resumeService := services.NewResumeService(resumeRepo, candidateRepo, fileStorage, aiProvider, cfg.MaxUploadSize, logger)
	matchingService := services.NewMatchingService(matchRepo, applicationRepo, jobRepo, resumeRepo, jobService, aiProvider, logger)
	applicationService := services.NewApplicationService(applicationRepo, jobRepo, candidateRepo, jobService, matchingService)

	if existing, _ := userRepo.FindByEmail(ctx, "john.doe@example.com"); existing != nil {
		fmt.Println("seed data already present (john.doe@example.com exists) -- skipping. Truncate tables first to reseed.")
		return
	}

	fmt.Println("seeding candidate: John Doe ...")
	candAuth, err := authService.RegisterCandidate(ctx, services.RegisterCandidateInput{
		Email: "john.doe@example.com", Password: "SecurePass123!", FullName: "John Doe", Phone: "555-0100",
	})
	must(err)

	fmt.Println("seeding company: Example Technologies ...")
	compAuth, err := authService.RegisterCompany(ctx, services.RegisterCompanyInput{
		Email: "hr@exampletech.com", Password: "SecurePass123!", CompanyName: "Example Technologies", Website: "https://exampletech.example.com",
	})
	must(err)

	fmt.Println("seeding job: Junior Go Backend Developer ...")
	job, err := jobService.Create(ctx, compAuth.User.ID, services.CreateJobInput{
		Title:                  "Junior Go Backend Developer",
		Description:            "Join our platform team building REST APIs and backend services in Go, backed by MySQL and deployed with Docker.",
		Location:               "Remote",
		EmploymentType:         models.EmploymentFullTime,
		ExperienceMinYears:     1,
		EducationRequirement:   "Bachelor's degree in Computer Science or related field",
		InterviewQuestionCount: 5,
		Skills: []services.JobSkillInput{
			{Name: "Go", IsRequired: true, Weight: 2},
			{Name: "MySQL", IsRequired: true, Weight: 1.5},
			{Name: "REST API", IsRequired: true, Weight: 1.5},
			{Name: "Docker", IsRequired: false, Weight: 1},
			{Name: "Git", IsRequired: false, Weight: 1},
		},
	})
	must(err)
	_, err = jobService.Publish(ctx, compAuth.User.ID, job.ID)
	must(err)

	fmt.Println("seeding John Doe's resume ...")
	candidate, err := candidateRepo.FindByUserID(ctx, candAuth.User.ID)
	must(err)

	resumeText := `John Doe
john.doe@example.com
555-0100
San Francisco, CA

SUMMARY
Backend developer with hands-on experience building REST APIs and services in Go, MySQL, and Docker.

EXPERIENCE
Backend Developer - Example Technologies
2023-01 - Present
Built and maintained REST APIs in Go for the company's core platform. Worked with MySQL for data
persistence and Docker for containerized deployments. Used Git for version control across the team.

PROJECTS
AI Resume Parser
An AI-powered resume parsing service built with Go, Gin, and MySQL. Exposes a REST API for uploading and parsing resumes.
Technologies: Go, MySQL, Docker, REST API

E-commerce API
A REST API for an e-commerce platform built with Go and MySQL, containerized with Docker.
Technologies: Go, MySQL, REST API, Docker

EDUCATION
B.S. Computer Science, State University, 2019 - 2023

SKILLS
Go, Golang, MySQL, REST APIs, Docker, Git`

	storageKey, err := fileStorage.Save("resumes", "john_doe_resume.txt", strings.NewReader(resumeText))
	must(err)
	resumeID, err := resumeRepo.Create(ctx, &models.Resume{
		CandidateID: candidate.ID, OriginalFilename: "john_doe_resume.txt", StoredFilename: storageKey,
		FilePath: fileStorage.AbsolutePath(storageKey), FileSize: uint32(len(resumeText)), MimeType: "text/plain",
		Status: models.ResumeStatusUploaded,
	})
	must(err)
	must(resumeRepo.SaveRawText(ctx, resumeID, resumeText))
	_, err = resumeService.ParseResume(ctx, resumeID)
	must(err)

	fmt.Println("seeding application + match calculation ...")
	app, err := applicationService.Apply(ctx, candAuth.User.ID, job.ID)
	must(err)

	match, err := matchingService.GetForApplication(ctx, app.ID)
	must(err)

	fmt.Println()
	fmt.Println("=== Seed complete ===")
	fmt.Printf("Candidate login:  john.doe@example.com / SecurePass123!\n")
	fmt.Printf("Company login:    hr@exampletech.com / SecurePass123!\n")
	fmt.Printf("Job:              %s (id=%d, published)\n", job.Title, job.ID)
	fmt.Printf("Application:      id=%d, overall match score=%.2f\n", app.ID, match.OverallScore)
}

func must(err error) {
	if err != nil {
		log.Fatalf("seed error: %v", err)
	}
}
