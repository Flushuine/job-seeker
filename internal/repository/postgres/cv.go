package postgres

import (
	"context"
	"errors"
	"github.com/aqil/ai-job-hunter/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

func (s *Store) CreateCVUpload(ctx context.Context, doc *domain.CVUpload) (domain.AnalysisTask, error) {
	var task domain.AnalysisTask
	e := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if e := tx.Exec("SELECT pg_advisory_xact_lock(735016)").Error; e != nil {
			return e
		}
		var p domain.Profile
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&p, "id = ?", ProfileID).Error; e != nil {
			return e
		}
		doc.Base = base("")
		doc.ProfileID = ProfileID
		doc.BaseProfileFingerprint = domain.ProfileContentFingerprint(p)
		if e := tx.Create(doc).Error; e != nil {
			return e
		}
		if e := tx.Model(&domain.Profile{}).Where("id = ?", ProfileID).Update("latest_cv_upload_id", doc.ID).Error; e != nil {
			return e
		}
		var e error
		task, e = (&Store{DB: tx}).Enqueue(ctx, "cv", doc.ID, 20)
		return e
	})
	return task, e
}
func (s *Store) CVUpload(ctx context.Context, id string) (domain.CVUpload, error) {
	var doc domain.CVUpload
	e := s.DB.WithContext(ctx).First(&doc, "id = ? AND profile_id = ?", id, ProfileID).Error
	return doc, e
}
func (s *Store) LatestCV(ctx context.Context) (domain.CVStatus, error) {
	status := domain.CVStatus{}
	p, e := s.Profile(ctx)
	if e != nil {
		return status, e
	}
	if p.LatestCVUploadID == nil {
		return status, nil
	}
	doc, e := s.CVUpload(ctx, *p.LatestCVUploadID)
	if e != nil {
		return status, e
	}
	status.Upload = &doc
	var task domain.AnalysisTask
	e = s.DB.WithContext(ctx).Where("kind = ? AND resource_id = ?", "cv", doc.ID).First(&task).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return status, nil
	}
	if e != nil {
		return status, e
	}
	status.Task = &task
	return status, nil
}
func (s *Store) CachedCV(ctx context.Context, doc domain.CVUpload) (*domain.CVAnalysis, error) {
	var cached domain.CVUpload
	e := s.DB.WithContext(ctx).Where("profile_id = ? AND fingerprint = ? AND model = ? AND prompt_version = ? AND ai_context = ? AND analysis IS NOT NULL", ProfileID, doc.Fingerprint, doc.Model, doc.PromptVersion, doc.AIContext).Order("created_at DESC").First(&cached).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return cached.Analysis, e
}
func (s *Store) ApplyCV(ctx context.Context, id string, analysis domain.CVAnalysis) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if e := domain.ValidateCVAnalysis(analysis); e != nil {
			return e
		}
		var p domain.Profile
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&p, "id = ?", ProfileID).Error; e != nil {
			return e
		}
		var doc domain.CVUpload
		if e := tx.First(&doc, "id = ? AND profile_id = ?", id, ProfileID).Error; e != nil {
			return e
		}
		doc.Analysis = &analysis
		if p.LatestCVUploadID == nil || *p.LatestCVUploadID != id {
			return tx.Model(&doc).Select("Analysis").Updates(&doc).Error
		}
		if p.CVFingerprint == doc.Fingerprint && doc.AppliedAt != nil {
			return nil
		}
		if domain.ProfileContentFingerprint(p) != doc.BaseProfileFingerprint {
			return domain.ErrProfileChanged
		}
		if analysis.Name != "" {
			p.Name = analysis.Name
		}
		if analysis.Location != "" {
			p.Location = analysis.Location
		}
		if analysis.YearsExperience != nil {
			p.YearsExperience = *analysis.YearsExperience
		}
		p.Summary = analysis.Summary
		p.Skills = analysis.Skills
		p.CVSource = doc.Filename
		if e := (&Store{DB: tx}).SaveProfile(ctx, &p); e != nil {
			return e
		}
		if e := tx.Model(&domain.Profile{}).Where("id = ?", ProfileID).Update("cv_fingerprint", doc.Fingerprint).Error; e != nil {
			return e
		}
		now := time.Now().UTC()
		doc.AppliedAt = &now
		return tx.Model(&doc).Select("Analysis", "AppliedAt").Updates(&doc).Error
	})
}
