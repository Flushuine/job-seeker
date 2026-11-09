package greenhouse

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aqil/ai-job-hunter/internal/domain"
	"github.com/aqil/ai-job-hunter/internal/provider/httpclient"
	"net/url"
	"strconv"
)

type Provider struct {
	Client  *httpclient.Client
	BaseURL string
}

func (p *Provider) Fetch(ctx context.Context, s domain.CompanySource) ([]domain.Posting, error) {
	base := p.BaseURL
	if base == "" {
		base = "https://boards-api.greenhouse.io/v1/boards"
	}
	var raw struct {
		Jobs []json.RawMessage `json:"jobs"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	if e := p.Client.Get(ctx, base+"/"+url.PathEscape(s.Board)+"/jobs?content=true", &raw); e != nil {
		return nil, e
	}
	if raw.Jobs == nil {
		return nil, fmt.Errorf("missing jobs array")
	}
	if raw.Meta.Total > len(raw.Jobs) {
		return nil, fmt.Errorf("incomplete board snapshot")
	}
	out := []domain.Posting{}
	for _, b := range raw.Jobs {
		var v struct {
			ID       int64  `json:"id"`
			Title    string `json:"title"`
			Content  string `json:"content"`
			URL      string `json:"absolute_url"`
			Location struct {
				Name string `json:"name"`
			} `json:"location"`
		}
		if e := json.Unmarshal(b, &v); e != nil {
			return nil, e
		}
		if v.ID == 0 {
			return nil, fmt.Errorf("missing Greenhouse id")
		}
		j, e := domain.Normalize(domain.Job{SourceName: "Greenhouse", CompanyID: s.CompanyID, Title: v.Title, Description: v.Content, Location: v.Location.Name, CanonicalURL: v.URL, SourcePayload: b})
		if e != nil {
			return nil, e
		}
		out = append(out, domain.Posting{ExternalID: strconv.FormatInt(v.ID, 10), Job: j})
	}
	return out, nil
}
