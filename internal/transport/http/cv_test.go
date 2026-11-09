package http

import (
	"bytes"
	"context"
	"github.com/aqil/ai-job-hunter/internal/config"
	"github.com/aqil/ai-job-hunter/internal/domain"
	"github.com/aqil/ai-job-hunter/internal/usecase"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"
)

type uploadRepo struct {
	usecase.Repository
	text string
}

func (r *uploadRepo) Preferences(context.Context) (domain.Preferences, error) {
	return domain.Preferences{AIEnabled: true, OllamaModel: "fixture", AIContext: 4096}, nil
}
func (r *uploadRepo) CreateCVUpload(_ context.Context, d *domain.CVUpload) (domain.AnalysisTask, error) {
	r.text = d.ExtractedText
	d.ID = "fixture"
	return domain.AnalysisTask{Kind: "cv", ResourceID: d.ID, Status: "queued"}, nil
}
func TestCVUploadRequiresAuthAndQueuesRedactedText(t *testing.T) {
	r := &uploadRepo{}
	h := New(nil, &usecase.Service{Repo: r, AIEnabled: true}, config.Config{Token: "test-token-with-at-least-24-characters"}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	cv := "Ada Example in Jakarta builds Go services using PostgreSQL and Docker with four years of experience in testing deployment and mentoring teams. ada@example.com"
	for _, tc := range []struct {
		name string
		auth bool
		code int
	}{{"cv.txt", false, 401}, {"cv.pdf", true, 422}, {"cv.txt", true, 202}} {
		var body bytes.Buffer
		form := multipart.NewWriter(&body)
		file, _ := form.CreateFormFile("file", tc.name)
		file.Write([]byte(cv))
		form.Close()
		req := httptest.NewRequest("POST", "/api/profile/cv", &body)
		req.Header.Set("Content-Type", form.FormDataContentType())
		if tc.auth {
			req.Header.Set("Authorization", "Bearer test-token-with-at-least-24-characters")
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != tc.code {
			t.Fatalf("%s: got %d: %s", tc.name, w.Code, w.Body)
		}
		if strings.Contains(w.Body.String(), "ada@example.com") || strings.Contains(w.Body.String(), "extracted_text") {
			t.Fatal("private CV text exposed")
		}
	}
	if strings.Contains(r.text, "ada@example.com") || !strings.Contains(r.text, "PostgreSQL") {
		t.Fatal("queued CV not redacted correctly")
	}
}
