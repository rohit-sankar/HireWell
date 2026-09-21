package services

import (
	"context"
	"errors"

	"github.com/resume-platform/backend/internal/models"
	"github.com/resume-platform/backend/internal/repository"
	"github.com/resume-platform/backend/internal/utils"
)

type JobService struct {
	jobs      *repository.JobRepository
	companies *repository.CompanyRepository
}

func NewJobService(jobs *repository.JobRepository, companies *repository.CompanyRepository) *JobService {
	return &JobService{jobs: jobs, companies: companies}
}

type JobSkillInput struct {
	Name       string
	IsRequired bool
	Weight     float64
}

type CreateJobInput struct {
	Title                  string
	Description            string
	Location               string
	EmploymentType         models.EmploymentType
	ExperienceMinYears     uint8
	ExperienceMaxYears     *uint8
	EducationRequirement   string
	InterviewQuestionCount uint8
	Skills                 []JobSkillInput
	Certifications         []string
}

// resolveCompanyOwner is the single choke point for the IDOR protection
// requirement: "a company/HR user must never be able to modify another
// company's private data". Every job-mutating method calls this first.
func (s *JobService) resolveCompanyOwner(ctx context.Context, userID int64) (*models.Company, error) {
	c, err := s.companies.FindByUserID(ctx, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, utils.NewNotFound("company profile")
	}
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	return c, nil
}

func (s *JobService) assertOwnership(job *models.Job, companyID int64) error {
	if job.CompanyID != companyID {
		return utils.NewForbidden("you do not have access to this job")
	}
	return nil
}

func (s *JobService) Create(ctx context.Context, userID int64, in CreateJobInput) (*models.JobFull, error) {
	company, err := s.resolveCompanyOwner(ctx, userID)
	if err != nil {
		return nil, err
	}
	if in.Title == "" || in.Description == "" {
		return nil, utils.NewValidationError("title and description are required", nil)
	}

	qCount := in.InterviewQuestionCount
	if qCount == 0 {
		qCount = 5
	}

	job := &models.Job{
		CompanyID:              company.ID,
		Title:                  in.Title,
		Description:            in.Description,
		Location:               strPtrOrNil(in.Location),
		EmploymentType:         in.EmploymentType,
		ExperienceMinYears:     in.ExperienceMinYears,
		ExperienceMaxYears:     in.ExperienceMaxYears,
		EducationRequirement:   strPtrOrNil(in.EducationRequirement),
		Status:                 models.JobStatusDraft,
		InterviewQuestionCount: qCount,
	}
	if job.EmploymentType == "" {
		job.EmploymentType = models.EmploymentFullTime
	}

	id, err := s.jobs.Create(ctx, job)
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	job.ID = id

	if err := s.replaceSkillsAndCerts(ctx, id, in.Skills, in.Certifications); err != nil {
		return nil, err
	}

	return s.GetFull(ctx, id)
}

func (s *JobService) replaceSkillsAndCerts(ctx context.Context, jobID int64, skillsIn []JobSkillInput, certsIn []string) error {
	skills := make([]models.JobSkill, 0, len(skillsIn))
	for _, si := range skillsIn {
		w := si.Weight
		if w <= 0 {
			w = 1
		}
		skills = append(skills, models.JobSkill{
			JobID: jobID, SkillName: si.Name, SkillSlug: utils.CanonicalSkillSlug(si.Name),
			IsRequired: si.IsRequired, Weight: w,
		})
	}
	if err := s.jobs.ReplaceSkills(ctx, jobID, skills); err != nil {
		return utils.NewDatabaseError(err)
	}

	certs := make([]models.JobCertification, 0, len(certsIn))
	for _, c := range certsIn {
		certs = append(certs, models.JobCertification{JobID: jobID, CertificationName: c, CertificationSlug: utils.Slugify(c)})
	}
	if err := s.jobs.ReplaceCertifications(ctx, jobID, certs); err != nil {
		return utils.NewDatabaseError(err)
	}
	return nil
}

func (s *JobService) Update(ctx context.Context, userID, jobID int64, in CreateJobInput) (*models.JobFull, error) {
	company, err := s.resolveCompanyOwner(ctx, userID)
	if err != nil {
		return nil, err
	}
	job, err := s.jobs.FindByID(ctx, jobID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, utils.NewNotFound("job")
	}
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	if err := s.assertOwnership(job, company.ID); err != nil {
		return nil, err
	}

	if in.Title != "" {
		job.Title = in.Title
	}
	if in.Description != "" {
		job.Description = in.Description
	}
	if in.Location != "" {
		job.Location = &in.Location
	}
	if in.EmploymentType != "" {
		job.EmploymentType = in.EmploymentType
	}
	job.ExperienceMinYears = in.ExperienceMinYears
	job.ExperienceMaxYears = in.ExperienceMaxYears
	if in.EducationRequirement != "" {
		job.EducationRequirement = &in.EducationRequirement
	}
	if in.InterviewQuestionCount > 0 {
		job.InterviewQuestionCount = in.InterviewQuestionCount
	}

	if err := s.jobs.Update(ctx, job); err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	if in.Skills != nil || in.Certifications != nil {
		if err := s.replaceSkillsAndCerts(ctx, jobID, in.Skills, in.Certifications); err != nil {
			return nil, err
		}
	}

	return s.GetFull(ctx, jobID)
}

func (s *JobService) changeStatus(ctx context.Context, userID, jobID int64, newStatus models.JobStatus, timestampCol string) (*models.Job, error) {
	company, err := s.resolveCompanyOwner(ctx, userID)
	if err != nil {
		return nil, err
	}
	job, err := s.jobs.FindByID(ctx, jobID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, utils.NewNotFound("job")
	}
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	if err := s.assertOwnership(job, company.ID); err != nil {
		return nil, err
	}

	if err := s.jobs.UpdateStatus(ctx, jobID, newStatus, timestampCol); err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	updated, err := s.jobs.FindByID(ctx, jobID)
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	return updated, nil
}

func (s *JobService) Publish(ctx context.Context, userID, jobID int64) (*models.Job, error) {
	return s.changeStatus(ctx, userID, jobID, models.JobStatusPublished, "published_at")
}

func (s *JobService) Close(ctx context.Context, userID, jobID int64) (*models.Job, error) {
	return s.changeStatus(ctx, userID, jobID, models.JobStatusClosed, "closed_at")
}

func (s *JobService) Archive(ctx context.Context, userID, jobID int64) (*models.Job, error) {
	return s.changeStatus(ctx, userID, jobID, models.JobStatusArchived, "")
}

func (s *JobService) Delete(ctx context.Context, userID, jobID int64) error {
	company, err := s.resolveCompanyOwner(ctx, userID)
	if err != nil {
		return err
	}
	job, err := s.jobs.FindByID(ctx, jobID)
	if errors.Is(err, repository.ErrNotFound) {
		return utils.NewNotFound("job")
	}
	if err != nil {
		return utils.NewDatabaseError(err)
	}
	if err := s.assertOwnership(job, company.ID); err != nil {
		return err
	}
	if err := s.jobs.Delete(ctx, jobID); err != nil {
		return utils.NewDatabaseError(err)
	}
	return nil
}

func (s *JobService) GetFull(ctx context.Context, jobID int64) (*models.JobFull, error) {
	job, err := s.jobs.FindByID(ctx, jobID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, utils.NewNotFound("job")
	}
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	skills, err := s.jobs.GetSkills(ctx, jobID)
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	certs, err := s.jobs.GetCertifications(ctx, jobID)
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	weights, err := s.jobs.GetMatchWeights(ctx, jobID)
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	return &models.JobFull{Job: *job, Skills: skills, Certifications: certs, MatchWeights: *weights}, nil
}

type ListJobsInput struct {
	CompanyID *int64
	Status    *models.JobStatus
	Search    string
	Page      int
	PageSize  int
}

// ListMine resolves the caller's company from userID and lists that
// company's jobs regardless of status (draft/published/closed/archived) --
// unlike the public List() path, which callers scope to published-only
// themselves.
func (s *JobService) ListMine(ctx context.Context, userID int64, in ListJobsInput) ([]models.Job, int, error) {
	company, err := s.resolveCompanyOwner(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	in.CompanyID = &company.ID
	return s.List(ctx, in)
}

func (s *JobService) List(ctx context.Context, in ListJobsInput) ([]models.Job, int, error) {
	page := in.Page
	if page < 1 {
		page = 1
	}
	pageSize := in.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	jobs, total, err := s.jobs.List(ctx, repository.JobListFilter{
		CompanyID: in.CompanyID, Status: in.Status, Search: in.Search,
		Limit: pageSize, Offset: (page - 1) * pageSize,
	})
	if err != nil {
		return nil, 0, utils.NewDatabaseError(err)
	}
	return jobs, total, nil
}

// AssertCompanyOwnsJob is a small exported helper other services (matching,
// applications) use to enforce the same IDOR protection when a company
// views job-scoped resources.
func (s *JobService) AssertCompanyOwnsJob(ctx context.Context, userID, jobID int64) (*models.Job, error) {
	company, err := s.resolveCompanyOwner(ctx, userID)
	if err != nil {
		return nil, err
	}
	job, err := s.jobs.FindByID(ctx, jobID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, utils.NewNotFound("job")
	}
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	if err := s.assertOwnership(job, company.ID); err != nil {
		return nil, err
	}
	return job, nil
}

func (s *JobService) UpdateMatchWeights(ctx context.Context, userID, jobID int64, w models.MatchWeights) error {
	if _, err := s.AssertCompanyOwnsJob(ctx, userID, jobID); err != nil {
		return err
	}
	sum := w.WeightSkill + w.WeightExperience + w.WeightEducation + w.WeightProject + w.WeightCertification
	if sum < 0.98 || sum > 1.02 {
		return utils.NewValidationError("match weights must sum to approximately 1.0", map[string]float64{"sum": sum})
	}
	if err := s.jobs.UpdateMatchWeights(ctx, jobID, w); err != nil {
		return utils.NewDatabaseError(err)
	}
	return nil
}
