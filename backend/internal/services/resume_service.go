package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"

	"github.com/resume-platform/backend/internal/ai"
	"github.com/resume-platform/backend/internal/models"
	"github.com/resume-platform/backend/internal/repository"
	"github.com/resume-platform/backend/internal/resume"
	"github.com/resume-platform/backend/internal/storage"
	"github.com/resume-platform/backend/internal/utils"
	"github.com/rs/zerolog"
)

type ResumeService struct {
	resumes    *repository.ResumeRepository
	candidates *repository.CandidateRepository
	storage    storage.FileStorage
	aiProvider ai.Provider
	maxUpload  int64
	logger     zerolog.Logger
}

func NewResumeService(resumes *repository.ResumeRepository, candidates *repository.CandidateRepository, store storage.FileStorage, provider ai.Provider, maxUpload int64, logger zerolog.Logger) *ResumeService {
	return &ResumeService{resumes: resumes, candidates: candidates, storage: store, aiProvider: provider, maxUpload: maxUpload, logger: logger}
}

// Upload implements pipeline steps 1-3: Validate -> Store -> create the
// resume DB row in "uploaded" status. Parsing (steps 4-9) happens in
// ParseResume, called synchronously right after in the controller for this
// delivery -- see the doc comment on ParseResume for the async-readiness
// note required by the spec.
func (s *ResumeService) Upload(ctx context.Context, candidateID int64, fh *multipart.FileHeader) (*models.Resume, error) {
	if err := resume.ValidateUpload(fh, s.maxUpload); err != nil {
		return nil, utils.NewFileError(err.Error())
	}

	f, err := fh.Open()
	if err != nil {
		return nil, utils.NewFileError("could not open uploaded file")
	}
	defer f.Close()

	storageKey, err := s.storage.Save("resumes", fh.Filename, f)
	if err != nil {
		return nil, utils.NewInternalError(fmt.Errorf("save resume file: %w", err))
	}

	mimeType := fh.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	r := &models.Resume{
		CandidateID:      candidateID,
		OriginalFilename: fh.Filename,
		StoredFilename:   storageKey,
		FilePath:         s.storage.AbsolutePath(storageKey),
		FileSize:         uint32(fh.Size),
		MimeType:         mimeType,
		Status:           models.ResumeStatusUploaded,
	}
	id, err := s.resumes.Create(ctx, r)
	if err != nil {
		_ = s.storage.Delete(storageKey)
		return nil, utils.NewDatabaseError(err)
	}
	r.ID = id
	return r, nil
}

// ParseResume implements pipeline steps 4-9: extract text -> normalize ->
// AI structured extraction -> validate AI output -> save structured
// profile -> mark parsed.
//
// PRODUCTION NOTE (async-readiness, per spec "design the code so processing
// can be moved to a worker architecture without rewriting business logic"):
// this method takes only a resumeID and has no HTTP/request-scoped state,
// so it can be called either synchronously from the upload controller (as
// this delivery does, for simplicity and to keep the demo/test flow
// single-request) or dispatched onto a queue (SQS/Redis/etc.) consumed by a
// separate worker process calling this exact same method -- no business
// logic would need to change, only the caller.
func (s *ResumeService) ParseResume(ctx context.Context, resumeID int64) (*models.ResumeFull, error) {
	r, err := s.resumes.FindByID(ctx, resumeID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, utils.NewNotFound("resume")
	}
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}

	if err := s.resumes.UpdateStatus(ctx, resumeID, models.ResumeStatusProcessing, nil); err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	_ = s.resumes.IncrementParseAttempts(ctx, resumeID)

	rc, err := s.storage.Open(r.StoredFilename)
	if err != nil {
		return nil, s.failParse(ctx, resumeID, fmt.Errorf("open stored file: %w", err))
	}
	defer rc.Close()

	fileBytes, err := io.ReadAll(rc)
	if err != nil {
		return nil, s.failParse(ctx, resumeID, fmt.Errorf("read stored file: %w", err))
	}

	rawText, err := resume.ExtractText(r.MimeType, r.OriginalFilename, fileBytes)
	if err != nil {
		return nil, s.failParse(ctx, resumeID, err)
	}
	if err := s.resumes.SaveRawText(ctx, resumeID, rawText); err != nil {
		s.logger.Warn().Err(err).Msg("failed to persist raw resume text (non-fatal, continuing)")
	}

	extraction, err := s.aiProvider.ParseResume(ctx, rawText)
	if err != nil {
		return nil, s.failParse(ctx, resumeID, fmt.Errorf("AI extraction: %w", err))
	}
	if err := ai.ValidateResumeExtraction(extraction); err != nil {
		return nil, s.failParse(ctx, resumeID, fmt.Errorf("AI output failed validation: %w", err))
	}

	structured := extractionToResumeFull(extraction)
	if err := s.resumes.SaveParsedStructuredData(ctx, resumeID, structured); err != nil {
		return nil, s.failParse(ctx, resumeID, fmt.Errorf("persist structured data: %w", err))
	}

	full, err := s.resumes.GetFull(ctx, resumeID)
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}

	if err := s.candidates.SetActiveResume(ctx, r.CandidateID, resumeID); err != nil {
		s.logger.Warn().Err(err).Msg("failed to set active resume (non-fatal)")
	}

	return full, nil
}

func (s *ResumeService) failParse(ctx context.Context, resumeID int64, cause error) error {
	msg := cause.Error()
	_ = s.resumes.UpdateStatus(ctx, resumeID, models.ResumeStatusFailed, &msg)
	s.logger.Error().Err(cause).Int64("resume_id", resumeID).Msg("resume parsing failed")
	return utils.NewAIError("resume parsing failed: "+msg, cause)
}

func (s *ResumeService) GetFull(ctx context.Context, resumeID int64) (*models.ResumeFull, error) {
	full, err := s.resumes.GetFull(ctx, resumeID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, utils.NewNotFound("resume")
	}
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	return full, nil
}

func (s *ResumeService) ListByCandidate(ctx context.Context, candidateID int64) ([]models.Resume, error) {
	list, err := s.resumes.ListByCandidate(ctx, candidateID)
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	return list, nil
}

// UpdateStructuredData implements "allow the candidate to review and edit
// extracted information; the candidate's edited information becomes the
// authoritative structured profile" -- it writes directly over the
// AI-derived rows using the same persistence path as the AI pipeline, and
// (correctly) does NOT re-run parsing or re-mark the resume as processing.
func (s *ResumeService) UpdateStructuredData(ctx context.Context, resumeID int64, data *models.ResumeFull) (*models.ResumeFull, error) {
	if _, err := s.resumes.FindByID(ctx, resumeID); errors.Is(err, repository.ErrNotFound) {
		return nil, utils.NewNotFound("resume")
	} else if err != nil {
		return nil, utils.NewDatabaseError(err)
	}

	if err := s.resumes.SaveParsedStructuredData(ctx, resumeID, data); err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	return s.GetFull(ctx, resumeID)
}

func extractionToResumeFull(e *ai.ResumeExtraction) *models.ResumeFull {
	full := &models.ResumeFull{}

	for _, edu := range e.Education {
		full.Education = append(full.Education, models.ResumeEducation{
			Institution: edu.Institution,
			Degree:      strPtrOrNil(edu.Degree),
			Field:       strPtrOrNil(edu.Field),
			StartDate:   strPtrOrNil(edu.StartDate),
			EndDate:     strPtrOrNil(edu.EndDate),
			GPA:         strPtrOrNil(edu.GPA),
		})
	}

	for _, exp := range e.Experience {
		full.Experience = append(full.Experience, models.ResumeExperience{
			Company:          exp.Company,
			Role:             exp.Role,
			StartDate:        strPtrOrNil(exp.StartDate),
			EndDate:          strPtrOrNil(exp.EndDate),
			IsCurrent:        exp.IsCurrent,
			Description:      strPtrOrNil(exp.Description),
			Technologies:     toJSONList(exp.Technologies),
			Responsibilities: toJSONList(exp.Responsibilities),
		})
	}

	for _, p := range e.Projects {
		links := models.JSONMap{}
		for i, l := range p.Links {
			links[fmt.Sprintf("link_%d", i+1)] = l
		}
		full.Projects = append(full.Projects, models.ResumeProject{
			Name:         p.Name,
			Description:  strPtrOrNil(p.Description),
			Technologies: toJSONList(p.Technologies),
			Role:         strPtrOrNil(p.Role),
			Links:        links,
		})
	}

	for _, sk := range e.Skills {
		full.Skills = append(full.Skills, models.ResumeSkill{
			SkillName: sk.Name,
			SkillSlug: utils.CanonicalSkillSlug(sk.Name),
			Category:  models.SkillCategory(sk.Category),
		})
	}

	for _, c := range e.Certifications {
		full.Certifications = append(full.Certifications, models.ResumeCertification{
			Name:         c.Name,
			NameSlug:     utils.Slugify(c.Name),
			Issuer:       strPtrOrNil(c.Issuer),
			IssueDate:    strPtrOrNil(c.IssueDate),
			CredentialID: strPtrOrNil(c.CredentialID),
		})
	}

	return full
}

func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func toJSONList(items []string) models.JSONList {
	out := make(models.JSONList, len(items))
	for i, s := range items {
		out[i] = s
	}
	return out
}
