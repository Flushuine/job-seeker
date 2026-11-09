package domain

import (
	"fmt"
	"math"
	"strings"
	"time"
)

func containsAny(s string, terms []string) bool {
	for _, v := range terms {
		if strings.Contains(strings.ToLower(s), strings.ToLower(v)) {
			return true
		}
	}
	return false
}
func has(s string, terms []string) bool {
	for _, v := range terms {
		if strings.EqualFold(s, v) {
			return true
		}
	}
	return false
}
func Filter(j Job, p Profile, pref Preferences, now time.Time) Eligibility {
	e := Eligibility{Eligible: true, Exclusions: []string{}, Preferences: []string{}, Unknowns: []string{}}
	exclude := func(s string) { e.Eligible = false; e.Exclusions = append(e.Exclusions, s) }
	if has(j.CompanyName, pref.ExcludedCompanies) {
		exclude("Excluded company")
	}
	if containsAny(j.Title+" "+j.Description, pref.ExcludedKeywords) {
		exclude("Excluded keyword")
	}
	if !j.Available {
		exclude("Job no longer available")
	}
	if len(pref.Roles) > 0 && !containsAny(j.Title, pref.Roles) {
		e.Preferences = append(e.Preferences, "Title differs from preferred roles")
	}
	for _, s := range pref.RequiredSkills {
		if !containsAny(j.Title+" "+j.Description, []string{s}) {
			e.Unknowns = append(e.Unknowns, "Required technology not stated: "+s)
		}
	}
	if j.MinExperience != nil && *j.MinExperience > p.YearsExperience {
		e.Preferences = append(e.Preferences, "Experience requirement exceeds profile")
	}
	if j.MinExperience == nil {
		e.Unknowns = append(e.Unknowns, "Minimum experience not normalised")
	}
	if j.Seniority == "" {
		e.Unknowns = append(e.Unknowns, "Seniority not stated")
	} else if len(pref.Seniorities) > 0 && !has(j.Seniority, pref.Seniorities) {
		e.Preferences = append(e.Preferences, "Seniority differs from preference")
	}
	if j.Remote == "unknown" || j.Remote == "" {
		e.Unknowns = append(e.Unknowns, "Working arrangement unknown")
	} else if len(pref.Remote) > 0 && !has(j.Remote, pref.Remote) {
		if pref.HardRemote {
			exclude("Working arrangement excluded")
		} else {
			e.Preferences = append(e.Preferences, "Working arrangement differs from preference")
		}
	}
	geo := Geography(j, pref)
	switch geo.Status {
	case "mismatch":
		if pref.HardLocation {
			exclude(geo.Explanation)
		} else {
			e.Preferences = append(e.Preferences, geo.Explanation)
		}
	case "unknown":
		e.Unknowns = append(e.Unknowns, geo.Explanation)
	}

	if j.EmploymentType == "" {
		e.Unknowns = append(e.Unknowns, "Employment type unknown")
	} else if len(pref.EmploymentTypes) > 0 && !has(j.EmploymentType, pref.EmploymentTypes) {
		e.Preferences = append(e.Preferences, "Employment type differs from preference")
	}
	if j.PostedAt == nil {
		e.Unknowns = append(e.Unknowns, "Original publication date unknown")
	} else if pref.MaxAgeDays > 0 && now.Sub(*j.PostedAt) > time.Duration(pref.MaxAgeDays)*24*time.Hour {
		exclude("Posting exceeds maximum age")
	}
	if j.SalaryMax == nil || j.Currency == "" || !ComparablePeriod(j.SalaryPeriod) {
		e.Unknowns = append(e.Unknowns, "Compensation unknown")
	} else if !strings.EqualFold(j.Currency, pref.Currency) {
		e.Unknowns = append(e.Unknowns, "Salary currency differs; no exchange rate applied")
	} else if pref.SalaryMin > 0 {
		max := Annual(*j.SalaryMax, j.SalaryPeriod)
		want := Annual(pref.SalaryMin, pref.SalaryPeriod)
		if max < want {
			e.Preferences = append(e.Preferences, "Salary below expectation")
		}
	}
	return e
}
func ComparablePeriod(period string) bool { return period == "monthly" || period == "annual" }
func Annual(v float64, period string) float64 {
	if period == "monthly" {
		return v * 12
	}
	return v
}

// Weighted scores use observed factors only. Coverage exposes missing weight explicitly.
func Rank(j Job, pref Preferences, a *Analysis) Ranking {
	r := Ranking{WorkplaceRisk: "Insufficient Evidence"}
	values := map[string]float64{}
	if j.Deterministic != nil && j.Deterministic.TechnicalScore != nil {
		v := *j.Deterministic.TechnicalScore
		for _, factor := range j.Deterministic.Factors {
			if factor.Name == "role" && factor.Score != nil {
				v = v*0.7 + *factor.Score*0.3
				break
			}
		}
		values["technical"] = v
	}
	if j.Remote != "" && j.Remote != "unknown" && len(pref.Remote) > 0 {
		values["remote"] = 0
		if has(j.Remote, pref.Remote) {
			values["remote"] = 100
		}
	}
	if j.SalaryMax != nil && pref.SalaryMin > 0 && strings.EqualFold(j.Currency, pref.Currency) && ComparablePeriod(j.SalaryPeriod) {
		values["compensation"] = math.Min(100, Annual(*j.SalaryMax, j.SalaryPeriod)/Annual(pref.SalaryMin, pref.SalaryPeriod)*100)
	}
	var sum, known, total float64
	for k, w := range pref.Weights {
		total += w
		if v, ok := values[k]; ok {
			known += w
			sum += v * w
		}
	}
	if total > 0 {
		r.Coverage = math.Round(known / total * 100)
	}
	if known > 0 {
		v := math.Round(sum / known)
		r.Score = &v
	}
	return r
}

var Recommendations = []string{"Strong Match", "Worth Considering", "Investigate Further", "Low Priority", "Not Suitable"}

func ValidateMatch(m Match) error {
	if math.IsNaN(m.OverallScore) || math.IsNaN(m.TechnicalScore) || m.OverallScore < 0 || m.OverallScore > 100 || m.TechnicalScore < 0 || m.TechnicalScore > 100 {
		return fmt.Errorf("scores must be between 0 and 100")
	}
	if !has(m.Recommendation, Recommendations) {
		return fmt.Errorf("invalid recommendation")
	}
	if !has(m.Confidence, []string{"low", "medium", "high"}) {
		return fmt.Errorf("invalid confidence")
	}
	if m.Explanation == "" || m.ExperienceCompatibility == "" || m.LocationCompatibility == "" || m.SalaryCompatibility == "" {
		return fmt.Errorf("missing compatibility explanation")
	}
	if m.MatchingSkills == nil || m.MissingSkills == nil || m.Advantages == nil || m.Disadvantages == nil || m.Concerns == nil || m.Unknowns == nil {
		return fmt.Errorf("missing required array")
	}
	return nil
}
func ValidatePreferences(p Preferences) error {
	for _, list := range [][]string{p.Roles, p.RequiredSkills, p.Seniorities, p.Countries, p.Remote, p.EmploymentTypes, p.ExcludedCompanies, p.ExcludedKeywords, p.NearbyLocations} {
		if len(list) > 50 {
			return fmt.Errorf("at most 50 values per preference")
		}
		for _, v := range list {
			if len(strings.TrimSpace(v)) == 0 || len(v) > 120 {
				return fmt.Errorf("preference values must contain 1–120 characters")
			}
		}
	}
	if (p.HardLocation || p.RequireLocationMatch) && len(p.Countries) == 0 && len(p.NearbyLocations) == 0 {
		return fmt.Errorf("configure nearby office areas or eligible countries for location restrictions")
	}
	if p.AIContext != 2048 && p.AIContext != 4096 {
		return fmt.Errorf("ai_context must be 2048 or 4096")
	}
	if p.OllamaModel == "" || len(p.OllamaModel) > 100 || strings.Contains(strings.ToLower(p.OllamaModel), "cloud") {
		return fmt.Errorf("choose an installed local Ollama model")
	}
	if p.AIKeepAlive != "0" && p.AIKeepAlive != "1m" && p.AIKeepAlive != "5m" {
		return fmt.Errorf("ai_keep_alive must be 0, 1m or 5m")
	}
	if p.DiscoveryMinutes < 5 || p.DiscoveryMinutes > 10080 {
		return fmt.Errorf("discovery_minutes must be 5–10080")
	}
	if p.MaxAgeDays < 0 || p.SalaryMin < 0 || p.SalaryMax < 0 || (p.SalaryMax > 0 && p.SalaryMax < p.SalaryMin) {
		return fmt.Errorf("invalid salary range or posting age")
	}
	if len(p.Currency) != 3 {
		return fmt.Errorf("currency must be a three-letter ISO code")
	}
	if !has(p.SalaryPeriod, []string{"monthly", "annual"}) {
		return fmt.Errorf("salary_period must be monthly or annual")
	}
	if len(p.Weights) != 5 {
		return fmt.Errorf("provide all five ranking weights")
	}
	var total float64
	for _, k := range []string{"technical", "work_life_balance", "remote", "compensation", "stability"} {
		v, ok := p.Weights[k]
		if !ok || v < 0 || v > 100 {
			return fmt.Errorf("invalid ranking weight: %s", k)
		}
		total += v
	}
	if math.Abs(total-100) > 0.01 {
		return fmt.Errorf("ranking weights must sum to 100")
	}
	for _, v := range p.Remote {
		if !has(v, []string{"remote", "hybrid", "onsite"}) {
			return fmt.Errorf("invalid remote preference")
		}
	}
	return nil
}
