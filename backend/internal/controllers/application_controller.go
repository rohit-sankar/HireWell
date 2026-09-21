package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/resume-platform/backend/internal/middleware"
	"github.com/resume-platform/backend/internal/models"
	"github.com/resume-platform/backend/internal/services"
	"github.com/resume-platform/backend/internal/utils"
	"github.com/resume-platform/backend/internal/validators"
)

type ApplicationController struct {
	applications *services.ApplicationService
	matching     *services.MatchingService
	jobs         *services.JobService
}

func NewApplicationController(applications *services.ApplicationService, matching *services.MatchingService, jobs *services.JobService) *ApplicationController {
	return &ApplicationController{applications: applications, matching: matching, jobs: jobs}
}

// Apply implements POST /jobs/:id/apply.
func (ac *ApplicationController) Apply(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		middleware.AbortUnauthenticated(c)
		return
	}
	jobID, err := parseIDParam(c, "id")
	if err != nil {
		c.Error(err)
		return
	}
	app, err := ac.applications.Apply(c.Request.Context(), userID, jobID)
	if err != nil {
		c.Error(err)
		return
	}
	utils.Success(c, http.StatusCreated, app, "application submitted")
}

func (ac *ApplicationController) ListMine(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		middleware.AbortUnauthenticated(c)
		return
	}
	list, err := ac.applications.ListForCandidate(c.Request.Context(), userID)
	if err != nil {
		c.Error(err)
		return
	}
	utils.Success(c, http.StatusOK, list, "")
}

func (ac *ApplicationController) ListForJob(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		middleware.AbortUnauthenticated(c)
		return
	}
	jobID, err := parseIDParam(c, "id")
	if err != nil {
		c.Error(err)
		return
	}
	list, err := ac.applications.ListForJob(c.Request.Context(), userID, jobID)
	if err != nil {
		c.Error(err)
		return
	}
	utils.Success(c, http.StatusOK, list, "")
}

func (ac *ApplicationController) Get(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		middleware.AbortUnauthenticated(c)
		return
	}
	role, _ := middleware.RoleFromContext(c)
	appID, err := parseIDParam(c, "id")
	if err != nil {
		c.Error(err)
		return
	}
	app, err := ac.applications.AssertAccess(c.Request.Context(), userID, role, appID)
	if err != nil {
		c.Error(err)
		return
	}
	utils.Success(c, http.StatusOK, app, "")
}

// UpdateStatus implements HR shortlist/reject/under_review/hired actions.
func (ac *ApplicationController) UpdateStatus(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		middleware.AbortUnauthenticated(c)
		return
	}
	appID, err := parseIDParam(c, "id")
	if err != nil {
		c.Error(err)
		return
	}
	var req validators.UpdateApplicationStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(utils.NewValidationError("invalid request body", err.Error()))
		return
	}
	app, err := ac.applications.UpdateStatus(c.Request.Context(), userID, appID, models.ApplicationStatus(req.Status))
	if err != nil {
		c.Error(err)
		return
	}
	utils.Success(c, http.StatusOK, app, "application status updated")
}

// GetMatch implements GET /applications/:id/match. Access is restricted to
// the candidate who owns the application or the company that owns the job
// -- see ApplicationService.AssertAccess.
func (ac *ApplicationController) GetMatch(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		middleware.AbortUnauthenticated(c)
		return
	}
	role, _ := middleware.RoleFromContext(c)
	appID, err := parseIDParam(c, "id")
	if err != nil {
		c.Error(err)
		return
	}
	if _, err := ac.applications.AssertAccess(c.Request.Context(), userID, role, appID); err != nil {
		c.Error(err)
		return
	}
	match, err := ac.matching.GetForApplication(c.Request.Context(), appID)
	if err != nil {
		c.Error(err)
		return
	}
	utils.Success(c, http.StatusOK, match, "")
}

// RecalculateMatch lets a company force a recalculation (e.g. after
// adjusting match weights) without requiring a brand new application.
// Restricted to the company that owns the job (candidates cannot trigger
// arbitrary recalculation of their own score).
func (ac *ApplicationController) RecalculateMatch(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		middleware.AbortUnauthenticated(c)
		return
	}
	role, _ := middleware.RoleFromContext(c)
	appID, err := parseIDParam(c, "id")
	if err != nil {
		c.Error(err)
		return
	}
	if role != models.RoleCompany {
		c.Error(utils.NewForbidden("only company accounts can recalculate a match"))
		return
	}
	if _, err := ac.applications.AssertAccess(c.Request.Context(), userID, role, appID); err != nil {
		c.Error(err)
		return
	}
	match, err := ac.matching.CalculateForApplication(c.Request.Context(), appID)
	if err != nil {
		c.Error(err)
		return
	}
	utils.Success(c, http.StatusOK, match, "match recalculated")
}

// GetJobMatches implements GET /jobs/:id/matches -- ranked candidates for a
// job, the "companies/HR receive ranked/recommended candidates" feature.
// Restricted to the company that owns the job.
func (ac *ApplicationController) GetJobMatches(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		middleware.AbortUnauthenticated(c)
		return
	}
	jobID, err := parseIDParam(c, "id")
	if err != nil {
		c.Error(err)
		return
	}
	if _, err := ac.jobs.AssertCompanyOwnsJob(c.Request.Context(), userID, jobID); err != nil {
		c.Error(err)
		return
	}
	ranked, err := ac.matching.RankedForJob(c.Request.Context(), jobID)
	if err != nil {
		c.Error(err)
		return
	}
	utils.Success(c, http.StatusOK, ranked, "")
}
