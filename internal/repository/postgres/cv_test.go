package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/aqil/ai-job-hunter/internal/domain"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestIntegrationCVUploadsApplyLatestAtomically(t *testing.T) {
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
	old, e := s.Profile(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := s.SaveProfile(ctx, &old); e != nil {
			t.Error(e)
		}
		s.DB.Model(&domain.Profile{}).Where("id = ?", ProfileID).Updates(map[string]any{"latest_cv_upload_id": old.LatestCVUploadID, "cv_fingerprint": old.CVFingerprint})
	}()
	pref, e := s.Preferences(ctx)
	if e != nil {
		t.Fatal(e)
	}
	upload := func(fingerprint string) domain.CVUpload {
		t.Helper()
		doc := domain.CVUpload{Filename: "fixture.txt", ExtractedText: "private fixture source", Fingerprint: fingerprint, Model: "fixture", PromptVersion: "v1", AIContext: 4096}
		task, e := s.CreateCVUpload(ctx, &doc)
		if e != nil {
			t.Fatal(e)
		}
		if task.Kind != "cv" || task.ResourceID != doc.ID {
			t.Fatal("upload not queued", task)
		}
		t.Cleanup(func() { s.DB.Where("resource_id = ? AND kind = ?", doc.ID, "cv").Delete(&domain.AnalysisTask{}) })
		return doc
	}
	a := domain.CVAnalysis{Name: "Ada Fixture", Location: "Jakarta", Skills: []string{"Go", "PostgreSQL"}, Summary: "Builds reliable Go services.", Unknowns: []string{}}
	first := upload("first")
	latest := upload("latest")
	if e = s.ApplyCV(ctx, first.ID, a); e != nil {
		t.Fatal(e)
	}
	before, _ := s.Profile(ctx)
	if before.Name != old.Name {
		t.Fatal("superseded upload overwrote profile")
	}
	if e = s.ApplyCV(ctx, latest.ID, a); e != nil {
		t.Fatal(e)
	}
	p, e := s.Profile(ctx)
	if e != nil || p.Name != a.Name || p.TimeZone != "Asia/Jakarta" || p.CVFingerprint != "latest" || !reflect.DeepEqual([]string(p.Skills), a.Skills) || p.YearsExperience != old.YearsExperience {
		t.Fatal("profile not updated safely", p, e)
	}
	var n int64
	s.DB.Table("profile_skills").Where("profile_id = ?", ProfileID).Count(&n)
	if n != 2 {
		t.Fatal("normalised skills stale", n)
	}
	afterPref, _ := s.Preferences(ctx)
	if !reflect.DeepEqual(pref, afterPref) {
		t.Fatal("CV changed job preferences")
	}
	status, e := s.LatestCV(ctx)
	if e != nil || status.Upload.ID != latest.ID || status.Upload.AppliedAt == nil {
		t.Fatal("latest status missing", status, e)
	}
	raw, _ := json.Marshal(status)
	if strings.Contains(string(raw), "private fixture source") || strings.Contains(string(raw), "base_profile_fingerprint") {
		t.Fatal("private upload context exposed")
	}
	repeated := upload("latest")
	cached, e := s.CachedCV(ctx, repeated)
	if e != nil || cached == nil || cached.Summary != a.Summary {
		t.Fatal("CV cache miss", cached, e)
	}
	changed, _ := s.Profile(ctx)
	changed.Summary = "Manually edited while analysis ran"
	if e = s.SaveProfile(ctx, &changed); e != nil {
		t.Fatal(e)
	}
	if e = s.ApplyCV(ctx, repeated.ID, a); !errors.Is(e, domain.ErrProfileChanged) {
		t.Fatal("concurrent manual edit not protected", e)
	}
	p, _ = s.Profile(ctx)
	if p.Summary != changed.Summary {
		t.Fatal("manual edit lost")
	}
}
