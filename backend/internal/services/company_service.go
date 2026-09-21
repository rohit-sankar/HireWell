package services

import (
	"context"
	"errors"

	"github.com/resume-platform/backend/internal/models"
	"github.com/resume-platform/backend/internal/repository"
	"github.com/resume-platform/backend/internal/utils"
)

type CompanyService struct {
	companies *repository.CompanyRepository
}

func NewCompanyService(companies *repository.CompanyRepository) *CompanyService {
	return &CompanyService{companies: companies}
}

func (s *CompanyService) GetByUserID(ctx context.Context, userID int64) (*models.Company, error) {
	c, err := s.companies.FindByUserID(ctx, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, utils.NewNotFound("company profile")
	}
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	return c, nil
}

func (s *CompanyService) GetByID(ctx context.Context, id int64) (*models.Company, error) {
	c, err := s.companies.FindByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, utils.NewNotFound("company")
	}
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	return c, nil
}

type UpdateCompanyInput struct {
	CompanyName string
	Website     string
	Description string
	LogoURL     string
	Industry    string
	CompanySize string
}

func (s *CompanyService) UpdateProfile(ctx context.Context, userID int64, in UpdateCompanyInput) (*models.Company, error) {
	c, err := s.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if in.CompanyName != "" {
		c.CompanyName = in.CompanyName
	}
	if in.Website != "" {
		c.Website = &in.Website
	}
	if in.Description != "" {
		c.Description = &in.Description
	}
	if in.LogoURL != "" {
		c.LogoURL = &in.LogoURL
	}
	if in.Industry != "" {
		c.Industry = &in.Industry
	}
	if in.CompanySize != "" {
		c.CompanySize = &in.CompanySize
	}

	if err := s.companies.Update(ctx, c); err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	return c, nil
}
