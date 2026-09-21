package validators

type UpdateCandidateProfileRequest struct {
	FullName string                 `json:"full_name" binding:"omitempty,max=255"`
	Phone    string                 `json:"phone" binding:"omitempty,max=50"`
	Location string                 `json:"location" binding:"omitempty,max=255"`
	Headline string                 `json:"headline" binding:"omitempty,max=255"`
	Summary  string                 `json:"summary" binding:"omitempty,max=2000"`
	Links    map[string]interface{} `json:"links"`
}

type UpdateCompanyProfileRequest struct {
	CompanyName string `json:"company_name" binding:"omitempty,max=255"`
	Website     string `json:"website" binding:"omitempty,max=255"`
	Description string `json:"description" binding:"omitempty,max=5000"`
	LogoURL     string `json:"logo_url" binding:"omitempty,max=500"`
	Industry    string `json:"industry" binding:"omitempty,max=150"`
	CompanySize string `json:"company_size" binding:"omitempty,max=50"`
}

// ResumeEducationRequest / etc. mirror models.ResumeFull's shape for the
// candidate-edit endpoint (PUT /candidates/me/resume/:id) -- the candidate
// can overwrite any AI-derived field, becoming the new authoritative data
// per the spec.
type ResumeEducationRequest struct {
	Institution string `json:"institution" binding:"required"`
	Degree      string `json:"degree"`
	Field       string `json:"field"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
	GPA         string `json:"gpa"`
}

type ResumeExperienceRequest struct {
	Company          string   `json:"company" binding:"required"`
	Role             string   `json:"role" binding:"required"`
	StartDate        string   `json:"start_date"`
	EndDate          string   `json:"end_date"`
	IsCurrent        bool     `json:"is_current"`
	Description      string   `json:"description"`
	Technologies     []string `json:"technologies"`
	Responsibilities []string `json:"responsibilities"`
}

type ResumeProjectRequest struct {
	Name         string   `json:"name" binding:"required"`
	Description  string   `json:"description"`
	Technologies []string `json:"technologies"`
	Role         string   `json:"role"`
	Links        []string `json:"links"`
}

type ResumeSkillRequest struct {
	Name     string `json:"name" binding:"required"`
	Category string `json:"category" binding:"required,oneof=technical programming_language framework database cloud tool soft_skill"`
}

type ResumeCertificationRequest struct {
	Name         string `json:"name" binding:"required"`
	Issuer       string `json:"issuer"`
	IssueDate    string `json:"issue_date"`
	CredentialID string `json:"credential_id"`
}

type UpdateResumeStructuredDataRequest struct {
	Education      []ResumeEducationRequest     `json:"education"`
	Experience     []ResumeExperienceRequest    `json:"experience"`
	Projects       []ResumeProjectRequest       `json:"projects"`
	Skills         []ResumeSkillRequest         `json:"skills" binding:"required,dive"`
	Certifications []ResumeCertificationRequest `json:"certifications"`
}
