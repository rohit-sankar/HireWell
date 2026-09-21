package services

import (
	"context"
	"errors"

	"github.com/resume-platform/backend/internal/ai"
	"github.com/resume-platform/backend/internal/matching"
	"github.com/resume-platform/backend/internal/models"
	"github.com/resume-platform/backend/internal/repository"
	"github.com/resume-platform/backend/internal/utils"
	"github.com/rs/zerolog"
)

type MatchingService struct {
	matches      *repository.MatchRepository
	applications *repository.ApplicationRepository
	jobs         *repository.JobRepository
	resumes      *repository.ResumeRepository
	jobService   *JobService
	aiProvider   ai.Provider
	logger       zerolog.Logger
}

func NewMatchingService(matches *repository.MatchRepository, applications *repository.ApplicationRepository, jobs *repository.JobRepository, resumes *repository.ResumeRepository, jobService *JobService, provider ai.Provider, logger zerolog.Logger) *MatchingService {
	return &MatchingService{matches: matches, applications: applications, jobs: jobs, resumes: resumes, jobService: jobService, aiProvider: provider, logger: logger}
}

// CalculateForApplication is the reproducible entry point described in the
// spec: "make matching reproducible" -- calling this twice for the same
// application always recomputes from the current job/resume state and
// overwrites the stored result (see MatchRepository.Upsert), rather than
// accumulating drifting history.
func (s *MatchingService) CalculateForApplication(ctx context.Context, applicationID int64) (*models.MatchResult, error) {
	app, err := s.applications.FindByID(ctx, applicationID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, utils.NewNotFound("application")
	}
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}

	jobFull, err := s.jobService.GetFull(ctx, app.JobID)
	if err != nil {
		return nil, err
	}
	resumeFull, err := s.resumes.GetFull(ctx, app.ResumeID)
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}

	// AI semantic assist is optional and best-effort: if the provider
	// errors out (rate limit, timeout, misconfiguration), matching still
	// proceeds on the deterministic score alone rather than failing the
	// whole operation -- AI assists the score, it does not gate it.
	var semanticExtra []matching.SemanticPair
	jobSkillNames := skillNames(jobFull.Skills)
	candidateSkillNames := resumeSkillNames(resumeFull.Skills)
	if len(jobSkillNames) > 0 && len(candidateSkillNames) > 0 {
		semResult, semErr := s.aiProvider.SemanticMatch(ctx, ai.SemanticMatchInput{JobSkills: jobSkillNames, CandidateSkills: candidateSkillNames})
		if semErr != nil {
			s.logger.Warn().Err(semErr).Int64("application_id", applicationID).Msg("semantic match assist failed, continuing with deterministic score only")
		} else {
			for _, pair := range semResult.AdditionalMatches {
				semanticExtra = append(semanticExtra, matching.SemanticPair{
					JobSkillSlug:       utils.CanonicalSkillSlug(pair.JobSkill),
					CandidateSkillSlug: utils.CanonicalSkillSlug(pair.CandidateSkill),
					Confidence:         pair.Confidence,
				})
			}
		}
	}

	result := matching.Calculate(matching.Input{Job: *jobFull, Resume: *resumeFull, SemanticExtra: semanticExtra})

	mr := &models.MatchResult{
		ApplicationID:      applicationID,
		SkillScore:         result.SkillScore,
		ExperienceScore:    result.ExperienceScore,
		EducationScore:     result.EducationScore,
		ProjectScore:       result.ProjectScore,
		CertificationScore: result.CertificationScore,
		OverallScore:       result.OverallScore,
		WeightsUsed:        matching.WeightsToMap(result.WeightsUsed),
		Explanation:        matching.ExplanationToMap(result.Explanation),
	}
	if err := s.matches.Upsert(ctx, mr); err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	return mr, nil
}

func (s *MatchingService) GetForApplication(ctx context.Context, applicationID int64) (*models.MatchResult, error) {
	m, err := s.matches.FindByApplicationID(ctx, applicationID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, utils.NewNotFound("match result")
	}
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	return m, nil
}

// RankedForJob implements "companies/HR receive ranked/recommended
// candidates". Ownership check happens via jobService in the controller
// layer before this is called.
func (s *MatchingService) RankedForJob(ctx context.Context, jobID int64) ([]repository.RankedApplication, error) {
	list, err := s.matches.RankedForJob(ctx, jobID)
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	return list, nil
}

func skillNames(skills []models.JobSkill) []string {
	out := make([]string, len(skills))
	for i, s := range skills {
		out[i] = s.SkillName
	}
	return out
}

func resumeSkillNames(skills []models.ResumeSkill) []string {
	out := make([]string, len(skills))
	for i, s := range skills {
		out[i] = s.SkillName
	}
	return out
}
