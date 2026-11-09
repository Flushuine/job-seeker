package http

import (
	"github.com/aqil/ai-job-hunter/internal/config"
	"io"
	"log/slog"
	"net/http/httptest"
	"testing"
)

func TestCompanyExclusionRequiresAuthAndValidID(t *testing.T) {
	h := New(nil, nil, config.Config{Token: "test-token-with-at-least-24-characters"}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	for _, tc := range []struct {
		auth bool
		code int
	}{{false, 401}, {true, 400}} {
		r := httptest.NewRequest("POST", "/api/companies/invalid-id/exclude", nil)
		if tc.auth {
			r.Header.Set("Authorization", "Bearer test-token-with-at-least-24-characters")
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != tc.code {
			t.Fatal(w.Code, w.Body.String())
		}
	}
}
