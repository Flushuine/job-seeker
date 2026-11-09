package http

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aqil/ai-job-hunter/internal/config"
	"github.com/aqil/ai-job-hunter/internal/domain"
	"github.com/aqil/ai-job-hunter/internal/repository/postgres"
	"github.com/aqil/ai-job-hunter/internal/usecase"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Handler struct {
	Store          *postgres.Store
	Service        *usecase.Service
	Config         config.Config
	Log            *slog.Logger
	mu             sync.Mutex
	mutationWindow time.Time
	mutations      int
}

func New(store *postgres.Store, s *usecase.Service, c config.Config, log *slog.Logger) http.Handler {
	h := &Handler{Store: store, Service: s, Config: c, Log: log}
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Cache-Control", "no-store")
			next.ServeHTTP(w, r)
		})
	})
	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		if e := store.Ping(r.Context()); e != nil {
			h.fail(w, r, e)
			return
		}
		write(w, 200, map[string]string{"status": "ok"})
	})
	r.Route("/api", func(r chi.Router) {
		r.Use(h.auth)
		r.Use(h.mutationLimit)
		r.Get("/status", func(w http.ResponseWriter, r *http.Request) {
			write(w, 200, map[string]any{"ai_enabled": c.AIEnabled, "model": c.Model, "queue_capacity": 20, "research_allowed_hosts": c.ResearchHosts})
		})
		r.Get("/jobs", h.jobs)
		r.Post("/jobs/discover", func(w http.ResponseWriter, r *http.Request) {
			run, e := s.Discover(r.Context())
			if e != nil {
				h.fail(w, r, e)
				return
			}
			write(w, 200, run)
		})
		r.Get("/jobs/{id}", h.job)
		r.Post("/jobs/{id}/analyse", func(w http.ResponseWriter, r *http.Request) {
			if !h.id(w, r) {
				return
			}
			if !c.AIEnabled {
				problem(w, 503, "ai_unconfigured", "Local AI is disabled. Deterministic matching remains available.")
				return
			}
			a, e := s.Enqueue(r.Context(), "job", chi.URLParam(r, "id"))
			if e != nil {
				h.fail(w, r, e)
				return
			}
			write(w, 202, a)
		})
		r.Get("/analysis-tasks/{id}", func(w http.ResponseWriter, r *http.Request) {
			if !h.id(w, r) {
				return
			}
			v, e := store.Task(r.Context(), chi.URLParam(r, "id"))
			h.respond(w, r, v, e)
		})
		r.Post("/companies/{id}/analyse", func(w http.ResponseWriter, r *http.Request) {
			if !h.id(w, r) {
				return
			}
			v, e := s.Enqueue(r.Context(), "company", chi.URLParam(r, "id"))
			if e != nil {
				h.fail(w, r, e)
				return
			}
			write(w, 202, v)
		})
		r.Post("/companies/{id}/evidence", h.addEvidence)
		r.Patch("/companies/{id}/profile", h.companyProfile)
		r.Post("/profile/cv", h.uploadCV)
		r.Get("/profile/cv", h.cvStatus)
		r.Get("/profile", func(w http.ResponseWriter, r *http.Request) {
			v, e := store.Profile(r.Context())
			h.respond(w, r, v, e)
		})
		r.Put("/profile", func(w http.ResponseWriter, r *http.Request) {
			var v domain.Profile
			if !decode(w, r, &v) {
				return
			}
			if strings.TrimSpace(v.Name) == "" || v.YearsExperience < 0 || v.YearsExperience > 80 || len(v.Skills) > 100 || len(v.Name) > 200 || len(v.Location) > 200 || len(v.Summary) > 5000 || len(v.CVSource) > 200 {
				problem(w, 422, "validation", "Provide a name, valid experience and at most 100 skills")
				return
			}
			for _, skill := range v.Skills {
				if len(strings.TrimSpace(skill)) == 0 || len(skill) > 80 {
					problem(w, 422, "validation", "Skill names must contain 1–80 characters")
					return
				}
			}
			e := store.SaveProfile(r.Context(), &v)
			h.respond(w, r, v, e)
		})
		r.Get("/preferences", func(w http.ResponseWriter, r *http.Request) {
			v, e := store.Preferences(r.Context())
			h.respond(w, r, v, e)
		})
		r.Put("/preferences", func(w http.ResponseWriter, r *http.Request) {
			var v domain.Preferences
			if !decode(w, r, &v) {
				return
			}
			if e := domain.ValidatePreferences(v); e != nil {
				problem(w, 422, "validation", e.Error())
				return
			}
			e := store.SavePreferences(r.Context(), &v)
			h.respond(w, r, v, e)
		})
		r.Get("/companies", func(w http.ResponseWriter, r *http.Request) {
			v, e := store.Companies(r.Context())
			h.respond(w, r, v, e)
		})
		r.Post("/companies", h.createCompany)
		r.Get("/companies/{id}", h.company)
		r.Post("/companies/{id}/exclude", func(w http.ResponseWriter, r *http.Request) {
			if !h.id(w, r) {
				return
			}
			pref, e := store.ExcludeCompany(r.Context(), chi.URLParam(r, "id"))
			h.respond(w, r, pref, e)
		})
		r.Post("/companies/{id}/research", func(w http.ResponseWriter, r *http.Request) {
			if !h.id(w, r) {
				return
			}
			v, e := s.Research(r.Context(), chi.URLParam(r, "id"))
			h.respond(w, r, v, e)
		})
		r.Get("/sources", func(w http.ResponseWriter, r *http.Request) {
			v, e := store.Sources(r.Context())
			h.respond(w, r, v, e)
		})
		r.Patch("/sources/{id}", func(w http.ResponseWriter, r *http.Request) {
			if !h.id(w, r) {
				return
			}
			var v struct {
				Enabled *bool `json:"enabled"`
			}
			if !decode(w, r, &v) {
				return
			}
			if v.Enabled == nil {
				problem(w, 422, "validation", "enabled is required")
				return
			}
			e := store.ToggleSource(r.Context(), chi.URLParam(r, "id"), *v.Enabled)
			h.respond(w, r, map[string]bool{"enabled": *v.Enabled}, e)
		})
		r.Get("/applications", func(w http.ResponseWriter, r *http.Request) {
			v, e := store.Applications(r.Context())
			h.respond(w, r, v, e)
		})
		r.Post("/applications", h.application)
		r.Patch("/applications/{id}", h.application)
		r.Get("/discovery-runs", func(w http.ResponseWriter, r *http.Request) { v, e := store.Runs(r.Context()); h.respond(w, r, v, e) })
		r.Get("/stats", h.stats)
	})
	r.NotFound(func(w http.ResponseWriter, r *http.Request) { problem(w, 404, "not_found", "Endpoint not found") })
	return r
}
func (h *Handler) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		want := sha256.Sum256([]byte(h.Config.Token))
		got := sha256.Sum256([]byte(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")))
		if subtle.ConstantTimeCompare(want[:], got[:]) != 1 {
			problem(w, 401, "unauthorized", "A valid application token is required")
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (h *Handler) mutationLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			h.mu.Lock()
			if time.Since(h.mutationWindow) > time.Minute {
				h.mutationWindow = time.Now()
				h.mutations = 0
			}
			h.mutations++
			over := h.mutations > 30
			h.mu.Unlock()
			if over {
				w.Header().Set("Retry-After", "60")
				problem(w, 429, "rate_limited", "Too many changes. Retry in one minute.")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
func (h *Handler) id(w http.ResponseWriter, r *http.Request) bool {
	if _, e := uuid.Parse(chi.URLParam(r, "id")); e != nil {
		problem(w, 400, "validation", "Invalid resource identifier")
		return false
	}
	return true
}
func (h *Handler) jobs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, e := integer(q.Get("page"), 1, 1, 100000)
	if e != nil {
		problem(w, 400, "validation", e.Error())
		return
	}
	limit, e := integer(q.Get("limit"), 20, 1, 100)
	if e != nil {
		problem(w, 400, "validation", e.Error())
		return
	}
	company := q.Get("company_id")
	if company != "" {
		if _, e = uuid.Parse(company); e != nil {
			problem(w, 400, "validation", "Invalid company_id")
			return
		}
	}
	includeExcluded := false
	if raw := q.Get("include_excluded"); raw != "" {
		includeExcluded, e = strconv.ParseBool(raw)
		if e != nil {
			problem(w, 400, "validation", "include_excluded must be true or false")
			return
		}
	}
	includeApplied := false
	if raw := q.Get("include_applied"); raw != "" {
		includeApplied, e = strconv.ParseBool(raw)
		if e != nil {
			problem(w, 400, "validation", "include_applied must be true or false")
			return
		}
	}
	v, e := h.Service.ListJobs(r.Context(), domain.JobQuery{IncludeApplied: includeApplied, IncludeExcluded: includeExcluded, Search: q.Get("q"), Remote: q.Get("remote"), CompanyID: company, Sort: q.Get("sort"), Status: q.Get("status"), Page: page, Limit: limit})
	if e != nil {
		h.fail(w, r, e)
		return
	}
	write(w, 200, v)
}
func (h *Handler) job(w http.ResponseWriter, r *http.Request) {
	if !h.id(w, r) {
		return
	}
	id := chi.URLParam(r, "id")
	j, e := h.Store.Job(r.Context(), id)
	if e != nil {
		h.fail(w, r, e)
		return
	}
	p, e := h.Store.Profile(r.Context())
	if e != nil {
		h.fail(w, r, e)
		return
	}
	pref, e := h.Store.Preferences(r.Context())
	if e != nil {
		h.fail(w, r, e)
		return
	}
	if e = h.Service.Enrich(r.Context(), &j, p, pref); e != nil {
		h.fail(w, r, e)
		return
	}
	write(w, 200, j)
}
func (h *Handler) company(w http.ResponseWriter, r *http.Request) {
	if !h.id(w, r) {
		return
	}
	id := chi.URLParam(r, "id")
	c, e := h.Store.Company(r.Context(), id)
	if e != nil {
		h.fail(w, r, e)
		return
	}
	ev, e := h.Store.Evidence(r.Context(), id)
	if e != nil {
		h.fail(w, r, e)
		return
	}
	research, e := h.Store.Research(r.Context(), id)
	h.respond(w, r, map[string]any{"company": c, "evidence": ev, "research": research}, e)
}

var boardPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,100}$`)

func (h *Handler) createCompany(w http.ResponseWriter, r *http.Request) {
	var v struct {
		Name         string   `json:"name"`
		Website      string   `json:"website"`
		ResearchURLs []string `json:"research_urls"`
		Provider     string   `json:"provider"`
		Board        string   `json:"board"`
		Country      string   `json:"country"`
	}
	if !decode(w, r, &v) {
		return
	}
	if strings.TrimSpace(v.Name) == "" || len(v.Name) > 200 || len(v.ResearchURLs) > 10 {
		problem(w, 422, "validation", "Company name required; at most ten research feeds")
		return
	}
	for _, raw := range append(append([]string{}, v.ResearchURLs...), v.Website) {
		if raw == "" {
			continue
		}
		u, e := url.Parse(raw)
		if e != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
			problem(w, 422, "validation", "Company and research URLs must use HTTPS")
			return
		}
	}
	var source *domain.CompanySource
	if v.Provider != "" {
		if v.Provider != "greenhouse" && v.Provider != "lever" && v.Provider != "lever-eu" && v.Provider != "himalayas" || !boardPattern.MatchString(v.Board) {
			problem(w, 422, "validation", "Select Greenhouse, Lever or Himalayas and a valid board identifier")
			return
		}
		v.Country = strings.ToUpper(strings.TrimSpace(v.Country))
		if v.Country != "" && (v.Provider != "himalayas" || !regexp.MustCompile(`^[A-Z]{2}$`).MatchString(v.Country)) {
			problem(w, 422, "validation", "Source country must be a two-letter code for Himalayas")
			return
		}
		source = &domain.CompanySource{Provider: v.Provider, Board: v.Board, Country: v.Country, Enabled: true}
	}
	c := domain.Company{Name: strings.TrimSpace(v.Name), Website: v.Website, ResearchURLs: v.ResearchURLs}
	e := h.Store.SaveCompany(r.Context(), &c, source)
	h.respond(w, r, c, e)
}
func (h *Handler) application(w http.ResponseWriter, r *http.Request) {
	var a domain.Application
	if !decode(w, r, &a) {
		return
	}
	valid := false
	for _, v := range []string{"Saved", "Applied", "Screening", "Interview", "Offer", "Rejected", "Withdrawn"} {
		if a.Stage == v {
			valid = true
		}
	}
	if !valid || len(a.Notes) > 10000 {
		problem(w, 422, "validation", "Invalid application stage or notes too long")
		return
	}
	create := r.Method == "POST"
	if create {
		if _, e := uuid.Parse(a.JobID); e != nil {
			problem(w, 422, "validation", "Valid job_id required")
			return
		}
	} else {
		if !h.id(w, r) {
			return
		}
		a.ID = chi.URLParam(r, "id")
	}
	e := h.Store.SaveApplication(r.Context(), &a, create)
	h.respond(w, r, a, e)
}
func (h *Handler) stats(w http.ResponseWriter, r *http.Request) {
	var v struct {
		Total        int64 `json:"total"`
		New          int64 `json:"new"`
		Strong       int64 `json:"strong"`
		Investigate  int64 `json:"investigate"`
		Applications int64 `json:"applications"`
	}
	db := h.Store.DB.WithContext(r.Context())
	checks := []func() error{func() error { return db.Model(&domain.Job{}).Count(&v.Total).Error }, func() error {
		return db.Model(&domain.Application{}).Where("stage <> ?", "Saved").Count(&v.Applications).Error
	}}
	for _, f := range checks {
		if e := f(); e != nil {
			h.fail(w, r, e)
			return
		}
	}
	p, e := h.Store.Profile(r.Context())
	if e != nil {
		h.fail(w, r, e)
		return
	}
	pref, e := h.Store.Preferences(r.Context())
	if e != nil {
		h.fail(w, r, e)
		return
	}
	for page := 1; ; page++ {
		jobs, e := h.Store.Jobs(r.Context(), domain.JobQuery{Page: page, Limit: 100})
		if e != nil {
			h.fail(w, r, e)
			return
		}
		for _, j := range jobs.Items {
			if !usecase.MatchesSearch(j, p, pref) {
				continue
			}
			if j.FirstSeen.After(time.Now().Add(-7 * 24 * time.Hour)) {
				v.New++
			}
			if e = h.Service.Enrich(r.Context(), &j, p, pref); e != nil {
				h.fail(w, r, e)
				return
			}
			if j.Deterministic != nil {
				switch j.Deterministic.Recommendation {
				case "Strong Match":
					v.Strong++
				case "Investigate Further":
					v.Investigate++
				}
			}
		}
		if page*100 >= int(jobs.Total) {
			break
		}
	}
	write(w, 200, v)
}
func integer(raw string, def, min, max int) (int, error) {
	if raw == "" {
		return def, nil
	}
	v, e := strconv.Atoi(raw)
	if e != nil || v < min || v > max {
		return 0, fmt.Errorf("pagination value must be %d–%d", min, max)
	}
	return v, nil
}
func decode(w http.ResponseWriter, r *http.Request, out any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if e := d.Decode(out); e != nil {
		problem(w, 400, "invalid_json", "Invalid JSON body or unknown field")
		return false
	}
	var extra any
	if e := d.Decode(&extra); e != io.EOF {
		problem(w, 400, "invalid_json", "Only one JSON object is allowed")
		return false
	}
	return true
}
func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
func problem(w http.ResponseWriter, status int, code, message string) {
	write(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
func (h *Handler) respond(w http.ResponseWriter, r *http.Request, v any, e error) {
	if e != nil {
		h.fail(w, r, e)
		return
	}
	write(w, 200, v)
}
func (h *Handler) fail(w http.ResponseWriter, r *http.Request, e error) {
	switch {
	case errors.Is(e, gorm.ErrRecordNotFound):
		problem(w, 404, "not_found", "Resource not found")
	case errors.Is(e, gorm.ErrForeignKeyViolated):
		problem(w, 422, "validation", "Referenced resource does not exist")
	case errors.Is(e, gorm.ErrDuplicatedKey):
		problem(w, 409, "conflict", "Resource already exists")
	case errors.Is(e, domain.ErrInvalidCV):
		problem(w, 422, "invalid_cv", e.Error())
	case errors.Is(e, domain.ErrAIUnavailable):
		problem(w, 503, "ai_unavailable", e.Error())
	case errors.Is(e, domain.ErrInvalidCompanyProfile):
		problem(w, 422, "validation", e.Error())
	case errors.Is(e, usecase.ErrSearchExcluded):
		problem(w, 422, "search_excluded", e.Error())
	case errors.Is(e, postgres.ErrCompanyExclusion):
		problem(w, 422, "validation", e.Error())
	case errors.Is(e, postgres.ErrBusy):
		problem(w, 409, "busy", e.Error())
	case errors.Is(e, postgres.ErrLimit):
		problem(w, 429, "queue_full", e.Error())
	default:
		h.Log.Error("request failed", "request_id", middleware.GetReqID(r.Context()), "path", r.URL.Path, "error", e)
		problem(w, 502, "operation_failed", "Operation failed. Check the server logs for details.")
	}
}

func (h *Handler) addEvidence(w http.ResponseWriter, r *http.Request) {
	if !h.id(w, r) {
		return
	}
	var v struct {
		SourceURL   string     `json:"source_url"`
		SourceName  string     `json:"source_name"`
		Summary     string     `json:"summary"`
		Category    string     `json:"category"`
		PublishedAt *time.Time `json:"published_at"`
	}
	if !decode(w, r, &v) {
		return
	}
	u, e := url.Parse(v.SourceURL)
	if e != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || strings.TrimSpace(v.SourceName) == "" || len(v.Summary) > 5000 || len(v.Summary) < 10 {
		problem(w, 422, "validation", "Provide an HTTPS source URL, source name, and excerpt of 10–5000 characters")
		return
	}
	allowed := map[string]bool{"employee_opinion": true, "unverified_allegation": true, "company_statement": true, "public_source_claim": true, "verified_fact": true}
	if !allowed[v.Category] {
		problem(w, 422, "validation", "Choose an evidence category")
		return
	}
	ev := domain.Evidence{CompanyID: chi.URLParam(r, "id"), SourceURL: v.SourceURL, SourceName: v.SourceName, Summary: v.Summary, Category: v.Category, PublishedAt: v.PublishedAt, RetrievedAt: time.Now().UTC(), Confidence: "low"}
	if v.Category == "verified_fact" {
		ev.Category = "user_reported_fact"
	}
	e = h.Store.SaveEvidence(r.Context(), &ev)
	h.respond(w, r, ev, e)
}

func (h *Handler) companyProfile(w http.ResponseWriter, r *http.Request) {
	if !h.id(w, r) {
		return
	}
	var updates domain.CompanyProfile
	if !decode(w, r, &updates) {
		return
	}
	if len(updates) == 0 {
		problem(w, 422, "validation", "Provide at least one company profile field")
		return
	}
	c, e := h.Store.PatchCompanyProfile(r.Context(), chi.URLParam(r, "id"), updates)
	h.respond(w, r, c, e)
}
