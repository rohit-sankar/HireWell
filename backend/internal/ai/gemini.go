package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// GeminiProvider implements Provider against Google's Gemini API using
// structured output (responseMimeType: application/json + responseSchema),
// which is what lets us skip fragile "please respond with only JSON" prompt
// engineering and get a schema-conformant response directly from the API.
//
// Docs: https://ai.google.dev/gemini-api/docs/structured-output
type GeminiProvider struct {
	apiKey     string
	model      string
	httpClient *http.Client
	maxRetries int
}

func NewGeminiProvider(apiKey, model string) *GeminiProvider {
	if model == "" {
		model = "gemini-2.0-flash"
	}
	return &GeminiProvider{
		apiKey:     apiKey,
		model:      model,
		httpClient: &http.Client{Timeout: 45 * time.Second},
		maxRetries: 3,
	}
}

func (g *GeminiProvider) Name() string { return "gemini:" + g.model }

// geminiRequest / geminiResponse are the minimal subset of the Gemini REST
// schema we need. We deliberately don't pull in a full SDK dependency --
// this sandbox's network allowlist can't reach a Go module proxy for one,
// and a raw REST call keeps this provider auditable in ~150 lines.
type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerationConfig struct {
	ResponseMimeType string      `json:"responseMimeType,omitempty"`
	ResponseSchema   interface{} `json:"responseSchema,omitempty"`
	Temperature      float64     `json:"temperature"`
}

type geminiRequest struct {
	Contents         []geminiContent        `json:"contents"`
	GenerationConfig geminiGenerationConfig `json:"generationConfig"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []geminiPart `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

// callStructured sends a single-turn prompt and unmarshals the JSON reply
// into target. It retries on transient failures (timeouts, 429, 5xx) with
// exponential backoff, and returns typed errors (ErrProviderRateLimit,
// ErrProviderTimeout, ErrProviderBadResponse) so callers can react
// appropriately instead of treating every failure the same way.
func (g *GeminiProvider) callStructured(ctx context.Context, systemPrompt, userPrompt string, schema interface{}, target interface{}) error {
	if g.apiKey == "" {
		return fmt.Errorf("%w: AI_API_KEY is not configured", ErrProviderUnavailable)
	}

	reqBody := geminiRequest{
		Contents: []geminiContent{
			{Role: "user", Parts: []geminiPart{{Text: systemPrompt + "\n\n" + userPrompt}}},
		},
		GenerationConfig: geminiGenerationConfig{
			ResponseMimeType: "application/json",
			ResponseSchema:   schema,
			Temperature:      0.2, // low temperature: we want consistent structured extraction, not creativity
		},
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal gemini request: %w", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", g.model, g.apiKey)

	var lastErr error
	for attempt := 0; attempt < g.maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt*attempt) * 500 * time.Millisecond
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
		if err != nil {
			return fmt.Errorf("build gemini request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := g.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("%w: %v", ErrProviderTimeout, err)
			continue
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = fmt.Errorf("%w: reading body: %v", ErrProviderBadResponse, readErr)
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			lastErr = ErrProviderRateLimit
			continue
		}
		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("%w: status %d", ErrProviderUnavailable, resp.StatusCode)
			continue
		}
		if resp.StatusCode >= 400 {
			// 4xx other than 429 (bad key, bad request) is not retryable.
			return fmt.Errorf("%w: status %d: %s", ErrProviderBadResponse, resp.StatusCode, truncateForError(string(body)))
		}

		var gr geminiResponse
		if err := json.Unmarshal(body, &gr); err != nil {
			lastErr = fmt.Errorf("%w: %v", ErrProviderBadResponse, err)
			continue
		}
		if gr.Error != nil {
			return fmt.Errorf("%w: %s", ErrProviderBadResponse, gr.Error.Message)
		}
		if len(gr.Candidates) == 0 || len(gr.Candidates[0].Content.Parts) == 0 {
			lastErr = fmt.Errorf("%w: empty candidates in response", ErrProviderBadResponse)
			continue
		}

		text := gr.Candidates[0].Content.Parts[0].Text
		if err := json.Unmarshal([]byte(text), target); err != nil {
			lastErr = fmt.Errorf("%w: response was not valid JSON matching the schema: %v", ErrProviderBadResponse, err)
			continue
		}

		return nil // success
	}

	return lastErr
}

func truncateForError(s string) string {
	if len(s) > 300 {
		return s[:300] + "..."
	}
	return s
}

// --- Provider methods -------------------------------------------------------

var resumeExtractionSchema = map[string]interface{}{
	"type": "object",
	"properties": map[string]interface{}{
		"personal_info": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"full_name": map[string]interface{}{"type": "string"},
				"email":     map[string]interface{}{"type": "string"},
				"phone":     map[string]interface{}{"type": "string"},
				"location":  map[string]interface{}{"type": "string"},
				"links":     map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
			},
		},
		"education": map[string]interface{}{
			"type": "array",
			"items": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"institution": map[string]interface{}{"type": "string"},
					"degree":      map[string]interface{}{"type": "string"},
					"field":       map[string]interface{}{"type": "string"},
					"start_date":  map[string]interface{}{"type": "string"},
					"end_date":    map[string]interface{}{"type": "string"},
					"gpa":         map[string]interface{}{"type": "string"},
				},
			},
		},
		"experience": map[string]interface{}{
			"type": "array",
			"items": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"company":          map[string]interface{}{"type": "string"},
					"role":             map[string]interface{}{"type": "string"},
					"start_date":       map[string]interface{}{"type": "string"},
					"end_date":         map[string]interface{}{"type": "string"},
					"is_current":       map[string]interface{}{"type": "boolean"},
					"description":      map[string]interface{}{"type": "string"},
					"technologies":     map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
					"responsibilities": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
				},
			},
		},
		"projects": map[string]interface{}{
			"type": "array",
			"items": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name":         map[string]interface{}{"type": "string"},
					"description":  map[string]interface{}{"type": "string"},
					"technologies": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
					"role":         map[string]interface{}{"type": "string"},
					"links":        map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
				},
			},
		},
		"skills": map[string]interface{}{
			"type": "array",
			"items": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{"type": "string"},
					"category": map[string]interface{}{
						"type": "string",
						"enum": []string{"technical", "programming_language", "framework", "database", "cloud", "tool", "soft_skill"},
					},
				},
			},
		},
		"certifications": map[string]interface{}{
			"type": "array",
			"items": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name":          map[string]interface{}{"type": "string"},
					"issuer":        map[string]interface{}{"type": "string"},
					"issue_date":    map[string]interface{}{"type": "string"},
					"credential_id": map[string]interface{}{"type": "string"},
				},
			},
		},
	},
}

func (g *GeminiProvider) ParseResume(ctx context.Context, rawText string) (*ResumeExtraction, error) {
	system := `You are a precise resume-parsing engine. Extract ONLY information that is explicitly present in the resume text below. Do not infer, guess, or fabricate any field. If a field is not present, leave it as an empty string or empty array. Categorize each skill into exactly one of: technical, programming_language, framework, database, cloud, tool, soft_skill.`
	var extraction ResumeExtraction
	if err := g.callStructured(ctx, system, "RESUME TEXT:\n"+rawText, resumeExtractionSchema, &extraction); err != nil {
		return nil, err
	}
	return &extraction, nil
}

var interviewQuestionsSchema = map[string]interface{}{
	"type": "array",
	"items": map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"question_text": map[string]interface{}{"type": "string"},
			"based_on":      map[string]interface{}{"type": "string"},
		},
	},
}

func (g *GeminiProvider) GenerateInterviewQuestions(ctx context.Context, ic InterviewContext) ([]GeneratedQuestion, error) {
	system := fmt.Sprintf(`You are a technical interviewer for the role "%s". Generate exactly %d interview questions.
CRITICAL RULE: You may ONLY ask about skills, technologies, projects, or experience that appear in the CONTEXT below.
Do NOT ask about anything not listed, even if it would normally be relevant to the role. Every question's "based_on"
field must name the exact job skill, candidate project, or candidate experience company it is grounded in
(format: "job_skill:X", "candidate_project:X", or "candidate_experience:X").`, ic.JobTitle, ic.QuestionCount)

	userPrompt := fmt.Sprintf(`REQUIRED JOB SKILLS: %v
PREFERRED JOB SKILLS: %v
CANDIDATE SKILLS: %v
CANDIDATE PROJECTS: %v
CANDIDATE EXPERIENCE: %v`, ic.RequiredSkills, ic.PreferredSkills, ic.CandidateSkills, ic.CandidateProjects, ic.CandidateExperience)

	var questions []GeneratedQuestion
	if err := g.callStructured(ctx, system, userPrompt, interviewQuestionsSchema, &questions); err != nil {
		return nil, err
	}
	return questions, nil
}

var answerEvaluationSchema = map[string]interface{}{
	"type": "object",
	"properties": map[string]interface{}{
		"technical_accuracy": map[string]interface{}{"type": "number"},
		"problem_solving":    map[string]interface{}{"type": "number"},
		"depth_of_knowledge": map[string]interface{}{"type": "number"},
		"communication":      map[string]interface{}{"type": "number"},
		"relevance":          map[string]interface{}{"type": "number"},
		"strengths":          map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
		"weaknesses":         map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
		"recommendation":     map[string]interface{}{"type": "string"},
	},
}

func (g *GeminiProvider) EvaluateInterviewAnswer(ctx context.Context, in AnswerEvaluationInput) (*AnswerEvaluation, error) {
	system := `You are evaluating a candidate's interview answer. Score each dimension from 0-100 based strictly on the
content of the answer relative to the question asked. Do not be swayed by answer length alone -- a short, precise
answer should score higher than a long, vague one.`
	userPrompt := fmt.Sprintf("QUESTION (grounded in %s): %s\n\nCANDIDATE ANSWER: %s", in.BasedOn, in.Question, in.Answer)

	var eval AnswerEvaluation
	if err := g.callStructured(ctx, system, userPrompt, answerEvaluationSchema, &eval); err != nil {
		return nil, err
	}
	return &eval, nil
}

func (g *GeminiProvider) GenerateCandidateSummary(ctx context.Context, in CandidateSummaryInput) (string, error) {
	system := `Write a concise, factual 2-3 sentence professional summary of this candidate, using ONLY information
present in the structured resume data provided. Do not invent achievements or skills not listed.`
	data, _ := json.Marshal(in.ResumeExtraction)

	schema := map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{"summary": map[string]interface{}{"type": "string"}},
	}
	var out struct {
		Summary string `json:"summary"`
	}
	if err := g.callStructured(ctx, system, string(data), schema, &out); err != nil {
		return "", err
	}
	return out.Summary, nil
}

var semanticMatchSchema = map[string]interface{}{
	"type": "object",
	"properties": map[string]interface{}{
		"additional_matches": map[string]interface{}{
			"type": "array",
			"items": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"job_skill":       map[string]interface{}{"type": "string"},
					"candidate_skill": map[string]interface{}{"type": "string"},
					"confidence":      map[string]interface{}{"type": "number"},
				},
			},
		},
	},
}

func (g *GeminiProvider) SemanticMatch(ctx context.Context, in SemanticMatchInput) (*SemanticMatchResult, error) {
	system := `Given a list of job-required skills and a list of candidate skills, identify pairs that are
semantically equivalent but not textually identical (e.g. "Postgres administration" ~ "relational databases").
Do NOT include pairs that are already an exact or near-exact textual match -- only genuinely non-obvious
equivalences. Assign a confidence from 0 to 1.`
	userPrompt := fmt.Sprintf("JOB SKILLS: %v\nCANDIDATE SKILLS: %v", in.JobSkills, in.CandidateSkills)

	var result SemanticMatchResult
	if err := g.callStructured(ctx, system, userPrompt, semanticMatchSchema, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
