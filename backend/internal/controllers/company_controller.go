package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/resume-platform/backend/internal/middleware"
	"github.com/resume-platform/backend/internal/services"
	"github.com/resume-platform/backend/internal/utils"
	"github.com/resume-platform/backend/internal/validators"
)

type CompanyController struct {
	companies *services.CompanyService
}

func NewCompanyController(companies *services.CompanyService) *CompanyController {
	return &CompanyController{companies: companies}
}

func (cc *CompanyController) Me(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		middleware.AbortUnauthenticated(c)
		return
	}
	company, err := cc.companies.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		c.Error(err)
		return
	}
	utils.Success(c, http.StatusOK, company, "")
}

func (cc *CompanyController) UpdateMe(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		middleware.AbortUnauthenticated(c)
		return
	}
	var req validators.UpdateCompanyProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(utils.NewValidationError("invalid request body", err.Error()))
		return
	}
	company, err := cc.companies.UpdateProfile(c.Request.Context(), userID, services.UpdateCompanyInput{
		CompanyName: req.CompanyName, Website: req.Website, Description: req.Description,
		LogoURL: req.LogoURL, Industry: req.Industry, CompanySize: req.CompanySize,
	})
	if err != nil {
		c.Error(err)
		return
	}
	utils.Success(c, http.StatusOK, company, "profile updated")
}
