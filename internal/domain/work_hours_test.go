package domain

import (
	"strings"
	"testing"
	"time"
)

func TestWorkingHoursConvertsExplicitWindows(t *testing.T) {
	now := time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct{ text, want string }{{"Work 09:00–17:00 UTC+1", "15:00–23:00"}, {"Overlap 9am-5pm EST", "21:00–05:00 (+1 day)"}, {"Hours 09:00 to 17:00 UTC+5:30", "10:30–18:30"}, {"Work 22:00-06:00 Asia/Jakarta", "22:00–06:00 (+1 day)"}, {"Hours 02:00-10:00 UTC+14", "19:00 (previous day)–03:00"}} {
		got := WorkingHours(Job{Description: tc.text}, Profile{Location: "Tangerang Selatan, Indonesia"}, now)
		if got.Status != "stated" || len(got.Examples) != 1 || got.Examples[0].YourHours != tc.want || got.Source == "" {
			t.Fatalf("%s: %+v", tc.text, got)
		}
	}
}
func TestWorkingHoursRegionsAreExamplesAndFollowDST(t *testing.T) {
	winter := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	summer := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)
	w := WorkingHours(Job{Location: "Remote EMEA"}, Profile{Location: "Tangerang Selatan, Indonesia"}, winter)
	s := WorkingHours(Job{Location: "Remote EMEA"}, Profile{Location: "Tangerang Selatan, Indonesia"}, summer)
	if w.Status != "example" || len(w.Examples) != 3 || w.Examples[0].YourHours != "16:00–00:00 (+1 day)" || s.Examples[0].YourHours != "15:00–23:00" || !strings.Contains(w.Note, "not the employer") {
		t.Fatal(w, s)
	}
	unknown := WorkingHours(Job{Location: "Worldwide", Description: "Choose your own hours"}, Profile{Location: "Tangerang Selatan, Indonesia"}, winter)
	if unknown.Status != "unknown" || len(unknown.Examples) != 0 {
		t.Fatal("invented working schedule", unknown)
	}
	invalid := WorkingHours(Job{Description: "Work 99:00-17:00 UTC+25"}, Profile{Location: "Tangerang Selatan, Indonesia"}, winter)
	if invalid.Status == "stated" {
		t.Fatal("invalid schedule accepted", invalid)
	}
}

func TestWorkingHoursDoesNotAddUTCToExplicitOffset(t *testing.T) {
	repeated := WorkingHours(Job{Description: "Work 09:00-17:00 UTC+1. Hours: 09:00-17:00 UTC+1."}, Profile{Location: "Tangerang Selatan, Indonesia"}, time.Now())
	if len(repeated.Examples) != 1 {
		t.Fatal("duplicate time windows", repeated)
	}
	got := WorkingHours(Job{Description: "Team timezone UTC+5:30. Hours not specified."}, Profile{Location: "Tangerang Selatan, Indonesia"}, time.Now())
	if got.Status != "example" || len(got.Examples) != 1 || got.Examples[0].Zone != "UTC+5:30" {
		t.Fatal(got)
	}
	ambiguous := WorkingHours(Job{Description: "Available 1-5pm EST"}, Profile{Location: "Tangerang Selatan, Indonesia"}, time.Now())
	if ambiguous.Status == "stated" {
		t.Fatal("ambiguous first endpoint must not become 01:00", ambiguous)
	}
}

func TestWorkingHoursUsesCVProfileLocation(t *testing.T) {
	now := time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC)
	job := Job{Description: "Work 09:00-17:00 UTC"}
	for _, tc := range []struct{ location, zone, hours string }{{"Tangerang Selatan, Banten, Indonesia", "Asia/Jakarta", "16:00–00:00 (+1 day)"}, {"Denpasar, Bali", "Asia/Makassar", "17:00–01:00 (+1 day)"}, {"Jayapura, Indonesia", "Asia/Jayapura", "18:00–02:00 (+1 day)"}, {"Europe/London", "Europe/London", "09:00–17:00"}} {
		got := WorkingHours(job, Profile{Location: tc.location}, now)
		if got.TargetZone != tc.zone || got.TargetLocation != tc.location || got.Examples[0].YourHours != tc.hours {
			t.Fatal(tc, got)
		}
	}
	for _, location := range []string{"Indonesia", "Jakarta or Bali", "Asia/Jakarta or Asia/Makassar", "Unknown city", ""} {
		got := WorkingHours(job, Profile{Location: location}, now)
		if got.TargetZone != "" || len(got.Examples) != 0 || got.Status != "unknown" {
			t.Fatal("ambiguous location defaulted to a zone", location, got)
		}
	}
}

func TestProfileTimeZoneFollowsDaylightSaving(t *testing.T) {
	got := WorkingHours(Job{Description: "Work 09:00-17:00 UTC"}, Profile{Location: "Europe/London"}, time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC))
	if got.Examples[0].YourHours != "10:00–18:00" || got.TargetLabel != "BST (GMT+1:00)" {
		t.Fatal(got)
	}
}
