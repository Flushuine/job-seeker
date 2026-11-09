package domain

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"
)

type WorkHours struct {
	TargetZone     string             `json:"target_zone"`
	TargetLabel    string             `json:"target_label"`
	TargetLocation string             `json:"target_location"`
	Status         string             `json:"status"`
	Source         string             `json:"source"`
	Examples       []WorkHoursExample `json:"examples"`
	Note           string             `json:"note"`
}
type WorkHoursExample struct {
	Zone       string `json:"zone"`
	LocalHours string `json:"local_hours"`
	YourHours  string `json:"your_hours"`
	Difference string `json:"difference"`
}

var explicitZone = regexp.MustCompile(`(?i)\b(?:UTC|GMT)\s*([+-])\s*(\d{1,2})(?::(\d{2}))?\b`)
var namedZone = regexp.MustCompile(`\b(?:Africa|America|Asia|Atlantic|Australia|Europe|Pacific)/[A-Za-z_]+(?:/[A-Za-z_]+)?\b`)
var hoursRange = regexp.MustCompile(`(?i)\b(\d{1,2})(?::([0-5]\d))?\s*(am|pm)?\s*(?:-|–|—|to)\s*(\d{1,2})(?::([0-5]\d))?\s*(am|pm)?\s*(?:\(?\s*)(UTC\s*[+-]\s*\d{1,2}(?::\d{2})?|GMT\s*[+-]\s*\d{1,2}(?::\d{2})?|UTC|GMT|CET|CEST|EST|EDT|PST|PDT|[A-Za-z_]+/[A-Za-z_/]+)\b`)
var regionEMEA = regexp.MustCompile(`(?i)\bEMEA\b|Europe,? Middle East`)
var regionEurope = regexp.MustCompile(`(?i)\bEurope(?:an)?\b`)

func scheduleZone(label string) (*time.Location, string) {
	clean := strings.ToUpper(strings.ReplaceAll(label, " ", ""))
	if m := explicitZone.FindStringSubmatch(clean); m != nil {
		h, _ := strconv.Atoi(m[2])
		min, _ := strconv.Atoi(m[3])
		if h > 14 || min > 59 || (h == 14 && min != 0) {
			return nil, ""
		}
		offset := h*3600 + min*60
		if m[1] == "-" {
			offset = -offset
		}
		return time.FixedZone(clean, offset), clean
	}
	fixed := map[string]int{"UTC": 0, "GMT": 0, "CET": 1, "CEST": 2, "EST": -5, "EDT": -4, "PST": -8, "PDT": -7}
	if offset, ok := fixed[clean]; ok {
		return time.FixedZone(clean, offset*3600), clean
	}
	if namedZone.MatchString(label) {
		if loc, e := time.LoadLocation(label); e == nil {
			return loc, label
		}
	}
	return nil, ""
}
func clockMinutes(h, m, period string) (int, bool) {
	hour, _ := strconv.Atoi(h)
	minute, _ := strconv.Atoi(m)
	if period != "" {
		if hour < 1 || hour > 12 {
			return 0, false
		}
		hour %= 12
		if strings.EqualFold(period, "pm") {
			hour += 12
		}
	} else if hour > 23 {
		return 0, false
	}
	return hour*60 + minute, true
}
func minuteLabel(minutes int) string {
	day := int(math.Floor(float64(minutes) / 1440))
	minutes -= day * 1440
	suffix := ""
	if day == 1 {
		suffix = " (+1 day)"
	} else if day == -1 {
		suffix = " (previous day)"
	} else if day != 0 {
		suffix = fmt.Sprintf(" (%+d days)", day)
	}
	return fmt.Sprintf("%02d:%02d%s", minutes/60, minutes%60, suffix)
}
func workExample(loc *time.Location, label string, start, end int, now time.Time, target *time.Location) WorkHoursExample {
	// Use the source zone's current date; named zones follow daylight saving rules.
	date := now.In(loc)
	mid := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, loc)
	if end <= start {
		end += 1440
	}
	convert := func(minutes int) int {
		local := time.Date(mid.Year(), mid.Month(), mid.Day()+minutes/1440, (minutes%1440)/60, minutes%60, 0, 0, loc)
		userTime := local.In(target)
		base := time.Date(mid.Year(), mid.Month(), mid.Day(), 0, 0, 0, 0, time.UTC)
		day := int(time.Date(userTime.Year(), userTime.Month(), userTime.Day(), 0, 0, 0, 0, time.UTC).Sub(base).Hours() / 24)
		return day*1440 + userTime.Hour()*60 + userTime.Minute()
	}
	_, offset := time.Date(date.Year(), date.Month(), date.Day(), start/60, start%60, 0, 0, loc).Zone()
	targetOffsetName, targetOffset := now.In(target).Zone()
	delta := targetOffset/60 - offset/60
	direction := "ahead of"
	if delta < 0 {
		direction = "behind"
		delta = -delta
	}
	diff := "Same time as your location"
	if delta != 0 {
		diff = fmt.Sprintf("Your time (%s) is %dh %02dm %s %s", targetOffsetName, delta/60, delta%60, direction, label)
	}
	return WorkHoursExample{Zone: label, LocalHours: minuteLabel(start) + "–" + minuteLabel(end), YourHours: minuteLabel(convert(start)) + "–" + minuteLabel(convert(end)), Difference: diff}
}

// WorkingHours reports source-stated windows or explicitly hypothetical local business hours.
func WorkingHours(j Job, p Profile, now time.Time) WorkHours {
	text := j.Title + "\n" + j.Location + "\n" + j.Description
	result := WorkHours{Status: "unknown", Examples: []WorkHoursExample{}, Note: "Working hours and time zone are not stated. Ask the recruiter about overlap with your time zone; a hiring country does not establish a work schedule."}

	result.TargetLocation = p.Location
	result.TargetZone = InferTimeZone(p.Location)
	if result.TargetZone == "" {
		result.Note = "Your CV/profile location does not identify a supported, unambiguous time zone. Update your profile location to calculate working hours."
		return result
	}
	target, err := time.LoadLocation(result.TargetZone)
	if err != nil {
		return result
	}
	name, offset := now.In(target).Zone()
	sign := "+"
	if offset < 0 {
		sign = "-"
		offset = -offset
	}
	result.TargetLabel = fmt.Sprintf("%s (GMT%s%d:%02d)", name, sign, offset/3600, (offset%3600)/60)
	seenWindows := map[string]bool{}
	for _, m := range hoursRange.FindAllStringSubmatch(text, -1) {
		// A trailing AM/PM alone leaves the first endpoint ambiguous (e.g. 1–5pm).
		if (m[3] == "") != (m[6] == "") {
			continue
		}
		start, ok := clockMinutes(m[1], m[2], m[3])
		end, okEnd := clockMinutes(m[4], m[5], m[6])
		loc, label := scheduleZone(m[7])
		if !ok || !okEnd || loc == nil {
			continue
		}
		key := fmt.Sprintf("%s/%d/%d", label, start, end)
		if seenWindows[key] {
			continue
		}
		seenWindows[key] = true
		result.Status = "stated"
		if result.Source != "" {
			result.Source += "; "
		}
		result.Source += m[0]
		result.Examples = append(result.Examples, workExample(loc, label, start, end, now, target))
		if len(result.Examples) == 3 {
			break
		}
	}
	if result.Status == "stated" {
		result.Note = "Converted from a time window in the posting; it may describe overlap or availability rather than a full shift. Confirm with the recruiter. Named zones use today's daylight saving rules; explicit offsets and abbreviations stay fixed."
		return result
	}
	zones := namedZone.FindAllString(text, -1)
	zones = append(zones, explicitZone.FindAllString(text, -1)...)
	withoutOffsets := explicitZone.ReplaceAllString(text, "")
	for _, abbr := range []string{"UTC", "GMT", "CET", "CEST", "EST", "EDT", "PST", "PDT"} {
		if regexp.MustCompile(`\b` + abbr + `\b`).MatchString(withoutOffsets) {
			zones = append(zones, abbr)
		}
	}
	if len(zones) == 0 {
		if regionEMEA.MatchString(text) {
			zones = []string{"Europe/London", "Europe/Berlin", "Asia/Dubai"}
		} else if regionEurope.MatchString(text) {
			zones = []string{"Europe/London", "Europe/Berlin"}
		}
	}
	seen := map[string]bool{}
	for _, zone := range zones {
		loc, label := scheduleZone(zone)
		if loc == nil || seen[label] {
			continue
		}
		seen[label] = true
		result.Examples = append(result.Examples, workExample(loc, label, 9*60, 17*60, now, target))
		if len(result.Examples) == 3 {
			break
		}
	}
	if len(result.Examples) > 0 {
		result.Status = "example"
		result.Note = "Illustrative 09:00–17:00 local hours, not the employer's confirmed shift. EMEA/Europe spans multiple time zones; these are examples only. Named zones use today's daylight saving rules and may shift seasonally. Confirm required hours with the recruiter."
	}
	return result
}
