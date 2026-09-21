// Package ai defines a provider-agnostic interface for every AI operation
// the platform needs (resume parsing, interview question generation,
// interview answer evaluation, candidate summaries, semantic skill
// matching). Controllers and services NEVER call a specific AI vendor's
// SDK/API directly -- they depend only on the Provider interface below.
// Swapping providers (Gemini -> OpenAI -> Anthropic -> local model) means
// writing one new file that implements this interface and changing the
// AI_PROVIDER env var; no other code changes.
//
// All operations return structured, schema-validated data -- never raw
// free-text dumped into the database. See ValidateResumeExtraction and the
// analogous validators below for the "never blindly trust AI output" rule.
package ai

import (
	"context"
	"errors"
)

var (
	ErrProviderTimeout     = errors.New("ai provider: request timed out")
	ErrProviderRateLimit   = errors.New("ai provider: rate limited")
	ErrProviderBadResponse = errors.New("ai provider: malformed or invalid response")
	ErrProviderUnavailable = errors.New("ai provider: unavailable")
)

// --- ParseResume ----------------------------------------------------------

type ParsedPersonalInfo struct {
	FullName string   `json:"full_name"`
	Email    string   `json:"email"`
	Phone    string   `json:"phone"`
	Location string   `json:"location"`
	Links    []string `json:"links"`
}

type ParsedEducation struct {
	Institution string `json:"institution"`
	Degree      string `json:"degree"`
	Field       string `json:"field"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
	GPA         string `json:"gpa"`
}

type ParsedExperience struct {
	Company          string   `json:"company"`
	Role             string   `json:"role"`
	StartDate        string   `json:"start_date"`
	EndDate          string   `json:"end_date"`
	IsCurrent        bool     `json:"is_current"`
	Description      string   `json:"description"`
	Technologies     []string `json:"technologies"`
	Responsibilities []string `json:"responsibilities"`
}

type ParsedProject struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Technologies []string `json:"technologies"`
	Role         string   `json:"role"`
	Links        []string `json:"links"`
}

type ParsedSkill struct {
	Name     string `json:"name"`
	Category string `json:"category"` // technical|programming_language|framework|database|cloud|tool|soft_skill
}

type ParsedCertification struct {
	Name         string `json:"name"`
	Issuer       string `json:"issuer"`
	IssueDate    string `json:"issue_date"`
	CredentialID string `json:"credential_id"`
}

// ResumeExtraction is the strict JSON schema every provider implementation
// must fill in and every caller must validate before persisting.
type ResumeExtraction struct {
	PersonalInfo   ParsedPersonalInfo    `json:"personal_info"`
	Education      []ParsedEducation     `json:"education"`
	Experience     []ParsedExperience    `json:"experience"`
	Projects       []ParsedProject       `json:"projects"`
	Skills         []ParsedSkill         `json:"skills"`
	Certifications []ParsedCertification `json:"certifications"`
}

// --- Interview questions ---------------------------------------------------

// InterviewContext is the ONLY information the question generator is given.
// This is what enforces "never ask about a skill the candidate doesn't have
// and the job doesn't require": the provider physically cannot see anything
// outside these lists, so it cannot hallucinate a question about, say,
// React if React appears in neither RequiredSkills/PreferredSkills nor
// CandidateSkills.
type InterviewContext struct {
	JobTitle            string
	RequiredSkills      []string
	PreferredSkills     []string
	CandidateSkills     []string
	CandidateProjects   []ParsedProject
	CandidateExperience []ParsedExperience
	QuestionCount       int
}

type GeneratedQuestion struct {
	QuestionText string `json:"question_text"`
	BasedOn      string `json:"based_on"` // must reference a skill/project/experience that was actually in context
}

// --- Interview answer evaluation -------------------------------------------

type AnswerEvaluationInput struct {
	Question string
	Answer   string
	BasedOn  string
}

type AnswerEvaluation struct {
	TechnicalAccuracy float64  `json:"technical_accuracy"` // 0-100
	ProblemSolving    float64  `json:"problem_solving"`
	DepthOfKnowledge  float64  `json:"depth_of_knowledge"`
	Communication     float64  `json:"communication"`
	Relevance         float64  `json:"relevance"`
	Strengths         []string `json:"strengths"`
	Weaknesses        []string `json:"weaknesses"`
	Recommendation    string   `json:"recommendation"`
}

// --- Semantic matching (assists, never replaces, deterministic scoring) ---

type SemanticMatchInput struct {
	JobSkills       []string
	CandidateSkills []string
}

type SemanticMatchResult struct {
	// AdditionalMatches lists candidate skills the deterministic slug-based
	// matcher (internal/matching) missed but that are semantically
	// equivalent to a required job skill (e.g. candidate lists "Golang",
	// job requires "Go" -- though the deterministic alias table already
	// catches that specific case; this exists for looser equivalences like
	// "Postgres administration" ~ "SQL databases").
	AdditionalMatches []SemanticMatchPair `json:"additional_matches"`
}

type SemanticMatchPair struct {
	JobSkill       string  `json:"job_skill"`
	CandidateSkill string  `json:"candidate_skill"`
	Confidence     float64 `json:"confidence"` // 0-1
}

// --- Candidate summary ------------------------------------------------------

type CandidateSummaryInput struct {
	ResumeExtraction ResumeExtraction
}

// --- Provider interface -----------------------------------------------------

type Provider interface {
	// ParseResume extracts structured data from raw resume text. Implementations
	// MUST return data that passes ValidateResumeExtraction, or an error --
	// never partially-invalid data silently.
	ParseResume(ctx context.Context, rawText string) (*ResumeExtraction, error)

	// GenerateInterviewQuestions produces up to ic.QuestionCount questions,
	// each grounded in ic's skills/projects/experience only.
	GenerateInterviewQuestions(ctx context.Context, ic InterviewContext) ([]GeneratedQuestion, error)

	// EvaluateInterviewAnswer scores a single answer against structured rubric
	// dimensions (see AnswerEvaluation).
	EvaluateInterviewAnswer(ctx context.Context, in AnswerEvaluationInput) (*AnswerEvaluation, error)

	// GenerateCandidateSummary produces a short natural-language profile
	// summary strictly derived from the candidate's own extracted resume data.
	GenerateCandidateSummary(ctx context.Context, in CandidateSummaryInput) (string, error)

	// SemanticMatch assists (does not replace) the deterministic matching
	// engine by finding skill-name equivalences the exact/alias matcher missed.
	SemanticMatch(ctx context.Context, in SemanticMatchInput) (*SemanticMatchResult, error)

	// Name identifies the active provider, e.g. "gemini" or "stub" -- used in
	// logs/health checks so it's always clear which provider produced a result.
	Name() string
}
