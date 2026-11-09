package domain

import (
	"context"
	"encoding/json"
	"time"
)

type Base struct {
	ID        string    `json:"id" gorm:"type:uuid;primaryKey"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
type Profile struct {
	TimeZone         string  `json:"time_zone" gorm:"-"`
	LatestCVUploadID *string `json:"latest_cv_upload_id" gorm:"type:uuid"`
	CVFingerprint    string  `json:"cv_fingerprint"`
	Summary          string  `json:"summary"`
	CVSource         string  `json:"cv_source"`
	Base
	Name            string   `json:"name"`
	YearsExperience int      `json:"years_experience"`
	Location        string   `json:"location"`
	Skills          []string `json:"skills" gorm:"serializer:json;type:jsonb"`
}
type Preferences struct {
	NearbyLocations      []string `json:"nearby_locations" gorm:"serializer:json;type:jsonb"`
	HardLocation         bool     `json:"hard_location"`
	RequireLocationMatch bool     `json:"require_location_match"`
	AIEnabled            bool     `json:"ai_enabled"`
	OllamaModel          string   `json:"ollama_model"`
	AIContext            int      `json:"ai_context"`
	AIKeepAlive          string   `json:"ai_keep_alive"`
	Base
	Roles             []string           `json:"roles" gorm:"serializer:json;type:jsonb"`
	RequiredSkills    []string           `json:"required_skills" gorm:"serializer:json;type:jsonb"`
	Seniorities       []string           `json:"seniorities" gorm:"serializer:json;type:jsonb"`
	Countries         []string           `json:"countries" gorm:"serializer:json;type:jsonb"`
	Remote            []string           `json:"remote" gorm:"serializer:json;type:jsonb"`
	EmploymentTypes   []string           `json:"employment_types" gorm:"serializer:json;type:jsonb"`
	ExcludedCompanies []string           `json:"excluded_companies" gorm:"serializer:json;type:jsonb"`
	ExcludedKeywords  []string           `json:"excluded_keywords" gorm:"serializer:json;type:jsonb"`
	SalaryMin         float64            `json:"salary_min"`
	SalaryMax         float64            `json:"salary_max"`
	Currency          string             `json:"currency"`
	SalaryPeriod      string             `json:"salary_period"`
	MaxAgeDays        int                `json:"max_age_days"`
	HardRemote        bool               `json:"hard_remote"`
	Weights           map[string]float64 `json:"weights" gorm:"serializer:json;type:jsonb"`
	DiscoveryMinutes  int                `json:"discovery_minutes"`
	AutoAnalyse       bool               `json:"auto_analyse"`
}

func (Preferences) TableName() string { return "job_search_preferences" }

type Company struct {
	Profile CompanyProfile `json:"profile" gorm:"serializer:json;type:jsonb"`
	Base
	Name         string   `json:"name"`
	Website      string   `json:"website"`
	ResearchURLs []string `json:"research_urls" gorm:"serializer:json;type:jsonb"`
}
type CompanySource struct {
	Country       string     `json:"country"`
	LastFetchedAt *time.Time `json:"last_fetched_at"`
	Base
	CompanyID string `json:"company_id" gorm:"type:uuid"`
	Provider  string `json:"provider"`
	Board     string `json:"board"`
	Enabled   bool   `json:"enabled"`
}
type Job struct {
	WorkHours         *WorkHours     `json:"work_hours,omitempty" gorm:"-"`
	EligibleCountries []string       `json:"eligible_countries" gorm:"serializer:json;type:jsonb"`
	Worldwide         *bool          `json:"worldwide"`
	Deterministic     *Deterministic `json:"deterministic,omitempty" gorm:"-"`
	SourceName        string         `json:"source_name"`
	SourceURL         string         `json:"source_url"`
	ApplicationURL    string         `json:"application_url"`
	ExpiresAt         *time.Time     `json:"expires_at"`
	Base
	CompanyID      string          `json:"company_id" gorm:"type:uuid"`
	CompanyName    string          `json:"company_name"`
	Title          string          `json:"title"`
	Description    string          `json:"description"`
	Location       string          `json:"location"`
	Country        string          `json:"country"`
	Remote         string          `json:"remote"`
	Seniority      string          `json:"seniority"`
	MinExperience  *int            `json:"min_experience"`
	EmploymentType string          `json:"employment_type"`
	SalaryMin      *float64        `json:"salary_min"`
	SalaryMax      *float64        `json:"salary_max"`
	Currency       string          `json:"currency"`
	SalaryPeriod   string          `json:"salary_period"`
	CanonicalURL   string          `json:"canonical_url"`
	DedupeKey      string          `json:"-"`
	Fingerprint    string          `json:"fingerprint"`
	SourcePayload  json.RawMessage `json:"source_payload" gorm:"type:jsonb"`
	PostedAt       *time.Time      `json:"posted_at"`
	FirstSeen      time.Time       `json:"first_seen"`
	LastSeen       time.Time       `json:"last_seen"`
	Available      bool            `json:"available"`
	Analysis       *Analysis       `json:"analysis,omitempty" gorm:"-"`
	Eligibility    *Eligibility    `json:"eligibility,omitempty" gorm:"-"`
	Ranking        *Ranking        `json:"ranking,omitempty" gorm:"-"`
}
type Observation struct {
	Base
	JobID       string          `json:"job_id" gorm:"type:uuid"`
	SourceID    string          `json:"source_id" gorm:"type:uuid"`
	ExternalID  string          `json:"external_id"`
	URL         string          `json:"url"`
	Payload     json.RawMessage `json:"payload" gorm:"type:jsonb"`
	LastSeen    time.Time       `json:"last_seen"`
	MissingRuns int             `json:"missing_runs"`
	Available   bool            `json:"available"`
}

func (Observation) TableName() string { return "job_observations" }

type Analysis struct {
	Base
	JobID            string   `json:"job_id" gorm:"type:uuid"`
	Model            string   `json:"model"`
	PromptVersion    string   `json:"prompt_version"`
	InputFingerprint string   `json:"input_fingerprint"`
	Result           Match    `json:"result" gorm:"serializer:json;type:jsonb"`
	InputTokens      int      `json:"input_tokens"`
	OutputTokens     int      `json:"output_tokens"`
	EstimatedCost    *float64 `json:"estimated_cost"`
}

func (Analysis) TableName() string { return "job_analyses" }

type Match struct {
	OverallScore            float64  `json:"overall_score"`
	TechnicalScore          float64  `json:"technical_score"`
	MatchingSkills          []string `json:"matching_skills"`
	MissingSkills           []string `json:"missing_skills"`
	ExperienceCompatibility string   `json:"experience_compatibility"`
	LocationCompatibility   string   `json:"location_compatibility"`
	SalaryCompatibility     string   `json:"salary_compatibility"`
	Advantages              []string `json:"advantages"`
	Disadvantages           []string `json:"disadvantages"`
	Concerns                []string `json:"concerns"`
	Unknowns                []string `json:"unknowns"`
	Recommendation          string   `json:"recommendation"`
	Confidence              string   `json:"confidence"`
	Explanation             string   `json:"explanation"`
}
type Evidence struct {
	Base
	CompanyID   string     `json:"company_id" gorm:"type:uuid"`
	SourceURL   string     `json:"source_url"`
	SourceName  string     `json:"source_name"`
	PublishedAt *time.Time `json:"published_at"`
	RetrievedAt time.Time  `json:"retrieved_at"`
	Summary     string     `json:"summary"`
	Category    string     `json:"category"`
	Confidence  string     `json:"confidence"`
}

func (Evidence) TableName() string { return "company_evidence" }

type Research struct {
	Interpretation map[string]any `json:"interpretation" gorm:"serializer:json;type:jsonb"`
	Base
	CompanyID string    `json:"company_id" gorm:"type:uuid"`
	Status    string    `json:"status"`
	Summary   string    `json:"summary"`
	Errors    []string  `json:"errors" gorm:"serializer:json;type:jsonb"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (Research) TableName() string { return "company_research" }

type Application struct {
	Base
	JobID string `json:"job_id" gorm:"type:uuid"`
	Stage string `json:"stage"`
	Notes string `json:"notes"`
}
type DiscoveryRun struct {
	Base
	Status     string     `json:"status"`
	JobsSeen   int        `json:"jobs_seen"`
	Errors     []string   `json:"errors" gorm:"serializer:json;type:jsonb"`
	FinishedAt *time.Time `json:"finished_at"`
}
type Posting struct {
	ExternalID string
	Job        Job
}
type JobProvider interface {
	Fetch(context.Context, CompanySource) ([]Posting, error)
}
type AIProvider interface {
	Analyse(context.Context, Job, Profile, Preferences) (Match, int, int, error)
	ModelID() string
}
type ResearchProvider interface {
	Research(context.Context, Company) ([]Evidence, []string)
}
type Eligibility struct {
	Eligible    bool     `json:"eligible"`
	Exclusions  []string `json:"exclusions"`
	Preferences []string `json:"preferences"`
	Unknowns    []string `json:"unknowns"`
}
type Ranking struct {
	Score         *float64 `json:"score"`
	Coverage      float64  `json:"coverage"`
	WorkplaceRisk string   `json:"workplace_risk"`
}
type JobQuery struct {
	Search, Remote, CompanyID, Sort, Status string
	Page, Limit                             int
	IncludeExcluded                         bool
	IncludeApplied                          bool
}
type JobPage struct {
	Items []Job `json:"items"`
	Total int64 `json:"total"`
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
}

type Factor struct {
	Name        string   `json:"name"`
	Status      string   `json:"status"`
	Explanation string   `json:"explanation"`
	Score       *float64 `json:"score"`
}
type Deterministic struct {
	TechnicalScore *float64 `json:"technical_score"`
	MatchingSkills []string `json:"matching_skills"`
	MissingSkills  []string `json:"missing_skills"`
	Unknowns       []string `json:"unknowns"`
	Factors        []Factor `json:"factors"`
	Recommendation string   `json:"recommendation"`
	Confidence     string   `json:"confidence"`
}
type AnalysisTask struct {
	Base
	ResourceID string `json:"resource_id" gorm:"type:uuid"`
	Kind       string `json:"kind"`
	Status     string `json:"status"`
	Error      string `json:"error"`
	Attempts   int    `json:"attempts"`
}
