package middleware

import "net/http"

// MaxUploadBytes caps every request body (CSV uploads are the largest payloads).
const MaxUploadBytes = 5 << 20

// MaxBodySize limits the request body to limit bytes. Reads beyond the limit
// fail with *http.MaxBytesError.
func MaxBodySize(limit int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, limit)
			}
			next.ServeHTTP(w, r)
		})
	}
}
