package postgres

import (
	"context"
	"github.com/aqil/ai-job-hunter/internal/domain"
	"github.com/google/uuid"
	"os"
	"testing"
	"time"
)

func TestIntegrationSnapshot(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to an isolated PostgreSQL database")
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
	if e = s.Migrate(ctx); e != nil {
		t.Fatalf("migration not idempotent: %v", e)
	}
	c := domain.Company{Name: "Fixture-" + uuid.NewString()}
	src := domain.CompanySource{Provider: "greenhouse", Board: uuid.NewString(), Enabled: true}
	if e = s.SaveCompany(ctx, &c, &src); e != nil {
		t.Fatal(e)
	}
	j, e := domain.Normalize(domain.Job{CompanyID: c.ID, CompanyName: c.Name, Title: "Go Engineer", CanonicalURL: "https://example.com/" + uuid.NewString(), SourcePayload: []byte(`{"fixture":true}`)})
	if e != nil {
		t.Fatal(e)
	}
	posts := []domain.Posting{{ExternalID: "1", Job: j}}
	now := time.Now().UTC()
	for i := 0; i < 2; i++ {
		if e = s.SaveSnapshot(ctx, src, posts, now); e != nil {
			t.Fatal(e)
		}
	}
	page, e := s.Jobs(ctx, domain.JobQuery{CompanyID: c.ID, Page: 1, Limit: 20, Status: "all"})
	if e != nil || page.Total != 1 {
		t.Fatalf("idempotence: %+v %v", page, e)
	}
	id := page.Items[0].ID
	// Same role, different provider URL should preserve a single cross-source job.
	src2 := domain.CompanySource{Provider: "lever", Board: uuid.NewString(), Enabled: true}
	if e = s.SaveCompany(ctx, &c, &src2); e != nil {
		t.Fatal(e)
	}
	j.CanonicalURL += "-alternate"
	if e = s.SaveSnapshot(ctx, src2, []domain.Posting{{ExternalID: "lever1", Job: j}}, now); e != nil {
		t.Fatal(e)
	}
	page, e = s.Jobs(ctx, domain.JobQuery{CompanyID: c.ID, Page: 1, Limit: 20, Status: "all"})
	if e != nil || page.Total != 1 {
		t.Fatalf("cross-source dedupe: %v %+v", e, page)
	}
	for i := 0; i < 2; i++ {
		if e = s.SaveSnapshot(ctx, src, nil, now.Add(time.Hour)); e != nil {
			t.Fatal(e)
		}
	}
	job, e := s.Job(ctx, id)
	if e != nil || !job.Available {
		t.Fatal("other live source must keep job open", e)
	}
	if e = s.SaveSnapshot(ctx, src2, nil, now.Add(time.Hour)); e != nil {
		t.Fatal(e)
	}
	job, _ = s.Job(ctx, id)
	if !job.Available {
		t.Fatal("first missing snapshot incorrectly closed job")
	}
	if e = s.SaveSnapshot(ctx, src2, nil, now.Add(2*time.Hour)); e != nil {
		t.Fatal(e)
	}
	job, _ = s.Job(ctx, id)
	if job.Available {
		t.Fatal("two missing successful snapshots should close job")
	}
	a := domain.Application{JobID: id, Stage: "Saved"}
	if e = s.SaveApplication(ctx, &a, true); e != nil {
		t.Fatal(e)
	}
	a.Stage = "Applied"
	if e = s.SaveApplication(ctx, &a, false); e != nil {
		t.Fatal(e)
	}
	task, e := s.Enqueue(ctx, "job", id, 20)
	if e != nil {
		t.Fatal(e)
	}
	again, e := s.Enqueue(ctx, "job", id, 20)
	if e != nil || again.ID != task.ID {
		t.Fatal("queue must deduplicate", e)
	}
	if e = s.FinishTask(ctx, task.ID, nil); e != nil {
		t.Fatal(e)
	}
}
func TestIntegrationEnsureCompanyPreservesSettings(t *testing.T) {
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
	c := domain.Company{Name: "Fixture-research-" + uuid.NewString(), ResearchURLs: []string{"https://example.org/feed"}}
	if e = s.SaveCompany(ctx, &c, nil); e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 2; i++ {
		actual, e := s.EnsureCompany(ctx, c.Name)
		if e != nil || actual.ID != c.ID || len(actual.ResearchURLs) != 1 {
			t.Fatal("ingestion overwrote company settings", actual, e)
		}
	}
}
func TestIntegrationCVAndLocationPreferences(t *testing.T) {
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
	p, e := s.Profile(ctx)
	if e != nil {
		t.Fatal(e)
	}
	p.Summary = "Fixture: production Go services and mentoring"
	p.CVSource = "fixture-cv.pdf"
	if e = s.SaveProfile(ctx, &p); e != nil {
		t.Fatal(e)
	}
	got, e := s.Profile(ctx)
	if e != nil || got.Summary != p.Summary || got.CVSource != p.CVSource {
		t.Fatal("CV context was not persisted", got, e)
	}
	pref, e := s.Preferences(ctx)
	if e != nil {
		t.Fatal(e)
	}
	old := pref
	defer func() {
		if e = s.SavePreferences(ctx, &old); e != nil {
			t.Error(e)
		}
	}()
	pref.NearbyLocations = []string{"Jakarta", "Tangerang Selatan"}
	pref.Countries = []string{"Indonesia"}
	pref.HardLocation = true
	pref.RequireLocationMatch = true
	if e = s.SavePreferences(ctx, &pref); e != nil {
		t.Fatal(e)
	}
	saved, e := s.Preferences(ctx)
	if e != nil || !saved.HardLocation || !saved.RequireLocationMatch || len(saved.NearbyLocations) != 2 {
		t.Fatal("location policy was not persisted", saved, e)
	}
}
func TestIntegrationStableJobPagination(t *testing.T) {
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
	c := domain.Company{Name: "Fixture-pagination-" + uuid.NewString()}
	if e = s.SaveCompany(ctx, &c, nil); e != nil {
		t.Fatal(e)
	}
	now := time.Now().UTC()
	jobs := []domain.Job{}
	for i := 0; i < 201; i++ {
		id := uuid.NewString()
		jobs = append(jobs, domain.Job{Base: base(id), CompanyID: c.ID, CompanyName: c.Name, Title: "Same fixture title", CanonicalURL: "https://example.org/" + id, SourcePayload: []byte(`{"fixture":true}`), FirstSeen: now, LastSeen: now, Available: true})
	}
	if e = s.DB.WithContext(ctx).Create(&jobs).Error; e != nil {
		t.Fatal(e)
	}
	for _, order := range []string{"", "title", "company"} {
		seen := map[string]bool{}
		prev := ""
		for page := 1; page <= 6; page++ {
			got, e := s.Jobs(ctx, domain.JobQuery{CompanyID: c.ID, Sort: order, Page: page, Limit: 37})
			if e != nil {
				t.Fatal(e)
			}
			for _, j := range got.Items {
				if seen[j.ID] || j.ID < prev {
					t.Fatal("unstable ordering or duplicate across pages", order, j.ID)
				}
				seen[j.ID] = true
				prev = j.ID
			}
		}
		if len(seen) != 201 {
			t.Fatal("pagination dropped jobs", order, len(seen))
		}
	}
}
func TestIntegrationCompanyProfileSurvivesIngestion(t *testing.T) {
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
	c := domain.Company{Name: "Fixture-profile-" + uuid.NewString()}
	if e = s.SaveCompany(ctx, &c, nil); e != nil {
		t.Fatal(e)
	}
	fact := domain.CompanyFact{Value: "Fixture CEO", SourceURL: "https://example.org/leadership", SourceName: "Fixture source"}
	c, e = s.PatchCompanyProfile(ctx, c.ID, domain.CompanyProfile{"ceo": fact})
	if e != nil || c.Profile["ceo"].Value != fact.Value {
		t.Fatal("profile write failed", c, e)
	}
	recorded := c.Profile["ceo"].RecordedAt
	c, e = s.EnsureCompany(ctx, c.Name)
	if e != nil || c.Profile["ceo"].Value != fact.Value {
		t.Fatal("ingestion erased company profile", c, e)
	}
	c, e = s.PatchCompanyProfile(ctx, c.ID, domain.CompanyProfile{"headquarters": {Value: "Jakarta", SourceURL: "https://example.org/contact", SourceName: "Fixture source"}})
	if e != nil || !c.Profile["ceo"].RecordedAt.Equal(*recorded) {
		t.Fatal("partial profile update reset leadership or date", c, e)
	}
	if e = s.SaveCompany(ctx, &c, nil); e != nil {
		t.Fatal(e)
	}
	again, e := s.Company(ctx, c.ID)
	if e != nil || again.Profile["ceo"].Value != fact.Value {
		t.Fatal("company configuration erased profile", again, e)
	}
}
func TestIntegrationAppliedJobsHiddenWithoutDeletingHistory(t *testing.T) {
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
	c := domain.Company{Name: "Fixture-applied-" + uuid.NewString()}
	src := domain.CompanySource{Provider: "greenhouse", Board: uuid.NewString(), Enabled: true}
	if e = s.SaveCompany(ctx, &c, &src); e != nil {
		t.Fatal(e)
	}
	posts := []domain.Posting{}
	for _, title := range []string{"Fixture Backend Engineer Untracked", "Fixture Backend Engineer Saved", "Fixture Backend Engineer Applied"} {
		j, e := domain.Normalize(domain.Job{CompanyID: c.ID, CompanyName: c.Name, Title: title, CanonicalURL: "https://example.org/" + uuid.NewString(), Location: "Jakarta", Remote: "hybrid", SourcePayload: []byte(`{"fixture":true}`)})
		if e != nil {
			t.Fatal(e)
		}
		posts = append(posts, domain.Posting{ExternalID: title, Job: j})
	}
	if e = s.SaveSnapshot(ctx, src, posts, time.Now()); e != nil {
		t.Fatal(e)
	}
	all, e := s.Jobs(ctx, domain.JobQuery{CompanyID: c.ID, Page: 1, Limit: 10, Sort: "title", IncludeApplied: true})
	if e != nil || all.Total != 3 {
		t.Fatal(all, e)
	}
	ids := map[string]string{}
	for _, j := range all.Items {
		ids[j.Title] = j.ID
	}
	saved := domain.Application{JobID: ids["Fixture Backend Engineer Saved"], Stage: "Saved", Notes: "Keep recruiter notes"}
	if e = s.SaveApplication(ctx, &saved, true); e != nil {
		t.Fatal(e)
	}
	original := saved
	retry := domain.Application{JobID: saved.JobID, Stage: "Saved", Notes: saved.Notes}
	if e = s.SaveApplication(ctx, &retry, true); e != nil || retry.ID != original.ID || !retry.CreatedAt.Equal(original.CreatedAt) {
		t.Fatal("upsert did not return canonical application", retry, e)
	}
	applied := domain.Application{JobID: ids["Fixture Backend Engineer Applied"], Stage: "Applied", Notes: "Application submitted manually"}
	if e = s.SaveApplication(ctx, &applied, true); e != nil {
		t.Fatal(e)
	}
	for _, stage := range []string{"Applied", "Screening", "Interview", "Offer", "Rejected", "Withdrawn"} {
		applied.Stage = stage
		if e = s.SaveApplication(ctx, &applied, false); e != nil {
			t.Fatal(e)
		}
		if applied.JobID == "" || applied.CreatedAt.IsZero() || applied.UpdatedAt.IsZero() {
			t.Fatal("patch response lost persisted application data", applied)
		}
		first, e := s.Jobs(ctx, domain.JobQuery{CompanyID: c.ID, Page: 1, Limit: 1})
		if e != nil || first.Total != 2 {
			t.Fatal("applied job not removed before counting and pagination", first, e)
		}
		second, e := s.Jobs(ctx, domain.JobQuery{CompanyID: c.ID, Page: 2, Limit: 1})
		if e != nil || len(second.Items) != 1 || first.Items[0].ID == applied.JobID || second.Items[0].ID == applied.JobID || first.Items[0].ID == second.Items[0].ID {
			t.Fatal("applied filtering broke pagination", first, second, e)
		}
	}
	visible, e := s.Jobs(ctx, domain.JobQuery{CompanyID: c.ID, Page: 1, Limit: 10, IncludeApplied: true})
	if e != nil || visible.Total != 3 {
		t.Fatal("show-applied option did not restore history", visible, e)
	}
	job, e := s.Job(ctx, applied.JobID)
	if e != nil || !job.Available {
		t.Fatal("filtering deleted or closed the actual job", job, e)
	}
	applied.Stage = "Saved"
	if e = s.SaveApplication(ctx, &applied, false); e != nil {
		t.Fatal(e)
	}
	restored, e := s.Jobs(ctx, domain.JobQuery{CompanyID: c.ID, Page: 1, Limit: 10})
	if e != nil || restored.Total != 3 {
		t.Fatal("moving back to Saved should restore the job", restored, e)
	}
}
