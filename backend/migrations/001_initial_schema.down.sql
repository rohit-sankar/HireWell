SET FOREIGN_KEY_CHECKS = 0;

DROP TABLE IF EXISTS interview_evaluations;
DROP TABLE IF EXISTS interview_answers;
DROP TABLE IF EXISTS interview_questions;
DROP TABLE IF EXISTS interviews;
DROP TABLE IF EXISTS match_results;
DROP TABLE IF EXISTS applications;
DROP TABLE IF EXISTS job_match_weights;
DROP TABLE IF EXISTS job_certifications;
DROP TABLE IF EXISTS job_skills;
DROP TABLE IF EXISTS jobs;
DROP TABLE IF EXISTS resume_certifications;
DROP TABLE IF EXISTS resume_skills;
DROP TABLE IF EXISTS resume_projects;
DROP TABLE IF EXISTS resume_experience;
DROP TABLE IF EXISTS resume_education;
ALTER TABLE candidates DROP FOREIGN KEY fk_candidates_active_resume;
DROP TABLE IF EXISTS resumes;
DROP TABLE IF EXISTS companies;
DROP TABLE IF EXISTS candidates;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS schema_migrations;

SET FOREIGN_KEY_CHECKS = 1;
