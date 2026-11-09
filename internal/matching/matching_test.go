package matching

import (
	"github.com/aqil/ai-job-hunter/internal/domain"
	"testing"
)

func TestDeterministicNoAI(t *testing.T) {
	p := domain.Profile{Skills: []string{"Go", "PostgreSQL"}, YearsExperience: 4}
	pref := domain.Preferences{Roles: []string{"Backend Engineer"}, Remote: []string{"remote"}}
	d := Evaluate(domain.Job{Title: "Backend Engineer", Description: "Golang and PostgreSQL", Remote: "remote", Available: true}, p, pref)
	if d.TechnicalScore == nil || *d.TechnicalScore != 100 || d.Recommendation != "Worth Considering" {
		t.Fatal(d)
	}
	d = Evaluate(domain.Job{Title: "Backend Engineer", Description: "MongoDB platform", Available: true}, domain.Profile{Skills: []string{"Go"}}, pref)
	for _, s := range d.MatchingSkills {
		if s == "Go" {
			t.Fatal("Go must not match MongoDB")
		}
	}
	d = Evaluate(domain.Job{Title: "Engineer", Available: true}, p, pref)
	if d.TechnicalScore != nil {
		t.Fatal("missing technologies must be unknown")
	}
}
func TestGenericGoAndRestAreNotTechnologies(t *testing.T) {
	d := Evaluate(domain.Job{Available: true, Title: "Contract Operations", Description: "Improve go-to-market operations, go live with launches and coordinate the rest of the team."}, domain.Profile{Skills: []string{"Go", "REST APIs"}}, domain.Preferences{})
	if len(d.MatchingSkills) != 0 || d.TechnicalScore != nil {
		t.Fatal("generic words mistaken for technologies", d)
	}
}
func TestRecogniseNonProfileBackendLanguages(t *testing.T) {
	d := Evaluate(domain.Job{Available: true, Title: "Ruby on Rails Software Engineer", Description: "Build with Ruby and Rails", Remote: "remote"}, domain.Profile{Skills: []string{"Go", "Node.js"}}, domain.Preferences{Roles: []string{"Software Engineer"}})
	if d.TechnicalScore == nil || *d.TechnicalScore != 0 || len(d.MissingSkills) != 2 {
		t.Fatal("explicit non-profile language requirements should not get a perfect technology score", d)
	}
}
