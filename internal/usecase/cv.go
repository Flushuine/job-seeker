package usecase

import (
	"context"
	"fmt"
	"github.com/aqil/ai-job-hunter/internal/domain"
	"github.com/aqil/ai-job-hunter/internal/matching"
	"github.com/aqil/ai-job-hunter/internal/provider/ollama"
)

type CVRepository interface {
	CreateCVUpload(context.Context, *domain.CVUpload) (domain.AnalysisTask, error)
	CVUpload(context.Context, string) (domain.CVUpload, error)
	LatestCV(context.Context) (domain.CVStatus, error)
	CachedCV(context.Context, domain.CVUpload) (*domain.CVAnalysis, error)
	ApplyCV(context.Context, string, domain.CVAnalysis) error
}

func (s *Service) UploadCV(ctx context.Context, filename, text string) (domain.CVStatus, error) {
	pref, e := s.Repo.Preferences(ctx)
	if e != nil {
		return domain.CVStatus{}, e
	}
	if !s.AIEnabled || !pref.AIEnabled {
		return domain.CVStatus{}, domain.ErrAIUnavailable
	}
	doc := domain.CVUpload{Filename: filename, ExtractedText: text, Fingerprint: domain.Hash(text), Model: pref.OllamaModel, PromptVersion: ollama.CVPromptVersion, AIContext: pref.AIContext}
	if doc.Model == "" {
		doc.Model = s.AI.ModelID()
	}
	task, e := s.Repo.CreateCVUpload(ctx, &doc)
	if e != nil {
		return domain.CVStatus{}, e
	}
	return domain.CVStatus{Upload: &doc, Task: &task}, nil
}
func (s *Service) AnalyseCV(ctx context.Context, id string) error {
	return s.Repo.WithLock(ctx, 735013, func() error {
		doc, e := s.Repo.CVUpload(ctx, id)
		if e != nil {
			return e
		}
		p, e := s.Repo.Profile(ctx)
		if e != nil {
			return e
		}
		if p.LatestCVUploadID == nil || *p.LatestCVUploadID != id {
			return nil
		}
		if doc.AppliedAt != nil {
			return nil
		}
		pref, e := s.Repo.Preferences(ctx)
		if e != nil {
			return e
		}
		if !s.AIEnabled || !pref.AIEnabled {
			return domain.ErrAIUnavailable
		}
		analysis, e := s.Repo.CachedCV(ctx, doc)
		if e != nil {
			return e
		}
		if analysis == nil {
			extractor, ok := s.AI.(interface {
				AnalyseCV(context.Context, string, domain.Preferences) (domain.CVAnalysis, error)
			})
			if !ok {
				return fmt.Errorf("local AI provider does not support CV extraction")
			}
			pref.OllamaModel = doc.Model
			pref.AIContext = doc.AIContext
			result, e := extractor.AnalyseCV(ctx, doc.ExtractedText, pref)
			if e != nil {
				return e
			}
			analysis = &result
		}
		grounded, e := matching.GroundCV(*analysis, doc.ExtractedText)
		if e != nil {
			return e
		}
		return s.Repo.ApplyCV(ctx, id, grounded)
	})
}
