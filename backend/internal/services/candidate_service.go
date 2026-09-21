package services

import (
	"context"
	"errors"

	"github.com/resume-platform/backend/internal/models"
	"github.com/resume-platform/backend/internal/repository"
	"github.com/resume-platform/backend/internal/utils"
)

type CandidateService struct {
	candidates *repository.CandidateRepository
	resumes    *repository.ResumeRepository
}

func NewCandidateService(candidates *repository.CandidateRepository, resumes *repository.ResumeRepository) *CandidateService {
	return &CandidateService{candidates: candidates, resumes: resumes}
}

func (s *CandidateService) GetByUserID(ctx context.Context, userID int64) (*models.Candidate, error) {
	c, err := s.candidates.FindByUserID(ctx, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, utils.NewNotFound("candidate profile")
	}
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	return c, nil
}

func (s *CandidateService) GetByID(ctx context.Context, id int64) (*models.Candidate, error) {
	c, err := s.candidates.FindByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, utils.NewNotFound("candidate")
	}
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	return c, nil
}

type UpdateCandidateInput struct {
	FullName string
	Phone    string
	Location string
	Headline string
	Summary  string
	Links    map[string]interface{}
}

func (s *CandidateService) UpdateProfile(ctx context.Context, userID int64, in UpdateCandidateInput) (*models.Candidate, error) {
	c, err := s.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if in.FullName != "" {
		c.FullName = in.FullName
	}
	if in.Phone != "" {
		c.Phone = &in.Phone
	}
	if in.Location != "" {
		c.Location = &in.Location
	}
	if in.Headline != "" {
		c.Headline = &in.Headline
	}
	if in.Summary != "" {
		c.Summary = &in.Summary
	}
	if in.Links != nil {
		c.Links = in.Links
	}

	c.ProfileCompletion = computeProfileCompletion(c)

	if err := s.candidates.Update(ctx, c); err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	return c, nil
}

// computeProfileCompletion is a simple, documented, deterministic
// percentage: each of these 6 fields present contributes ~16-17%. Not AI
// generated -- profile completeness is exactly the kind of thing that
// should be transparent and predictable to the user.
func computeProfileCompletion(c *models.Candidate) uint8 {
	total := 6
	filled := 0
	if c.FullName != "" {
		filled++
	}
	if c.Phone != nil && *c.Phone != "" {
		filled++
	}
	if c.Location != nil && *c.Location != "" {
		filled++
	}
	if c.Headline != nil && *c.Headline != "" {
		filled++
	}
	if c.Summary != nil && *c.Summary != "" {
		filled++
	}
	if c.ActiveResumeID != nil {
		filled++
	}
	return uint8((filled * 100) / total)
}
