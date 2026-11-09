package usecase

import (
	"context"
	"github.com/aqil/ai-job-hunter/internal/domain"
	"sort"
	"time"
)

// MatchesSearch separates an explicit mismatch from unconfirmed location data.
// Strict location confirmation hides unknowns from the default shortlist without
// inventing a negative score; the explorer can still show them for investigation.
func MatchesSearch(j domain.Job, p domain.Profile, pref domain.Preferences) bool {
	if !domain.Filter(j, p, pref, time.Now()).Eligible {
		return false
	}
	if pref.RequireLocationMatch {
		if domain.Geography(j, pref).Status != "match" {
			return false
		}
		if pref.HardRemote && (j.Remote == "" || j.Remote == "unknown") {
			return false
		}
	}
	return true
}

// Apply eligibility before pagination so counts and page boundaries reflect the
// user's full search constraints, rather than filtering only the current page.
func (s *Service) ListJobs(ctx context.Context, q domain.JobQuery) (domain.JobPage, error) {
	p, e := s.Repo.Profile(ctx)
	if e != nil {
		return domain.JobPage{}, e
	}
	pref, e := s.Repo.Preferences(ctx)
	if e != nil {
		return domain.JobPage{}, e
	}
	all := []domain.Job{}
	searchMatches := map[string]bool{}
	scan := q
	scan.Page = 1
	scan.Limit = 100
	if scan.Sort == "score" {
		scan.Sort = ""
	}
	for {
		batch, e := s.Repo.Jobs(ctx, scan)
		if e != nil {
			return domain.JobPage{}, e
		}
		for _, j := range batch.Items {
			matches := MatchesSearch(j, p, pref)
			if q.IncludeExcluded || matches {
				searchMatches[j.ID] = matches
				all = append(all, j)
			}
		}
		if scan.Page*scan.Limit >= int(batch.Total) || len(batch.Items) == 0 {
			break
		}
		scan.Page++
	}
	if q.Sort == "score" {
		for i := range all {
			if e = s.Enrich(ctx, &all[i], p, pref); e != nil {
				return domain.JobPage{}, e
			}
		}
		sort.SliceStable(all, func(i, j int) bool {
			// Confirmed eligibility precedes a high technical score on an excluded job.
			aEligible, bEligible := searchMatches[all[i].ID], searchMatches[all[j].ID]
			if aEligible != bEligible {
				return aEligible
			}
			roleMatch := func(j domain.Job) bool {
				for _, f := range j.Deterministic.Factors {
					if f.Name == "role" {
						return f.Status == "match"
					}
				}
				return false
			}
			ar, br := roleMatch(all[i]), roleMatch(all[j])
			if ar != br {
				return ar
			}
			at, bt := all[i].Deterministic.TechnicalScore != nil, all[j].Deterministic.TechnicalScore != nil
			if at != bt {
				return at
			}
			a, b := all[i].Ranking, all[j].Ranking
			if a.Score == nil {
				return false
			}
			if b.Score == nil {
				return true
			}
			if *a.Score == *b.Score {
				return a.Coverage > b.Coverage
			}
			return *a.Score > *b.Score
		})
	}
	start := (q.Page - 1) * q.Limit
	if start > len(all) {
		start = len(all)
	}
	end := start + q.Limit
	if end > len(all) {
		end = len(all)
	}
	items := all[start:end]
	if q.Sort != "score" {
		for i := range items {
			if e = s.Enrich(ctx, &items[i], p, pref); e != nil {
				return domain.JobPage{}, e
			}
		}
	}
	return domain.JobPage{Items: items, Total: int64(len(all)), Page: q.Page, Limit: q.Limit}, nil
}
