package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/resume-platform/backend/internal/models"
)

type JobRepository struct {
	db *sql.DB
}

func NewJobRepository(db *sql.DB) *JobRepository {
	return &JobRepository{db: db}
}

func (r *JobRepository) Create(ctx context.Context, j *models.Job) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO jobs (company_id, title, description, location, employment_type, experience_min_years,
		                    experience_max_years, education_requirement, status, interview_question_count)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		j.CompanyID, j.Title, j.Description, j.Location, j.EmploymentType, j.ExperienceMinYears,
		j.ExperienceMaxYears, j.EducationRequirement, j.Status, j.InterviewQuestionCount,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	// default match weights row, per job, editable later via UpdateMatchWeights
	_, err = r.db.ExecContext(ctx, `INSERT INTO job_match_weights (job_id) VALUES (?)`, id)
	return id, err
}

const jobSelectCols = `j.id, j.company_id, j.title, j.description, j.location, j.employment_type,
	j.experience_min_years, j.experience_max_years, j.education_requirement, j.status,
	j.interview_question_count, j.created_at, j.updated_at, j.published_at, j.closed_at, co.company_name`

func (r *JobRepository) FindByID(ctx context.Context, id int64) (*models.Job, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+jobSelectCols+` FROM jobs j JOIN companies co ON co.id = j.company_id WHERE j.id = ?`, id)
	return scanJob(row)
}

func scanJob(row *sql.Row) (*models.Job, error) {
	var j models.Job
	err := row.Scan(&j.ID, &j.CompanyID, &j.Title, &j.Description, &j.Location, &j.EmploymentType,
		&j.ExperienceMinYears, &j.ExperienceMaxYears, &j.EducationRequirement, &j.Status,
		&j.InterviewQuestionCount, &j.CreatedAt, &j.UpdatedAt, &j.PublishedAt, &j.ClosedAt, &j.CompanyName)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &j, nil
}

type JobListFilter struct {
	CompanyID *int64
	Status    *models.JobStatus
	Search    string
	Limit     int
	Offset    int
}

func (r *JobRepository) List(ctx context.Context, f JobListFilter) ([]models.Job, int, error) {
	where := " WHERE 1=1 "
	args := []interface{}{}

	if f.CompanyID != nil {
		where += " AND j.company_id = ? "
		args = append(args, *f.CompanyID)
	}
	if f.Status != nil {
		where += " AND j.status = ? "
		args = append(args, *f.Status)
	}
	if f.Search != "" {
		where += " AND (j.title LIKE ? OR j.description LIKE ?) "
		like := "%" + f.Search + "%"
		args = append(args, like, like)
	}

	var total int
	countQuery := `SELECT COUNT(*) FROM jobs j` + where
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	query := `SELECT ` + jobSelectCols + ` FROM jobs j JOIN companies co ON co.id = j.company_id` + where +
		` ORDER BY j.created_at DESC LIMIT ? OFFSET ?`
	args = append(args, limit, f.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]models.Job, 0)
	for rows.Next() {
		var j models.Job
		if err := rows.Scan(&j.ID, &j.CompanyID, &j.Title, &j.Description, &j.Location, &j.EmploymentType,
			&j.ExperienceMinYears, &j.ExperienceMaxYears, &j.EducationRequirement, &j.Status,
			&j.InterviewQuestionCount, &j.CreatedAt, &j.UpdatedAt, &j.PublishedAt, &j.ClosedAt, &j.CompanyName); err != nil {
			return nil, 0, err
		}
		out = append(out, j)
	}
	return out, total, rows.Err()
}

func (r *JobRepository) Update(ctx context.Context, j *models.Job) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE jobs SET title = ?, description = ?, location = ?, employment_type = ?, experience_min_years = ?,
		                  experience_max_years = ?, education_requirement = ?, interview_question_count = ?
		 WHERE id = ?`,
		j.Title, j.Description, j.Location, j.EmploymentType, j.ExperienceMinYears,
		j.ExperienceMaxYears, j.EducationRequirement, j.InterviewQuestionCount, j.ID,
	)
	return err
}

func (r *JobRepository) UpdateStatus(ctx context.Context, id int64, status models.JobStatus, timestampCol string) error {
	query := `UPDATE jobs SET status = ?`
	if timestampCol != "" {
		query += `, ` + timestampCol + ` = NOW()`
	}
	query += ` WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, status, id)
	return err
}

func (r *JobRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM jobs WHERE id = ?`, id)
	return err
}

// --- job skills ---

func (r *JobRepository) ReplaceSkills(ctx context.Context, jobID int64, skills []models.JobSkill) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM job_skills WHERE job_id = ?`, jobID); err != nil {
		return err
	}
	for _, s := range skills {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO job_skills (job_id, skill_name, skill_slug, is_required, weight) VALUES (?, ?, ?, ?, ?)`,
			jobID, s.SkillName, s.SkillSlug, s.IsRequired, s.Weight); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *JobRepository) GetSkills(ctx context.Context, jobID int64) ([]models.JobSkill, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, job_id, skill_name, skill_slug, is_required, weight FROM job_skills WHERE job_id = ?`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.JobSkill, 0)
	for rows.Next() {
		var s models.JobSkill
		if err := rows.Scan(&s.ID, &s.JobID, &s.SkillName, &s.SkillSlug, &s.IsRequired, &s.Weight); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// --- job certifications ---

func (r *JobRepository) ReplaceCertifications(ctx context.Context, jobID int64, certs []models.JobCertification) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM job_certifications WHERE job_id = ?`, jobID); err != nil {
		return err
	}
	for _, c := range certs {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO job_certifications (job_id, certification_name, certification_slug) VALUES (?, ?, ?)`,
			jobID, c.CertificationName, c.CertificationSlug); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *JobRepository) GetCertifications(ctx context.Context, jobID int64) ([]models.JobCertification, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, job_id, certification_name, certification_slug FROM job_certifications WHERE job_id = ?`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.JobCertification, 0)
	for rows.Next() {
		var c models.JobCertification
		if err := rows.Scan(&c.ID, &c.JobID, &c.CertificationName, &c.CertificationSlug); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// --- match weights ---

func (r *JobRepository) GetMatchWeights(ctx context.Context, jobID int64) (*models.MatchWeights, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, job_id, weight_skill, weight_experience, weight_education, weight_project, weight_certification
		 FROM job_match_weights WHERE job_id = ?`, jobID)
	var w models.MatchWeights
	err := row.Scan(&w.ID, &w.JobID, &w.WeightSkill, &w.WeightExperience, &w.WeightEducation, &w.WeightProject, &w.WeightCertification)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (r *JobRepository) UpdateMatchWeights(ctx context.Context, jobID int64, w models.MatchWeights) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE job_match_weights SET weight_skill = ?, weight_experience = ?, weight_education = ?,
		 weight_project = ?, weight_certification = ? WHERE job_id = ?`,
		w.WeightSkill, w.WeightExperience, w.WeightEducation, w.WeightProject, w.WeightCertification, jobID,
	)
	return err
}
