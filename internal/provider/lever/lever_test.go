package lever

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
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"id":"abc","text":"Backend Engineer","descriptionPlain":"Go APIs","hostedUrl":"https://jobs.example.com/abc","workplaceType":"hybrid","categories":{"location":"Jakarta","commitment":"Full-time"},"salaryRange":{"min":100,"max":200,"currency":"IDR","interval":"per-month"}}]`))
	}))
	defer s.Close()
	p := Provider{httpclient.New(time.Second), s.URL}
	v, e := p.Fetch(context.Background(), domain.CompanySource{Board: "test"})
	if e != nil {
		t.Fatal(e)
	}
	if len(v) != 1 || v[0].Job.SalaryPeriod != "monthly" || *v[0].Job.SalaryMax != 200 {
		t.Fatalf("unexpected %+v", v)
	}
}
