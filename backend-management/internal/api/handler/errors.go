package handler

import (
	"net/http"

	"github.com/knirpsenstadt/kita-apps/backend-management/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/service"
)

func writeServiceError(w http.ResponseWriter, err error) {
	switch service.GetCode(err) {
	case service.ErrCodeNotFound:
		response.NotFound(w, err.Error())
	case service.ErrCodeBadRequest:
		response.BadRequest(w, err.Error())
	case service.ErrCodeUnauthorized:
		response.Unauthorized(w, err.Error())
	case service.ErrCodeForbidden:
		response.Forbidden(w, err.Error())
	case service.ErrCodeConflict:
		response.Conflict(w, err.Error())
	default:
		response.InternalError(w, "Ein interner Fehler ist aufgetreten")
	}
}
