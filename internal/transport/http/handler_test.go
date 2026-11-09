package http

import (
	"bytes"
	"context"
	"github.com/aqil/ai-job-hunter/internal/config"
	"github.com/aqil/ai-job-hunter/internal/domain"
	"github.com/aqil/ai-job-hunter/internal/provider/ollama"
	"github.com/aqil/ai-job-hunter/internal/repository/postgres"
	"github.com/aqil/ai-job-hunter/internal/usecase"
	"github.com/google/uuid"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestAuthAndValidation(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := New(nil, nil, config.Config{Token: "test-token-with-at-least-24-characters"}, log)
	req := httptest.NewRequest("POST", "/api/jobs/discover", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Fatalf("unprotected resource %d", w.Code)
	}
	req = httptest.NewRequest("GET", "/api/jobs/not-a-uuid", nil)
	req.Header.Set("Authorization", "Bearer test-token-with-at-least-24-characters")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Fatalf("expected invalid id, got %d", w.Code)
	}
}
func TestIntegrationAPI(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL required")
	}
	s, e := postgres.Open(context.Background(), dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if e = s.Migrate(context.Background()); e != nil {
		t.Fatal(e)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := &usecase.Service{Repo: s, AI: &ollama.Provider{Model: "qwen3:1.7b"}, Log: log}
	h := New(s, svc, config.Config{Token: "test-token-with-at-least-24-characters"}, log)
	for _, path := range []string{"/api/health", "/api/profile", "/api/preferences", "/api/jobs", "/api/companies", "/api/applications", "/api/discovery-runs", "/api/stats"} {
		req := httptest.NewRequest("GET", path, nil)
		req.Header.Set("Authorization", "Bearer test-token-with-at-least-24-characters")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("%s: %d %s", path, w.Code, w.Body)
		}
	}
	req := httptest.NewRequest("POST", "/api/companies", bytes.NewBufferString(`{"name":"invalid","provider":"greenhouse","board":"../bad"}`))
	req.Header.Set("Authorization", "Bearer test-token-with-at-least-24-characters")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 422 {
		t.Fatalf("invalid board accepted %d", w.Code)
	}
}
func TestIntegrationCompanyProfileAPI(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL required")
	}
	ctx := context.Background()
	store, e := postgres.Open(ctx, dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer store.Close()
	if e = store.Migrate(ctx); e != nil {
		t.Fatal(e)
	}
	c := domain.Company{Name: "Fixture-api-profile-" + uuid.NewString()}
	if e = store.SaveCompany(ctx, &c, nil); e != nil {
		t.Fatal(e)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := New(store, &usecase.Service{Repo: store}, config.Config{Token: "test-token-with-at-least-24-characters"}, log)
	request := func(body string, auth bool) *httptest.ResponseRecorder {
		r := httptest.NewRequest("PATCH", "/api/companies/"+c.ID+"/profile", bytes.NewBufferString(body))
		if auth {
			r.Header.Set("Authorization", "Bearer test-token-with-at-least-24-characters")
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	if w := request(`{"ceo":{"value":"Fixture CEO"}}`, false); w.Code != 401 {
		t.Fatal("profile mutation not protected", w.Code)
	}
	if w := request(`{"ceo":{"value":"Fixture CEO"}}`, true); w.Code != 422 {
		t.Fatal("unsourced CEO accepted", w.Code, w.Body)
	}
	if w := request(`{"ceo":{"value":"Fixture CEO","source_url":"https://example.org/leadership","source_name":"Fixture official page"}}`, true); w.Code != 200 {
		t.Fatal("sourced CEO not saved", w.Code, w.Body)
	}
	if w := request(`{"ceo_nationality":{"value":"Indonesian","source_url":"https://example.org/bio","source_name":"Fixture biography"}}`, true); w.Code != 200 {
		t.Fatal("nationality not saved", w.Code, w.Body)
	}
	saved, e := store.Company(ctx, c.ID)
	if e != nil || saved.Profile["ceo_nationality"].Value != "Indonesian" {
		t.Fatal("profile API did not persist", e)
	}
}
func TestInvalidAppliedFilter(t *testing.T) {
	h := New(nil, nil, config.Config{Token: "test-token-with-at-least-24-characters"}, slog.Default())
	r := httptest.NewRequest("GET", "/api/jobs?include_applied=invalid", nil)
	r.Header.Set("Authorization", "Bearer test-token-with-at-least-24-characters")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 400 {
		t.Fatal("invalid applied filter accepted", w.Code, w.Body)
	}
}
