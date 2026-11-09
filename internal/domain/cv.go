package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrInvalidCV = errors.New("invalid CV")
var ErrAIUnavailable = errors.New("enable local AI and start Ollama to analyse a CV; your current profile is unchanged")
var ErrProfileChanged = errors.New("profile changed while the CV was being analysed; upload again to apply it safely")

type CVAnalysis struct {
	Name            string   `json:"name"`
	Location        string   `json:"location"`
	YearsExperience *int     `json:"years_experience"`
	Skills          []string `json:"skills"`
	Summary         string   `json:"summary"`
	Unknowns        []string `json:"unknowns"`
}
type CVUpload struct {
	Base
	ProfileID              string      `json:"-" gorm:"type:uuid"`
	Filename               string      `json:"filename"`
	Fingerprint            string      `json:"fingerprint"`
	ExtractedText          string      `json:"-"`
	BaseProfileFingerprint string      `json:"-"`
	Model                  string      `json:"model"`
	PromptVersion          string      `json:"prompt_version"`
	AIContext              int         `json:"ai_context"`
	Analysis               *CVAnalysis `json:"analysis" gorm:"serializer:json;type:jsonb"`
	AppliedAt              *time.Time  `json:"applied_at"`
}

func (CVUpload) TableName() string { return "cv_uploads" }

type CVStatus struct {
	Upload *CVUpload     `json:"upload"`
	Task   *AnalysisTask `json:"task"`
}

func ProfileContentFingerprint(p Profile) string {
	return Hash([]any{p.Name, p.Location, p.YearsExperience, p.Skills, p.Summary, p.CVSource, p.CVFingerprint})
}
func ValidateCVAnalysis(a CVAnalysis) error {
	if len(a.Name) > 200 || len(a.Location) > 200 || strings.TrimSpace(a.Summary) == "" || len(a.Summary) > 5000 || len(a.Skills) == 0 || len(a.Skills) > 100 || a.Unknowns == nil || len(a.Unknowns) > 30 {
		return fmt.Errorf("%w: incomplete or oversized analysis", ErrInvalidCV)
	}
	if a.YearsExperience != nil && (*a.YearsExperience < 0 || *a.YearsExperience > 80) {
		return fmt.Errorf("%w: invalid experience", ErrInvalidCV)
	}
	for _, s := range a.Skills {
		if strings.TrimSpace(s) == "" || len(s) > 80 {
			return fmt.Errorf("%w: invalid skill", ErrInvalidCV)
		}
	}
	for _, s := range a.Unknowns {
		if len(s) > 500 {
			return fmt.Errorf("%w: oversized unknown", ErrInvalidCV)
		}
	}
	return nil
}
