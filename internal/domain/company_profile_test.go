package domain

import (
	"errors"
	"testing"
	"time"
)

func companyFact(value string) CompanyFact {
	return CompanyFact{Value: value, SourceURL: "https://example.org/leadership", SourceName: "Fixture company"}
}
func TestCompanyProfileRequiresExplicitSources(t *testing.T) {
	now := time.Now()
	for name, updates := range map[string]CompanyProfile{"missing source": {"ceo": {Value: "Fixture CEO"}}, "unsafe URL": {"ceo": {Value: "Fixture CEO", SourceURL: "javascript:alert(1)", SourceName: "Fixture"}}, "unknown field": {"rating": companyFact("100")}, "nationality without CEO": {"ceo_nationality": companyFact("Unknown")}} {
		t.Run(name, func(t *testing.T) {
			_, e := MergeCompanyProfile(nil, updates, now)
			if !errors.Is(e, ErrInvalidCompanyProfile) {
				t.Fatal("accepted invalid company profile", e)
			}
		})
	}
	profile, e := MergeCompanyProfile(nil, CompanyProfile{"ceo": companyFact("Fixture CEO"), "headquarters": companyFact("Jakarta, Indonesia")}, now)
	if e != nil {
		t.Fatal(e)
	}
	if _, ok := profile["ceo_nationality"]; ok {
		t.Fatal("nationality must not be inferred from headquarters")
	}
	if !profile["ceo"].RecordedAt.Equal(now) {
		t.Fatal("recording timestamp not assigned")
	}
}
func TestCompanyProfileMergeClearAndDates(t *testing.T) {
	first := time.Now().Add(-24 * time.Hour)
	current, e := MergeCompanyProfile(nil, CompanyProfile{"origin_country": companyFact("Indonesia"), "ceo": companyFact("First CEO"), "ceo_nationality": companyFact("Indonesian")}, first)
	if e != nil {
		t.Fatal(e)
	}
	now := time.Now()
	next, e := MergeCompanyProfile(current, CompanyProfile{"headquarters": companyFact("Jakarta")}, now)
	if e != nil {
		t.Fatal(e)
	}
	if next["origin_country"].Value != "Indonesia" || !next["ceo"].RecordedAt.Equal(first) {
		t.Fatal("partial update lost fields or manufactured freshness")
	}
	same, e := MergeCompanyProfile(next, CompanyProfile{"ceo": next["ceo"]}, now)
	if e != nil || !same["ceo"].RecordedAt.Equal(first) {
		t.Fatal("retry must not change unchanged source timestamps")
	}
	changed, e := MergeCompanyProfile(next, CompanyProfile{"ceo": companyFact("Second CEO")}, now)
	if e != nil {
		t.Fatal(e)
	}
	if _, ok := changed["ceo_nationality"]; ok {
		t.Fatal("old nationality was transferred to a new leader")
	}
	if current["ceo"].Value != "First CEO" {
		t.Fatal("merge mutated the previous profile")
	}
	cleared, e := MergeCompanyProfile(next, CompanyProfile{"ceo": {}}, now)
	if e != nil || cleared["ceo"].Value != "" || cleared["ceo_nationality"].Value != "" {
		t.Fatal("clearing CEO should clear related nationality", cleared, e)
	}
}
func TestCompanyProfileRejectsStaleNationalityOnNewCEO(t *testing.T) {
	current := CompanyProfile{"ceo": companyFact("First CEO"), "ceo_nationality": companyFact("American")}
	_, e := MergeCompanyProfile(current, CompanyProfile{"ceo": companyFact("Second CEO"), "ceo_nationality": current["ceo_nationality"]}, time.Now())
	if !errors.Is(e, ErrInvalidCompanyProfile) {
		t.Fatal("stale nationality copied to changed CEO")
	}
}
