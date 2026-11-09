package ollama

import (
	"context"
	"encoding/json"
	"github.com/aqil/ai-job-hunter/internal/domain"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func valid() domain.Match {
	return domain.Match{OverallScore: 80, TechnicalScore: 90, MatchingSkills: []string{"Go"}, MissingSkills: []string{}, ExperienceCompatibility: "unknown", LocationCompatibility: "unknown", SalaryCompatibility: "unknown", Advantages: []string{"Go"}, Disadvantages: []string{}, Concerns: []string{}, Unknowns: []string{"Salary"}, Recommendation: "Worth Considering", Confidence: "low", Explanation: "Go role; location eligibility and salary unknown"}
}
func TestStructuredRequest(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" {
			t.Error(r.URL.Path)
		}
		var b map[string]any
		json.NewDecoder(r.Body).Decode(&b)
		if b["stream"] != false || b["think"] != false || b["format"] == nil || b["keep_alive"] != "1m" {
			t.Errorf("unsafe local options %+v", b)
		}
		m, _ := json.Marshal(valid())
		json.NewEncoder(w).Encode(map[string]any{"response": string(m), "done": true, "done_reason": "stop", "prompt_eval_count": 12, "eval_count": 20})
	}))
	defer s.Close()
	p := Provider{Client: s.Client(), BaseURL: s.URL, Model: "qwen3:1.7b"}
	m, in, out, e := p.Analyse(context.Background(), domain.Job{}, domain.Profile{}, domain.Preferences{})
	if e != nil || m.TechnicalScore != 90 || in != 12 || out != 20 {
		t.Fatal(m, in, out, e)
	}
}
func TestValidation(t *testing.T) {
	b, _ := json.Marshal(valid())
	if _, e := DecodeMatch(b); e != nil {
		t.Fatal(e)
	}
	for _, raw := range []string{`{}`, `{"technical_score":900}`, `null`, string(b[:len(b)-1]) + `,"extra":1}`} {
		if _, e := DecodeMatch([]byte(raw)); e == nil {
			t.Fatalf("accepted invalid JSON %s", raw)
		}
	}
	m := valid()
	m.OverallScore = 101
	b, _ = json.Marshal(m)
	if _, e := DecodeMatch(b); e == nil {
		t.Fatal("accepted out-of-range score")
	}
}
func TestCancellation(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(100 * time.Millisecond):
		}
	}))
	defer s.Close()
	p := Provider{Client: s.Client(), BaseURL: s.URL, Model: "fixture"}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, _, _, e := p.Analyse(ctx, domain.Job{}, domain.Profile{}, domain.Preferences{}); e == nil {
		t.Fatal("expected cancellation")
	}
}
func TestTruncation(t *testing.T) {
	if got := Truncate("你好世界", 2); got != "你好 [truncated; later requirements may be unknown]" {
		t.Fatal(got)
	}
}
