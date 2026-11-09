// Package research retrieves only explicitly allowlisted public RSS/Atom feeds.
// Feed entries are source claims, not verified workplace facts or sentiment scores.
package research

import (
	"context"
	"encoding/xml"
	"fmt"
	"github.com/aqil/ai-job-hunter/internal/domain"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Provider struct {
	Client       *http.Client
	AllowedHosts map[string]bool
}

func New(hosts string) *Provider {
	allowed := map[string]bool{}
	for _, h := range strings.Split(hosts, ",") {
		if h = strings.TrimSpace(strings.ToLower(h)); h != "" {
			allowed[h] = true
		}
	}
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.Proxy = nil
	tr.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, e := net.SplitHostPort(address)
		if e != nil {
			return nil, e
		}
		ips, e := net.DefaultResolver.LookupIPAddr(ctx, host)
		if e != nil {
			return nil, e
		}
		for _, ip := range ips {
			if !publicIP(ip.IP) {
				return nil, fmt.Errorf("private network research target denied")
			}
		}
		if len(ips) == 0 {
			return nil, fmt.Errorf("no public IP")
		}
		return (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
	}
	p := &Provider{AllowedHosts: allowed}
	p.Client = &http.Client{Transport: tr, Timeout: 20 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return fmt.Errorf("research redirects are disabled")
	}}
	return p
}
func publicIP(ip net.IP) bool {
	return ip.IsGlobalUnicast() && !ip.IsPrivate() && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() && !ip.IsUnspecified()
}
func (p *Provider) allowed(raw string) bool {
	u, e := url.Parse(raw)
	return e == nil && u.Scheme == "https" && u.User == nil && (u.Port() == "" || u.Port() == "443") && p.AllowedHosts[strings.ToLower(u.Hostname())]
}
func (p *Provider) Research(ctx context.Context, c domain.Company) ([]domain.Evidence, []string) {
	evidence := []domain.Evidence{}
	errs := []string{}
	for _, raw := range c.ResearchURLs {
		if !p.allowed(raw) {
			errs = append(errs, "Feed URL is not HTTPS or its host is not allowlisted: "+raw)
			continue
		}
		req, e := http.NewRequestWithContext(ctx, "GET", raw, nil)
		if e != nil {
			errs = append(errs, e.Error())
			continue
		}
		req.Header.Set("User-Agent", "AIJobHunter/1.0 public-feed-reader")
		res, e := p.Client.Do(req)
		if e != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", raw, e))
			continue
		}
		b, e := io.ReadAll(io.LimitReader(res.Body, 2<<20+1))
		res.Body.Close()
		if e != nil || len(b) > 2<<20 || res.StatusCode != 200 {
			errs = append(errs, fmt.Sprintf("%s: unavailable feed (HTTP %d)", raw, res.StatusCode))
			continue
		}
		var feed struct {
			XMLName xml.Name
			Channel struct {
				Items []struct {
					Title       string `xml:"title"`
					Link        string `xml:"link"`
					Description string `xml:"description"`
					Date        string `xml:"pubDate"`
				} `xml:"item"`
			} `xml:"channel"`
			Entries []struct {
				Title string `xml:"title"`
				Links []struct {
					Href string `xml:"href,attr"`
					Rel  string `xml:"rel,attr"`
				} `xml:"link"`
				Summary string `xml:"summary"`
				Updated string `xml:"updated"`
			} `xml:"entry"`
		}
		if e = xml.Unmarshal(b, &feed); e != nil || (feed.XMLName.Local != "rss" && feed.XMLName.Local != "feed") {
			errs = append(errs, raw+": invalid RSS/Atom feed")
			continue
		}
		add := func(title, summary, link, date string) {
			text := domain.PlainText(title + " " + summary)
			if !strings.Contains(strings.ToLower(text), strings.ToLower(c.Name)) {
				return
			}
			u, e := domain.CanonicalURL(link)
			if e != nil {
				return
			}
			host, _ := url.Parse(raw)
			var published *time.Time
			for _, layout := range []string{time.RFC1123Z, time.RFC1123, time.RFC3339, time.RFC822Z} {
				if v, e := time.Parse(layout, date); e == nil {
					published = &v
					break
				}
			}
			if len(text) > 2500 {
				text = text[:2500]
			}
			evidence = append(evidence, domain.Evidence{CompanyID: c.ID, SourceURL: u, SourceName: host.Hostname(), PublishedAt: published, RetrievedAt: time.Now().UTC(), Summary: text, Category: "public_source_claim", Confidence: "low"})
		}
		for _, v := range feed.Channel.Items {
			add(v.Title, v.Description, v.Link, v.Date)
		}
		for _, v := range feed.Entries {
			link := ""
			for _, l := range v.Links {
				if l.Rel == "alternate" || l.Rel == "" {
					link = l.Href
					break
				}
			}
			add(v.Title, v.Summary, link, v.Updated)
		}
	}
	return evidence, errs
}
