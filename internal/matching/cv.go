package matching

import (
	"fmt"
	"github.com/aqil/ai-job-hunter/internal/domain"
	"regexp"
	"strconv"
	"strings"
)

var cvYears = regexp.MustCompile(`(?i)\b(\d{1,2})\s*\+?\s*(?:years?|yrs?|tahun)\b`)

func GroundCV(a domain.CVAnalysis, text string) (domain.CVAnalysis, error) {
	if e := domain.ValidateCVAnalysis(a); e != nil {
		return a, e
	}
	flat := strings.ToLower(strings.Join(strings.Fields(text), " "))
	for _, field := range []struct {
		name  string
		value *string
	}{{"Name", &a.Name}, {"Location", &a.Location}} {
		*field.value = strings.TrimSpace(*field.value)
		if *field.value != "" && !strings.Contains(flat, strings.ToLower(strings.Join(strings.Fields(*field.value), " "))) {
			*field.value = ""
			a.Unknowns = append(a.Unknowns, field.name+" was not explicitly supported by the CV and was left unchanged.")
		}
	}
	if a.YearsExperience != nil {
		supported := false
		for _, match := range cvYears.FindAllStringSubmatch(text, -1) {
			n, _ := strconv.Atoi(match[1])
			if n == *a.YearsExperience {
				supported = true
			}
		}
		if !supported {
			a.YearsExperience = nil
			a.Unknowns = append(a.Unknowns, "Experience was not explicitly stated and was left unchanged.")
		}
	}
	skills := []string{}
	seen := map[string]bool{}
	for _, skill := range a.Skills {
		skill = strings.TrimSpace(skill)
		key := strings.ToLower(skill)
		if !seen[key] && mentions(text, skill) {
			seen[key] = true
			skills = append(skills, skill)
		}
	}
	if len(skills) == 0 {
		return a, fmt.Errorf("%w: no extracted skills could be grounded in the CV; existing profile retained", domain.ErrInvalidCV)
	}
	if len(skills) < len(a.Skills) {
		a.Unknowns = append(a.Unknowns, "Unsupported or duplicate skill suggestions were discarded.")
	}
	a.Skills = skills
	return a, nil
}
