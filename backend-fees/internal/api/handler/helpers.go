package handler

import (
	"errors"
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

// parseUpload parses a multipart upload. The body size is capped globally by
// middleware.MaxBodySize; exceeding it yields 413. Other parse failures yield 400.
func parseUpload(w http.ResponseWriter, r *http.Request) bool {
	err := r.ParseMultipartForm(10 << 20)
	if err == nil {
		return true
	}
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		response.TooLarge(w, "Datei zu groß (maximal 5 MB)")
		return false
	}
	response.BadRequest(w, "ungültiger Upload: "+err.Error())
	return false
}
