package ai

import "testing"

func TestValidateResumeExtraction_RejectsMissingName(t *testing.T) {
	r := &ResumeExtraction{}
	if err := ValidateResumeExtraction(r); err == nil {
		t.Error("expected validation error for missing full_name")
	}
}

func TestValidateResumeExtraction_RejectsInvalidSkillCategory(t *testing.T) {
	r := &ResumeExtraction{
		PersonalInfo: ParsedPersonalInfo{FullName: "Jane Doe"},
		Skills:       []ParsedSkill{{Name: "Go", Category: "not_a_real_category"}},
	}
	if err := ValidateResumeExtraction(r); err == nil {
		t.Error("expected validation error for invalid skill category")
	}
}

func TestValidateResumeExtraction_AcceptsValidData(t *testing.T) {
	r := &ResumeExtraction{
		PersonalInfo: ParsedPersonalInfo{FullName: "Jane Doe", Email: "jane@example.com"},
		Skills:       []ParsedSkill{{Name: "Go", Category: "programming_language"}},
		Education:    []ParsedEducation{{Institution: "State University"}},
		Experience:   []ParsedExperience{{Company: "Acme", Role: "Engineer"}},
		Projects:     []ParsedProject{{Name: "Cool Project"}},
	}
	if err := ValidateResumeExtraction(r); err != nil {
		t.Errorf("expected valid data to pass, got: %v", err)
	}
}

func TestValidateResumeExtraction_RejectsExperienceMissingRole(t *testing.T) {
	r := &ResumeExtraction{
		PersonalInfo: ParsedPersonalInfo{FullName: "Jane Doe"},
		Experience:   []ParsedExperience{{Company: "Acme", Role: ""}},
	}
	if err := ValidateResumeExtraction(r); err == nil {
		t.Error("expected validation error for experience entry missing role")
	}
}

// --- interview question grounding: the core "never ask about a skill the
// candidate doesn't have and the job doesn't require" enforcement point ---

func TestValidateGeneratedQuestions_AcceptsGroundedQuestion(t *testing.T) {
	allowed := map[string]bool{"go": true, "mysql": true}
	questions := []GeneratedQuestion{
		{QuestionText: "Tell me about Go.", BasedOn: "job_skill:Go"},
	}
	valid, rejected := ValidateGeneratedQuestions(questions, allowed)
	if len(valid) != 1 {
		t.Errorf("expected 1 valid question, got %d (rejected: %v)", len(valid), rejected)
	}
}

func TestValidateGeneratedQuestions_RejectsUngroundedQuestion(t *testing.T) {
	// allowed context does NOT include "react" -- simulates the AI
	// hallucinating a question about a skill neither the job nor the
	// candidate ever mentioned.
	allowed := map[string]bool{"go": true, "mysql": true}
	questions := []GeneratedQuestion{
		{QuestionText: "Tell me about React hooks.", BasedOn: "job_skill:React"},
	}
	valid, rejected := ValidateGeneratedQuestions(questions, allowed)
	if len(valid) != 0 {
		t.Errorf("expected 0 valid questions for an ungrounded skill, got %d", len(valid))
	}
	if len(rejected) != 1 {
		t.Errorf("expected 1 rejected question logged, got %d", len(rejected))
	}
}

func TestValidateGeneratedQuestions_MixedBatch(t *testing.T) {
	allowed := map[string]bool{"go": true, "docker": true}
	questions := []GeneratedQuestion{
		{QuestionText: "Tell me about Go.", BasedOn: "job_skill:Go"},
		{QuestionText: "Tell me about Kubernetes.", BasedOn: "job_skill:Kubernetes"}, // not in allowed set
		{QuestionText: "Tell me about Docker.", BasedOn: "job_skill:Docker"},
	}
	valid, rejected := ValidateGeneratedQuestions(questions, allowed)
	if len(valid) != 2 {
		t.Errorf("expected 2 valid grounded questions, got %d", len(valid))
	}
	if len(rejected) != 1 {
		t.Errorf("expected exactly 1 rejected ungrounded question, got %d", len(rejected))
	}
}

func TestValidateGeneratedQuestions_RejectsEmptyQuestionText(t *testing.T) {
	allowed := map[string]bool{"go": true}
	questions := []GeneratedQuestion{{QuestionText: "  ", BasedOn: "job_skill:Go"}}
	valid, rejected := ValidateGeneratedQuestions(questions, allowed)
	if len(valid) != 0 || len(rejected) != 1 {
		t.Errorf("expected empty question text to be rejected, got valid=%d rejected=%d", len(valid), len(rejected))
	}
}

// --- answer evaluation score clamping ---

func TestValidateAnswerEvaluation_RejectsOutOfRangeScore(t *testing.T) {
	e := &AnswerEvaluation{TechnicalAccuracy: 150, ProblemSolving: 50, DepthOfKnowledge: 50, Communication: 50, Relevance: 50}
	if err := ValidateAnswerEvaluation(e); err == nil {
		t.Error("expected validation error for score > 100")
	}
}

func TestValidateAnswerEvaluation_RejectsNegativeScore(t *testing.T) {
	e := &AnswerEvaluation{TechnicalAccuracy: -5, ProblemSolving: 50, DepthOfKnowledge: 50, Communication: 50, Relevance: 50}
	if err := ValidateAnswerEvaluation(e); err == nil {
		t.Error("expected validation error for negative score")
	}
}

func TestValidateAnswerEvaluation_AcceptsValidScores(t *testing.T) {
	e := &AnswerEvaluation{TechnicalAccuracy: 80, ProblemSolving: 75, DepthOfKnowledge: 60, Communication: 90, Relevance: 85}
	if err := ValidateAnswerEvaluation(e); err != nil {
		t.Errorf("expected valid scores to pass, got: %v", err)
	}
}
