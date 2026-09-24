package middleware

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMaxBodySize(t *testing.T) {
	const limit = 16
	tests := []struct {
		name    string
		size    int
		wantErr bool
	}{
		{name: "below limit", size: limit - 1},
		{name: "exactly at limit", size: limit},
		{name: "above limit", size: limit + 1, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var readErr error
			var read int
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				data, err := io.ReadAll(r.Body)
				read, readErr = len(data), err
			})
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bytes.Repeat([]byte("x"), tt.size)))

			MaxBodySize(limit)(next).ServeHTTP(httptest.NewRecorder(), req)

			var tooLarge *http.MaxBytesError
			if tt.wantErr {
				if !errors.As(readErr, &tooLarge) {
					t.Fatalf("err = %v, want *http.MaxBytesError", readErr)
				}
				return
			}
			if readErr != nil || read != tt.size {
				t.Fatalf("read %d bytes, err %v", read, readErr)
			}
		})
	}
}

func TestMaxBodySize_NilBody(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Body = nil

	MaxBodySize(1)(next).ServeHTTP(httptest.NewRecorder(), req)

	if !called {
		t.Fatal("next handler not called for request without body")
	}
}
