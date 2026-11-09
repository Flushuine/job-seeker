package ollama

import (
	"strings"
	"testing"
)

func TestDecodeCVAnalysis(t *testing.T) {
	valid := `{"name":"Ada","location":"Jakarta","years_experience":null,"skills":["Go"],"summary":"Builds services.","unknowns":[]}`
	if _, e := DecodeCVAnalysis([]byte(valid)); e != nil {
		t.Fatal(e)
	}
	for _, bad := range []string{strings.Replace(valid, `"skills":["Go"]`, `"skills":null`, 1), strings.Replace(valid, `"years_experience":null`, `"years_experience":90`, 1), strings.Replace(valid, `"summary":"Builds services."`, `"summary":""`, 1), strings.Replace(valid, `,"unknowns":[]`, "", 1), valid + "{}"} {
		if _, e := DecodeCVAnalysis([]byte(bad)); e == nil {
			t.Fatal("invalid analysis accepted", bad)
		}
	}
}
func TestCVExcerptRetainsBeginningAndEnd(t *testing.T) {
	text := "HEAD" + strings.Repeat("文", 100) + "TAIL"
	got := CVExcerpt(text, 30)
	if !strings.HasPrefix(got, "HEAD") || !strings.HasSuffix(got, "TAIL") || !strings.Contains(got, "omitted") {
		t.Fatal(got)
	}
	if CVExcerpt("short", 30) != "short" {
		t.Fatal("short CV truncated")
	}
}
