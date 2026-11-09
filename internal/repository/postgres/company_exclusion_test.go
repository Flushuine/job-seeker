package postgres

import (
	"context"
	"github.com/aqil/ai-job-hunter/internal/domain"
	"github.com/google/uuid"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestIntegrationExcludeCompanyPreservesPreferencesAndHistory(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL required")
	}
	ctx := context.Background()
	s, e := Open(ctx, dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if e = s.Migrate(ctx); e != nil {
		t.Fatal(e)
	}
	original, e := s.Preferences(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := s.SavePreferences(ctx, &original); e != nil {
			t.Error(e)
		}
	}()
	companies := []domain.Company{{Name: "Excluded-fixture-" + uuid.NewString()}, {Name: "Excluded-fixture-" + uuid.NewString()}}
	for i := range companies {
		if e = s.SaveCompany(ctx, &companies[i], nil); e != nil {
			t.Fatal(e)
		}
	}
	job := domain.Job{Base: base(""), CompanyID: companies[0].ID, CompanyName: companies[0].Name, Title: "Backend Engineer", CanonicalURL: "https://example.org/" + uuid.NewString(), SourcePayload: []byte(`{}`), Available: true, Remote: "remote", FirstSeen: time.Now(), LastSeen: time.Now()}
	if e = s.DB.Create(&job).Error; e != nil {
		t.Fatal(e)
	}
	application := domain.Application{JobID: job.ID, Stage: "Applied", Notes: "Keep this application history"}
	if e = s.SaveApplication(ctx, &application, true); e != nil {
		t.Fatal(e)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 4)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(index int) { defer wg.Done(); _, e := s.ExcludeCompany(ctx, companies[index%2].ID); errs <- e }(i)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	saved, e := s.Preferences(ctx)
	if e != nil {
		t.Fatal(e)
	}
	for _, company := range companies {
		count := 0
		for _, name := range saved.ExcludedCompanies {
			if strings.EqualFold(name, company.Name) {
				count++
			}
		}
		if count != 1 {
			t.Fatal("exclusion lost or duplicated", saved.ExcludedCompanies)
		}
	}
	eligibility := domain.Filter(job, domain.Profile{}, saved, time.Now())
	if eligibility.Eligible {
		t.Fatal("excluded company's jobs still eligible")
	}
	saved.ExcludedCompanies = original.ExcludedCompanies
	saved.UpdatedAt = original.UpdatedAt
	if !reflect.DeepEqual(saved, original) {
		t.Fatal("exclusion changed other preferences")
	}
	var actual domain.Application
	if e = s.DB.First(&actual, "id = ?", application.ID).Error; e != nil || actual.Notes != application.Notes || actual.Stage != "Applied" {
		t.Fatal("application history changed", actual, e)
	}
	if _, e = s.ExcludeCompany(ctx, uuid.NewString()); e == nil {
		t.Fatal("missing company accepted")
	}
}
