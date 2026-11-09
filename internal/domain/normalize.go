package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"golang.org/x/net/html"
	stdhtml "html"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

func Hash(v any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func PlainText(s string) string {
	doc, e := html.Parse(strings.NewReader(stdhtml.UnescapeString(s)))
	if e != nil {
		return s
	}
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style") {
			return
		}
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
			b.WriteByte(' ')
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return strings.Join(strings.Fields(b.String()), " ")
}
func CanonicalURL(raw string) (string, error) {
	u, e := url.Parse(raw)
	if e != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil {
		return "", fmt.Errorf("invalid posting URL")
	}
	u.Fragment = ""
	u.Host = strings.ToLower(u.Host)
	q := u.Query()
	for k := range q {
		if strings.HasPrefix(strings.ToLower(k), "utm_") || k == "source" || k == "ref" {
			q.Del(k)
		}
	}
	u.RawQuery = q.Encode()
	u.Path = strings.TrimRight(u.Path, "/")
	return u.String(), nil
}

var yearsPattern = regexp.MustCompile(`(?i)(?:at least|minimum(?: of)?|min\.?)[ ]+(\d{1,2})[ ]*\+?[ ]+years?|\b(\d{1,2})\+[ ]+years?`)

func Normalize(j Job) (Job, error) {
	var e error
	j.CanonicalURL, e = CanonicalURL(j.CanonicalURL)
	if e != nil {
		return j, e
	}
	j.Title = strings.Join(strings.Fields(j.Title), " ")
	if j.Title == "" {
		return j, fmt.Errorf("empty job title")
	}
	j.Description = PlainText(j.Description)
	if j.MinExperience == nil {
		m := yearsPattern.FindStringSubmatch(j.Description)
		if len(m) > 0 {
			raw := m[1]
			if raw == "" {
				raw = m[2]
			}
			if n, e := strconv.Atoi(raw); e == nil && n <= 40 {
				j.MinExperience = &n
			}
		}
	}
	if j.ApplicationURL == "" {
		j.ApplicationURL = j.CanonicalURL
	}
	if j.SourceURL == "" {
		j.SourceURL = j.CanonicalURL
	}
	if j.Remote != "remote" && j.Remote != "hybrid" && j.Remote != "onsite" {
		j.Remote = "unknown"
	}
	loc := strings.ToLower(j.Location)
	if j.Remote == "unknown" {
		if strings.Contains(loc, "hybrid") {
			j.Remote = "hybrid"
		} else if strings.Contains(loc, "remote") {
			j.Remote = "remote"
		}
	}
	title := strings.ToLower(j.Title)
	for _, s := range []string{"intern", "junior", "mid", "senior", "staff", "principal", "lead"} {
		if j.Seniority == "" && strings.Contains(title, s) {
			j.Seniority = s
			break
		}
	}
	j.DedupeKey = Hash([]string{strings.ToLower(strings.Join(strings.Fields(j.Title), " ")), strings.ToLower(strings.Join(strings.Fields(j.Location), " "))})
	j.Fingerprint = Hash([]any{j.Title, j.Description, j.MinExperience, j.Location, j.Country, j.EligibleCountries, j.Worldwide, j.Remote, j.Seniority, j.EmploymentType, j.SalaryMin, j.SalaryMax, j.Currency, j.SalaryPeriod})
	return j, nil
}
