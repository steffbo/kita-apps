package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/response"
)

// parseUUIDParam extracts and parses a UUID from URL parameters.
// Returns the parsed UUID and true if successful, or writes an error response and returns false.
func parseUUIDParam(w http.ResponseWriter, r *http.Request, paramName string) (uuid.UUID, bool) {
	param := chi.URLParam(r, paramName)
	id, err := uuid.Parse(param)
	if err != nil {
		response.BadRequest(w, "invalid "+paramName)
		return uuid.Nil, false
	}
	return id, true
}
