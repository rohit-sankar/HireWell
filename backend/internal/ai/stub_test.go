package ai

import (
	"context"
	"strings"
	"testing"
)

func TestStubProvider_ParseResume_ExtractsRealData(t *testing.T) {
	p := NewStubProvider()
	text := "John Doe\njohn.doe@example.com\n555-1234\nExperienced with Go, MySQL, Docker, and REST APIs."

	result, err := p.ParseResume(context.Background(), text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.PersonalInfo.Email != "john.doe@example.com" {
		t.Errorf("expected extracted email, got %q", result.PersonalInfo.Email)
	}
	if len(result.Skills) == 0 {
		t.Error("expected at least one skill extracted from text mentioning Go/MySQL/Docker")
	}

	foundGo := false
	for _, s := range result.Skills {
		if strings.EqualFold(s.Name, "go") {
			foundGo = true
		}
	}
	if !foundGo {
		t.Error("expected 'Go' to be recognized as a skill in the resume text")
	}
}

func TestStubProvider_ParseResume_DoesNotFabricateEducationOrExperience(t *testing.T) {
	p := NewStubProvider()
	text := "Just some random text with no structured resume sections at all."

	result, err := p.ParseResume(context.Background(), text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Education) != 0 {
		t.Error("stub provider must not fabricate education entries it cannot reliably extract")
	}
	if len(result.Experience) != 0 {
		t.Error("stub provider must not fabricate experience entries it cannot reliably extract")
	}
}

func TestStubProvider_GenerateInterviewQuestions_OnlyUsesGroundedSkills(t *testing.T) {
	p := NewStubProvider()
	ic := InterviewContext{
		JobTitle:        "Backend Developer",
		RequiredSkills:  []string{"Go", "React"}, // React required by job but candidate doesn't have it
		PreferredSkills: []string{"Docker"},
		CandidateSkills: []string{"Go", "MySQL"}, // candidate has Go and MySQL, not React or Docker
		QuestionCount:   10,
	}

	questions, err := p.GenerateInterviewQuestions(context.Background(), ic)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, q := range questions {
		if strings.Contains(strings.ToLower(q.QuestionText), "react") {
			t.Errorf("stub must never generate a question about a skill (React) the candidate doesn't have, even though the job requires it: %q", q.QuestionText)
		}
		if strings.Contains(strings.ToLower(q.QuestionText), "docker") {
			t.Errorf("stub must never generate a question about a skill (Docker) the candidate doesn't have: %q", q.QuestionText)
		}
	}

	// Should have generated exactly one skill-based question: Go (the only
	// skill in the intersection of required-and-candidate-has).
	foundGoQuestion := false
	for _, q := range questions {
		if q.BasedOn == "job_skill:Go" {
			foundGoQuestion = true
		}
	}
	if !foundGoQuestion {
		t.Error("expected a question grounded in 'Go', the one skill both required by the job and possessed by the candidate")
	}
}

func TestStubProvider_GenerateInterviewQuestions_GroundedInProjectsAndExperience(t *testing.T) {
	p := NewStubProvider()
	ic := InterviewContext{
		JobTitle:            "Backend Developer",
		CandidateProjects:   []ParsedProject{{Name: "Inventory System"}},
		CandidateExperience: []ParsedExperience{{Company: "Acme Corp", Role: "Engineer"}},
		QuestionCount:       5,
	}

	questions, err := p.GenerateInterviewQuestions(context.Background(), ic)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundProjectQ, foundExpQ := false, false
	for _, q := range questions {
		if strings.Contains(q.BasedOn, "Inventory System") {
			foundProjectQ = true
		}
		if strings.Contains(q.BasedOn, "Acme Corp") {
			foundExpQ = true
		}
	}
	if !foundProjectQ {
		t.Error("expected a question grounded in the candidate's own project")
	}
	if !foundExpQ {
		t.Error("expected a question grounded in the candidate's own experience")
	}
}

func TestStubProvider_EvaluateInterviewAnswer_ReturnsValidRangeScores(t *testing.T) {
	p := NewStubProvider()
	eval, err := p.EvaluateInterviewAnswer(context.Background(), AnswerEvaluationInput{
		Question: "Tell me about Go.", Answer: "I have used Go extensively in production backend services for three years.",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ValidateAnswerEvaluation(eval); err != nil {
		t.Errorf("stub provider output failed its own validation contract: %v", err)
	}
}

func TestStubProvider_Name(t *testing.T) {
	p := NewStubProvider()
	if p.Name() != "stub" {
		t.Errorf("expected provider name 'stub', got %q", p.Name())
	}
}
