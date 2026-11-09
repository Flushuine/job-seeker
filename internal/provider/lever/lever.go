package lever

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aqil/ai-job-hunter/internal/domain"
	"github.com/aqil/ai-job-hunter/internal/provider/httpclient"
	"net/url"
	"strconv"
	"strings"
)

type Provider struct {
	Client  *httpclient.Client
	BaseURL string
}

func (p *Provider) Fetch(ctx context.Context, s domain.CompanySource) ([]domain.Posting, error) {
	base := p.BaseURL
	if base == "" {
		base = "https://api.lever.co/v0/postings"
		if s.Provider == "lever-eu" {
			base = "https://api.eu.lever.co/v0/postings"
		}
	}
	out := []domain.Posting{}
	seen := map[string]bool{}
	for skip := 0; skip < 20000; skip += 100 {
		var raw []json.RawMessage
		if e := p.Client.Get(ctx, base+"/"+url.PathEscape(s.Board)+"?mode=json&limit=100&skip="+strconv.Itoa(skip), &raw); e != nil {
			return nil, e
		}
		if raw == nil {
			return nil, fmt.Errorf("missing Lever array")
		}
		for _, b := range raw {
			var v struct {
				ID          string `json:"id"`
				Text        string `json:"text"`
				Description string `json:"descriptionPlain"`
				Additional  string `json:"additionalPlain"`
				URL         string `json:"hostedUrl"`
				Workplace   string `json:"workplaceType"`
				Categories  struct {
					Location   string `json:"location"`
					Commitment string `json:"commitment"`
				} `json:"categories"`
				Lists []struct {
					Text    string `json:"text"`
					Content string `json:"content"`
				} `json:"lists"`
				Salary *struct {
					Min      *float64 `json:"min"`
					Max      *float64 `json:"max"`
					Currency string   `json:"currency"`
					Interval string   `json:"interval"`
				} `json:"salaryRange"`
			}
			if e := json.Unmarshal(b, &v); e != nil {
				return nil, e
			}
			if v.ID == "" || seen[v.ID] {
				return nil, fmt.Errorf("missing or repeated Lever id; refusing incomplete snapshot")
			}
			seen[v.ID] = true
			desc := v.Description + " " + v.Additional
			for _, l := range v.Lists {
				desc += " " + l.Text + " " + l.Content
			}
			j := domain.Job{SourceName: "Lever", CompanyID: s.CompanyID, Title: v.Text, Description: desc, Location: v.Categories.Location, EmploymentType: strings.ToLower(v.Categories.Commitment), CanonicalURL: v.URL, Remote: v.Workplace, SourcePayload: b}
			if j.Remote == "on-site" {
				j.Remote = "onsite"
			}
			if v.Salary != nil {
				j.SalaryMin = v.Salary.Min
				j.SalaryMax = v.Salary.Max
				j.Currency = v.Salary.Currency
				switch v.Salary.Interval {
				case "per-year", "yearly":
					j.SalaryPeriod = "annual"
				case "per-month", "monthly":
					j.SalaryPeriod = "monthly"
				}
			}
			j, e := domain.Normalize(j)
			if e != nil {
				return nil, e
			}
			out = append(out, domain.Posting{ExternalID: v.ID, Job: j})
		}
		if len(raw) < 100 {
			return out, nil
		}
	}
	return nil, fmt.Errorf("Lever pagination safety limit exceeded")
}
