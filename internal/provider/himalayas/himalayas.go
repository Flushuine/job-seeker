package himalayas

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aqil/ai-job-hunter/internal/domain"
	"github.com/aqil/ai-job-hunter/internal/provider/httpclient"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Provider struct {
	Client  *httpclient.Client
	BaseURL string
}

func (p *Provider) Fetch(ctx context.Context, s domain.CompanySource) ([]domain.Posting, error) {
	base := p.BaseURL
	if base == "" {
		base = "https://himalayas.app/jobs/api/search"
	}
	out := []domain.Posting{}
	seen := map[string]bool{}
	for page := 1; page <= 100; page++ {
		var raw struct {
			Jobs []json.RawMessage `json:"jobs"`
		}
		query := url.Values{"q": {s.Board}, "page": {strconv.Itoa(page)}}
		if s.Country != "" {
			query.Set("country", s.Country)
		}
		if e := p.Client.Get(ctx, base+"?"+query.Encode(), &raw); e != nil {
			return nil, e
		}
		if raw.Jobs == nil {
			return nil, fmt.Errorf("missing Himalayas jobs array")
		}
		if len(raw.Jobs) == 0 {
			return out, nil
		}
		for _, b := range raw.Jobs {
			var v struct {
				Title, CompanyName, EmploymentType, Currency, Description, ApplicationLink, Guid, SalaryPeriod string
				MinSalary, MaxSalary                                                                           *float64
				Seniority                                                                                      json.RawMessage
				LocationRestrictions                                                                           []json.RawMessage
				PubDate, ExpiryDate                                                                            int64
			}
			if e := json.Unmarshal(b, &v); e != nil {
				return nil, e
			}
			if v.Guid == "" || v.CompanyName == "" {
				return nil, fmt.Errorf("missing Himalayas identity")
			}
			if seen[v.Guid] {
				return nil, fmt.Errorf("repeated Himalayas page; refusing partial snapshot")
			}
			seen[v.Guid] = true
			locs := []string{}
			for _, rawLoc := range v.LocationRestrictions {
				var name string
				if json.Unmarshal(rawLoc, &name) != nil {
					var obj struct{ Name string }
					if e := json.Unmarshal(rawLoc, &obj); e != nil {
						return nil, e
					}
					name = obj.Name
				}
				locs = append(locs, name)
			}
			location := "Remote — hiring countries not provided"
			if v.LocationRestrictions != nil && len(locs) == 0 {
				location = "Worldwide"
			}
			if len(locs) > 0 {
				location = strings.Join(locs, ", ")
			}
			j := domain.Job{CompanyName: v.CompanyName, Title: v.Title, Description: v.Description, Remote: "remote", Location: location, SalaryMin: v.MinSalary, SalaryMax: v.MaxSalary, Currency: v.Currency, SalaryPeriod: v.SalaryPeriod, EmploymentType: strings.ToLower(strings.ReplaceAll(v.EmploymentType, " ", "-")), CanonicalURL: v.Guid, ApplicationURL: v.ApplicationLink, SourceName: "Himalayas", SourceURL: v.Guid, SourcePayload: b}
			if v.LocationRestrictions != nil {
				j.EligibleCountries = locs
				worldwide := len(locs) == 0
				j.Worldwide = &worldwide
			}
			if len(locs) == 1 {
				j.Country = locs[0]
			}
			if v.ExpiryDate > 0 {
				d := time.Unix(v.ExpiryDate, 0).UTC()
				j.ExpiresAt = &d
			}
			var seniority string
			var seniors []string
			if json.Unmarshal(v.Seniority, &seniority) != nil && json.Unmarshal(v.Seniority, &seniors) == nil && len(seniors) > 0 {
				seniority = seniors[0]
			}
			j.Seniority = strings.ToLower(strings.TrimSuffix(seniority, "-level"))
			if v.ApplicationLink == "" {
				return nil, fmt.Errorf("Himalayas missing application link")
			}
			if _, e := domain.CanonicalURL(v.ApplicationLink); e != nil {
				return nil, e
			}
			j, e := domain.Normalize(j)
			if e != nil {
				return nil, e
			}
			out = append(out, domain.Posting{ExternalID: v.Guid, Job: j})
		}
	}
	return nil, fmt.Errorf("Himalayas pagination limit reached; no partial snapshot ingested")
}
