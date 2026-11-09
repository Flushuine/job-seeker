package matching

import (
	"math"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/aqil/ai-job-hunter/internal/domain"
)

var catalog = []string{"Go", "Node.js", "PostgreSQL", "MongoDB", "Redis", "MySQL", "Docker", "AWS", "Kubernetes", "Python", "Java", "Ruby", "Rails", "Scala", "Kotlin", "C#", "Elixir", "Django", "Spring", "TypeScript", "Rust", "C++", "GraphQL", "Kafka", "REST", "GitHub Actions", "OpenTelemetry"}
var aliases = map[string][]string{"go": {"golang", "go"}, "node.js": {"node.js", "nodejs"}, "postgresql": {"postgresql", "postgres"}, "rest apis": {"rest", "restful"}, "aws s3": {"aws s3", "amazon s3", "s3"}, "rest": {"rest", "restful"}}

var goNamePattern = regexp.MustCompile(`(?i)\bgolang\b`)
var goTokenPattern = regexp.MustCompile(`(^|[^a-zA-Z0-9-])Go($|[^a-zA-Z0-9-])`)
var restPattern = regexp.MustCompile(`(^|[^a-zA-Z0-9])REST($|[^a-zA-Z0-9])|(?i)\brestful\b`)
var patterns sync.Map

func mentions(text, skill string) bool {
	terms := aliases[strings.ToLower(skill)]
	if strings.EqualFold(skill, "Go") {
		return goNamePattern.MatchString(text) || goTokenPattern.MatchString(text)
	}
	if strings.EqualFold(skill, "REST") || strings.EqualFold(skill, "REST APIs") {
		return restPattern.MatchString(text)
	}
	if len(terms) == 0 {
		terms = []string{skill}
	}
	for _, v := range terms {
		cached, ok := patterns.Load(v)
		if !ok {
			compiled := regexp.MustCompile(`(?i)(^|[^a-z0-9])` + regexp.QuoteMeta(v) + `([^a-z0-9]|$)`)
			cached, _ = patterns.LoadOrStore(v, compiled)
		}
		re := cached.(*regexp.Regexp)
		if re.MatchString(text) {
			return true
		}
	}
	return false
}
func has(list []string, v string) bool {
	for _, s := range list {
		if strings.EqualFold(s, v) || mentions(s, v) {
			return true
		}
	}
	return false
}
func number(v float64) *float64 { return &v }
func Evaluate(j domain.Job, p domain.Profile, pref domain.Preferences) domain.Deterministic {
	d := domain.Deterministic{MatchingSkills: []string{}, MissingSkills: []string{}, Unknowns: []string{}, Factors: []domain.Factor{}, Recommendation: "Investigate Further", Confidence: "low"}
	text := j.Title + " " + j.Description
	skills := append(append([]string{}, catalog...), p.Skills...)
	seen := map[string]bool{}
	var known, matched float64
	for _, skill := range skills {
		k := strings.ToLower(skill)
		if seen[k] {
			continue
		}
		seen[k] = true
		if !mentions(text, skill) {
			continue
		}
		known++
		if has(p.Skills, skill) {
			matched++
			d.MatchingSkills = append(d.MatchingSkills, skill)
		} else {
			d.MissingSkills = append(d.MissingSkills, skill)
		}
	}
	if known > 0 {
		d.TechnicalScore = number(math.Round(matched / known * 100))
	} else {
		d.Unknowns = append(d.Unknowns, "No recognised technologies stated")
	}
	for _, skill := range pref.RequiredSkills {
		if !mentions(text, skill) {
			d.Unknowns = append(d.Unknowns, "Required skill not stated: "+skill)
		}
	}
	add := func(name, status, why string, score *float64) {
		d.Factors = append(d.Factors, domain.Factor{Name: name, Status: status, Explanation: why, Score: score})
	}
	status := "unknown"
	if d.TechnicalScore != nil {
		status = "match"
		if *d.TechnicalScore < 60 {
			status = "mismatch"
		}
	}
	add("technical", status, "Recognised technologies in the posting compared with profile skills; mentions may be optional requirements.", d.TechnicalScore)
	role := false
	for _, v := range pref.Roles {
		if strings.Contains(strings.ToLower(j.Title), strings.ToLower(v)) {
			role = true
		}
	}
	if len(pref.Roles) == 0 {
		add("role", "unknown", "No preferred roles configured", nil)
	} else if role {
		add("role", "match", "Title matches a preferred role", number(100))
	} else {
		add("role", "mismatch", "Title differs from preferred roles", number(0))
	}
	if j.Remote == "unknown" || j.Remote == "" {
		add("remote", "unknown", "Working arrangement not stated", nil)
	} else if len(pref.Remote) == 0 {
		add("remote", "unknown", "No working arrangement preference configured", nil)
	} else if has(pref.Remote, j.Remote) {
		add("remote", "match", "Working arrangement matches preference; geographic eligibility still needs confirmation", number(100))
	} else {
		add("remote", "mismatch", "Working arrangement differs from preference", number(0))
	}
	geo := domain.Geography(j, pref)
	d.Factors = append(d.Factors, geo)

	if j.MinExperience == nil {
		add("experience", "unknown", "Minimum experience is not available as structured data", nil)
	} else if p.YearsExperience >= *j.MinExperience {
		add("experience", "match", "Experience meets stated minimum", number(100))
	} else {
		add("experience", "mismatch", "Experience below stated minimum", number(0))
	}
	if j.Seniority == "" || len(pref.Seniorities) == 0 {
		add("seniority", "unknown", "Seniority or preference unknown", nil)
	} else if has(pref.Seniorities, j.Seniority) {
		add("seniority", "match", "Seniority matches preference", number(100))
	} else {
		add("seniority", "mismatch", "Seniority differs from preference", number(0))
	}
	if j.EmploymentType == "" || len(pref.EmploymentTypes) == 0 {
		add("employment", "unknown", "Employment type or preference unknown", nil)
	} else if has(pref.EmploymentTypes, j.EmploymentType) {
		add("employment", "match", "Employment type matches", number(100))
	} else {
		add("employment", "mismatch", "Employment type differs", number(0))
	}
	if j.SalaryMax == nil || pref.SalaryMin == 0 || !strings.EqualFold(j.Currency, pref.Currency) || !domain.ComparablePeriod(j.SalaryPeriod) {
		add("compensation", "unknown", "Salary undisclosed, no expectation set, or currencies differ; no FX conversion assumed", nil)
	} else {
		score := math.Min(100, domain.Annual(*j.SalaryMax, j.SalaryPeriod)/domain.Annual(pref.SalaryMin, pref.SalaryPeriod)*100)
		status := "match"
		if score < 100 {
			status = "mismatch"
		}
		add("compensation", status, "Stated salary maximum compared with minimum expectation in the same currency", number(score))
	}
	elig := domain.Filter(j, p, pref, time.Now())
	d.Unknowns = append(d.Unknowns, elig.Unknowns...)
	if !elig.Eligible {
		d.Recommendation = "Not Suitable"
	} else if d.TechnicalScore != nil {
		switch {
		case *d.TechnicalScore < 40 || !role:
			d.Recommendation = "Low Priority"
		case *d.TechnicalScore >= 80 && len(elig.Preferences) == 0:
			d.Recommendation = "Worth Considering"
		default:
			d.Recommendation = "Investigate Further"
		}
	}
	knownFactors := 0
	mismatches := 0
	for _, f := range d.Factors {
		if f.Score != nil {
			knownFactors++
		}
		if f.Status == "mismatch" {
			mismatches++
		}
	}
	if knownFactors >= 5 {
		d.Confidence = "medium"
	}
	if pref.RequireLocationMatch && (geo.Status == "unknown" || j.Remote == "unknown" || j.Remote == "") && elig.Eligible {
		d.Recommendation = "Investigate Further"
	}
	if elig.Eligible && knownFactors == len(d.Factors) && mismatches == 0 && d.TechnicalScore != nil && *d.TechnicalScore >= 80 {
		d.Recommendation = "Strong Match"
		d.Confidence = "high"
	}
	return d
}
