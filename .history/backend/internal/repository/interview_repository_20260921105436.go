package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/resume-platform/backend/internal/models"
)

type InterviewRepository struct {
	db *sql.DB
}

func NewInterviewRepository(db *sql.DB) *InterviewRepository {
	return &InterviewRepository{db: db}
}

func (r *InterviewRepository) Create(ctx context.Context, applicationID int64) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO interviews (application_id, status) VALUES (?, ?)`, applicationID, models.InterviewPending)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *InterviewRepository) FindByID(ctx context.Context, id int64) (*models.Interview, error) {
	var iv models.Interview
	err := r.db.QueryRowContext(ctx,
		`SELECT id, application_id, status, started_at, completed_at, created_at FROM interviews WHERE id = ?`, id,
	).Scan(&iv.ID, &iv.ApplicationID, &iv.Status, &iv.StartedAt, &iv.CompletedAt, &iv.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &iv, nil
}

func (r *InterviewRepository) FindByApplicationID(ctx context.Context, applicationID int64) (*models.Interview, error) {
	var iv models.Interview
	err := r.db.QueryRowContext(ctx,
		`SELECT id, application_id, status, started_at, completed_at, created_at FROM interviews WHERE application_id = ?`, applicationID,
	).Scan(&iv.ID, &iv.ApplicationID, &iv.Status, &iv.StartedAt, &iv.CompletedAt, &iv.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &iv, nil
}

func (r *InterviewRepository) UpdateStatus(ctx context.Context, id int64, status models.InterviewStatus, setStarted, setCompleted bool) error {
	query := `UPDATE interviews SET status = ?`
	if setStarted {
		query += `, started_at = NOW()`
	}
	if setCompleted {
		query += `, completed_at = NOW()`
	}
	query += ` WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, status, id)
	return err
}

func (r *InterviewRepository) AddQuestions(ctx context.Context, interviewID int64, questions []models.InterviewQuestion) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for i, q := range questions {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO interview_questions (interview_id, question_text, based_on, order_index) VALUES (?, ?, ?, ?)`,
			interviewID, q.QuestionText, q.BasedOn, i); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *InterviewRepository) GetQuestions(ctx context.Context, interviewID int64) ([]models.InterviewQuestion, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, interview_id, question_text, based_on, order_index FROM interview_questions
		 WHERE interview_id = ? ORDER BY order_index`, interviewID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.InterviewQuestion, 0)
	for rows.Next() {
		var q models.InterviewQuestion
		if err := rows.Scan(&q.ID, &q.InterviewID, &q.QuestionText, &q.BasedOn, &q.OrderIndex); err != nil {
			return nil, err
		}
		out = append(out, q)
	}
	return out, rows.Err()
}

func (r *InterviewRepository) FindQuestionByID(ctx context.Context, questionID int64) (*models.InterviewQuestion, error) {
	var q models.InterviewQuestion
	err := r.db.QueryRowContext(ctx,
		`SELECT id, interview_id, question_text, based_on, order_index FROM interview_questions WHERE id = ?`, questionID,
	).Scan(&q.ID, &q.InterviewID, &q.QuestionText, &q.BasedOn, &q.OrderIndex)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &q, nil
}

func (r *InterviewRepository) SaveAnswer(ctx context.Context, questionID int64, answerText string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO interview_answers (question_id, answer_text) VALUES (?, ?)
		 ON DUPLICATE KEY UPDATE answer_text = VALUES(answer_text), submitted_at = NOW()`,
		questionID, answerText)
	return err
}

func (r *InterviewRepository) GetAnswers(ctx context.Context, interviewID int64) (map[int64]models.InterviewAnswer, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT ia.id, ia.question_id, ia.answer_text, ia.submitted_at
		 FROM interview_answers ia
		 JOIN interview_questions iq ON iq.id = ia.question_id
		 WHERE iq.interview_id = ?`, interviewID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[int64]models.InterviewAnswer{}
	for rows.Next() {
		var a models.InterviewAnswer
		if err := rows.Scan(&a.ID, &a.QuestionID, &a.AnswerText, &a.SubmittedAt); err != nil {
			return nil, err
		}
		out[a.QuestionID] = a
	}
	return out, rows.Err()
}

func (r *InterviewRepository) SaveEvaluation(ctx context.Context, e *models.InterviewEvaluation) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO interview_evaluations
		    (interview_id, technical_accuracy, problem_solving, depth_of_knowledge, communication, relevance,
		     overall_score, weights_used, strengths, weaknesses, recommendation)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE
		    technical_accuracy = VALUES(technical_accuracy), problem_solving = VALUES(problem_solving),
		    depth_of_knowledge = VALUES(depth_of_knowledge), communication = VALUES(communication),
		    relevance = VALUES(relevance), overall_score = VALUES(overall_score),
		    weights_used = VALUES(weights_used), strengths = VALUES(strengths),
		    weaknesses = VALUES(weaknesses), recommendation = VALUES(recommendation), evaluated_at = NOW()`,
		e.InterviewID, e.TechnicalAccuracy, e.ProblemSolving, e.DepthOfKnowledge, e.Communication, e.Relevance,
		e.OverallScore, e.WeightsUsed, e.Strengths, e.Weaknesses, e.Recommendation,
	)
	return err
}

func (r *InterviewRepository) GetEvaluation(ctx context.Context, interviewID int64) (*models.InterviewEvaluation, error) {
	var e models.InterviewEvaluation
	err := r.db.QueryRowContext(ctx,
		`SELECT id, interview_id, technical_accuracy, problem_solving, depth_of_knowledge, communication, relevance,
		        overall_score, weights_used, strengths, weaknesses, recommendation, evaluated_at
		 FROM interview_evaluations WHERE interview_id = ?`, interviewID,
	).Scan(&e.ID, &e.InterviewID, &e.TechnicalAccuracy, &e.ProblemSolving, &e.DepthOfKnowledge, &e.Communication,
		&e.Relevance, &e.OverallScore, &e.WeightsUsed, &e.Strengths, &e.Weaknesses, &e.Recommendation, &e.EvaluatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}
