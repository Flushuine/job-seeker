package domain

import (
	"testing"
	"time"
)

func nearbyPreferences() Preferences {
	return Preferences{Countries: []string{"Indonesia"}, NearbyLocations: []string{"Jakarta", "Bogor", "Depok", "Tangerang", "Tangerang Selatan", "Bekasi", "Jabodetabek", "Greater Jakarta", "South Tangerang", "BSD", "Bintaro", "Serpong"}, HardLocation: true, RequireLocationMatch: true, HardRemote: true, Remote: []string{"remote", "hybrid"}}
}
func TestJakartaLocationPolicy(t *testing.T) {
	yes := true
	cases := []struct {
		name string
		job  Job
		want string
	}{
		{"hybrid Jakarta", Job{Remote: "hybrid", Location: "Jakarta, Indonesia"}, "match"},
		{"hybrid Tangsel", Job{Remote: "hybrid", Location: "Tangerang Selatan, Banten"}, "match"},
		{"hybrid BSD", Job{Remote: "hybrid", Location: "BSD City"}, "match"},
		{"hybrid Bekasi", Job{Remote: "hybrid", Location: "Bekasi"}, "match"},
		{"hybrid Bandung", Job{Remote: "hybrid", Location: "Bandung, Indonesia"}, "mismatch"},
		{"hybrid Singapore", Job{Remote: "hybrid", Location: "Singapore"}, "mismatch"},
		{"unknown office city", Job{Remote: "hybrid", Location: "Indonesia"}, "unknown"},
		{"remote generic", Job{Remote: "remote", Location: "Remote"}, "unknown"},
		{"unknown arrangement local", Job{Remote: "unknown", Location: "Jakarta"}, "unknown"},
		{"US-only remote", Job{Remote: "remote", Location: "Remote - US"}, "mismatch"},
		{"Europe remote", Job{Remote: "remote", Location: "Remote, Europe"}, "mismatch"},
		{"APAC remote", Job{Remote: "remote", Location: "APAC"}, "match"},
		{"Indonesian remote", Job{Remote: "remote", Country: "Indonesia"}, "match"},
		{"worldwide remote", Job{Remote: "remote", Worldwide: &yes}, "match"},
		{"structured multi-country excludes ID", Job{Remote: "remote", EligibleCountries: []string{"United States", "Canada"}}, "mismatch"},
		{"structured multi-country includes ID", Job{Remote: "remote", EligibleCountries: []string{"Singapore", "Indonesia"}}, "match"},
		{"mandatory US overrides worldwide", Job{Remote: "remote", Worldwide: &yes, Description: "Candidates must be based in the United States."}, "mismatch"},
		{"US-only description", Job{Remote: "remote", Location: "Worldwide", Description: "This is a fully remote role (US only)."}, "mismatch"},
		{"explicit Indonesian exclusion", Job{Remote: "remote", Location: "Worldwide excluding Indonesia"}, "mismatch"},
		{"excluded Asia region", Job{Remote: "remote", Location: "Worldwide except APAC"}, "mismatch"},
		{"foreign clients not eligibility", Job{Remote: "remote", Location: "Indonesia", Description: "We serve clients in the United States and Singapore."}, "match"},
		{"unknown city is not geocoded", Job{Remote: "remote", Location: "London"}, "unknown"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Geography(c.job, nearbyPreferences())
			if got.Status != c.want {
				t.Fatalf("want %s, got %+v", c.want, got)
			}
		})
	}
}
func TestHardLocationExclusionsAndUnknowns(t *testing.T) {
	pref := nearbyPreferences()
	for _, job := range []Job{{Available: true, Remote: "onsite", Location: "Jakarta"}, {Available: true, Remote: "hybrid", Location: "Bandung"}, {Available: true, Remote: "remote", Location: "United States"}} {
		if Filter(job, Profile{}, pref, time.Now()).Eligible {
			t.Fatal("incompatible job was allowed", job)
		}
	}
	e := Filter(Job{Available: true, Remote: "remote", Location: "Remote"}, Profile{}, pref, time.Now())
	if !e.Eligible || len(e.Unknowns) == 0 {
		t.Fatal("missing geography must remain unknown, not a fabricated mismatch", e)
	}
}
func TestSpecificRegionsAndTimezones(t *testing.T) {
	pref := nearbyPreferences()
	for label, want := range map[string]string{"South Asia": "mismatch", "East Asia": "mismatch", "Central Asia": "mismatch", "Worldwide except US": "match", "Worldwide (US hours)": "match", "Worldwide, Europe": "unknown"} {
		t.Run(label, func(t *testing.T) {
			if got := Geography(Job{Remote: "remote", Location: label}, pref); got.Status != want {
				t.Fatal(label, got)
			}
		})
	}
}
