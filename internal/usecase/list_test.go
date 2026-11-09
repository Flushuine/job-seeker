package usecase

import (
	"context"
	"github.com/aqil/ai-job-hunter/internal/domain"
	"github.com/aqil/ai-job-hunter/internal/provider/ollama"
	"testing"
)

type listRepo struct {
	Repository
	jobs    []domain.Job
	profile domain.Profile
	pref    domain.Preferences
}

func (r *listRepo) Profile(context.Context) (domain.Profile, error)         { return r.profile, nil }
func (r *listRepo) Preferences(context.Context) (domain.Preferences, error) { return r.pref, nil }
func (r *listRepo) Jobs(ctx context.Context, q domain.JobQuery) (domain.JobPage, error) {
	start := (q.Page - 1) * q.Limit
	if start > len(r.jobs) {
		start = len(r.jobs)
	}
	end := start + q.Limit
	if end > len(r.jobs) {
		end = len(r.jobs)
	}
	return domain.JobPage{Items: r.jobs[start:end], Total: int64(len(r.jobs))}, nil
}
func (r *listRepo) CachedAnalysis(context.Context, string, string) (*domain.Analysis, error) {
	return nil, nil
}
func TestLocationFilterBeforePagination(t *testing.T) {
	r := &listRepo{profile: domain.Profile{Skills: []string{"Go"}}, pref: domain.Preferences{Remote: []string{"hybrid", "remote"}, HardRemote: true, HardLocation: true, RequireLocationMatch: true, NearbyLocations: []string{"Jakarta", "Tangerang"}, Countries: []string{"Indonesia"}}, jobs: []domain.Job{{Base: domain.Base{ID: "far"}, Available: true, Remote: "hybrid", Location: "Singapore"}, {Base: domain.Base{ID: "near"}, Available: true, Remote: "hybrid", Location: "Jakarta"}, {Base: domain.Base{ID: "unknown"}, Available: true, Remote: "remote", Location: "Remote"}, {Base: domain.Base{ID: "remote-id"}, Available: true, Remote: "remote", Location: "Indonesia"}}}
	s := Service{Repo: r, AI: &ollama.Provider{Model: "fixture"}}
	for page, id := range map[int]string{1: "near", 2: "remote-id"} {
		got, e := s.ListJobs(context.Background(), domain.JobQuery{Page: page, Limit: 1})
		if e != nil || got.Total != 2 || len(got.Items) != 1 || got.Items[0].ID != id {
			t.Fatal("location filtering broke pagination", got, e)
		}
	}
	all, e := s.ListJobs(context.Background(), domain.JobQuery{Page: 1, Limit: 10, IncludeExcluded: true})
	if e != nil || all.Total != 4 {
		t.Fatal("excluded jobs must remain inspectable", all, e)
	}
	unknown := r.jobs[2]
	if MatchesSearch(unknown, r.profile, r.pref) {
		t.Fatal("unconfirmed location shown in strict shortlist")
	}
	r.pref.RequireLocationMatch = false
	if !MatchesSearch(unknown, r.profile, r.pref) {
		t.Fatal("unknown job should be investigatable when strict confirmation is off")
	}
}
func TestProfileSummaryInvalidatesCache(t *testing.T) {
	s := Service{AI: &ollama.Provider{Model: "fixture"}}
	p := domain.Profile{Skills: []string{"Go"}}
	first := s.Fingerprint(domain.Job{}, p, domain.Preferences{})
	p.Summary = "Production Go services, payments, mentoring"
	if first == s.Fingerprint(domain.Job{}, p, domain.Preferences{}) {
		t.Fatal("CV changes must invalidate analysis cache")
	}
}
func TestPreferredRolePrecedesUnknownTechnicalPerfectScore(t *testing.T) {
	r := &listRepo{profile: domain.Profile{Skills: []string{"Go"}}, pref: domain.Preferences{Roles: []string{"Backend Engineer"}, Remote: []string{"remote"}, Weights: map[string]float64{"technical": 30, "remote": 20}}, jobs: []domain.Job{{Base: domain.Base{ID: "unrelated"}, Available: true, Title: "Operations Specialist", Remote: "remote"}, {Base: domain.Base{ID: "backend"}, Available: true, Title: "Backend Engineer", Description: "Ruby", Remote: "remote"}}}
	s := Service{Repo: r, AI: &ollama.Provider{Model: "fixture"}}
	got, e := s.ListJobs(context.Background(), domain.JobQuery{Sort: "score", Page: 1, Limit: 10})
	if e != nil || got.Items[0].ID != "backend" {
		t.Fatal("unrelated role with sparse evidence incorrectly outranked engineering role", got, e)
	}
}
