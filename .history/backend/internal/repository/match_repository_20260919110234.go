package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/resume-platform/backend/internal/models"
)

type MatchRepository struct {
	db *sql.DB
}

func NewMatchRepository(db *sql.DB) *MatchRepository {
	return &MatchRepository{db: db}
}

// Upsert stores (or replaces) the match result for an application, keeping
// matching reproducible: re-running the engine for the same application
// simply recalculates and overwrites this single row.
func (r *MatchRepository) Upsert(ctx context.Context, m *models.MatchResult) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO match_results
		    (application_id, skill_score, experience_score, education_score, project_score, certification_score,
		     overall_score, weights_used, explanation)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE
		    skill_score = VALUES(skill_score),
		    experience_score = VALUES(experience_score),
		    education_score = VALUES(education_score),
		    project_score = VALUES(project_score),
		    certification_score = VALUES(certification_score),
		    overall_score = VALUES(overall_score),
		    weights_used = VALUES(weights_used),
		    explanation = VALUES(explanation),
		    calculated_at = NOW()`,
		m.ApplicationID, m.SkillScore, m.ExperienceScore, m.EducationScore, m.ProjectScore, m.CertificationScore,
		m.OverallScore, m.WeightsUsed, m.Explanation,
	)
	return err
}

func (r *MatchRepository) FindByApplicationID(ctx context.Context, applicationID int64) (*models.MatchResult, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, application_id, skill_score, experience_score, education_score, project_score, certification_score,
		        overall_score, weights_used, explanation, calculated_at
		 FROM match_results WHERE application_id = ?`, applicationID)

	var m models.MatchResult
	err := row.Scan(&m.ID, &m.ApplicationID, &m.SkillScore, &m.ExperienceScore, &m.EducationScore, &m.ProjectScore,
		&m.CertificationScore, &m.OverallScore, &m.WeightsUsed, &m.Explanation, &m.CalculatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// RankedForJob returns every application for a job with its match score,
// ordered best-first -- this is the "company receives ranked candidates"
// requirement.
type RankedApplication struct {
	models.Application
	MatchResult *models.MatchResult
}

func (r *MatchRepository) RankedForJob(ctx context.Context, jobID int64) ([]RankedApplication, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT a.id, a.job_id, a.candidate_id, a.resume_id, a.status, a.applied_at, a.updated_at,
		        j.title, co.company_name, c.full_name,
		        mr.id, mr.skill_score, mr.experience_score, mr.education_score, mr.project_score,
		        mr.certification_score, mr.overall_score, mr.weights_used, mr.explanation, mr.calculated_at
		 FROM applications a
		 JOIN jobs j ON j.id = a.job_id
		 JOIN companies co ON co.id = j.company_id
		 JOIN candidates c ON c.id = a.candidate_id
		 LEFT JOIN match_results mr ON mr.application_id = a.id
		 WHERE a.job_id = ?
		 ORDER BY mr.overall_score DESC, a.applied_at ASC`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]RankedApplication, 0)
	for rows.Next() {
		var ra RankedApplication
		var mID sql.NullInt64
		var skillScore, expScore, eduScore, projScore, certScore, overallScore sql.NullFloat64
		var weightsUsed, explanation models.JSONMap
		var calculatedAt sql.NullTime

		if err := rows.Scan(&ra.ID, &ra.JobID, &ra.CandidateID, &ra.ResumeID, &ra.Status, &ra.AppliedAt, &ra.UpdatedAt,
			&ra.JobTitle, &ra.CompanyName, &ra.CandidateName,
			&mID, &skillScore, &expScore, &eduScore, &projScore, &certScore, &overallScore, &weightsUsed, &explanation, &calculatedAt); err != nil {
			return nil, err
		}

		if mID.Valid {
			ra.MatchResult = &models.MatchResult{
				ID: mID.Int64, ApplicationID: ra.ID,
				SkillScore: skillScore.Float64, ExperienceScore: expScore.Float64,
				EducationScore: eduScore.Float64, ProjectScore: projScore.Float64,
				CertificationScore: certScore.Float64, OverallScore: overallScore.Float64,
				WeightsUsed: weightsUsed, Explanation: explanation,
			}
			if calculatedAt.Valid {
				ra.MatchResult.CalculatedAt = calculatedAt.Time
			}
		}
		out = append(out, ra)
	}
	return out, rows.Err()
}
