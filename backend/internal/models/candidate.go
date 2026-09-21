package models

import "time"

type Candidate struct {
	ID                int64     `json:"id" db:"id"`
	UserID            int64     `json:"user_id" db:"user_id"`
	FullName          string    `json:"full_name" db:"full_name"`
	Phone             *string   `json:"phone" db:"phone"`
	Location          *string   `json:"location" db:"location"`
	Links             JSONMap   `json:"links" db:"links"`
	Headline          *string   `json:"headline" db:"headline"`
	Summary           *string   `json:"summary" db:"summary"`
	ActiveResumeID    *int64    `json:"active_resume_id" db:"active_resume_id"`
	ProfileCompletion uint8     `json:"profile_completion" db:"profile_completion"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
	Email             string    `json:"email,omitempty" db:"-"` // populated by joins, not a real column
}

type ResumeStatus string

const (
	ResumeStatusUploaded   ResumeStatus = "uploaded"
	ResumeStatusProcessing ResumeStatus = "processing"
	ResumeStatusParsed     ResumeStatus = "parsed"
	ResumeStatusFailed     ResumeStatus = "failed"
)

type Resume struct {
	ID               int64        `json:"id" db:"id"`
	CandidateID      int64        `json:"candidate_id" db:"candidate_id"`
	OriginalFilename string       `json:"original_filename" db:"original_filename"`
	StoredFilename   string       `json:"stored_filename" db:"stored_filename"`
	FilePath         string       `json:"-" db:"file_path"` // internal disk path, never exposed
	FileSize         uint32       `json:"file_size" db:"file_size"`
	MimeType         string       `json:"mime_type" db:"mime_type"`
	Status           ResumeStatus `json:"status" db:"status"`
	RawText          *string      `json:"-" db:"raw_text"` // large, never sent to clients directly
	ParseError       *string      `json:"parse_error,omitempty" db:"parse_error"`
	ParseAttempts    uint8        `json:"parse_attempts" db:"parse_attempts"`
	UploadedAt       time.Time    `json:"uploaded_at" db:"uploaded_at"`
	ParsedAt         *time.Time   `json:"parsed_at,omitempty" db:"parsed_at"`
}

type ResumeEducation struct {
	ID          int64   `json:"id" db:"id"`
	ResumeID    int64   `json:"resume_id" db:"resume_id"`
	Institution string  `json:"institution" db:"institution"`
	Degree      *string `json:"degree" db:"degree"`
	Field       *string `json:"field" db:"field"`
	StartDate   *string `json:"start_date" db:"start_date"` // YYYY-MM-DD or partial, kept as string for flexibility
	EndDate     *string `json:"end_date" db:"end_date"`
	GPA         *string `json:"gpa" db:"gpa"`
	SortOrder   uint16  `json:"sort_order" db:"sort_order"`
}

type ResumeExperience struct {
	ID               int64    `json:"id" db:"id"`
	ResumeID         int64    `json:"resume_id" db:"resume_id"`
	Company          string   `json:"company" db:"company"`
	Role             string   `json:"role" db:"role"`
	StartDate        *string  `json:"start_date" db:"start_date"`
	EndDate          *string  `json:"end_date" db:"end_date"`
	IsCurrent        bool     `json:"is_current" db:"is_current"`
	Description      *string  `json:"description" db:"description"`
	Technologies     JSONList `json:"technologies" db:"technologies"`
	Responsibilities JSONList `json:"responsibilities" db:"responsibilities"`
	SortOrder        uint16   `json:"sort_order" db:"sort_order"`
}

type ResumeProject struct {
	ID           int64    `json:"id" db:"id"`
	ResumeID     int64    `json:"resume_id" db:"resume_id"`
	Name         string   `json:"name" db:"name"`
	Description  *string  `json:"description" db:"description"`
	Technologies JSONList `json:"technologies" db:"technologies"`
	Role         *string  `json:"role" db:"role"`
	Links        JSONMap  `json:"links" db:"links"`
	SortOrder    uint16   `json:"sort_order" db:"sort_order"`
}

type SkillCategory string

const (
	SkillTechnical       SkillCategory = "technical"
	SkillProgrammingLang SkillCategory = "programming_language"
	SkillFramework       SkillCategory = "framework"
	SkillDatabase        SkillCategory = "database"
	SkillCloud           SkillCategory = "cloud"
	SkillTool            SkillCategory = "tool"
	SkillSoft            SkillCategory = "soft_skill"
)

type ResumeSkill struct {
	ID        int64         `json:"id" db:"id"`
	ResumeID  int64         `json:"resume_id" db:"resume_id"`
	SkillName string        `json:"skill_name" db:"skill_name"`
	SkillSlug string        `json:"skill_slug" db:"skill_slug"`
	Category  SkillCategory `json:"category" db:"category"`
}

type ResumeCertification struct {
	ID           int64   `json:"id" db:"id"`
	ResumeID     int64   `json:"resume_id" db:"resume_id"`
	Name         string  `json:"name" db:"name"`
	NameSlug     string  `json:"name_slug" db:"name_slug"`
	Issuer       *string `json:"issuer" db:"issuer"`
	IssueDate    *string `json:"issue_date" db:"issue_date"`
	CredentialID *string `json:"credential_id" db:"credential_id"`
}

// ResumeFull aggregates a resume with all its structured child records --
// this is what gets returned to the candidate for review/edit and what the
// matching engine reads from.
type ResumeFull struct {
	Resume
	Education      []ResumeEducation     `json:"education"`
	Experience     []ResumeExperience    `json:"experience"`
	Projects       []ResumeProject       `json:"projects"`
	Skills         []ResumeSkill         `json:"skills"`
	Certifications []ResumeCertification `json:"certifications"`
}
