package research

import (
	"context"
	"github.com/aqil/ai-job-hunter/internal/domain"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
)

type transport func(*http.Request) (*http.Response, error)

func (f transport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestPublicEvidence(t *testing.T) {
	p := New("news.example.org")
	p.Client = &http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`<rss><channel><item><title>Fixture Company announces changes</title><description>Source claims a restructuring.</description><link>https://news.example.org/story</link><pubDate>Mon, 02 Nov 2026 09:00:00 +0000</pubDate></item><item><title>Unrelated company</title><link>https://news.example.org/other</link></item></channel></rss>`))}, nil
	})}
	ev, errs := p.Research(context.Background(), domain.Company{Name: "Fixture Company", ResearchURLs: []string{"https://news.example.org/rss", "https://not-allowed.example/feed"}})
	if len(ev) != 1 || len(errs) != 1 || ev[0].Category != "public_source_claim" || ev[0].PublishedAt == nil || ev[0].RetrievedAt.IsZero() {
		t.Fatal(ev, errs)
	}
}
func TestDenyPrivateNetworks(t *testing.T) {
	for _, raw := range []string{"127.0.0.1", "10.0.0.1", "169.254.169.254", "::1", "fc00::1"} {
		if publicIP(net.ParseIP(raw)) {
			t.Fatalf("accepted private network %s", raw)
		}
	}
	p := New("example.com")
	if p.allowed("http://example.com/feed") || p.allowed("https://example.com:11434/feed") || p.allowed("https://example.net/feed") {
		t.Fatal("unsafe URL accepted")
	}
}
