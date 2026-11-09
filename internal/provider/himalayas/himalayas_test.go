package himalayas

import (
	"context"
	"github.com/aqil/ai-job-hunter/internal/domain"
	"github.com/aqil/ai-job-hunter/internal/provider/httpclient"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPaginationAttribution(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("country") != "ID" {
			t.Error("country search constraint not forwarded")
		}
		if r.URL.Query().Get("page") == "1" {
			w.Write([]byte(`{"jobs":[{"title":"Go Engineer","companyName":"Test fixture","guid":"https://himalayas.app/companies/test/jobs/go","applicationLink":"https://himalayas.app/companies/test/jobs/go?apply=true","description":"Go","seniority":["Senior"],"locationRestrictions":[{"name":"Indonesia"}],"expiryDate":2000000000}]}`))
		} else {
			w.Write([]byte(`{"jobs":[]}`))
		}
	}))
	defer s.Close()
	client := httpclient.New(time.Second)
	client.Interval = 0
	p := Provider{client, s.URL}
	v, e := p.Fetch(context.Background(), domain.CompanySource{Board: "backend", Country: "ID"})
	if e != nil || len(v) != 1 {
		t.Fatal(v, e)
	}
	if v[0].Job.Country != "Indonesia" || v[0].Job.SourceName != "Himalayas" || v[0].Job.ApplicationURL != "https://himalayas.app/companies/test/jobs/go?apply=true" {
		t.Fatal(v[0])
	}
}
func TestMissingRestrictionsNotWorldwide(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") == "1" {
			w.Write([]byte(`{"jobs":[{"title":"Backend Engineer","companyName":"Fixture","guid":"https://example.org/job","applicationLink":"https://example.org/apply","description":"Go"}]}`))
		} else {
			w.Write([]byte(`{"jobs":[]}`))
		}
	}))
	defer s.Close()
	c := httpclient.New(time.Second)
	c.Interval = 0
	p := Provider{c, s.URL}
	jobs, e := p.Fetch(context.Background(), domain.CompanySource{Board: "backend"})
	if e != nil {
		t.Fatal(e)
	}
	if jobs[0].Job.Worldwide != nil || jobs[0].Job.Location == "Worldwide" {
		t.Fatal("missing country restrictions invented worldwide eligibility")
	}
}
