package models

import "time"

type ApplicationStatus string

const (
	AppStatusApplied     ApplicationStatus = "applied"
	AppStatusUnderReview ApplicationStatus = "under_review"
	AppStatusShortlisted ApplicationStatus = "shortlisted"
	AppStatusRejected    ApplicationStatus = "rejected"
	AppStatusHired       ApplicationStatus = "hired"
)

type Application struct {
	ID          int64             `json:"id" db:"id"`
	JobID       int64             `json:"job_id" db:"job_id"`
	CandidateID int64             `json:"candidate_id" db:"candidate_id"`
	ResumeID    int64             `json:"resume_id" db:"resume_id"`
	Status      ApplicationStatus `json:"status" db:"status"`
	AppliedAt   time.Time         `json:"applied_at" db:"applied_at"`
	UpdatedAt   time.Time         `json:"updated_at" db:"updated_at"`

	// populated by joins for list/detail views, not real columns:
	JobTitle      string `json:"job_title,omitempty" db:"-"`
	CompanyName   string `json:"company_name,omitempty" db:"-"`
	CandidateName string `json:"candidate_name,omitempty" db:"-"`
}

// MatchResult is the explainable, reproducible output of the deterministic
// matching engine (internal/matching). See that package's doc comment for
// the exact scoring formula.
type MatchResult struct {
	ID                 int64     `json:"id" db:"id"`
	ApplicationID      int64     `json:"application_id" db:"application_id"`
	SkillScore         float64   `json:"skill_score" db:"skill_score"`
	ExperienceScore    float64   `json:"experience_score" db:"experience_score"`
	EducationScore     float64   `json:"education_score" db:"education_score"`
	ProjectScore       float64   `json:"project_score" db:"project_score"`
	CertificationScore float64   `json:"certification_score" db:"certification_score"`
	OverallScore       float64   `json:"overall_score" db:"overall_score"`
	WeightsUsed        JSONMap   `json:"weights_used" db:"weights_used"`
	Explanation        JSONMap   `json:"explanation" db:"explanation"`
	CalculatedAt       time.Time `json:"calculated_at" db:"calculated_at"`
}
