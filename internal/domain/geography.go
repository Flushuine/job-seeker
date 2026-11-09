package domain

import (
	"regexp"
	"sort"
	"strings"
	"sync"
)

// Country aliases and region membership are deterministic vocabulary, not user
// preferences. Nearby office locations and eligible countries remain editable.
var countryAliases = map[string][]string{
	"indonesia": {"Indonesia", "IDN"}, "united states": {"United States", "USA", "U.S.", "United States of America"},
	"united kingdom": {"United Kingdom", "UK", "U.K.", "Great Britain"}, "canada": {"Canada"}, "australia": {"Australia"},
	"singapore": {"Singapore"}, "malaysia": {"Malaysia"}, "vietnam": {"Vietnam", "Viet Nam"}, "thailand": {"Thailand"}, "philippines": {"Philippines"},
	"india": {"India"}, "japan": {"Japan"}, "china": {"China"}, "south korea": {"South Korea", "Korea"}, "new zealand": {"New Zealand"},
	"germany": {"Germany", "Deutschland"}, "france": {"France"}, "netherlands": {"Netherlands"}, "spain": {"Spain"}, "poland": {"Poland"},
	"portugal": {"Portugal"}, "ireland": {"Ireland"}, "sweden": {"Sweden"}, "switzerland": {"Switzerland"}, "brazil": {"Brazil"}, "mexico": {"Mexico"},
	"argentina": {"Argentina"}, "israel": {"Israel"}, "south africa": {"South Africa"}, "united arab emirates": {"United Arab Emirates", "UAE"},
}
var countryCodes = map[string]string{"ID": "indonesia", "US": "united states", "GB": "united kingdom", "CA": "canada", "AU": "australia", "SG": "singapore", "MY": "malaysia", "VN": "vietnam", "TH": "thailand", "PH": "philippines", "IN": "india", "JP": "japan", "DE": "germany", "FR": "france", "NL": "netherlands", "NZ": "new zealand"}
var regions = map[string][]string{
	"south asia": {"india"}, "east asia": {"china", "japan", "south korea"}, "central asia": {}, "west asia": {"united arab emirates", "israel"},
	"apac":           {"indonesia", "singapore", "malaysia", "vietnam", "thailand", "philippines", "india", "japan", "china", "south korea", "australia", "new zealand"},
	"asia pacific":   {"indonesia", "singapore", "malaysia", "vietnam", "thailand", "philippines", "india", "japan", "china", "south korea", "australia", "new zealand"},
	"asia":           {"indonesia", "singapore", "malaysia", "vietnam", "thailand", "philippines", "india", "japan", "china", "south korea"},
	"southeast asia": {"indonesia", "singapore", "malaysia", "vietnam", "thailand", "philippines"},
	"asean":          {"indonesia", "singapore", "malaysia", "vietnam", "thailand", "philippines"},
	"emea":           {"united kingdom", "germany", "france", "netherlands", "spain", "poland", "portugal", "ireland", "sweden", "switzerland", "south africa", "israel", "united arab emirates"},
	"europe":         {"united kingdom", "germany", "france", "netherlands", "spain", "poland", "portugal", "ireland", "sweden", "switzerland"},
	"european union": {"germany", "france", "netherlands", "spain", "poland", "portugal", "ireland", "sweden"},
	"north america":  {"united states", "canada", "mexico"}, "americas": {"united states", "canada", "mexico", "brazil", "argentina"},
	"latin america": {"mexico", "brazil", "argentina"}, "latam": {"mexico", "brazil", "argentina"},
}

var onlyScopePattern = regexp.MustCompile(`(?i)\b(?:United States|United Kingdom|US|USA|UK|Canada|Australia|Singapore|Malaysia|Vietnam|Thailand|Philippines|India|Japan|Germany|France|Netherlands|Indonesia|Europe|EMEA|APAC|North America|Latin America|LATAM)[ -]+(?:(?:based|residents|candidates|applicants)[ -]+)?only\b`)

var locationPatterns sync.Map

func locationPattern(key, pattern string) *regexp.Regexp {
	p, ok := locationPatterns.Load(key)
	if !ok {
		p, _ = locationPatterns.LoadOrStore(key, regexp.MustCompile(pattern))
	}
	return p.(*regexp.Regexp)
}
func locationToken(text, term string) bool {
	return locationPattern(term, `(?i)(^|[^a-z0-9])`+regexp.QuoteMeta(term)+`([^a-z0-9]|$)`).MatchString(text)
}
func canonicalCountry(s string) string {
	for code, country := range countryCodes {
		if strings.EqualFold(strings.TrimSpace(s), code) {
			return country
		}
	}
	for name, aliases := range countryAliases {
		if strings.EqualFold(s, name) || has(s, aliases) {
			return name
		}
	}
	return strings.ToLower(strings.TrimSpace(s))
}
func allowedCountry(country string, targets []string) bool {
	for _, target := range targets {
		if canonicalCountry(target) == canonicalCountry(country) {
			return true
		}
	}
	return false
}
func localLocation(text string, nearby []string) bool {
	for _, term := range nearby {
		if locationToken(text, term) {
			return true
		}
	}
	return false
}

// scope matches explicit country/region labels. Unknown place names are never
// geocoded or assigned a distance. In particular, a city-only remote label does
// not prove which countries can employ the applicant.
var regionNames = func() []string {
	keys := []string{}
	for k := range regions {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return len(keys[i]) > len(keys[j]) })
	return keys
}()

func scope(text string, targets []string) (known, allowed bool) {
	text = strings.ReplaceAll(text, "-", " ")
	for name, aliases := range countryAliases {
		for _, alias := range append([]string{name}, aliases...) {
			if locationToken(text, alias) {
				known = true
				if allowedCountry(name, targets) {
					allowed = true
				}
			}
		}
	}
	for code, name := range countryCodes {
		if locationPattern("code:"+code, `(^|[^A-Za-z0-9])`+code+`([^A-Za-z0-9]|$)`).MatchString(text) {
			known = true
			if allowedCountry(name, targets) {
				allowed = true
			}
		}
	}
	regionText := text
	for _, region := range regionNames {
		members := regions[region]
		if locationToken(regionText, region) {
			known = true
			regionText = locationPattern(region, `(?i)(^|[^a-z0-9])`+regexp.QuoteMeta(region)+`([^a-z0-9]|$)`).ReplaceAllString(regionText, " ")
			for _, member := range members {
				if allowedCountry(member, targets) {
					allowed = true
				}
			}
		}
	}
	return
}
func geoResult(status, why string) Factor {
	f := Factor{Name: "geography", Status: status, Explanation: why}
	if status != "unknown" {
		v := 0.0
		if status == "match" {
			v = 100
		}
		f.Score = &v
	}
	return f
}
func Geography(j Job, pref Preferences) Factor {
	if len(pref.NearbyLocations) == 0 && len(pref.Countries) == 0 {
		return geoResult("unknown", "No office area or country eligibility preference configured")
	}
	remote := j.Remote == "remote"
	if !remote && j.Remote != "hybrid" && j.Remote != "onsite" {
		return geoResult("unknown", "Working arrangement must be confirmed before assessing office distance")
	}
	if !remote && len(pref.NearbyLocations) > 0 {
		if strings.TrimSpace(j.Location) == "" || strings.EqualFold(j.Location, "unknown") {
			return geoResult("unknown", "Office location is not stated; confirm that it is in your preferred area")
		}
		if localLocation(j.Location, pref.NearbyLocations) {
			return geoResult("match", "Office location matches your configured nearby areas")
		}
		// A country or remote label alone does not identify an office city.
		if allowedCountry(j.Location, pref.Countries) || strings.EqualFold(j.Location, "hybrid") || strings.EqualFold(j.Location, "onsite") {
			return geoResult("unknown", "Office city is not stated; country alone does not establish a nearby commute")
		}
		return geoResult("mismatch", "Office location is outside your configured nearby areas: "+j.Location)
	}
	if len(pref.Countries) == 0 {
		return geoResult("unknown", "Configure eligible countries to check remote hiring restrictions")
	}
	// Explicit mandatory statements in the description can clarify generic labels.
	for _, part := range strings.FieldsFunc(j.Location+";"+j.Description, func(r rune) bool { return r == '.' || r == '\n' || r == ';' }) {
		lower := strings.ToLower(part)
		onlyClauses := onlyScopePattern.FindAllString(part, -1)
		if len(onlyClauses) > 0 {
			known, allowed := scope(strings.Join(onlyClauses, ", "), pref.Countries)
			if known && !allowed {
				return geoResult("mismatch", "The posting explicitly limits remote hiring to another country or region")
			}
		}
		mandatory := containsAny(lower, []string{"must be based in", "must reside in", "must be located in", "only hiring in", "only open to candidates in", "remote within", "candidates must be based in"})
		if mandatory {
			known, allowed := scope(part, pref.Countries)
			if known && !allowed {
				return geoResult("mismatch", "The posting explicitly requires residence outside your eligible countries")
			}
		}
		for _, phrase := range []string{"excluding ", "except "} {
			if i := strings.Index(lower, phrase); i >= 0 {
				known, excluded := scope(part[i+len(phrase):], pref.Countries)
				if known && excluded {
					return geoResult("mismatch", "The posting explicitly excludes your country or hiring region")
				}
			}
		}
		for _, target := range pref.Countries {
			for _, alias := range append([]string{target}, countryAliases[canonicalCountry(target)]...) {
				for _, phrase := range []string{"excluding ", "except ", "not available in ", "not hiring in ", "cannot hire in "} {
					if strings.Contains(lower, phrase+strings.ToLower(alias)) {
						return geoResult("mismatch", "The posting explicitly excludes your eligible country")
					}
				}
			}
		}
	}
	if j.EligibleCountries != nil && len(j.EligibleCountries) > 0 {
		for _, country := range j.EligibleCountries {
			if allowedCountry(country, pref.Countries) {
				return geoResult("match", "The source explicitly lists an eligible hiring country")
			}
		}
		return geoResult("mismatch", "The source restricts hiring to other countries")
	}
	if j.Worldwide != nil && *j.Worldwide {
		return geoResult("match", "The source explicitly advertises worldwide hiring; confirm employer and time-zone requirements")
	}
	if j.Country != "" {
		if allowedCountry(j.Country, pref.Countries) {
			return geoResult("match", "The source lists an eligible country")
		}
		return geoResult("mismatch", "The source lists a different hiring country: "+j.Country)
	}
	label := j.Location
	if containsAny(label, []string{"worldwide", "anywhere", "global"}) {
		cleaned := locationPattern("label-exceptions", `(?i)\b(?:except|excluding)\b.*`).ReplaceAllString(label, "")
		cleaned = locationPattern("label-timezones", `(?i)\b(?:US|UK|United States|United Kingdom|Europe)[ -]+(?:(?:business|working)[ -]+)?(?:hours|time[ -]?zones?)\b`).ReplaceAllString(cleaned, "")
		known, allowed := scope(cleaned, pref.Countries)
		if known && !allowed {
			return geoResult("unknown", "Worldwide label conflicts with a regional hiring label; confirm eligibility")
		}
		return geoResult("match", "The posting advertises worldwide remote work; confirm employer and time-zone requirements")
	}
	known, allowed := scope(label, pref.Countries)
	if known {
		if allowed {
			return geoResult("match", "Advertised hiring location or region includes your eligible country; confirm specific employer requirements")
		}
		return geoResult("mismatch", "Advertised hiring countries or region exclude your location: "+j.Location)
	}
	if remote && localLocation(j.Location, pref.NearbyLocations) {
		return geoResult("match", "Remote posting is associated with your configured local area; confirm employment eligibility")
	}
	return geoResult("unknown", "Remote work is stated, but permission to work from your country is unconfirmed")
}
