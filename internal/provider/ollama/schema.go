package ollama

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/aqil/ai-job-hunter/internal/domain"
)

const PromptVersion = "local-match-v3-cv-location"

func Schema() map[string]any {
	props := map[string]any{}
	for _, k := range []string{"overall_score", "technical_score"} {
		props[k] = map[string]any{"type": "number", "minimum": 0, "maximum": 100}
	}
	for _, k := range []string{"matching_skills", "missing_skills", "advantages", "disadvantages", "concerns", "unknowns"} {
		props[k] = map[string]any{"type": "array", "items": map[string]any{"type": "string"}}
	}
	for _, k := range []string{"experience_compatibility", "location_compatibility", "salary_compatibility", "explanation"} {
		props[k] = map[string]any{"type": "string"}
	}
	props["recommendation"] = map[string]any{"type": "string", "enum": domain.Recommendations}
	props["confidence"] = map[string]any{"type": "string", "enum": []string{"low", "medium", "high"}}
	required := []string{}
	for k := range props {
		required = append(required, k)
	}
	return map[string]any{"type": "object", "additionalProperties": false, "properties": props, "required": required}
}
func DecodeMatch(b []byte) (domain.Match, error) {
	var raw map[string]json.RawMessage
	if e := json.Unmarshal(b, &raw); e != nil {
		return domain.Match{}, e
	}
	props := Schema()["properties"].(map[string]any)
	if len(raw) != len(props) {
		return domain.Match{}, fmt.Errorf("unexpected or missing analysis fields")
	}
	for k := range props {
		if len(raw[k]) == 0 || string(raw[k]) == "null" {
			return domain.Match{}, fmt.Errorf("missing or null field %s", k)
		}
	}
	var m domain.Match
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if e := d.Decode(&m); e != nil {
		return m, e
	}
	return m, domain.ValidateMatch(m)
}
