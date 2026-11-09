package usecase

import (
	"context"
	"errors"
	"github.com/aqil/ai-job-hunter/internal/domain"
	"testing"
)

type cvRepo struct {
	Repository
	doc     domain.CVUpload
	profile domain.Profile
	cached  *domain.CVAnalysis
	applied bool
	enabled bool
}

func (r *cvRepo) WithLock(_ context.Context, _ int64, fn func() error) error { return fn() }
func (r *cvRepo) CVUpload(context.Context, string) (domain.CVUpload, error)  { return r.doc, nil }
func (r *cvRepo) Profile(context.Context) (domain.Profile, error)            { return r.profile, nil }
func (r *cvRepo) Preferences(context.Context) (domain.Preferences, error) {
	return domain.Preferences{AIEnabled: r.enabled}, nil
}
func (r *cvRepo) CachedCV(context.Context, domain.CVUpload) (*domain.CVAnalysis, error) {
	return r.cached, nil
}
func (r *cvRepo) ApplyCV(context.Context, string, domain.CVAnalysis) error {
	r.applied = true
	return nil
}

type cvAI struct {
	domain.AIProvider
	calls int
	err   error
}

func (a *cvAI) AnalyseCV(context.Context, string, domain.Preferences) (domain.CVAnalysis, error) {
	a.calls++
	return domain.CVAnalysis{Skills: []string{"Go"}, Summary: "Go engineer", Unknowns: []string{}}, a.err
}
func TestCVAnalysisSkipsSupersededAndReusesCache(t *testing.T) {
	id := "latest"
	r := &cvRepo{profile: domain.Profile{LatestCVUploadID: &id}, doc: domain.CVUpload{ExtractedText: "Go engineer"}, enabled: true}
	ai := &cvAI{}
	s := Service{Repo: r, AI: ai, AIEnabled: true}
	if e := s.AnalyseCV(context.Background(), "old"); e != nil || ai.calls != 0 || r.applied {
		t.Fatal("superseded CV analysed", e)
	}
	if e := s.AnalyseCV(context.Background(), id); e != nil || ai.calls != 1 || !r.applied {
		t.Fatal("CV not applied", e)
	}
	r.applied = false
	r.cached = &domain.CVAnalysis{Skills: []string{"Go"}, Summary: "Go engineer", Unknowns: []string{}}
	if e := s.AnalyseCV(context.Background(), id); e != nil || ai.calls != 1 || !r.applied {
		t.Fatal("cached CV called model", e)
	}
	r.cached = nil
	r.applied = false
	ai.err = errors.New("model unavailable")
	if e := s.AnalyseCV(context.Background(), id); e == nil || r.applied {
		t.Fatal("failed analysis overwrote profile")
	}
	r.enabled = false
	if _, e := s.UploadCV(context.Background(), "cv.txt", "Go engineer"); !errors.Is(e, domain.ErrAIUnavailable) {
		t.Fatal("disabled AI accepted upload", e)
	}
}
