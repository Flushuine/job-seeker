package usecase

import (
	"context"
	"errors"
	"github.com/aqil/ai-job-hunter/internal/domain"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"
)

type discoveryRepo struct {
	Repository
	mu        sync.Mutex
	snapshots []string
	runs      []domain.DiscoveryRun
}

func (r *discoveryRepo) WithLock(ctx context.Context, key int64, fn func() error) error { return fn() }
func (r *discoveryRepo) SaveRun(ctx context.Context, v *domain.DiscoveryRun) error {
	r.runs = append(r.runs, *v)
	return nil
}
func (r *discoveryRepo) Sources(context.Context) ([]domain.CompanySource, error) {
	return []domain.CompanySource{{Base: domain.Base{ID: "good"}, CompanyID: "company", Provider: "good", Enabled: true}, {Base: domain.Base{ID: "bad"}, CompanyID: "company", Provider: "bad", Enabled: true}}, nil
}
func (r *discoveryRepo) Company(context.Context, string) (domain.Company, error) {
	return domain.Company{Name: "Fixture company"}, nil
}
func (r *discoveryRepo) SaveSnapshot(_ context.Context, s domain.CompanySource, _ []domain.Posting, _ time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.snapshots = append(r.snapshots, s.ID)
	return nil
}

type providerFunc func(context.Context, domain.CompanySource) ([]domain.Posting, error)

func (f providerFunc) Fetch(c context.Context, s domain.CompanySource) ([]domain.Posting, error) {
	return f(c, s)
}
func TestDiscoveryFailureIsolation(t *testing.T) {
	r := &discoveryRepo{}
	s := Service{Repo: r, Log: slog.New(slog.NewTextHandler(io.Discard, nil)), Providers: map[string]domain.JobProvider{"good": providerFunc(func(context.Context, domain.CompanySource) ([]domain.Posting, error) {
		return []domain.Posting{{ExternalID: "fixture"}}, nil
	}), "bad": providerFunc(func(context.Context, domain.CompanySource) ([]domain.Posting, error) {
		return nil, errors.New("upstream temporarily unavailable")
	})}}
	run, e := s.Discover(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	if run.Status != "partial_failure" || run.JobsSeen != 1 || len(run.Errors) != 1 || len(r.snapshots) != 1 || r.snapshots[0] != "good" {
		t.Fatal(run, r.snapshots)
	}
	if len(r.runs) != 2 || run.FinishedAt == nil {
		t.Fatal("execution was not recorded")
	}
}
