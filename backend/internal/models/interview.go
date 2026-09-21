package models

import "time"

type InterviewStatus string

const (
	InterviewPending    InterviewStatus = "pending"
	InterviewInProgress InterviewStatus = "in_progress"
	InterviewCompleted  InterviewStatus = "completed"
)

type Interview struct {
	ID            int64           `json:"id" db:"id"`
	ApplicationID int64           `json:"application_id" db:"application_id"`
	Status        InterviewStatus `json:"status" db:"status"`
	StartedAt     *time.Time      `json:"started_at" db:"started_at"`
	CompletedAt   *time.Time      `json:"completed_at" db:"completed_at"`
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
}

type InterviewQuestion struct {
	ID           int64  `json:"id" db:"id"`
	InterviewID  int64  `json:"interview_id" db:"interview_id"`
	QuestionText string `json:"question_text" db:"question_text"`
	BasedOn      string `json:"based_on" db:"based_on"` // e.g. "job_skill:Go"
	OrderIndex   uint16 `json:"order_index" db:"order_index"`
}

type InterviewAnswer struct {
	ID          int64     `json:"id" db:"id"`
	QuestionID  int64     `json:"question_id" db:"question_id"`
	AnswerText  string    `json:"answer_text" db:"answer_text"`
	SubmittedAt time.Time `json:"submitted_at" db:"submitted_at"`
}

type InterviewEvaluation struct {
	ID                int64     `json:"id" db:"id"`
	InterviewID       int64     `json:"interview_id" db:"interview_id"`
	TechnicalAccuracy float64   `json:"technical_accuracy" db:"technical_accuracy"`
	ProblemSolving    float64   `json:"problem_solving" db:"problem_solving"`
	DepthOfKnowledge  float64   `json:"depth_of_knowledge" db:"depth_of_knowledge"`
	Communication     float64   `json:"communication" db:"communication"`
	Relevance         float64   `json:"relevance" db:"relevance"`
	OverallScore      float64   `json:"overall_score" db:"overall_score"`
	WeightsUsed       JSONMap   `json:"weights_used" db:"weights_used"`
	Strengths         JSONList  `json:"strengths" db:"strengths"`
	Weaknesses        JSONList  `json:"weaknesses" db:"weaknesses"`
	Recommendation    *string   `json:"recommendation" db:"recommendation"`
	EvaluatedAt       time.Time `json:"evaluated_at" db:"evaluated_at"`
}
