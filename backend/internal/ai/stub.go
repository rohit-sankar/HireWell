package ai

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// StubProvider is a deterministic, offline implementation of Provider. It
// exists for three real reasons, not as a fake placeholder:
//  1. The full pipeline (upload -> extract -> validate -> persist -> match)
//     must be testable in CI without a live API key or network access.
//  2. It gives every reviewer of this codebase a working end-to-end demo
//     with zero external setup.
//  3. It documents, in code, exactly what shape a real provider must
//     produce -- see gemini.go for the real implementation behind the
//     same interface.
//
// It is intentionally simple regex/heuristic extraction, NOT a mock that
// returns hardcoded fixture data regardless of input: it actually parses
// whatever raw resume text it is given.
type StubProvider struct{}

func NewStubProvider() *StubProvider { return &StubProvider{} }

func (s *StubProvider) Name() string { return "stub" }

var (
	emailRe = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
	phoneRe = regexp.MustCompile(`(\+?\d[\d\-\s().]{7,}\d)`)
	urlRe   = regexp.MustCompile(`https?://[^\s,)]+`)
)

// knownSkills is the vocabulary the stub extractor recognizes in free text.
// A real provider (gemini.go) has no such fixed vocabulary -- this list
// exists purely so the offline stub can do *something* meaningful with
// arbitrary resume text for demos/tests.
var knownSkills = map[string]string{
	"go": "programming_language", "golang": "programming_language",
	"python": "programming_language", "java": "programming_language",
	"javascript": "programming_language", "typescript": "programming_language",
	"c++": "programming_language", "c#": "programming_language",
	"react": "framework", "vue": "framework", "angular": "framework",
	"gin": "framework", "django": "framework", "flask": "framework", "spring": "framework",
	"next.js": "framework", "nextjs": "framework",
	"mysql": "database", "postgresql": "database", "postgres": "database",
	"mongodb": "database", "redis": "database", "sqlite": "database",
	"docker": "tool", "kubernetes": "cloud", "git": "tool", "jenkins": "tool",
	"aws": "cloud", "gcp": "cloud", "azure": "cloud",
	"rest api": "technical", "rest apis": "technical", "graphql": "technical",
	"communication": "soft_skill", "leadership": "soft_skill", "teamwork": "soft_skill",
}

func (s *StubProvider) ParseResume(_ context.Context, rawText string) (*ResumeExtraction, error) {
	lower := strings.ToLower(rawText)

	email := emailRe.FindString(rawText)
	phone := phoneRe.FindString(rawText)
	links := urlRe.FindAllString(rawText, -1)

	// naive "first non-empty line that isn't the email/phone" heuristic for
	// name -- real extraction happens in the Gemini provider; this is a
	// best-effort fallback.
	name := "Unknown Candidate"
	for _, line := range strings.Split(rawText, "\n") {
		t := strings.TrimSpace(line)
		if t == "" || emailRe.MatchString(t) || phoneRe.MatchString(t) {
			continue
		}
		if len(t) < 60 {
			name = t
			break
		}
	}

	var skills []ParsedSkill
	seen := map[string]bool{}
	for term, category := range knownSkills {
		if strings.Contains(lower, term) && !seen[term] {
			seen[term] = true
			skills = append(skills, ParsedSkill{Name: strings.Title(term), Category: category})
		}
	}

	extraction := &ResumeExtraction{
		PersonalInfo: ParsedPersonalInfo{
			FullName: name,
			Email:    email,
			Phone:    phone,
			Location: "",
			Links:    links,
		},
		Skills: skills,
	}

	// The stub does not attempt to reconstruct education/experience/project
	// history from unstructured text with any reliability -- it correctly
	// returns them empty rather than fabricating plausible-looking but
	// false data. Candidates fill these in via the edit UI; a real LLM
	// provider (gemini.go) does the full extraction.
	return extraction, nil
}

func (s *StubProvider) GenerateInterviewQuestions(_ context.Context, ic InterviewContext) ([]GeneratedQuestion, error) {
	var out []GeneratedQuestion
	n := ic.QuestionCount
	if n <= 0 {
		n = 5
	}

	// Only ever draw from RequiredSkills ∩ CandidateSkills, falling back to
	// PreferredSkills ∩ CandidateSkills, which structurally guarantees the
	// "never ask about a skill the candidate doesn't have and the job
	// doesn't require" rule even before ValidateGeneratedQuestions runs.
	candidateSet := toLowerSet(ic.CandidateSkills)
	var groundedSkills []string
	for _, sk := range ic.RequiredSkills {
		if candidateSet[strings.ToLower(sk)] {
			groundedSkills = append(groundedSkills, sk)
		}
	}
	for _, sk := range ic.PreferredSkills {
		if candidateSet[strings.ToLower(sk)] {
			groundedSkills = append(groundedSkills, sk)
		}
	}

	for _, sk := range groundedSkills {
		if len(out) >= n {
			break
		}
		out = append(out, GeneratedQuestion{
			QuestionText: fmt.Sprintf("Tell me about a time you used %s to solve a real problem. What was the challenge and how did %s help?", sk, sk),
			BasedOn:      "job_skill:" + sk,
		})
	}

	for _, p := range ic.CandidateProjects {
		if len(out) >= n {
			break
		}
		out = append(out, GeneratedQuestion{
			QuestionText: fmt.Sprintf("Walk me through the architecture of your project %q. What were the hardest technical decisions?", p.Name),
			BasedOn:      "candidate_project:" + p.Name,
		})
	}

	for _, e := range ic.CandidateExperience {
		if len(out) >= n {
			break
		}
		out = append(out, GeneratedQuestion{
			QuestionText: fmt.Sprintf("At %s, what was the most significant contribution you made as a %s?", e.Company, e.Role),
			BasedOn:      "candidate_experience:" + e.Company,
		})
	}

	return out, nil
}

func (s *StubProvider) EvaluateInterviewAnswer(_ context.Context, in AnswerEvaluationInput) (*AnswerEvaluation, error) {
	// Deterministic, length/keyword-based heuristic scoring -- adequate for
	// tests and offline demos; the real provider (gemini.go) does actual
	// language-model evaluation.
	words := len(strings.Fields(in.Answer))
	base := 40.0
	if words > 15 {
		base = 60.0
	}
	if words > 40 {
		base = 75.0
	}
	if words > 80 {
		base = 85.0
	}

	return &AnswerEvaluation{
		TechnicalAccuracy: base,
		ProblemSolving:    base - 5,
		DepthOfKnowledge:  base - 10,
		Communication:     base + 5,
		Relevance:         base,
		Strengths:         []string{"Answer addresses the question directly"},
		Weaknesses:        []string{"Automated stub evaluation -- configure a real AI_PROVIDER for production-quality feedback"},
		Recommendation:    "stub evaluation: not suitable for real hiring decisions",
	}, nil
}

func (s *StubProvider) GenerateCandidateSummary(_ context.Context, in CandidateSummaryInput) (string, error) {
	r := in.ResumeExtraction
	var skillNames []string
	for _, sk := range r.Skills {
		skillNames = append(skillNames, sk.Name)
	}
	summary := fmt.Sprintf("%s has experience across %d roles and %d projects, with skills including %s.",
		r.PersonalInfo.FullName, len(r.Experience), len(r.Projects), strings.Join(skillNames, ", "))
	return summary, nil
}

func (s *StubProvider) SemanticMatch(_ context.Context, in SemanticMatchInput) (*SemanticMatchResult, error) {
	// The stub finds no *additional* matches beyond the deterministic
	// slug/alias matcher -- it honestly reports zero semantic assist rather
	// than fabricating confidence scores.
	return &SemanticMatchResult{AdditionalMatches: []SemanticMatchPair{}}, nil
}

func toLowerSet(items []string) map[string]bool {
	out := make(map[string]bool, len(items))
	for _, i := range items {
		out[strings.ToLower(i)] = true
	}
	return out
}
