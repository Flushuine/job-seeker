package domain

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

var ErrInvalidCompanyProfile = errors.New("invalid company profile")

// A profile fact is an attributed claim, not an inferred workplace assessment.
// RecordedAt is controlled by the server and records when the claim was saved;
// it does not imply the linked page was fetched or independently verified.
type CompanyFact struct {
	Value      string     `json:"value"`
	SourceURL  string     `json:"source_url"`
	SourceName string     `json:"source_name"`
	Notes      string     `json:"notes"`
	RecordedAt *time.Time `json:"recorded_at"`
}
type CompanyProfile map[string]CompanyFact

var CompanyProfileFields = map[string]string{"summary": "Company overview", "legal_name": "Legal entity", "origin_country": "Origin / incorporation country", "headquarters": "Headquarters", "ceo": "CEO / co-CEOs", "ceo_nationality": "CEO nationality / citizenship"}

func MergeCompanyProfile(current, updates CompanyProfile, now time.Time) (CompanyProfile, error) {
	merged := CompanyProfile{}
	for key, value := range current {
		merged[key] = value
	}
	invalid := func(message string) (CompanyProfile, error) {
		return nil, fmt.Errorf("%w: %s", ErrInvalidCompanyProfile, message)
	}
	for key, fact := range updates {
		if _, ok := CompanyProfileFields[key]; !ok {
			return invalid("unknown field: " + key)
		}
		fact.Value = strings.TrimSpace(fact.Value)
		fact.SourceURL = strings.TrimSpace(fact.SourceURL)
		fact.SourceName = strings.TrimSpace(fact.SourceName)
		fact.Notes = strings.TrimSpace(fact.Notes)
		if fact.Value == "" {
			delete(merged, key)
			continue
		}
		limit := 300
		if key == "summary" {
			limit = 2000
		}
		if len(fact.Value) > limit || len(fact.SourceURL) > 2048 || len(fact.SourceName) > 120 || len(fact.Notes) > 600 {
			return invalid("field or source metadata is too long: " + key)
		}
		u, e := url.Parse(fact.SourceURL)
		if e != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || fact.SourceName == "" {
			return invalid("provide an HTTPS source URL and source name for " + key)
		}
		previous, exists := current[key]
		if exists && previous.Value == fact.Value && previous.SourceURL == fact.SourceURL && previous.SourceName == fact.SourceName && previous.Notes == fact.Notes {
			fact.RecordedAt = previous.RecordedAt
		} else {
			recorded := now.UTC()
			fact.RecordedAt = &recorded
		}
		merged[key] = fact
	}
	// Leadership changes invalidate the old leader's nationality claim. Never
	// transfer nationality by name, birthplace, headquarters or company origin.
	if current["ceo"].Value != merged["ceo"].Value {
		if _, explicit := updates["ceo_nationality"]; !explicit {
			delete(merged, "ceo_nationality")
		} else if merged["ceo_nationality"].Value != "" && current["ceo_nationality"].Value == merged["ceo_nationality"].Value && current["ceo_nationality"].SourceURL == merged["ceo_nationality"].SourceURL && current["ceo"].Value != "" {
			return invalid("CEO changed; clear nationality or provide a source for the new CEO")
		}
	}
	if merged["ceo_nationality"].Value != "" && merged["ceo"].Value == "" {
		return invalid("identify the CEO before recording nationality")
	}
	return merged, nil
}
