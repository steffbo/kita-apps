package handler

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/middleware"
)

func multipartBody(t *testing.T, size int) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", "export.csv")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write(bytes.Repeat([]byte("a"), size)); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	return &buf, mw.FormDataContentType()
}

func TestParseUpload(t *testing.T) {
	tests := []struct {
		name       string
		size       int
		wantOK     bool
		wantStatus int
	}{
		{name: "small file", size: 1024, wantOK: true, wantStatus: http.StatusOK},
		{name: "too large", size: middleware.MaxUploadBytes + 1, wantStatus: http.StatusRequestEntityTooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, contentType := multipartBody(t, tt.size)
			var ok bool
			h := middleware.MaxBodySize(middleware.MaxUploadBytes)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ok = parseUpload(w, r)
			}))
			req := httptest.NewRequest(http.MethodPost, "/import/upload", body)
			req.Header.Set("Content-Type", contentType)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if ok != tt.wantOK || rec.Code != tt.wantStatus {
				t.Fatalf("ok=%v status=%d, want ok=%v status=%d", ok, rec.Code, tt.wantOK, tt.wantStatus)
			}
		})
	}

	t.Run("not multipart", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/import/upload", bytes.NewBufferString("x"))
		rec := httptest.NewRecorder()
		if parseUpload(rec, req) || rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})
}
