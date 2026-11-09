package domain

import (
	"regexp"
	"strings"
)

// InferTimeZone resolves known Indonesian cities or an explicitly supplied IANA zone.
// Country-only or conflicting locations remain unknown because Indonesia spans three zones.
func InferTimeZone(location string) string {
	found := ""
	for _, zone := range namedZone.FindAllString(location, -1) {
		if loc, _ := scheduleZone(zone); loc != nil {
			if found != "" && found != zone {
				return ""
			}
			found = zone
		}
	}

	for _, group := range []struct{ zone, places string }{
		{"Asia/Jakarta", `jakarta|tangerang(?: selatan)?|bogor|depok|bekasi|jabodetabek|banten|bandung|surabaya|yogyakarta|semarang|medan|padang|palembang|aceh|pekanbaru|pontianak`},
		{"Asia/Makassar", `makassar|bali|denpasar|lombok|mataram|kupang|samarinda|balikpapan|manado`},
		{"Asia/Jayapura", `jayapura|ambon|sorong|manokwari|timika`},
	} {
		if regexp.MustCompile(`(?i)\b(?:` + group.places + `)\b`).MatchString(strings.TrimSpace(location)) {
			if found != "" && found != group.zone {
				return ""
			}
			found = group.zone
		}
	}
	return found
}
