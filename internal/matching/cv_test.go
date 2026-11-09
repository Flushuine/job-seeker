package matching

import (
	"github.com/aqil/ai-job-hunter/internal/domain"
	"testing"
)

func TestGroundCVPreservesOnlySupportedProfileFields(t *testing.T) {
	years := 9
	a := domain.CVAnalysis{Name: "Invented Person", Location: "Singapore", YearsExperience: &years, Skills: []string{"Go", "PostgreSQL", "Go", "Kubernetes"}, Summary: "Builds Go services.", Unknowns: []string{}}
	got, e := GroundCV(a, "Ada Example in Jakarta. 4+ years of experience with Go and PostgreSQL.")
	if e != nil {
		t.Fatal(e)
	}
	if got.Name != "" || got.Location != "" || got.YearsExperience != nil || len(got.Skills) != 2 || len(got.Unknowns) != 4 {
		t.Fatalf("unsupported facts survived: %+v", got)
	}
	a.Name = "Ada Example"
	a.Location = "Jakarta"
	years = 4
	got, e = GroundCV(a, "Ada Example in Jakarta. 4+ years of experience with Go and PostgreSQL.")
	if e != nil || got.Name != a.Name || got.Location != a.Location || got.YearsExperience == nil || *got.YearsExperience != 4 {
		t.Fatal(got, e)
	}
	a.Skills = []string{"Kubernetes"}
	if _, e = GroundCV(a, "Go and PostgreSQL"); e == nil {
		t.Fatal("unsupported skills must not replace profile")
	}
}
