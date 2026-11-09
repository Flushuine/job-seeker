package greenhouse

import (
	"context"
	"github.com/aqil/ai-job-hunter/internal/domain"
	"github.com/aqil/ai-job-hunter/internal/provider/httpclient"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/example/jobs" || r.URL.Query().Get("content") != "true" {
			t.Errorf("unexpected request %s", r.URL)
		}
		w.Write([]byte(`{"jobs":[{"id":42,"title":"Senior Go Engineer","content":"&lt;p&gt;Build APIs&lt;/p&gt;","location":{"name":"Remote"},"absolute_url":"https://example.com/jobs/42?utm_source=test"}],"meta":{"total":1}}`))
	}))
	defer server.Close()
	p := Provider{httpclient.New(time.Second), server.URL}
	jobs, e := p.Fetch(context.Background(), domain.CompanySource{Board: "example"})
	if e != nil {
		t.Fatal(e)
	}
	if len(jobs) != 1 || jobs[0].ExternalID != "42" || jobs[0].Job.Remote != "remote" || jobs[0].Job.Seniority != "senior" {
		t.Fatalf("unexpected normalization: %+v", jobs)
	}
}
func TestIncompleteResponse(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"jobs":[],"meta":{"total":2}}`)) }))
	defer s.Close()
	p := Provider{httpclient.New(time.Second), s.URL}
	if _, e := p.Fetch(context.Background(), domain.CompanySource{}); e == nil {
		t.Fatal("accepted incomplete snapshot")
	}
}
