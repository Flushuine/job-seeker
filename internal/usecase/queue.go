package usecase

import (
	"context"
	"fmt"
	"github.com/aqil/ai-job-hunter/internal/domain"
)

type QueueStore interface {
	Enqueue(context.Context, string, string, int) (domain.AnalysisTask, error)
	SaveInterpretation(context.Context, string, map[string]any) error
}

func (s *Service) Enqueue(ctx context.Context, kind, id string) (domain.AnalysisTask, error) {
	pref, e := s.Repo.Preferences(ctx)
	if e != nil {
		return domain.AnalysisTask{}, e
	}
	if !s.AIEnabled || !pref.AIEnabled {
		return domain.AnalysisTask{}, fmt.Errorf("local AI is disabled; deterministic analysis remains available")
	}
	if kind == "job" {
		j, e := s.Repo.Job(ctx, id)
		if e != nil {
			return domain.AnalysisTask{}, e
		}
		profile, e := s.Repo.Profile(ctx)
		if e != nil {
			return domain.AnalysisTask{}, e
		}
		if !MatchesSearch(j, profile, pref) {
			return domain.AnalysisTask{}, ErrSearchExcluded
		}
	} else {
		if _, e = s.Repo.Company(ctx, id); e != nil {
			return domain.AnalysisTask{}, e
		}
	}
	return s.Repo.Enqueue(ctx, kind, id, 20)
}
func (s *Service) InterpretCompany(ctx context.Context, id string) error {
	pref, e := s.Repo.Preferences(ctx)
	if e != nil {
		return e
	}
	if !s.AIEnabled || !pref.AIEnabled {
		return fmt.Errorf("local AI disabled")
	}
	if _, e := s.Research(ctx, id); e != nil {
		return e
	}
	ev, e := s.Repo.Evidence(ctx, id)
	if e != nil {
		return e
	}
	if len(ev) == 0 {
		return fmt.Errorf("Insufficient Evidence: add sources before analysis")
	}
	type interpreter interface {
		Interpret(context.Context, []domain.Evidence, domain.Preferences) (map[string]any, error)
	}
	p, ok := s.AI.(interpreter)
	if !ok {
		return fmt.Errorf("AI provider cannot interpret evidence")
	}
	return s.Repo.WithLock(ctx, 735013, func() error {
		v, e := p.Interpret(ctx, ev, pref)
		if e != nil {
			return e
		}
		v["evidence_fingerprint"] = domain.Hash(ev)
		v["model"] = pref.OllamaModel
		return s.Repo.SaveInterpretation(ctx, id, v)
	})
}
