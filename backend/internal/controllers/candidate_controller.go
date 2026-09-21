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

type CandidateController struct {
	candidates *services.CandidateService
	resumes    *services.ResumeService
}

func NewCandidateController(candidates *services.CandidateService, resumes *services.ResumeService) *CandidateController {
	return &CandidateController{candidates: candidates, resumes: resumes}
}

func (cc *CandidateController) Me(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		middleware.AbortUnauthenticated(c)
		return
	}
	candidate, err := cc.candidates.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		c.Error(err)
		return
	}
	utils.Success(c, http.StatusOK, candidate, "")
}

func (cc *CandidateController) UpdateMe(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		middleware.AbortUnauthenticated(c)
		return
	}
	var req validators.UpdateCandidateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(utils.NewValidationError("invalid request body", err.Error()))
		return
	}
	candidate, err := cc.candidates.UpdateProfile(c.Request.Context(), userID, services.UpdateCandidateInput{
		FullName: req.FullName, Phone: req.Phone, Location: req.Location,
		Headline: req.Headline, Summary: req.Summary, Links: req.Links,
	})
	if err != nil {
		c.Error(err)
		return
	}
	utils.Success(c, http.StatusOK, candidate, "profile updated")
}

// UploadResume implements POST /candidates/me/resume: the full pipeline's
// upload step, followed synchronously by the parse step for this delivery
// (see ResumeService.ParseResume's doc comment on async-worker readiness).
func (cc *CandidateController) UploadResume(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		middleware.AbortUnauthenticated(c)
		return
	}
	candidate, err := cc.candidates.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		c.Error(err)
		return
	}

	fh, err := c.FormFile("resume")
	if err != nil {
		c.Error(utils.NewValidationError("resume file is required (multipart field name: \"resume\")", nil))
		return
	}

	resume, err := cc.resumes.Upload(c.Request.Context(), candidate.ID, fh)
	if err != nil {
		c.Error(err)
		return
	}

	full, err := cc.resumes.ParseResume(c.Request.Context(), resume.ID)
	if err != nil {
		// Upload succeeded even though parsing failed -- return 201 with the
		// resume in "failed" status rather than masking the successful
		// upload as a total failure. The candidate can retry parsing.
		utils.Success(c, http.StatusCreated, gin.H{
			"resume_id": resume.ID, "status": "failed", "parse_error": err.Error(),
		}, "resume uploaded but parsing failed")
		return
	}

	utils.Success(c, http.StatusCreated, full, "resume uploaded and parsed")
}

func (cc *CandidateController) GetResume(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		middleware.AbortUnauthenticated(c)
		return
	}
	candidate, err := cc.candidates.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		c.Error(err)
		return
	}
	if candidate.ActiveResumeID == nil {
		c.Error(utils.NewNotFound("resume"))
		return
	}
	full, err := cc.resumes.GetFull(c.Request.Context(), *candidate.ActiveResumeID)
	if err != nil {
		c.Error(err)
		return
	}
	utils.Success(c, http.StatusOK, full, "")
}

func (cc *CandidateController) ListResumes(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		middleware.AbortUnauthenticated(c)
		return
	}
	candidate, err := cc.candidates.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		c.Error(err)
		return
	}
	list, err := cc.resumes.ListByCandidate(c.Request.Context(), candidate.ID)
	if err != nil {
		c.Error(err)
		return
	}
	utils.Success(c, http.StatusOK, list, "")
}

// UpdateResumeProfile implements "allow the candidate to review and edit
// extracted information; edited data becomes authoritative". Ownership is
// enforced: a candidate may only edit their own active/owned resume.
func (cc *CandidateController) UpdateResumeProfile(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		middleware.AbortUnauthenticated(c)
		return
	}
	resumeIDStr := c.Param("id")
	resumeID, err := strconv.ParseInt(resumeIDStr, 10, 64)
	if err != nil {
		c.Error(utils.NewValidationError("invalid resume id", nil))
		return
	}

	candidate, err := cc.candidates.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		c.Error(err)
		return
	}

	existing, err := cc.resumes.GetFull(c.Request.Context(), resumeID)
	if err != nil {
		c.Error(err)
		return
	}
	if existing.CandidateID != candidate.ID {
		c.Error(utils.NewForbidden("you do not have access to this resume"))
		return
	}

	var req validators.UpdateResumeStructuredDataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(utils.NewValidationError("invalid request body", err.Error()))
		return
	}

	data := requestToResumeFull(req)
	updated, err := cc.resumes.UpdateStructuredData(c.Request.Context(), resumeID, data)
	if err != nil {
		c.Error(err)
		return
	}
	utils.Success(c, http.StatusOK, updated, "resume profile updated")
}

func requestToResumeFull(req validators.UpdateResumeStructuredDataRequest) *models.ResumeFull {
	full := &models.ResumeFull{}
	for _, e := range req.Education {
		full.Education = append(full.Education, models.ResumeEducation{
			Institution: e.Institution, Degree: ptrOrNil(e.Degree), Field: ptrOrNil(e.Field),
			StartDate: ptrOrNil(e.StartDate), EndDate: ptrOrNil(e.EndDate), GPA: ptrOrNil(e.GPA),
		})
	}
	for _, e := range req.Experience {
		full.Experience = append(full.Experience, models.ResumeExperience{
			Company: e.Company, Role: e.Role, StartDate: ptrOrNil(e.StartDate), EndDate: ptrOrNil(e.EndDate),
			IsCurrent: e.IsCurrent, Description: ptrOrNil(e.Description),
			Technologies: stringsToJSONList(e.Technologies), Responsibilities: stringsToJSONList(e.Responsibilities),
		})
	}
	for _, p := range req.Projects {
		links := models.JSONMap{}
		for i, l := range p.Links {
			links[strconv.Itoa(i+1)] = l
		}
		full.Projects = append(full.Projects, models.ResumeProject{
			Name: p.Name, Description: ptrOrNil(p.Description), Technologies: stringsToJSONList(p.Technologies),
			Role: ptrOrNil(p.Role), Links: links,
		})
	}
	for _, s := range req.Skills {
		full.Skills = append(full.Skills, models.ResumeSkill{
			SkillName: s.Name, SkillSlug: utils.CanonicalSkillSlug(s.Name), Category: models.SkillCategory(s.Category),
		})
	}
	for _, cert := range req.Certifications {
		full.Certifications = append(full.Certifications, models.ResumeCertification{
			Name: cert.Name, NameSlug: utils.Slugify(cert.Name), Issuer: ptrOrNil(cert.Issuer),
			IssueDate: ptrOrNil(cert.IssueDate), CredentialID: ptrOrNil(cert.CredentialID),
		})
	}
	return full
}

func ptrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func stringsToJSONList(items []string) models.JSONList {
	out := make(models.JSONList, len(items))
	for i, s := range items {
		out[i] = s
	}
	return out
}
