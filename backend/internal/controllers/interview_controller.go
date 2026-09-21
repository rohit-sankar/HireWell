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

type InterviewController struct {
	interviews   *services.InterviewService
	applications *services.ApplicationService
}

func NewInterviewController(interviews *services.InterviewService, applications *services.ApplicationService) *InterviewController {
	return &InterviewController{interviews: interviews, applications: applications}
}

type createInterviewRequest struct {
	ApplicationID int64 `json:"application_id" binding:"required"`
}

// Create implements POST /interviews. Only the candidate who owns the
// application may start an interview for it.
func (ic *InterviewController) Create(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		middleware.AbortUnauthenticated(c)
		return
	}
	var req createInterviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(utils.NewValidationError("invalid request body", err.Error()))
		return
	}

	if _, err := ic.applications.AssertAccess(c.Request.Context(), userID, models.RoleCandidate, req.ApplicationID); err != nil {
		c.Error(err)
		return
	}

	interview, err := ic.interviews.Create(c.Request.Context(), userID, req.ApplicationID)
	if err != nil {
		c.Error(err)
		return
	}
	utils.Success(c, http.StatusCreated, interview, "interview created")
}

// authorizeInterviewAccess is the shared IDOR guard for every /interviews/:id
// sub-route: it loads the interview's parent application and re-uses the
// exact same role-aware access rule as application endpoints.
func (ic *InterviewController) authorizeInterviewAccess(c *gin.Context, interviewID int64) (*models.Interview, []models.InterviewQuestion, bool) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		middleware.AbortUnauthenticated(c)
		return nil, nil, false
	}
	role, _ := middleware.RoleFromContext(c)

	interview, questions, err := ic.interviews.GetByID(c.Request.Context(), interviewID)
	if err != nil {
		c.Error(err)
		return nil, nil, false
	}
	if _, err := ic.applications.AssertAccess(c.Request.Context(), userID, role, interview.ApplicationID); err != nil {
		c.Error(err)
		return nil, nil, false
	}
	return interview, questions, true
}

func (ic *InterviewController) Get(c *gin.Context) {
	interviewID, err := parseIDParam(c, "id")
	if err != nil {
		c.Error(err)
		return
	}
	interview, questions, ok := ic.authorizeInterviewAccess(c, interviewID)
	if !ok {
		return
	}
	utils.Success(c, http.StatusOK, gin.H{"interview": interview, "questions": questions}, "")
}

func (ic *InterviewController) Start(c *gin.Context) {
	interviewID, err := parseIDParam(c, "id")
	if err != nil {
		c.Error(err)
		return
	}
	if _, _, ok := ic.authorizeInterviewAccess(c, interviewID); !ok {
		return
	}
	interview, err := ic.interviews.Start(c.Request.Context(), interviewID)
	if err != nil {
		c.Error(err)
		return
	}
	utils.Success(c, http.StatusOK, interview, "interview started")
}

func (ic *InterviewController) Answer(c *gin.Context) {
	interviewID, err := parseIDParam(c, "id")
	if err != nil {
		c.Error(err)
		return
	}
	if _, _, ok := ic.authorizeInterviewAccess(c, interviewID); !ok {
		return
	}

	var req validators.SubmitAnswerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(utils.NewValidationError("invalid request body", err.Error()))
		return
	}
	if err := ic.interviews.SubmitAnswer(c.Request.Context(), req.QuestionID, req.Answer); err != nil {
		c.Error(err)
		return
	}
	utils.Success(c, http.StatusOK, nil, "answer submitted")
}

func (ic *InterviewController) Complete(c *gin.Context) {
	interviewID, err := parseIDParam(c, "id")
	if err != nil {
		c.Error(err)
		return
	}
	if _, _, ok := ic.authorizeInterviewAccess(c, interviewID); !ok {
		return
	}
	evaluation, err := ic.interviews.Complete(c.Request.Context(), interviewID)
	if err != nil {
		c.Error(err)
		return
	}
	utils.Success(c, http.StatusOK, evaluation, "interview completed and evaluated")
}

func (ic *InterviewController) Result(c *gin.Context) {
	interviewID, err := parseIDParam(c, "id")
	if err != nil {
		c.Error(err)
		return
	}
	if _, _, ok := ic.authorizeInterviewAccess(c, interviewID); !ok {
		return
	}
	evaluation, err := ic.interviews.GetEvaluation(c.Request.Context(), interviewID)
	if err != nil {
		c.Error(err)
		return
	}
	utils.Success(c, http.StatusOK, evaluation, "")
}
