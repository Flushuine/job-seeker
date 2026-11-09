package resume

import (
	"bytes"
	"context"
	"fmt"
	"github.com/aqil/ai-job-hunter/internal/domain"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

const MaxFileBytes = 5 << 20
const MaxTextBytes = 200000

var email = regexp.MustCompile(`(?i)\b[A-Z0-9._%+\-]+@[A-Z0-9.\-]+\.[A-Z]{2,}\b`)
var phone = regexp.MustCompile(`(?:\+?\d[\d ()\-]{7,}\d)`)
var url = regexp.MustCompile(`(?i)https?://[^\s<>]+`)

func RedactContacts(s string) string {
	s = email.ReplaceAllString(s, "[email removed]")
	s = phone.ReplaceAllString(s, "[phone removed]")
	return url.ReplaceAllString(s, "[URL removed]")
}

type limitedBuffer struct {
	bytes.Buffer
	max int
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if b.Len()+len(p) > b.max {
		return 0, fmt.Errorf("extracted CV text exceeds limit")
	}
	return b.Buffer.Write(p)
}
func Extract(ctx context.Context, filename string, data []byte) (string, error) {
	if len(data) == 0 || len(data) > MaxFileBytes {
		return "", fmt.Errorf("%w: upload a PDF or UTF-8 TXT file up to 5 MB", domain.ErrInvalidCV)
	}
	var text string
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".txt":
		if !utf8.Valid(data) || bytes.ContainsRune(data, '\x00') {
			return "", fmt.Errorf("%w: TXT file must contain UTF-8 text", domain.ErrInvalidCV)
		}
		text = string(data)
	case ".pdf":
		if !bytes.HasPrefix(data, []byte("%PDF-")) {
			return "", fmt.Errorf("%w: file is not a PDF", domain.ErrInvalidCV)
		}
		binary, e := exec.LookPath("pdftotext")
		if e != nil {
			return "", fmt.Errorf("%w: PDF extraction needs Poppler (brew install poppler), or upload a TXT file", domain.ErrInvalidCV)
		}
		extractCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		command := exec.CommandContext(extractCtx, binary, "-layout", "-enc", "UTF-8", "-", "-")
		command.Stdin = bytes.NewReader(data)
		out := &limitedBuffer{max: MaxTextBytes}
		command.Stdout = out
		if e = command.Run(); e != nil {
			if ctx.Err() != nil {
				return "", ctx.Err()
			}
			return "", fmt.Errorf("%w: cannot extract PDF text; use an unencrypted text-based PDF or TXT export", domain.ErrInvalidCV)
		}
		text = out.String()
	default:
		return "", fmt.Errorf("%w: supported CV formats are PDF and TXT", domain.ErrInvalidCV)
	}
	text = strings.TrimSpace(strings.ReplaceAll(text, "\r\n", "\n"))
	if len(text) > MaxTextBytes || len(strings.Fields(text)) < 20 {
		return "", fmt.Errorf("%w: CV must contain readable text (scanned PDFs need a text/OCR export)", domain.ErrInvalidCV)
	}
	return RedactContacts(text), nil
}
