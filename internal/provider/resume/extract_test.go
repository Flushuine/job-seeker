package resume

import (
	"context"
	"errors"
	"github.com/aqil/ai-job-hunter/internal/domain"
	"strings"
	"testing"
)

func TestExtractTextAndRejectInvalidDocuments(t *testing.T) {
	cv := "Ada Example\nJakarta\nSoftware engineer with 4+ years building Go services using PostgreSQL and Docker. Experienced in testing, deployment, mentoring and reliable production systems.\nada@example.com +62 812-3456-7890 https://example.com/profile"
	got, e := Extract(context.Background(), "cv.TXT", []byte(cv))
	if e != nil {
		t.Fatal(e)
	}
	for _, secret := range []string{"ada@example.com", "812-3456", "https://example.com"} {
		if strings.Contains(got, secret) {
			t.Fatal("contact information was retained")
		}
	}
	if !strings.Contains(got, "4+ years") || !strings.Contains(got, "PostgreSQL") {
		t.Fatal("professional context lost", got)
	}
	for _, tc := range []struct {
		name string
		data []byte
	}{{"cv.docx", []byte(cv)}, {"cv.pdf", []byte(cv)}, {"cv.txt", []byte("too short")}, {"cv.txt", []byte{0xff, 0x00}}, {"cv.txt", make([]byte, MaxFileBytes+1)}} {
		if _, e := Extract(context.Background(), tc.name, tc.data); !errors.Is(e, domain.ErrInvalidCV) {
			t.Fatalf("%s: expected invalid CV, got %v", tc.name, e)
		}
	}
}
