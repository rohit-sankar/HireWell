package ai

import (
	"fmt"
	"strings"
)

// ValidationError collects every problem found so callers can log/report
// everything wrong at once rather than failing on the first issue.
type ValidationError struct {
	Issues []string
}

func (e *ValidationError) Error() string {
	return "ai output failed validation: " + strings.Join(e.Issues, "; ")
}

// ValidateResumeExtraction enforces the "structured data, not raw AI text"
// requirement: every required field must be present and every enum-like
// field must be one of the allowed values. This is the boundary between
// "what the AI returned" and "what we're willing to write to the database".
func ValidateResumeExtraction(r *ResumeExtraction) error {
	var issues []string

	if strings.TrimSpace(r.PersonalInfo.FullName) == "" {
		issues = append(issues, "personal_info.full_name is required")
	}

	validCategories := map[string]bool{
		"technical": true, "programming_language": true, "framework": true,
		"database": true, "cloud": true, "tool": true, "soft_skill": true,
	}
	for i, s := range r.Skills {
		if strings.TrimSpace(s.Name) == "" {
			issues = append(issues, fmt.Sprintf("skills[%d].name is empty", i))
		}
		if !validCategories[s.Category] {
			issues = append(issues, fmt.Sprintf("skills[%d].category %q is not a recognized category", i, s.Category))
		}
	}

	for i, e := range r.Education {
		if strings.TrimSpace(e.Institution) == "" {
			issues = append(issues, fmt.Sprintf("education[%d].institution is empty", i))
		}
	}

	for i, e := range r.Experience {
		if strings.TrimSpace(e.Company) == "" || strings.TrimSpace(e.Role) == "" {
			issues = append(issues, fmt.Sprintf("experience[%d] missing company or role", i))
		}
	}

	for i, p := range r.Projects {
		if strings.TrimSpace(p.Name) == "" {
			issues = append(issues, fmt.Sprintf("projects[%d].name is empty", i))
		}
	}

	if len(issues) > 0 {
		return &ValidationError{Issues: issues}
	}
	return nil
}

// ValidateGeneratedQuestions is the enforcement point for "the AI interviewer
// must never ask about a skill/technology that isn't actually present in the
// job requirements or the candidate's profile". allowedTerms is the
// lowercased union of every skill/project/experience term the question
// generator was given as context; any question whose BasedOn tag doesn't
// reference one of those terms is rejected rather than silently kept.
func ValidateGeneratedQuestions(questions []GeneratedQuestion, allowedTerms map[string]bool) ([]GeneratedQuestion, []string) {
	var valid []GeneratedQuestion
	var rejected []string

	for _, q := range questions {
		if strings.TrimSpace(q.QuestionText) == "" {
			rejected = append(rejected, "empty question text")
			continue
		}
		basedOnLower := strings.ToLower(q.BasedOn)
		grounded := false
		for term := range allowedTerms {
			if term != "" && strings.Contains(basedOnLower, term) {
				grounded = true
				break
			}
		}
		if !grounded {
			rejected = append(rejected, fmt.Sprintf("question %q rejected: based_on %q is not grounded in job/candidate context", q.QuestionText, q.BasedOn))
			continue
		}
		valid = append(valid, q)
	}
	return valid, rejected
}

// ValidateAnswerEvaluation clamps/validates score fields into the documented
// 0-100 range so a malformed AI response can't corrupt downstream averages.
func ValidateAnswerEvaluation(e *AnswerEvaluation) error {
	fields := map[string]float64{
		"technical_accuracy": e.TechnicalAccuracy,
		"problem_solving":    e.ProblemSolving,
		"depth_of_knowledge": e.DepthOfKnowledge,
		"communication":      e.Communication,
		"relevance":          e.Relevance,
	}
	var issues []string
	for name, v := range fields {
		if v < 0 || v > 100 {
			issues = append(issues, fmt.Sprintf("%s = %v is outside 0-100", name, v))
		}
	}
	if len(issues) > 0 {
		return &ValidationError{Issues: issues}
	}
	return nil
}
