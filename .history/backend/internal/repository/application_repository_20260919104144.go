package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/resume-platform/backend/internal/models"
)

type ApplicationRepository struct {
	db *sql.DB
}

func NewApplicationRepository(db *sql.DB) *ApplicationRepository {
	return &ApplicationRepository{db: db}
}

func (r *ApplicationRepository) Create(ctx context.Context, a *models.Application) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO applications (job_id, candidate_id, resume_id, status) VALUES (?, ?, ?, ?)`,
		a.JobID, a.CandidateID, a.ResumeID, models.AppStatusApplied,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

const applicationSelectCols = `a.id, a.job_id, a.candidate_id, a.resume_id, a.status, a.applied_at, a.updated_at,
	j.title, co.company_name, c.full_name`

func (r *ApplicationRepository) FindByID(ctx context.Context, id int64) (*models.Application, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+applicationSelectCols+`
		 FROM applications a
		 JOIN jobs j ON j.id = a.job_id
		 JOIN companies co ON co.id = j.company_id
		 JOIN candidates c ON c.id = a.candidate_id
		 WHERE a.id = ?`, id)
	return scanApplication(row)
}

func (r *ApplicationRepository) FindByJobAndCandidate(ctx context.Context, jobID, candidateID int64) (*models.Application, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+applicationSelectCols+`
		 FROM applications a
		 JOIN jobs j ON j.id = a.job_id
		 JOIN companies co ON co.id = j.company_id
		 JOIN candidates c ON c.id = a.candidate_id
		 WHERE a.job_id = ? AND a.candidate_id = ?`, jobID, candidateID)
	return scanApplication(row)
}

func scanApplication(row *sql.Row) (*models.Application, error) {
	var a models.Application
	err := row.Scan(&a.ID, &a.JobID, &a.CandidateID, &a.ResumeID, &a.Status, &a.AppliedAt, &a.UpdatedAt,
		&a.JobTitle, &a.CompanyName, &a.CandidateName)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *ApplicationRepository) ListByCandidate(ctx context.Context, candidateID int64) ([]models.Application, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+applicationSelectCols+`
		 FROM applications a
		 JOIN jobs j ON j.id = a.job_id
		 JOIN companies co ON co.id = j.company_id
		 JOIN candidates c ON c.id = a.candidate_id
		 WHERE a.candidate_id = ? ORDER BY a.applied_at DESC`, candidateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanApplicationRows(rows)
}

func (r *ApplicationRepository) ListByJob(ctx context.Context, jobID int64) ([]models.Application, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+applicationSelectCols+`
		 FROM applications a
		 JOIN jobs j ON j.id = a.job_id
		 JOIN companies co ON co.id = j.company_id
		 JOIN candidates c ON c.id = a.candidate_id
		 WHERE a.job_id = ? ORDER BY a.applied_at DESC`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanApplicationRows(rows)
}

func scanApplicationRows(rows *sql.Rows) ([]models.Application, error) {
	out := make([]models.Application, 0)
	for rows.Next() {
		var a models.Application
		if err := rows.Scan(&a.ID, &a.JobID, &a.CandidateID, &a.ResumeID, &a.Status, &a.AppliedAt, &a.UpdatedAt,
			&a.JobTitle, &a.CompanyName, &a.CandidateName); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *ApplicationRepository) UpdateStatus(ctx context.Context, id int64, status models.ApplicationStatus) error {
	_, err := r.db.ExecContext(ctx, `UPDATE applications SET status = ? WHERE id = ?`, status, id)
	return err
}
