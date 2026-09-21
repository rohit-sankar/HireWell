package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/resume-platform/backend/internal/models"
)

type ResumeRepository struct {
	db *sql.DB
}

func NewResumeRepository(db *sql.DB) *ResumeRepository {
	return &ResumeRepository{db: db}
}

func (r *ResumeRepository) Create(ctx context.Context, res *models.Resume) (int64, error) {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO resumes (candidate_id, original_filename, stored_filename, file_path, file_size, mime_type, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		res.CandidateID, res.OriginalFilename, res.StoredFilename, res.FilePath, res.FileSize, res.MimeType, res.Status,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *ResumeRepository) FindByID(ctx context.Context, id int64) (*models.Resume, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, candidate_id, original_filename, stored_filename, file_path, file_size, mime_type, status,
		        raw_text, parse_error, parse_attempts, uploaded_at, parsed_at
		 FROM resumes WHERE id = ?`, id)
	return scanResume(row)
}

func scanResume(row *sql.Row) (*models.Resume, error) {
	var res models.Resume
	err := row.Scan(&res.ID, &res.CandidateID, &res.OriginalFilename, &res.StoredFilename, &res.FilePath,
		&res.FileSize, &res.MimeType, &res.Status, &res.RawText, &res.ParseError, &res.ParseAttempts,
		&res.UploadedAt, &res.ParsedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *ResumeRepository) ListByCandidate(ctx context.Context, candidateID int64) ([]models.Resume, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, candidate_id, original_filename, stored_filename, file_path, file_size, mime_type, status,
		        raw_text, parse_error, parse_attempts, uploaded_at, parsed_at
		 FROM resumes WHERE candidate_id = ? ORDER BY uploaded_at DESC`, candidateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.Resume, 0)
	for rows.Next() {
		var res models.Resume
		if err := rows.Scan(&res.ID, &res.CandidateID, &res.OriginalFilename, &res.StoredFilename, &res.FilePath,
			&res.FileSize, &res.MimeType, &res.Status, &res.RawText, &res.ParseError, &res.ParseAttempts,
			&res.UploadedAt, &res.ParsedAt); err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, rows.Err()
}

func (r *ResumeRepository) UpdateStatus(ctx context.Context, id int64, status models.ResumeStatus, parseError *string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE resumes SET status = ?, parse_error = ? WHERE id = ?`, status, parseError, id)
	return err
}

func (r *ResumeRepository) IncrementParseAttempts(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE resumes SET parse_attempts = parse_attempts + 1 WHERE id = ?`, id)
	return err
}

func (r *ResumeRepository) SaveRawText(ctx context.Context, id int64, text string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE resumes SET raw_text = ? WHERE id = ?`, text, id)
	return err
}

// SaveParsedStructuredData persists the full structured extraction result in
// a single transaction: it wipes any prior structured rows for this resume
// (in case of re-parse) and inserts the new ones, then marks the resume
// parsed. This keeps "resume row + all its child rows" atomic.
func (r *ResumeRepository) SaveParsedStructuredData(ctx context.Context, resumeID int64, data *models.ResumeFull) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	tables := []string{"resume_education", "resume_experience", "resume_projects", "resume_skills", "resume_certifications"}
	for _, t := range tables {
		if _, err := tx.ExecContext(ctx, "DELETE FROM "+t+" WHERE resume_id = ?", resumeID); err != nil {
			return err
		}
	}

	for i, e := range data.Education {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO resume_education (resume_id, institution, degree, field, start_date, end_date, gpa, sort_order)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			resumeID, e.Institution, e.Degree, e.Field, e.StartDate, e.EndDate, e.GPA, i); err != nil {
			return err
		}
	}

	for i, e := range data.Experience {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO resume_experience (resume_id, company, role, start_date, end_date, is_current, description, technologies, responsibilities, sort_order)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			resumeID, e.Company, e.Role, e.StartDate, e.EndDate, e.IsCurrent, e.Description, e.Technologies, e.Responsibilities, i); err != nil {
			return err
		}
	}

	for i, p := range data.Projects {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO resume_projects (resume_id, name, description, technologies, role, links, sort_order)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			resumeID, p.Name, p.Description, p.Technologies, p.Role, p.Links, i); err != nil {
			return err
		}
	}

	for _, s := range data.Skills {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO resume_skills (resume_id, skill_name, skill_slug, category) VALUES (?, ?, ?, ?)`,
			resumeID, s.SkillName, s.SkillSlug, s.Category); err != nil {
			return err
		}
	}

	for _, cert := range data.Certifications {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO resume_certifications (resume_id, name, name_slug, issuer, issue_date, credential_id)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			resumeID, cert.Name, cert.NameSlug, cert.Issuer, cert.IssueDate, cert.CredentialID); err != nil {
			return err
		}
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE resumes SET status = ?, parsed_at = NOW() WHERE id = ?`, models.ResumeStatusParsed, resumeID); err != nil {
		return err
	}

	return tx.Commit()
}

// GetFull loads a resume plus all structured child records -- what the
// candidate reviews/edits and what the matching engine reads from.
func (r *ResumeRepository) GetFull(ctx context.Context, resumeID int64) (*models.ResumeFull, error) {
	res, err := r.FindByID(ctx, resumeID)
	if err != nil {
		return nil, err
	}
	full := &models.ResumeFull{Resume: *res}

	eduRows, err := r.db.QueryContext(ctx,
		`SELECT id, resume_id, institution, degree, field, start_date, end_date, gpa, sort_order
		 FROM resume_education WHERE resume_id = ? ORDER BY sort_order`, resumeID)
	if err != nil {
		return nil, err
	}
	for eduRows.Next() {
		var e models.ResumeEducation
		if err := eduRows.Scan(&e.ID, &e.ResumeID, &e.Institution, &e.Degree, &e.Field, &e.StartDate, &e.EndDate, &e.GPA, &e.SortOrder); err != nil {
			eduRows.Close()
			return nil, err
		}
		full.Education = append(full.Education, e)
	}
	eduRows.Close()

	expRows, err := r.db.QueryContext(ctx,
		`SELECT id, resume_id, company, role, start_date, end_date, is_current, description, technologies, responsibilities, sort_order
		 FROM resume_experience WHERE resume_id = ? ORDER BY sort_order`, resumeID)
	if err != nil {
		return nil, err
	}
	for expRows.Next() {
		var e models.ResumeExperience
		if err := expRows.Scan(&e.ID, &e.ResumeID, &e.Company, &e.Role, &e.StartDate, &e.EndDate, &e.IsCurrent,
			&e.Description, &e.Technologies, &e.Responsibilities, &e.SortOrder); err != nil {
			expRows.Close()
			return nil, err
		}
		full.Experience = append(full.Experience, e)
	}
	expRows.Close()

	projRows, err := r.db.QueryContext(ctx,
		`SELECT id, resume_id, name, description, technologies, role, links, sort_order
		 FROM resume_projects WHERE resume_id = ? ORDER BY sort_order`, resumeID)
	if err != nil {
		return nil, err
	}
	for projRows.Next() {
		var p models.ResumeProject
		if err := projRows.Scan(&p.ID, &p.ResumeID, &p.Name, &p.Description, &p.Technologies, &p.Role, &p.Links, &p.SortOrder); err != nil {
			projRows.Close()
			return nil, err
		}
		full.Projects = append(full.Projects, p)
	}
	projRows.Close()

	skillRows, err := r.db.QueryContext(ctx,
		`SELECT id, resume_id, skill_name, skill_slug, category FROM resume_skills WHERE resume_id = ?`, resumeID)
	if err != nil {
		return nil, err
	}
	for skillRows.Next() {
		var s models.ResumeSkill
		if err := skillRows.Scan(&s.ID, &s.ResumeID, &s.SkillName, &s.SkillSlug, &s.Category); err != nil {
			skillRows.Close()
			return nil, err
		}
		full.Skills = append(full.Skills, s)
	}
	skillRows.Close()

	certRows, err := r.db.QueryContext(ctx,
		`SELECT id, resume_id, name, name_slug, issuer, issue_date, credential_id FROM resume_certifications WHERE resume_id = ?`, resumeID)
	if err != nil {
		return nil, err
	}
	for certRows.Next() {
		var cert models.ResumeCertification
		if err := certRows.Scan(&cert.ID, &cert.ResumeID, &cert.Name, &cert.NameSlug, &cert.Issuer, &cert.IssueDate, &cert.CredentialID); err != nil {
			certRows.Close()
			return nil, err
		}
		full.Certifications = append(full.Certifications, cert)
	}
	certRows.Close()

	return full, nil
}
