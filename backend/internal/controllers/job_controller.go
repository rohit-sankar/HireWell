package controllers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/resume-platform/backend/internal/middleware"
	"github.com/resume-platform/backend/internal/models"
	"github.com/resume-platform/backend/internal/services"
	"github.com/resume-platform/backend/internal/utils"
	"github.com/resume-platform/backend/internal/validators"
)

type JobController struct {
	jobs *services.JobService
}

func NewJobController(jobs *services.JobService) *JobController {
	return &JobController{jobs: jobs}
}

func (jc *JobController) Create(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		middleware.AbortUnauthenticated(c)
		return
	}
	var req validators.CreateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(utils.NewValidationError("invalid request body", err.Error()))
		return
	}
	job, err := jc.jobs.Create(c.Request.Context(), userID, requestToCreateJobInput(req))
	if err != nil {
		c.Error(err)
		return
	}
	utils.Success(c, http.StatusCreated, job, "job created")
}

func (jc *JobController) Update(c *gin.Context) {
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
	var req validators.CreateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(utils.NewValidationError("invalid request body", err.Error()))
		return
	}
	job, err := jc.jobs.Update(c.Request.Context(), userID, jobID, requestToCreateJobInput(req))
	if err != nil {
		c.Error(err)
		return
	}
	utils.Success(c, http.StatusOK, job, "job updated")
}

func (jc *JobController) Get(c *gin.Context) {
	jobID, err := parseIDParam(c, "id")
	if err != nil {
		c.Error(err)
		return
	}
	job, err := jc.jobs.GetFull(c.Request.Context(), jobID)
	if err != nil {
		c.Error(err)
		return
	}
	utils.Success(c, http.StatusOK, job, "")
}

// List implements GET /jobs: public candidate-facing job search (published
// only, unless a company is querying their own jobs via company_id being
// implicitly scoped elsewhere) plus company's own job list via query param.
func (jc *JobController) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("page_size"))

	filter := services.ListJobsInput{Search: c.Query("search"), Page: page, PageSize: pageSize}

	// Public job search only ever shows published jobs unless the caller is
	// an authenticated company viewing their own listings (mine=true).
	if c.Query("mine") == "true" {
		userID, ok := middleware.UserIDFromContext(c)
		if !ok {
			middleware.AbortUnauthenticated(c)
			return
		}
		role, _ := middleware.RoleFromContext(c)
		if role != models.RoleCompany {
			c.Error(utils.NewForbidden("only company accounts can list their own jobs"))
			return
		}
		// resolve company id indirectly through the job service's ownership
		// helper is overkill here; ListJobsInput.CompanyID needs a real id,
		// so we fetch it via the company-scoped listing path in the service
		// layer instead: pass userID through a dedicated method.
		jobs, total, err := jc.jobs.ListMine(c.Request.Context(), userID, filter)
		if err != nil {
			c.Error(err)
			return
		}
		utils.SuccessWithMeta(c, http.StatusOK, jobs, "", gin.H{"total": total, "page": filter.Page})
		return
	}

	published := models.JobStatusPublished
	filter.Status = &published
	jobs, total, err := jc.jobs.List(c.Request.Context(), filter)
	if err != nil {
		c.Error(err)
		return
	}
	utils.SuccessWithMeta(c, http.StatusOK, jobs, "", gin.H{"total": total, "page": filter.Page})
}

func (jc *JobController) Delete(c *gin.Context) {
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
	if err := jc.jobs.Delete(c.Request.Context(), userID, jobID); err != nil {
		c.Error(err)
		return
	}
	utils.Success(c, http.StatusOK, nil, "job deleted")
}

func (jc *JobController) Publish(c *gin.Context) {
	jc.statusAction(c, func(ctx *gin.Context, userID, jobID int64) (interface{}, error) {
		return jc.jobs.Publish(ctx.Request.Context(), userID, jobID)
	})
}

func (jc *JobController) Close(c *gin.Context) {
	jc.statusAction(c, func(ctx *gin.Context, userID, jobID int64) (interface{}, error) {
		return jc.jobs.Close(ctx.Request.Context(), userID, jobID)
	})
}

func (jc *JobController) Archive(c *gin.Context) {
	jc.statusAction(c, func(ctx *gin.Context, userID, jobID int64) (interface{}, error) {
		return jc.jobs.Archive(ctx.Request.Context(), userID, jobID)
	})
}

func (jc *JobController) statusAction(c *gin.Context, action func(*gin.Context, int64, int64) (interface{}, error)) {
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
	result, err := action(c, userID, jobID)
	if err != nil {
		c.Error(err)
		return
	}
	utils.Success(c, http.StatusOK, result, "status updated")
}

func (jc *JobController) UpdateMatchWeights(c *gin.Context) {
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
	var req validators.UpdateMatchWeightsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(utils.NewValidationError("invalid request body", err.Error()))
		return
	}
	weights := models.MatchWeights{
		WeightSkill: req.WeightSkill, WeightExperience: req.WeightExperience, WeightEducation: req.WeightEducation,
		WeightProject: req.WeightProject, WeightCertification: req.WeightCertification,
	}
	if err := jc.jobs.UpdateMatchWeights(c.Request.Context(), userID, jobID, weights); err != nil {
		c.Error(err)
		return
	}
	utils.Success(c, http.StatusOK, weights, "match weights updated")
}

func requestToCreateJobInput(req validators.CreateJobRequest) services.CreateJobInput {
	skills := make([]services.JobSkillInput, 0, len(req.Skills))
	for _, s := range req.Skills {
		skills = append(skills, services.JobSkillInput{Name: s.Name, IsRequired: s.IsRequired, Weight: s.Weight})
	}
	return services.CreateJobInput{
		Title: req.Title, Description: req.Description, Location: req.Location,
		EmploymentType: models.EmploymentType(req.EmploymentType), ExperienceMinYears: req.ExperienceMinYears,
		ExperienceMaxYears: req.ExperienceMaxYears, EducationRequirement: req.EducationRequirement,
		InterviewQuestionCount: req.InterviewQuestionCount, Skills: skills, Certifications: req.Certifications,
	}
}

func parseIDParam(c *gin.Context, name string) (int64, error) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil {
		return 0, utils.NewValidationError("invalid "+name, nil)
	}
	return id, nil
}
