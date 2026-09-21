package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/resume-platform/backend/internal/models"
)

type CandidateRepository struct {
	db *sql.DB
}

func NewCandidateRepository(db *sql.DB) *CandidateRepository {
	return &CandidateRepository{db: db}
}

func (r *CandidateRepository) Create(ctx context.Context, c *models.Candidate) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO candidates (user_id, full_name, phone, location, links, headline, summary)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		c.UserID, c.FullName, c.Phone, c.Location, c.Links, c.Headline, c.Summary,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

const candidateSelectCols = `c.id, c.user_id, c.full_name, c.phone, c.location, c.links, c.headline, c.summary,
	c.active_resume_id, c.profile_completion, c.created_at, c.updated_at, u.email`

func (r *CandidateRepository) FindByUserID(ctx context.Context, userID int64) (*models.Candidate, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+candidateSelectCols+` FROM candidates c JOIN users u ON u.id = c.user_id WHERE c.user_id = ?`, userID)
	return scanCandidate(row)
}

func (r *CandidateRepository) FindByID(ctx context.Context, id int64) (*models.Candidate, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+candidateSelectCols+` FROM candidates c JOIN users u ON u.id = c.user_id WHERE c.id = ?`, id)
	return scanCandidate(row)
}

func scanCandidate(row *sql.Row) (*models.Candidate, error) {
	var c models.Candidate
	err := row.Scan(&c.ID, &c.UserID, &c.FullName, &c.Phone, &c.Location, &c.Links, &c.Headline, &c.Summary,
		&c.ActiveResumeID, &c.ProfileCompletion, &c.CreatedAt, &c.UpdatedAt, &c.Email)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CandidateRepository) Update(ctx context.Context, c *models.Candidate) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE candidates SET full_name = ?, phone = ?, location = ?, links = ?, headline = ?, summary = ?, profile_completion = ?
		 WHERE id = ?`,
		c.FullName, c.Phone, c.Location, c.Links, c.Headline, c.Summary, c.ProfileCompletion, c.ID,
	)
	return err
}

func (r *CandidateRepository) SetActiveResume(ctx context.Context, candidateID, resumeID int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE candidates SET active_resume_id = ? WHERE id = ?`, resumeID, candidateID)
	return err
}
