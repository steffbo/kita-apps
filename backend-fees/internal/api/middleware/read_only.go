package middleware

import (
	"net/http"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/response"
)

// ReadOnly lets the given roles through on safe methods only (GET, HEAD,
// OPTIONS) and answers every other method with 403. It runs after
// AuthMiddleware and is a no-op for all other roles.
//
// Use it on every route group a read-only role (BOARD) can reach. A role added
// to RequireRole without ReadOnly can write.
func ReadOnly(roles ...string) func(http.Handler) http.Handler {
	restricted := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		restricted[role] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userCtx := GetUserFromContext(r)
			if userCtx == nil {
				next.ServeHTTP(w, r)
				return
			}
			if _, ok := restricted[userCtx.Role]; ok {
				switch r.Method {
				case http.MethodGet, http.MethodHead, http.MethodOptions:
				default:
					response.Error(w, http.StatusForbidden, "Diese Rolle hat nur Leserechte")
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
