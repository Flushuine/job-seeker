package http

import (
	"fmt"
	"github.com/aqil/ai-job-hunter/internal/provider/resume"
	"io"
	"net/http"
	"path/filepath"
	"strings"
)

func (h *Handler) uploadCV(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, resume.MaxFileBytes+(64<<10))
	if e := r.ParseMultipartForm(resume.MaxFileBytes); e != nil {
		problem(w, 400, "invalid_upload", "Upload one PDF or TXT file up to 5 MB")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	if r.MultipartForm == nil || len(r.MultipartForm.File) != 1 || len(r.MultipartForm.File["file"]) != 1 {
		problem(w, 400, "invalid_upload", "Provide exactly one CV in the file field")
		return
	}
	file, header, e := r.FormFile("file")
	if e != nil {
		problem(w, 400, "invalid_upload", "Choose a CV file to upload")
		return
	}
	defer file.Close()
	filename := filepath.Base(strings.ReplaceAll(header.Filename, "\\", "/"))
	if len(filename) > 200 || filename == "." || filename == "" {
		problem(w, 422, "validation", "CV filename must contain 1–200 characters")
		return
	}
	data, e := io.ReadAll(io.LimitReader(file, resume.MaxFileBytes+1))
	if e != nil {
		h.fail(w, r, fmt.Errorf("read CV upload: %w", e))
		return
	}
	text, e := resume.Extract(r.Context(), filename, data)
	if e != nil {
		h.fail(w, r, e)
		return
	}
	status, e := h.Service.UploadCV(r.Context(), filename, text)
	if e != nil {
		h.fail(w, r, e)
		return
	}
	write(w, http.StatusAccepted, status)
}
func (h *Handler) cvStatus(w http.ResponseWriter, r *http.Request) {
	status, e := h.Store.LatestCV(r.Context())
	h.respond(w, r, status, e)
}
