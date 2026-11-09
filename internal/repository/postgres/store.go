package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aqil/ai-job-hunter/internal/domain"
	"github.com/aqil/ai-job-hunter/migrations"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

const ProfileID = "00000000-0000-0000-0000-000000000001"

var ErrBusy = errors.New("operation already running")
var ErrLimit = errors.New("analysis queue is full")

type Store struct{ DB *gorm.DB }

func Open(ctx context.Context, dsn string) (*Store, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent), TranslateError: true})
	if err != nil {
		return nil, err
	}
	raw, err := db.DB()
	if err != nil {
		return nil, err
	}
	raw.SetMaxOpenConns(12)
	raw.SetMaxIdleConns(4)
	raw.SetConnMaxLifetime(30 * time.Minute)
	if err = raw.PingContext(ctx); err != nil {
		return nil, err
	}
	return &Store{db}, nil
}
func (s *Store) Close() error {
	db, e := s.DB.DB()
	if e != nil {
		return e
	}
	return db.Close()
}
func (s *Store) Migrate(ctx context.Context) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(735010)").Error; err != nil {
			return err
		}
		if err := tx.Exec("CREATE TABLE IF NOT EXISTS schema_migrations (version text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())").Error; err != nil {
			return err
		}
		files, e := migrations.Files.ReadDir(".")
		if e != nil {
			return e
		}
		for _, f := range files {
			if !strings.HasSuffix(f.Name(), ".sql") {
				continue
			}
			var count int64
			if e = tx.Table("schema_migrations").Where("version = ?", f.Name()).Count(&count).Error; e != nil {
				return e
			}
			if count > 0 {
				continue
			}
			b, e := migrations.Files.ReadFile(f.Name())
			if e != nil {
				return e
			}
			if e = tx.Exec(string(b)).Error; e != nil {
				return fmt.Errorf("migration %s: %w", f.Name(), e)
			}
			if e = tx.Exec("INSERT INTO schema_migrations(version) VALUES (?)", f.Name()).Error; e != nil {
				return e
			}
		}
		p := domain.Profile{Base: base(ProfileID), Name: "My profile", YearsExperience: 4, Location: "Indonesia", Skills: []string{"Go", "Node.js", "PostgreSQL", "MongoDB", "Redis", "MySQL", "REST APIs", "microservices", "Docker", "AWS S3", "GitHub Actions", "OpenTelemetry"}}
		if e = tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&p).Error; e != nil {
			return e
		}
		pref := domain.Preferences{NearbyLocations: []string{}, RequiredSkills: []string{}, Countries: []string{}, EmploymentTypes: []string{}, ExcludedCompanies: []string{}, ExcludedKeywords: []string{}, Base: base(ProfileID), Roles: []string{"Backend Engineer", "Golang Developer", "Software Engineer", "Backend Developer"}, Remote: []string{"remote", "hybrid"}, Seniorities: []string{"mid", "senior"}, Currency: "IDR", SalaryPeriod: "monthly", DiscoveryMinutes: 360, AIEnabled: true, OllamaModel: "qwen3:1.7b", AIContext: 4096, AIKeepAlive: "1m", Weights: map[string]float64{"technical": 30, "work_life_balance": 30, "remote": 20, "compensation": 15, "stability": 5}}
		return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&pref).Error
	})
}
func base(id string) domain.Base {
	if id == "" {
		id = uuid.NewString()
	}
	return domain.Base{ID: id}
}
func (s *Store) Ping(ctx context.Context) error {
	var n int
	return s.DB.WithContext(ctx).Raw("SELECT 1").Scan(&n).Error
}
func (s *Store) Profile(ctx context.Context) (domain.Profile, error) {
	var p domain.Profile
	e := s.DB.WithContext(ctx).First(&p, "id = ?", ProfileID).Error
	p.TimeZone = domain.InferTimeZone(p.Location)
	return p, e
}
func (s *Store) SaveProfile(ctx context.Context, p *domain.Profile) error {
	p.TimeZone = domain.InferTimeZone(p.Location)
	p.ID = ProfileID
	if p.Skills == nil {
		p.Skills = []string{}
	}
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if e := tx.Model(&domain.Profile{}).Where("id = ?", ProfileID).Select("Name", "YearsExperience", "Location", "Skills", "Summary", "CVSource").Updates(p).Error; e != nil {
			return e
		}
		if e := tx.Exec("DELETE FROM profile_skills WHERE profile_id = ?", ProfileID).Error; e != nil {
			return e
		}
		for _, skill := range p.Skills {
			if e := tx.Exec("INSERT INTO profile_skills(id,profile_id,name) VALUES (?,?,?) ON CONFLICT DO NOTHING", uuid.NewString(), ProfileID, skill).Error; e != nil {
				return e
			}
		}
		return nil
	})
}
func (s *Store) Preferences(ctx context.Context) (domain.Preferences, error) {
	var p domain.Preferences
	e := s.DB.WithContext(ctx).First(&p, "id = ?", ProfileID).Error
	return p, e
}
func (s *Store) SavePreferences(ctx context.Context, p *domain.Preferences) error {
	p.ID = ProfileID
	for _, list := range []*[]string{&p.NearbyLocations, &p.Roles, &p.RequiredSkills, &p.Seniorities, &p.Countries, &p.Remote, &p.EmploymentTypes, &p.ExcludedCompanies, &p.ExcludedKeywords} {
		if *list == nil {
			*list = []string{}
		}
	}
	return s.DB.WithContext(ctx).Model(&domain.Preferences{}).Where("id = ?", ProfileID).Select("NearbyLocations", "HardLocation", "RequireLocationMatch", "AIEnabled", "OllamaModel", "AIContext", "AIKeepAlive", "Roles", "RequiredSkills", "Seniorities", "Countries", "Remote", "EmploymentTypes", "ExcludedCompanies", "ExcludedKeywords", "SalaryMin", "SalaryMax", "Currency", "SalaryPeriod", "MaxAgeDays", "HardRemote", "Weights", "DiscoveryMinutes", "AutoAnalyse").Updates(p).Error
}
func (s *Store) Companies(ctx context.Context) ([]domain.Company, error) {
	v := []domain.Company{}
	e := s.DB.WithContext(ctx).Order("name").Find(&v).Error
	return v, e
}
func (s *Store) Company(ctx context.Context, id string) (domain.Company, error) {
	var v domain.Company
	e := s.DB.WithContext(ctx).First(&v, "id = ?", id).Error
	return v, e
}
func (s *Store) SaveCompany(ctx context.Context, c *domain.Company, source *domain.CompanySource) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if c.ResearchURLs == nil {
			c.ResearchURLs = []string{}
		}
		c.Base = base(c.ID)
		if e := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "name"}}, DoUpdates: clause.AssignmentColumns([]string{"website", "research_urls", "updated_at"})}).Create(c).Error; e != nil {
			return e
		}
		var existing domain.Company
		if e := tx.First(&existing, "name = ?", c.Name).Error; e != nil {
			return e
		}
		*c = existing
		if source != nil {
			source.Base = base(source.ID)
			source.CompanyID = c.ID
			return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "provider"}, {Name: "board"}, {Name: "country"}}, DoUpdates: clause.AssignmentColumns([]string{"enabled", "updated_at"})}).Create(source).Error
		}
		return nil
	})
}
func (s *Store) Sources(ctx context.Context) ([]domain.CompanySource, error) {
	v := []domain.CompanySource{}
	e := s.DB.WithContext(ctx).Order("created_at").Find(&v).Error
	return v, e
}
func (s *Store) ToggleSource(ctx context.Context, id string, enabled bool) error {
	r := s.DB.WithContext(ctx).Model(&domain.CompanySource{}).Where("id = ?", id).Update("enabled", enabled)
	if r.Error != nil {
		return r.Error
	}
	if r.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// A successful complete board snapshot is committed atomically. Failed fetches never reach here.
func (s *Store) SaveSnapshot(ctx context.Context, source domain.CompanySource, postings []domain.Posting, now time.Time) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		seen := []string{}
		for _, p := range postings {
			j := p.Job
			var obs domain.Observation
			err := tx.First(&obs, "source_id = ? AND external_id = ?", source.ID, p.ExternalID).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			var existing domain.Job
			if err == nil {
				if e := tx.First(&existing, "id = ?", obs.JobID).Error; e != nil {
					return e
				}
			} else {
				e := tx.Where("canonical_url = ? OR (company_id = ? AND dedupe_key = ?)", j.CanonicalURL, j.CompanyID, j.DedupeKey).Order("first_seen").First(&existing).Error
				if e != nil && !errors.Is(e, gorm.ErrRecordNotFound) {
					return e
				}
			}
			j.Base = base(existing.ID)
			j.FirstSeen = now
			if existing.ID != "" {
				j.CreatedAt = existing.CreatedAt
				j.FirstSeen = existing.FirstSeen
			}
			j.LastSeen = now
			j.Available = true
			if e := tx.Save(&j).Error; e != nil {
				return e
			}
			o := domain.Observation{Base: base(obs.ID), JobID: j.ID, SourceID: source.ID, ExternalID: p.ExternalID, URL: j.CanonicalURL, Payload: j.SourcePayload, LastSeen: now, Available: true}
			if obs.ID != "" {
				o.CreatedAt = obs.CreatedAt
			}
			if e := tx.Save(&o).Error; e != nil {
				return e
			}
			seen = append(seen, p.ExternalID)
		}
		q := tx.Model(&domain.Observation{}).Where("source_id = ?", source.ID)
		if len(seen) > 0 {
			q = q.Where("external_id NOT IN ?", seen)
		}
		if e := q.Updates(map[string]any{"missing_runs": gorm.Expr("missing_runs + 1"), "updated_at": now}).Error; e != nil {
			return e
		}
		if e := tx.Model(&domain.Observation{}).Where("source_id = ? AND missing_runs >= 2", source.ID).Update("available", false).Error; e != nil {
			return e
		}
		if e := tx.Exec("UPDATE jobs SET available = EXISTS (SELECT 1 FROM job_observations o WHERE o.job_id = jobs.id AND o.available) AND (expires_at IS NULL OR expires_at > ?) WHERE id IN (SELECT job_id FROM job_observations WHERE source_id = ?)", now, source.ID).Error; e != nil {
			return e
		}
		return tx.Model(&domain.CompanySource{}).Where("id = ?", source.ID).Update("last_fetched_at", now).Error
	})
}
func (s *Store) Jobs(ctx context.Context, q domain.JobQuery) (domain.JobPage, error) {
	p := domain.JobPage{Items: []domain.Job{}, Page: q.Page, Limit: q.Limit}
	db := s.DB.WithContext(ctx).Model(&domain.Job{})
	if !q.IncludeApplied {
		db = db.Where("NOT EXISTS (SELECT 1 FROM applications a WHERE a.job_id = jobs.id AND a.stage <> ?)", "Saved")
	}
	if q.Search != "" {
		db = db.Where("title ILIKE ? OR company_name ILIKE ? OR description ILIKE ?", "%"+q.Search+"%", "%"+q.Search+"%", "%"+q.Search+"%")
	}
	if q.Remote != "" {
		db = db.Where("remote = ?", q.Remote)
	}
	if q.CompanyID != "" {
		db = db.Where("company_id = ?", q.CompanyID)
	}
	if q.Status != "all" {
		if q.Status == "closed" {
			db = db.Where("available = false OR expires_at <= ?", time.Now())
		} else {
			db = db.Where("available = true AND (expires_at IS NULL OR expires_at > ?)", time.Now())
		}
	}
	if e := db.Count(&p.Total).Error; e != nil {
		return p, e
	}
	order := "first_seen DESC, id ASC"
	if q.Sort == "company" {
		order = "company_name ASC, first_seen DESC, id ASC"
	}
	if q.Sort == "title" {
		order = "title ASC, id ASC"
	}

	e := db.Order(order).Limit(q.Limit).Offset((q.Page - 1) * q.Limit).Find(&p.Items).Error
	return p, e
}
func (s *Store) Job(ctx context.Context, id string) (domain.Job, error) {
	var j domain.Job
	e := s.DB.WithContext(ctx).First(&j, "id = ?", id).Error
	return j, e
}
func (s *Store) CachedAnalysis(ctx context.Context, id, fp string) (*domain.Analysis, error) {
	var a domain.Analysis
	db := s.DB.WithContext(ctx).Where("job_id = ?", id)
	if fp != "" {
		db = db.Where("input_fingerprint = ?", fp)
	}
	e := db.Order("created_at DESC").First(&a).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &a, e
}
func (s *Store) SaveAnalysis(ctx context.Context, a *domain.Analysis) error {
	a.Base = base("")
	return s.DB.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(a).Error
}
func (s *Store) Evidence(ctx context.Context, id string) ([]domain.Evidence, error) {
	v := []domain.Evidence{}
	e := s.DB.WithContext(ctx).Where("company_id = ?", id).Order("retrieved_at DESC").Find(&v).Error
	return v, e
}
func (s *Store) Research(ctx context.Context, id string) (*domain.Research, error) {
	var v domain.Research
	e := s.DB.WithContext(ctx).First(&v, "company_id = ?", id).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &v, e
}
func (s *Store) SaveResearch(ctx context.Context, r *domain.Research, ev []domain.Evidence) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, v := range ev {
			v.Base = base("")
			if e := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "company_id"}, {Name: "source_url"}}, DoUpdates: clause.AssignmentColumns([]string{"summary", "retrieved_at", "category", "confidence", "updated_at"})}).Create(&v).Error; e != nil {
				return e
			}
		}
		r.Base = base("")
		return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "company_id"}}, DoUpdates: clause.AssignmentColumns([]string{"status", "summary", "errors", "expires_at", "interpretation", "updated_at"})}).Create(r).Error
	})
}
func (s *Store) Applications(ctx context.Context) ([]domain.Application, error) {
	v := []domain.Application{}
	e := s.DB.WithContext(ctx).Order("updated_at DESC").Find(&v).Error
	return v, e
}
func (s *Store) SaveApplication(ctx context.Context, a *domain.Application, create bool) error {
	if create {
		a.Base = base("")
		return s.DB.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "job_id"}}, DoUpdates: clause.AssignmentColumns([]string{"stage", "notes", "updated_at"})}, clause.Returning{}).Create(a).Error
	}
	r := s.DB.WithContext(ctx).Model(&domain.Application{}).Where("id = ?", a.ID).Updates(map[string]any{"stage": a.Stage, "notes": a.Notes, "updated_at": time.Now()})
	if r.Error != nil {
		return r.Error
	}
	if r.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	var saved domain.Application
	if e := s.DB.WithContext(ctx).First(&saved, "id = ?", a.ID).Error; e != nil {
		return e
	}
	*a = saved
	return nil
}
func (s *Store) SaveRun(ctx context.Context, r *domain.DiscoveryRun) error {
	if r.ID == "" {
		r.Base = base("")
	}
	return s.DB.WithContext(ctx).Save(r).Error
}
func (s *Store) Runs(ctx context.Context) ([]domain.DiscoveryRun, error) {
	v := []domain.DiscoveryRun{}
	e := s.DB.WithContext(ctx).Order("created_at DESC").Limit(50).Find(&v).Error
	return v, e
}

// Dedicated connection keeps the session advisory lock on the same PostgreSQL session.
func (s *Store) WithLock(ctx context.Context, key int64, fn func() error) error {
	return s.DB.WithContext(ctx).Connection(func(db *gorm.DB) error {
		var ok bool
		if e := db.Raw("SELECT pg_try_advisory_lock(?)", key).Scan(&ok).Error; e != nil {
			return e
		}
		if !ok {
			return ErrBusy
		}
		defer db.WithContext(context.WithoutCancel(ctx)).Exec("SELECT pg_advisory_unlock(?)", key)
		return fn()
	})
}

// Feed ingestion must not overwrite user-maintained company research settings.
func (s *Store) EnsureCompany(ctx context.Context, name string) (domain.Company, error) {
	c := domain.Company{Base: base(""), Name: name, ResearchURLs: []string{}}
	if e := s.DB.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&c).Error; e != nil {
		return c, e
	}
	var existing domain.Company
	e := s.DB.WithContext(ctx).Where("name = ?", name).First(&existing).Error
	return existing, e
}
