package models

import "time"

type Company struct {
	ID          int64     `json:"id" db:"id"`
	UserID      int64     `json:"user_id" db:"user_id"`
	CompanyName string    `json:"company_name" db:"company_name"`
	Website     *string   `json:"website" db:"website"`
	Description *string   `json:"description" db:"description"`
	LogoURL     *string   `json:"logo_url" db:"logo_url"`
	Industry    *string   `json:"industry" db:"industry"`
	CompanySize *string   `json:"company_size" db:"company_size"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
	Email       string    `json:"email,omitempty" db:"-"`
}

type EmploymentType string

const (
	EmploymentFullTime   EmploymentType = "full_time"
	EmploymentPartTime   EmploymentType = "part_time"
	EmploymentContract   EmploymentType = "contract"
	EmploymentInternship EmploymentType = "internship"
	EmploymentRemote     EmploymentType = "remote"
)

type JobStatus string

const (
	JobStatusDraft     JobStatus = "draft"
	JobStatusPublished JobStatus = "published"
	JobStatusClosed    JobStatus = "closed"
	JobStatusArchived  JobStatus = "archived"
)

type Job struct {
	ID                     int64          `json:"id" db:"id"`
	CompanyID              int64          `json:"company_id" db:"company_id"`
	Title                  string         `json:"title" db:"title"`
	Description            string         `json:"description" db:"description"`
	Location               *string        `json:"location" db:"location"`
	EmploymentType         EmploymentType `json:"employment_type" db:"employment_type"`
	ExperienceMinYears     uint8          `json:"experience_min_years" db:"experience_min_years"`
	ExperienceMaxYears     *uint8         `json:"experience_max_years" db:"experience_max_years"`
	EducationRequirement   *string        `json:"education_requirement" db:"education_requirement"`
	Status                 JobStatus      `json:"status" db:"status"`
	InterviewQuestionCount uint8          `json:"interview_question_count" db:"interview_question_count"`
	CreatedAt              time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time      `json:"updated_at" db:"updated_at"`
	PublishedAt            *time.Time     `json:"published_at" db:"published_at"`
	ClosedAt               *time.Time     `json:"closed_at" db:"closed_at"`
	CompanyName            string         `json:"company_name,omitempty" db:"-"`
}

type JobSkill struct {
	ID         int64   `json:"id" db:"id"`
	JobID      int64   `json:"job_id" db:"job_id"`
	SkillName  string  `json:"skill_name" db:"skill_name"`
	SkillSlug  string  `json:"skill_slug" db:"skill_slug"`
	IsRequired bool    `json:"is_required" db:"is_required"`
	Weight     float64 `json:"weight" db:"weight"`
}

type JobCertification struct {
	ID                int64  `json:"id" db:"id"`
	JobID             int64  `json:"job_id" db:"job_id"`
	CertificationName string `json:"certification_name" db:"certification_name"`
	CertificationSlug string `json:"certification_slug" db:"certification_slug"`
}

// MatchWeights are the configurable weights documented in the matching
// engine (internal/matching). They must sum to ~1.0; the service validates
// this on save so scores stay interpretable.
type MatchWeights struct {
	ID                  int64   `json:"id" db:"id"`
	JobID               int64   `json:"job_id" db:"job_id"`
	WeightSkill         float64 `json:"weight_skill" db:"weight_skill"`
	WeightExperience    float64 `json:"weight_experience" db:"weight_experience"`
	WeightEducation     float64 `json:"weight_education" db:"weight_education"`
	WeightProject       float64 `json:"weight_project" db:"weight_project"`
	WeightCertification float64 `json:"weight_certification" db:"weight_certification"`
}

// JobFull aggregates a job with its required/preferred skills and
// certifications for the job-detail and matching-engine views.
type JobFull struct {
	Job
	Skills         []JobSkill         `json:"skills"`
	Certifications []JobCertification `json:"certifications"`
	MatchWeights   MatchWeights       `json:"match_weights"`
}
