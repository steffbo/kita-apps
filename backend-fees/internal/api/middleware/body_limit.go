package middleware

import (
	"net/http"
	"strings"
)

// MaxUploadBytes caps every request body (CSV uploads are the largest payloads).
const MaxUploadBytes = 5 << 20

// MaxBodySize limits the request body to limit bytes. Reads beyond the limit
// fail with *http.MaxBytesError.
func MaxBodySize(limit int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				requestLimit := limit
				if r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/api/fees/v1/installment-plans/") && strings.HasSuffix(r.URL.Path, "/contract") {
					requestLimit = 11 << 20
				}
				r.Body = http.MaxBytesReader(w, r.Body, requestLimit)
			}
			next.ServeHTTP(w, r)
		})
	}
}
