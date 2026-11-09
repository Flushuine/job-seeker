package domain

import (
	"testing"
	"time"
)

func TestUnknownNotExcluded(t *testing.T) {
	e := Filter(Job{Available: true}, Profile{}, Preferences{HardRemote: true, Remote: []string{"remote"}}, time.Now())
	if !e.Eligible || len(e.Unknowns) == 0 {
		t.Fatalf("unknown excluded: %+v", e)
	}
}
func TestHardExclusions(t *testing.T) {
	e := Filter(Job{Available: true, Remote: "onsite", Description: "unpaid trial"}, Profile{}, Preferences{HardRemote: true, Remote: []string{"remote"}, ExcludedKeywords: []string{"unpaid"}}, time.Now())
	if e.Eligible || len(e.Exclusions) != 2 {
		t.Fatal(e)
	}
}
func TestRankCoverage(t *testing.T) {
	p := Preferences{Remote: []string{"remote"}, Weights: map[string]float64{"technical": 30, "remote": 20, "work_life_balance": 30, "compensation": 15, "stability": 5}}
	r := Rank(Job{Remote: "remote", Deterministic: &Deterministic{TechnicalScore: func() *float64 { v := 80.0; return &v }()}}, p, &Analysis{Result: Match{TechnicalScore: 80}})
	if r.Score == nil || *r.Score != 88 || r.Coverage != 50 || r.WorkplaceRisk != "Insufficient Evidence" {
		t.Fatal(r)
	}
	r = Rank(Job{}, p, nil)
	if r.Score != nil || r.Coverage != 0 {
		t.Fatal(r)
	}
}
func TestNormalizationDedupe(t *testing.T) {
	a, e := Normalize(Job{Title: "Senior Go Engineer", Location: "Remote", Description: "<p>Go</p>", CanonicalURL: "https://EXAMPLE.com/jobs/1/?utm_source=x"})
	if e != nil {
		t.Fatal(e)
	}
	b, _ := Normalize(Job{Title: " Senior  Go Engineer ", Location: "Remote", Description: "Go", CanonicalURL: "https://example.com/jobs/1"})
	if a.DedupeKey != b.DedupeKey || a.CanonicalURL != b.CanonicalURL || a.Fingerprint != b.Fingerprint {
		t.Fatalf("inconsistent normalization %+v %+v", a, b)
	}
}
func TestSalaryPeriod(t *testing.T) {
	if Annual(100, "monthly") != 1200 || Annual(100, "annual") != 100 {
		t.Fatal("incorrect annual conversion")
	}
}
func TestRejectInvalidMatch(t *testing.T) {
	if ValidateMatch(Match{OverallScore: 101}) == nil {
		t.Fatal("invalid AI accepted")
	}
}
func TestUnsupportedSalaryPeriodUnknown(t *testing.T) {
	max := 20.0
	j := Job{Available: true, SalaryMax: &max, Currency: "USD", SalaryPeriod: "hourly"}
	e := Filter(j, Profile{}, Preferences{Currency: "USD", SalaryPeriod: "annual", SalaryMin: 40000}, time.Now())
	if len(e.Preferences) != 0 {
		t.Fatal("hourly salary must not be compared as annual", e)
	}
	r := Rank(j, Preferences{Currency: "USD", SalaryPeriod: "annual", SalaryMin: 40000, Weights: map[string]float64{"compensation": 100}}, nil)
	if r.Score != nil {
		t.Fatal("unsupported pay period scored")
	}
}
