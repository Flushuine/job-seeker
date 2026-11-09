package usecase

import (
	"context"
	"errors"
	"fmt"
	"github.com/aqil/ai-job-hunter/internal/domain"
	"github.com/aqil/ai-job-hunter/internal/matching"
	ai "github.com/aqil/ai-job-hunter/internal/provider/ollama"
	"log/slog"
	"sync"
	"time"
)

var ErrSearchExcluded = errors.New("job is excluded or has unconfirmed location eligibility")

type Repository interface {
	CVRepository
	QueueStore
	Profile(context.Context) (domain.Profile, error)
	Preferences(context.Context) (domain.Preferences, error)
	Companies(context.Context) ([]domain.Company, error)
	Company(context.Context, string) (domain.Company, error)
	Sources(context.Context) ([]domain.CompanySource, error)
	SaveSnapshot(context.Context, domain.CompanySource, []domain.Posting, time.Time) error
	EnsureCompany(context.Context, string) (domain.Company, error)
	Job(context.Context, string) (domain.Job, error)
	Jobs(context.Context, domain.JobQuery) (domain.JobPage, error)
	CachedAnalysis(context.Context, string, string) (*domain.Analysis, error)
	SaveAnalysis(context.Context, *domain.Analysis) error
	Evidence(context.Context, string) ([]domain.Evidence, error)
	Research(context.Context, string) (*domain.Research, error)
	SaveResearch(context.Context, *domain.Research, []domain.Evidence) error
	SaveRun(context.Context, *domain.DiscoveryRun) error
	Runs(context.Context) ([]domain.DiscoveryRun, error)
	WithLock(context.Context, int64, func() error) error
}
type Service struct {
	Repo       Repository
	Providers  map[string]domain.JobProvider
	AI         domain.AIProvider
	Researcher domain.ResearchProvider
	AIEnabled  bool

	Log *slog.Logger
}

func (s *Service) Discover(ctx context.Context) (domain.DiscoveryRun, error) {
	r := domain.DiscoveryRun{Status: "running", Errors: []string{}}
	e := s.Repo.WithLock(ctx, 735011, func() error {
		if e := s.Repo.SaveRun(ctx, &r); e != nil {
			return e
		}
		sources, e := s.Repo.Sources(ctx)
		if e != nil {
			return e
		}
		var mu sync.Mutex
		sem := make(chan struct{}, 3)
		var wg sync.WaitGroup
		for _, source := range sources {
			if !source.Enabled || (source.Provider == "himalayas" && source.LastFetchedAt != nil && time.Since(*source.LastFetchedAt) < 24*time.Hour) {
				continue
			}
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				break
			}
			if ctx.Err() != nil {
				break
			}
			wg.Add(1)
			go func(src domain.CompanySource) {
				defer wg.Done()
				defer func() { <-sem }()
				provider, ok := s.Providers[src.Provider]
				var err error
				count := 0
				if !ok {
					err = fmt.Errorf("unsupported provider")
				} else {
					company, e := s.Repo.Company(ctx, src.CompanyID)
					err = e
					if err == nil {
						var postings []domain.Posting
						postings, err = provider.Fetch(ctx, src)
						if err == nil {
							for i := range postings {
								if src.Provider == "himalayas" {
									actual, e := s.Repo.EnsureCompany(ctx, postings[i].Job.CompanyName)
									if e != nil {
										err = e
										break
									}
									postings[i].Job.CompanyID = actual.ID
								} else {
									postings[i].Job.CompanyName = company.Name
								}
							}
							if err == nil {
								err = s.Repo.SaveSnapshot(ctx, src, postings, time.Now().UTC())
							}
							count = len(postings)
						}
					}
				}
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					r.Errors = append(r.Errors, fmt.Sprintf("%s/%s: %v", src.Provider, src.Board, err))
					s.Log.Warn("board discovery failed", "provider", src.Provider, "board", src.Board, "error", err)
				} else {
					r.JobsSeen += count
				}
			}(source)
		}
		wg.Wait()
		r.Status = "completed"
		if len(r.Errors) > 0 {
			r.Status = "partial_failure"
		}
		if ctx.Err() != nil {
			r.Status = "cancelled"
			r.Errors = append(r.Errors, ctx.Err().Error())
		}
		now := time.Now().UTC()
		r.FinishedAt = &now
		saveCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		return s.Repo.SaveRun(saveCtx, &r)
	})
	return r, e
}
func (s *Service) Fingerprint(j domain.Job, p domain.Profile, pref domain.Preferences) string {
	return domain.Hash([]any{j.Fingerprint, p.Name, p.YearsExperience, p.Location, p.Skills, p.Summary, p.CVSource, p.CVFingerprint, pref, s.AI.ModelID(), ai.PromptVersion})
}
func (s *Service) Analyse(ctx context.Context, id string) (*domain.Analysis, error) {
	var result *domain.Analysis
	e := s.Repo.WithLock(ctx, 735013, func() error {
		j, e := s.Repo.Job(ctx, id)
		if e != nil {
			return e
		}
		p, e := s.Repo.Profile(ctx)
		if e != nil {
			return e
		}
		pref, e := s.Repo.Preferences(ctx)
		if e != nil {
			return e
		}
		if !MatchesSearch(j, p, pref) {
			return ErrSearchExcluded
		}
		fp := s.Fingerprint(j, p, pref)
		result, e = s.Repo.CachedAnalysis(ctx, id, fp)
		if e != nil || result != nil {
			return e
		}
		if !s.AIEnabled || !pref.AIEnabled {
			return fmt.Errorf("local AI is disabled; deterministic matching remains available")
		}
		m, in, out, e := s.AI.Analyse(ctx, j, p, pref)
		if e != nil {
			return e
		}
		result = &domain.Analysis{JobID: id, Model: pref.OllamaModel, PromptVersion: ai.PromptVersion, InputFingerprint: fp, Result: m, InputTokens: in, OutputTokens: out}
		return s.Repo.SaveAnalysis(ctx, result)
	})
	return result, e
}
func (s *Service) Enrich(ctx context.Context, j *domain.Job, p domain.Profile, pref domain.Preferences) error {
	a, e := s.Repo.CachedAnalysis(ctx, j.ID, s.Fingerprint(*j, p, pref))
	if e != nil {
		return e
	}
	j.Analysis = a
	d := matching.Evaluate(*j, p, pref)
	j.Deterministic = &d
	eligible := domain.Filter(*j, p, pref, time.Now())
	j.Eligibility = &eligible
	r := domain.Rank(*j, pref, a)
	j.Ranking = &r
	hours := domain.WorkingHours(*j, p, time.Now())
	j.WorkHours = &hours
	return nil
}
func (s *Service) Research(ctx context.Context, id string) (*domain.Research, error) {
	var r *domain.Research
	e := s.Repo.WithLock(ctx, 735014, func() error {
		c, e := s.Repo.Company(ctx, id)
		if e != nil {
			return e
		}
		r, e = s.Repo.Research(ctx, id)
		if e != nil {
			return e
		}
		if r != nil && r.ExpiresAt.After(time.Now()) {
			return nil
		}
		ev, errs := s.Researcher.Research(ctx, c)
		r = &domain.Research{CompanyID: id, Status: "Insufficient Evidence", Summary: "Public source claims require review. No workplace score is inferred from feed entries.", Errors: errs, ExpiresAt: time.Now().Add(24 * time.Hour)}
		if len(c.ResearchURLs) == 0 {
			r.Summary = "No public research feeds configured. Add permitted RSS/Atom URLs in company settings."
		}
		return s.Repo.SaveResearch(ctx, r, ev)
	})
	return r, e
}
func (s *Service) Automate(ctx context.Context) {
	pref, e := s.Repo.Preferences(ctx)
	if e != nil {
		s.Log.Error("load worker preferences", "error", e)
		return
	}
	companies, e := s.Repo.Companies(ctx)
	if e != nil {
		s.Log.Error("load companies", "error", e)
	} else {
		for _, c := range companies {
			if ctx.Err() != nil {
				return
			}
			if len(c.ResearchURLs) > 0 {
				if _, e := s.Research(ctx, c.ID); e != nil {
					s.Log.Warn("company research", "company", c.ID, "error", e)
				}
			}
		}
	}
	if !pref.AutoAnalyse || !pref.AIEnabled || !s.AIEnabled {
		return
	}
	for page := 1; ; page++ {
		jobs, e := s.Repo.Jobs(ctx, domain.JobQuery{Page: page, Limit: 50})
		if e != nil {
			s.Log.Error("load jobs for matching", "error", e)
			return
		}
		p, e := s.Repo.Profile(ctx)
		if e != nil {
			s.Log.Error("load profile", "error", e)
			return
		}
		for _, j := range jobs.Items {
			if ctx.Err() != nil {
				return
			}
			elig := domain.Filter(j, p, pref, time.Now())
			if !MatchesSearch(j, p, pref) || len(elig.Preferences) > 0 {
				continue
			}
			if _, e := s.Enqueue(ctx, "job", j.ID); e != nil {
				s.Log.Warn("automatic analysis stopped", "error", e)
				return
			}
		}
		if page*50 >= int(jobs.Total) {
			return
		}
	}
}
