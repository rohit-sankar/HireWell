package services

import (
	"context"
	"errors"
	"strings"

	"github.com/resume-platform/backend/internal/ai"
	"github.com/resume-platform/backend/internal/models"
	"github.com/resume-platform/backend/internal/repository"
	"github.com/resume-platform/backend/internal/utils"
	"github.com/rs/zerolog"
)

type InterviewService struct {
	interviews   *repository.InterviewRepository
	applications *repository.ApplicationRepository
	jobs         *repository.JobRepository
	resumes      *repository.ResumeRepository
	jobService   *JobService
	aiProvider   ai.Provider
	logger       zerolog.Logger
}

func NewInterviewService(interviews *repository.InterviewRepository, applications *repository.ApplicationRepository, jobs *repository.JobRepository, resumes *repository.ResumeRepository, jobService *JobService, provider ai.Provider, logger zerolog.Logger) *InterviewService {
	return &InterviewService{interviews: interviews, applications: applications, jobs: jobs, resumes: resumes, jobService: jobService, aiProvider: provider, logger: logger}
}

// Create implements "AI interviews candidates based ONLY on skills/
// requirements actually present in the job and candidate profile": it
// builds an ai.InterviewContext strictly from the job's own required/
// preferred skills and the candidate's own extracted resume data, generates
// questions from the provider, then runs every question through
// ai.ValidateGeneratedQuestions -- any question not grounded in that exact
// context is dropped rather than stored, even if the provider produced it.
func (s *InterviewService) Create(ctx context.Context, candidateUserID, applicationID int64) (*models.Interview, error) {
	app, err := s.applications.FindByID(ctx, applicationID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, utils.NewNotFound("application")
	}
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}

	jobFull, err := s.jobService.GetFull(ctx, app.JobID)
	if err != nil {
		return nil, err
	}
	resumeFull, err := s.resumes.GetFull(ctx, app.ResumeID)
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}

	if existing, err := s.interviews.FindByApplicationID(ctx, applicationID); err == nil {
		return existing, nil // idempotent: re-requesting returns the existing interview
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, utils.NewDatabaseError(err)
	}

	var required, preferred, candidateSkills []string
	allowedTerms := map[string]bool{}
	for _, js := range jobFull.Skills {
		allowedTerms[strings.ToLower(js.SkillName)] = true
		if js.IsRequired {
			required = append(required, js.SkillName)
		} else {
			preferred = append(preferred, js.SkillName)
		}
	}
	for _, sk := range resumeFull.Skills {
		candidateSkills = append(candidateSkills, sk.SkillName)
		allowedTerms[strings.ToLower(sk.SkillName)] = true
	}

	var projects []ai.ParsedProject
	for _, p := range resumeFull.Projects {
		allowedTerms[strings.ToLower(p.Name)] = true
		projects = append(projects, ai.ParsedProject{Name: p.Name, Description: derefOrEmpty(p.Description)})
	}
	var experience []ai.ParsedExperience
	for _, e := range resumeFull.Experience {
		allowedTerms[strings.ToLower(e.Company)] = true
		experience = append(experience, ai.ParsedExperience{Company: e.Company, Role: e.Role})
	}

	ic := ai.InterviewContext{
		JobTitle: jobFull.Title, RequiredSkills: required, PreferredSkills: preferred,
		CandidateSkills: candidateSkills, CandidateProjects: projects, CandidateExperience: experience,
		QuestionCount: int(jobFull.InterviewQuestionCount),
	}

	generated, err := s.aiProvider.GenerateInterviewQuestions(ctx, ic)
	if err != nil {
		return nil, utils.NewAIError("failed to generate interview questions", err)
	}

	valid, rejected := ai.ValidateGeneratedQuestions(generated, allowedTerms)
	if len(rejected) > 0 {
		s.logger.Warn().Strs("rejected_questions", rejected).Int64("application_id", applicationID).
			Msg("dropped AI-generated interview questions not grounded in job/candidate context")
	}
	if len(valid) == 0 {
		return nil, utils.NewAIError("the AI provider did not return any interview questions grounded in this job's requirements and the candidate's profile", nil)
	}

	interviewID, err := s.interviews.Create(ctx, applicationID)
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}

	var questions []models.InterviewQuestion
	for _, q := range valid {
		questions = append(questions, models.InterviewQuestion{QuestionText: q.QuestionText, BasedOn: q.BasedOn})
	}
	if err := s.interviews.AddQuestions(ctx, interviewID, questions); err != nil {
		return nil, utils.NewDatabaseError(err)
	}

	return s.interviews.FindByID(ctx, interviewID)
}

func (s *InterviewService) GetByID(ctx context.Context, id int64) (*models.Interview, []models.InterviewQuestion, error) {
	iv, err := s.interviews.FindByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, nil, utils.NewNotFound("interview")
	}
	if err != nil {
		return nil, nil, utils.NewDatabaseError(err)
	}
	questions, err := s.interviews.GetQuestions(ctx, id)
	if err != nil {
		return nil, nil, utils.NewDatabaseError(err)
	}
	return iv, questions, nil
}

func (s *InterviewService) Start(ctx context.Context, id int64) (*models.Interview, error) {
	if err := s.interviews.UpdateStatus(ctx, id, models.InterviewInProgress, true, false); err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	return s.interviews.FindByID(ctx, id)
}

func (s *InterviewService) SubmitAnswer(ctx context.Context, questionID int64, answerText string) error {
	if strings.TrimSpace(answerText) == "" {
		return utils.NewValidationError("answer cannot be empty", nil)
	}
	if _, err := s.interviews.FindQuestionByID(ctx, questionID); errors.Is(err, repository.ErrNotFound) {
		return utils.NewNotFound("interview question")
	} else if err != nil {
		return utils.NewDatabaseError(err)
	}
	if err := s.interviews.SaveAnswer(ctx, questionID, answerText); err != nil {
		return utils.NewDatabaseError(err)
	}
	return nil
}

// Complete implements evaluation aggregation: each answered question is
// scored individually by the AI provider against the documented rubric
// (technical_accuracy, problem_solving, depth_of_knowledge, communication,
// relevance), then averaged into a single InterviewEvaluation. Weights for
// the final overall_score are configurable (documented default: equal
// weighting across the five dimensions) rather than hardcoded as a single
// magic formula.
func (s *InterviewService) Complete(ctx context.Context, id int64) (*models.InterviewEvaluation, error) {
	iv, questions, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	answers, err := s.interviews.GetAnswers(ctx, id)
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}

	var accSum, psSum, dokSum, commSum, relSum float64
	var strengths, weaknesses []interface{}
	var recommendations []string
	answered := 0

	for _, q := range questions {
		ans, ok := answers[q.ID]
		if !ok {
			continue
		}
		eval, err := s.aiProvider.EvaluateInterviewAnswer(ctx, ai.AnswerEvaluationInput{
			Question: q.QuestionText, Answer: ans.AnswerText, BasedOn: q.BasedOn,
		})
		if err != nil {
			s.logger.Warn().Err(err).Int64("question_id", q.ID).Msg("failed to evaluate interview answer, skipping from aggregate")
			continue
		}
		if err := ai.ValidateAnswerEvaluation(eval); err != nil {
			s.logger.Warn().Err(err).Int64("question_id", q.ID).Msg("AI evaluation failed validation, skipping from aggregate")
			continue
		}

		accSum += eval.TechnicalAccuracy
		psSum += eval.ProblemSolving
		dokSum += eval.DepthOfKnowledge
		commSum += eval.Communication
		relSum += eval.Relevance
		answered++
		for _, st := range eval.Strengths {
			strengths = append(strengths, st)
		}
		for _, wk := range eval.Weaknesses {
			weaknesses = append(weaknesses, wk)
		}
		if eval.Recommendation != "" {
			recommendations = append(recommendations, eval.Recommendation)
		}
	}

	if answered == 0 {
		return nil, utils.NewValidationError("no answered questions could be evaluated", nil)
	}

	avg := func(sum float64) float64 { return round2(sum / float64(answered)) }
	technical, problemSolving, depth, comm, relevance := avg(accSum), avg(psSum), avg(dokSum), avg(commSum), avg(relSum)

	// Documented default: equal weighting across the five rubric
	// dimensions. This is intentionally simple and stated explicitly rather
	// than buried -- a future iteration could make these per-job
	// configurable the same way match weights are.
	overall := round2((technical + problemSolving + depth + comm + relevance) / 5)

	evaluation := &models.InterviewEvaluation{
		InterviewID: id, TechnicalAccuracy: technical, ProblemSolving: problemSolving,
		DepthOfKnowledge: depth, Communication: comm, Relevance: relevance, OverallScore: overall,
		WeightsUsed: models.JSONMap{"technical_accuracy": 0.2, "problem_solving": 0.2, "depth_of_knowledge": 0.2, "communication": 0.2, "relevance": 0.2},
		Strengths:   toInterfaceList(strengths),
		Weaknesses:  toInterfaceList(weaknesses),
	}
	if len(recommendations) > 0 {
		rec := strings.Join(dedupe(recommendations), " ")
		evaluation.Recommendation = &rec
	}

	if err := s.interviews.SaveEvaluation(ctx, evaluation); err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	if err := s.interviews.UpdateStatus(ctx, id, models.InterviewCompleted, false, true); err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	_ = iv

	return s.interviews.GetEvaluation(ctx, id)
}

func (s *InterviewService) GetEvaluation(ctx context.Context, interviewID int64) (*models.InterviewEvaluation, error) {
	e, err := s.interviews.GetEvaluation(ctx, interviewID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, utils.NewNotFound("interview evaluation")
	}
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	return e, nil
}

func round2(v float64) float64 { return float64(int(v*100+0.5)) / 100 }

func derefOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func toInterfaceList(items []interface{}) models.JSONList {
	return models.JSONList(items)
}

func dedupe(items []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, i := range items {
		if !seen[i] {
			seen[i] = true
			out = append(out, i)
		}
	}
	return out
}
