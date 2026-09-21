package services

import (
	"context"
	"errors"

	"github.com/resume-platform/backend/internal/models"
	"github.com/resume-platform/backend/internal/repository"
	"github.com/resume-platform/backend/internal/utils"
)

type ApplicationService struct {
	applications *repository.ApplicationRepository
	jobs         *repository.JobRepository
	candidates   *repository.CandidateRepository
	jobService   *JobService
	matching     *MatchingService
}

func NewApplicationService(applications *repository.ApplicationRepository, jobs *repository.JobRepository, candidates *repository.CandidateRepository, jobService *JobService, matching *MatchingService) *ApplicationService {
	return &ApplicationService{applications: applications, jobs: jobs, candidates: candidates, jobService: jobService, matching: matching}
}

// Apply implements "candidates can apply to registered companies/jobs".
// A candidate may only apply once per job (enforced both here and by the
// applications.uq_applications_job_candidate DB constraint -- defense in
// depth), the job must be published, and the candidate must have an active
// (parsed) resume, since applying without one would produce a meaningless
// match score. After creating the application it synchronously triggers
// match calculation (see MatchingService.CalculateForApplication).
func (s *ApplicationService) Apply(ctx context.Context, userID, jobID int64) (*models.Application, error) {
	candidate, err := s.candidates.FindByUserID(ctx, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, utils.NewNotFound("candidate profile")
	}
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	if candidate.ActiveResumeID == nil {
		return nil, utils.NewValidationError("you must upload and parse a resume before applying", nil)
	}

	job, err := s.jobs.FindByID(ctx, jobID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, utils.NewNotFound("job")
	}
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	if job.Status != models.JobStatusPublished {
		return nil, utils.NewValidationError("this job is not currently accepting applications", nil)
	}

	if _, err := s.applications.FindByJobAndCandidate(ctx, jobID, candidate.ID); err == nil {
		return nil, utils.NewConflict("you have already applied to this job")
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, utils.NewDatabaseError(err)
	}

	app := &models.Application{JobID: jobID, CandidateID: candidate.ID, ResumeID: *candidate.ActiveResumeID}
	id, err := s.applications.Create(ctx, app)
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	app.ID = id

	full, err := s.applications.FindByID(ctx, id)
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}

	// Compute the match score immediately so the candidate and company see
	// it right away. A failure here must not fail the application itself
	// (the candidate has successfully applied regardless) -- it's logged by
	// MatchingService and can be recalculated later via the matches
	// endpoint if needed.
	_, _ = s.matching.CalculateForApplication(ctx, id)

	return full, nil
}

// AssertAccess enforces the IDOR protection rule for endpoints reachable by
// both roles (e.g. an application's match score): the authenticated user
// must either be the candidate who owns the application, or belong to the
// company that owns the application's job. Every other caller is
// forbidden, regardless of what the URL claims.
func (s *ApplicationService) AssertAccess(ctx context.Context, userID int64, role models.Role, applicationID int64) (*models.Application, error) {
	app, err := s.GetByID(ctx, applicationID)
	if err != nil {
		return nil, err
	}

	switch role {
	case models.RoleCandidate:
		candidate, err := s.candidates.FindByUserID(ctx, userID)
		if errors.Is(err, repository.ErrNotFound) {
			return nil, utils.NewNotFound("candidate profile")
		}
		if err != nil {
			return nil, utils.NewDatabaseError(err)
		}
		if app.CandidateID != candidate.ID {
			return nil, utils.NewForbidden("you do not have access to this application")
		}
		return app, nil
	case models.RoleCompany:
		if _, err := s.jobService.AssertCompanyOwnsJob(ctx, userID, app.JobID); err != nil {
			return nil, err
		}
		return app, nil
	default:
		return nil, utils.NewForbidden("you do not have access to this application")
	}
}

func (s *ApplicationService) GetByID(ctx context.Context, id int64) (*models.Application, error) {
	a, err := s.applications.FindByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, utils.NewNotFound("application")
	}
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	return a, nil
}

func (s *ApplicationService) ListForCandidate(ctx context.Context, userID int64) ([]models.Application, error) {
	candidate, err := s.candidates.FindByUserID(ctx, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, utils.NewNotFound("candidate profile")
	}
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	list, err := s.applications.ListByCandidate(ctx, candidate.ID)
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	return list, nil
}

// ListForJob enforces the same IDOR protection as job mutation: a company
// can only list applicants for jobs it owns.
func (s *ApplicationService) ListForJob(ctx context.Context, userID, jobID int64) ([]models.Application, error) {
	if _, err := s.jobService.AssertCompanyOwnsJob(ctx, userID, jobID); err != nil {
		return nil, err
	}
	list, err := s.applications.ListByJob(ctx, jobID)
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	return list, nil
}

// UpdateStatus implements HR shortlist/reject actions. Ownership is
// re-verified via the application's parent job, not trusted from the
// request.
func (s *ApplicationService) UpdateStatus(ctx context.Context, userID, applicationID int64, status models.ApplicationStatus) (*models.Application, error) {
	app, err := s.GetByID(ctx, applicationID)
	if err != nil {
		return nil, err
	}
	if _, err := s.jobService.AssertCompanyOwnsJob(ctx, userID, app.JobID); err != nil {
		return nil, err
	}

	validTransitions := map[models.ApplicationStatus]bool{
		models.AppStatusUnderReview: true, models.AppStatusShortlisted: true,
		models.AppStatusRejected: true, models.AppStatusHired: true,
	}
	if !validTransitions[status] {
		return nil, utils.NewValidationError("invalid application status", nil)
	}

	if err := s.applications.UpdateStatus(ctx, applicationID, status); err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	app.Status = status
	return app, nil
}
