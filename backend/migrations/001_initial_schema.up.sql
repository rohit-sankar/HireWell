-- ============================================================================
-- AI Resume Parser & Intelligent Recruitment Platform
-- Initial normalized schema (MySQL 8.0)
-- ============================================================================

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 1;

-- ----------------------------------------------------------------------------
-- users: single auth table for all roles (candidate / company / admin)
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS users (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    email           VARCHAR(255) NOT NULL,
    password_hash   VARCHAR(255) NULL,                 -- NULL when account is Google-OAuth-only
    role            ENUM('candidate','company','admin') NOT NULL,
    google_id       VARCHAR(255) NULL,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    email_verified  BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uq_users_email (email),
    UNIQUE KEY uq_users_google_id (google_id),
    KEY idx_users_role (role)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ----------------------------------------------------------------------------
-- refresh_tokens: rotation-capable refresh token store (hashed, never plaintext)
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id     BIGINT UNSIGNED NOT NULL,
    token_hash  VARCHAR(255) NOT NULL,
    expires_at  DATETIME NOT NULL,
    revoked     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_refresh_tokens_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE KEY uq_refresh_token_hash (token_hash),
    KEY idx_refresh_tokens_user (user_id),
    KEY idx_refresh_tokens_expires (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ----------------------------------------------------------------------------
-- candidates
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS candidates (
    id                   BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id              BIGINT UNSIGNED NOT NULL,
    full_name            VARCHAR(255) NOT NULL,
    phone                VARCHAR(50) NULL,
    location             VARCHAR(255) NULL,
    links                JSON NULL,                    -- {linkedin, github, portfolio, ...}
    headline             VARCHAR(255) NULL,
    summary              TEXT NULL,
    active_resume_id     BIGINT UNSIGNED NULL,          -- points at resumes.id (the current authoritative resume)
    profile_completion   TINYINT UNSIGNED NOT NULL DEFAULT 0,
    created_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_candidates_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE KEY uq_candidates_user (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ----------------------------------------------------------------------------
-- companies
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS companies (
    id             BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id        BIGINT UNSIGNED NOT NULL,
    company_name   VARCHAR(255) NOT NULL,
    website        VARCHAR(255) NULL,
    description    TEXT NULL,
    logo_url       VARCHAR(500) NULL,
    industry       VARCHAR(150) NULL,
    company_size   VARCHAR(50) NULL,
    created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_companies_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE KEY uq_companies_user (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ----------------------------------------------------------------------------
-- resumes: one row per uploaded file + parsing lifecycle
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS resumes (
    id                 BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    candidate_id       BIGINT UNSIGNED NOT NULL,
    original_filename  VARCHAR(255) NOT NULL,
    stored_filename    VARCHAR(255) NOT NULL,           -- random safe filename on disk
    file_path          VARCHAR(500) NOT NULL,
    file_size          INT UNSIGNED NOT NULL,
    mime_type          VARCHAR(100) NOT NULL,
    status             ENUM('uploaded','processing','parsed','failed') NOT NULL DEFAULT 'uploaded',
    raw_text           LONGTEXT NULL,
    parse_error        TEXT NULL,
    parse_attempts     TINYINT UNSIGNED NOT NULL DEFAULT 0,
    uploaded_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    parsed_at          DATETIME NULL,
    CONSTRAINT fk_resumes_candidate FOREIGN KEY (candidate_id) REFERENCES candidates(id) ON DELETE CASCADE,
    KEY idx_resumes_candidate (candidate_id),
    KEY idx_resumes_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

ALTER TABLE candidates
    ADD CONSTRAINT fk_candidates_active_resume FOREIGN KEY (active_resume_id) REFERENCES resumes(id) ON DELETE SET NULL;

-- ----------------------------------------------------------------------------
-- resume_education
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS resume_education (
    id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    resume_id     BIGINT UNSIGNED NOT NULL,
    institution   VARCHAR(255) NOT NULL,
    degree        VARCHAR(255) NULL,
    field         VARCHAR(255) NULL,
    start_date    DATE NULL,
    end_date      DATE NULL,
    gpa           VARCHAR(20) NULL,
    sort_order    SMALLINT UNSIGNED NOT NULL DEFAULT 0,
    CONSTRAINT fk_resume_education_resume FOREIGN KEY (resume_id) REFERENCES resumes(id) ON DELETE CASCADE,
    KEY idx_resume_education_resume (resume_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ----------------------------------------------------------------------------
-- resume_experience
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS resume_experience (
    id                BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    resume_id         BIGINT UNSIGNED NOT NULL,
    company           VARCHAR(255) NOT NULL,
    role              VARCHAR(255) NOT NULL,
    start_date        DATE NULL,
    end_date          DATE NULL,
    is_current        BOOLEAN NOT NULL DEFAULT FALSE,
    description       TEXT NULL,
    technologies      JSON NULL,
    responsibilities  JSON NULL,
    sort_order        SMALLINT UNSIGNED NOT NULL DEFAULT 0,
    CONSTRAINT fk_resume_experience_resume FOREIGN KEY (resume_id) REFERENCES resumes(id) ON DELETE CASCADE,
    KEY idx_resume_experience_resume (resume_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ----------------------------------------------------------------------------
-- resume_projects
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS resume_projects (
    id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    resume_id     BIGINT UNSIGNED NOT NULL,
    name          VARCHAR(255) NOT NULL,
    description   TEXT NULL,
    technologies  JSON NULL,
    role          VARCHAR(255) NULL,
    links         JSON NULL,
    sort_order    SMALLINT UNSIGNED NOT NULL DEFAULT 0,
    CONSTRAINT fk_resume_projects_resume FOREIGN KEY (resume_id) REFERENCES resumes(id) ON DELETE CASCADE,
    KEY idx_resume_projects_resume (resume_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ----------------------------------------------------------------------------
-- resume_skills (normalized skill rows -> enables real matching queries)
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS resume_skills (
    id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    resume_id     BIGINT UNSIGNED NOT NULL,
    skill_name    VARCHAR(150) NOT NULL,
    skill_slug    VARCHAR(150) NOT NULL,   -- lowercase/normalized, used for matching joins
    category      ENUM('technical','programming_language','framework','database','cloud','tool','soft_skill') NOT NULL DEFAULT 'technical',
    CONSTRAINT fk_resume_skills_resume FOREIGN KEY (resume_id) REFERENCES resumes(id) ON DELETE CASCADE,
    KEY idx_resume_skills_resume (resume_id),
    KEY idx_resume_skills_slug (skill_slug)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ----------------------------------------------------------------------------
-- resume_certifications
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS resume_certifications (
    id             BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    resume_id      BIGINT UNSIGNED NOT NULL,
    name           VARCHAR(255) NOT NULL,
    name_slug      VARCHAR(255) NOT NULL,
    issuer         VARCHAR(255) NULL,
    issue_date     DATE NULL,
    credential_id  VARCHAR(150) NULL,
    CONSTRAINT fk_resume_certifications_resume FOREIGN KEY (resume_id) REFERENCES resumes(id) ON DELETE CASCADE,
    KEY idx_resume_certifications_resume (resume_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ----------------------------------------------------------------------------
-- jobs
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS jobs (
    id                      BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    company_id              BIGINT UNSIGNED NOT NULL,
    title                   VARCHAR(255) NOT NULL,
    description             TEXT NOT NULL,
    location                VARCHAR(255) NULL,
    employment_type         ENUM('full_time','part_time','contract','internship','remote') NOT NULL DEFAULT 'full_time',
    experience_min_years    TINYINT UNSIGNED NOT NULL DEFAULT 0,
    experience_max_years    TINYINT UNSIGNED NULL,
    education_requirement   VARCHAR(255) NULL,
    status                  ENUM('draft','published','closed','archived') NOT NULL DEFAULT 'draft',
    interview_question_count TINYINT UNSIGNED NOT NULL DEFAULT 5,
    created_at              DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at              DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    published_at            DATETIME NULL,
    closed_at               DATETIME NULL,
    CONSTRAINT fk_jobs_company FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE,
    KEY idx_jobs_company (company_id),
    KEY idx_jobs_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ----------------------------------------------------------------------------
-- job_skills: required vs preferred, with matching weight
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS job_skills (
    id           BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    job_id       BIGINT UNSIGNED NOT NULL,
    skill_name   VARCHAR(150) NOT NULL,
    skill_slug   VARCHAR(150) NOT NULL,
    is_required  BOOLEAN NOT NULL DEFAULT TRUE,     -- TRUE = required, FALSE = preferred
    weight       DECIMAL(4,2) NOT NULL DEFAULT 1.00, -- relative importance within its group
    CONSTRAINT fk_job_skills_job FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE,
    KEY idx_job_skills_job (job_id),
    KEY idx_job_skills_slug (skill_slug)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ----------------------------------------------------------------------------
-- job_certifications
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS job_certifications (
    id                    BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    job_id                BIGINT UNSIGNED NOT NULL,
    certification_name    VARCHAR(255) NOT NULL,
    certification_slug    VARCHAR(255) NOT NULL,
    CONSTRAINT fk_job_certifications_job FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE,
    KEY idx_job_certifications_job (job_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ----------------------------------------------------------------------------
-- job_match_weights: configurable scoring weights per job (documented, not magic numbers)
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS job_match_weights (
    id                   BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    job_id               BIGINT UNSIGNED NOT NULL,
    weight_skill         DECIMAL(4,3) NOT NULL DEFAULT 0.400,
    weight_experience    DECIMAL(4,3) NOT NULL DEFAULT 0.250,
    weight_education     DECIMAL(4,3) NOT NULL DEFAULT 0.100,
    weight_project       DECIMAL(4,3) NOT NULL DEFAULT 0.150,
    weight_certification DECIMAL(4,3) NOT NULL DEFAULT 0.100,
    CONSTRAINT fk_job_match_weights_job FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE,
    UNIQUE KEY uq_job_match_weights_job (job_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ----------------------------------------------------------------------------
-- applications
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS applications (
    id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    job_id        BIGINT UNSIGNED NOT NULL,
    candidate_id  BIGINT UNSIGNED NOT NULL,
    resume_id     BIGINT UNSIGNED NOT NULL,          -- snapshot: which resume version was applied with
    status        ENUM('applied','under_review','shortlisted','rejected','hired') NOT NULL DEFAULT 'applied',
    applied_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_applications_job FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE,
    CONSTRAINT fk_applications_candidate FOREIGN KEY (candidate_id) REFERENCES candidates(id) ON DELETE CASCADE,
    CONSTRAINT fk_applications_resume FOREIGN KEY (resume_id) REFERENCES resumes(id) ON DELETE RESTRICT,
    UNIQUE KEY uq_applications_job_candidate (job_id, candidate_id),
    KEY idx_applications_job (job_id),
    KEY idx_applications_candidate (candidate_id),
    KEY idx_applications_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ----------------------------------------------------------------------------
-- match_results: explainable, reproducible scoring output (see internal/matching)
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS match_results (
    id                   BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    application_id       BIGINT UNSIGNED NOT NULL,
    skill_score          DECIMAL(5,2) NOT NULL,
    experience_score     DECIMAL(5,2) NOT NULL,
    education_score      DECIMAL(5,2) NOT NULL,
    project_score        DECIMAL(5,2) NOT NULL,
    certification_score  DECIMAL(5,2) NOT NULL,
    overall_score         DECIMAL(5,2) NOT NULL,
    weights_used         JSON NOT NULL,        -- weight snapshot used for this calculation
    explanation          JSON NOT NULL,        -- matched/missing skills, breakdown detail
    calculated_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_match_results_application FOREIGN KEY (application_id) REFERENCES applications(id) ON DELETE CASCADE,
    UNIQUE KEY uq_match_results_application (application_id),
    KEY idx_match_results_overall (overall_score)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ----------------------------------------------------------------------------
-- interviews (schema included for full data model; service layer is a
-- documented stub in this vertical-slice delivery -- see README limitations)
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS interviews (
    id             BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    application_id BIGINT UNSIGNED NOT NULL,
    status         ENUM('pending','in_progress','completed') NOT NULL DEFAULT 'pending',
    started_at     DATETIME NULL,
    completed_at   DATETIME NULL,
    created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_interviews_application FOREIGN KEY (application_id) REFERENCES applications(id) ON DELETE CASCADE,
    UNIQUE KEY uq_interviews_application (application_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS interview_questions (
    id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    interview_id  BIGINT UNSIGNED NOT NULL,
    question_text TEXT NOT NULL,
    based_on      VARCHAR(255) NULL,     -- e.g. "job_skill:Go", "candidate_project:AI Resume Parser"
    order_index   SMALLINT UNSIGNED NOT NULL DEFAULT 0,
    CONSTRAINT fk_interview_questions_interview FOREIGN KEY (interview_id) REFERENCES interviews(id) ON DELETE CASCADE,
    KEY idx_interview_questions_interview (interview_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS interview_answers (
    id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    question_id   BIGINT UNSIGNED NOT NULL,
    answer_text   TEXT NOT NULL,
    submitted_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_interview_answers_question FOREIGN KEY (question_id) REFERENCES interview_questions(id) ON DELETE CASCADE,
    UNIQUE KEY uq_interview_answers_question (question_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS interview_evaluations (
    id                   BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    interview_id         BIGINT UNSIGNED NOT NULL,
    technical_accuracy   DECIMAL(5,2) NOT NULL,
    problem_solving      DECIMAL(5,2) NOT NULL,
    depth_of_knowledge   DECIMAL(5,2) NOT NULL,
    communication        DECIMAL(5,2) NOT NULL,
    relevance            DECIMAL(5,2) NOT NULL,
    overall_score        DECIMAL(5,2) NOT NULL,
    weights_used         JSON NOT NULL,
    strengths            JSON NULL,
    weaknesses           JSON NULL,
    recommendation       TEXT NULL,
    evaluated_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_interview_evaluations_interview FOREIGN KEY (interview_id) REFERENCES interviews(id) ON DELETE CASCADE,
    UNIQUE KEY uq_interview_evaluations_interview (interview_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ----------------------------------------------------------------------------
-- schema_migrations: tracks which migration files have been applied
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS schema_migrations (
    version     VARCHAR(50) NOT NULL PRIMARY KEY,
    applied_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
