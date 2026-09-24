package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/middleware"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/request"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

// WarningListResponse represents a paginated list of warnings
// @Description Paginated warnings list as returned by the warnings endpoint
type WarningListResponse struct {
	Data       []domain.TransactionWarning `json:"data"`
	Total      int                         `json:"total" example:"3"`
	Page       int                         `json:"page" example:"1"`
	PerPage    int                         `json:"perPage" example:"20"`
	TotalPages int                         `json:"totalPages" example:"1"`
} //@name WarningList

// GetWarnings handles GET /import/warnings
// @Summary Get import warnings
// @Description Get a paginated list of warnings generated during import processing
// @Tags Import
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param perPage query int false "Items per page" default(20)
// @Success 200 {object} WarningListResponse "Import warnings"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /import/warnings [get]
func (h *ImportHandler) GetWarnings(w http.ResponseWriter, r *http.Request) {
	pagination := request.GetPagination(r)

	warnings, total, err := h.importService.GetWarnings(r.Context(), pagination.Offset, pagination.PerPage)
	if err != nil {
		response.InternalError(w, "failed to get warnings")
		return
	}

	response.Paginated(w, warnings, total, pagination.Page, pagination.PerPage)
}

// DismissWarningRequest represents a request to dismiss a warning.
// @Description Request body for dismissing a warning
type DismissWarningRequest struct {
	Note string `json:"note" example:"Differenz wurde bar ausgeglichen"`
} //@name DismissWarningRequest

// DismissWarning handles POST /import/warnings/{id}/dismiss
// @Summary Dismiss a warning
// @Description Dismiss an import warning with an optional note
// @Tags Import
// @Accept json
// @Security BearerAuth
// @Param id path string true "Warning ID (UUID)"
// @Param dismiss body DismissWarningRequest true "Dismissal note"
// @Success 204 "Warning dismissed"
// @Failure 400 {object} response.ErrorBody "Invalid warning ID"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Warning not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /import/warnings/{id}/dismiss [post]
func (h *ImportHandler) DismissWarning(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(w, "invalid warning ID")
		return
	}

	var req DismissWarningRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	userCtx := middleware.GetUserFromContext(r)
	if userCtx == nil {
		response.Unauthorized(w, "not authenticated")
		return
	}

	userID, _ := uuid.Parse(userCtx.UserID)

	err = h.importService.DismissWarning(r.Context(), id, userID, req.Note)
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "warning not found")
			return
		}
		response.InternalError(w, "failed to dismiss warning")
		return
	}

	response.NoContent(w)
}

// ResolveLateFeeResponse represents the result of resolving a late payment warning
// @Description Late payment resolution result
type ResolveLateFeeResponse struct {
	WarningID     string  `json:"warningId" example:"550e8400-e29b-41d4-a716-446655440000"`
	LateFeeID     string  `json:"lateFeeId" example:"550e8400-e29b-41d4-a716-446655440001"`
	LateFeeAmount float64 `json:"lateFeeAmount" example:"10.00"`
} //@name ResolveLateFeeResponse

// ResolveLateFee handles POST /import/warnings/{id}/resolve-late-fee
// @Summary Resolve late payment warning by creating late fee
// @Description Resolves a LATE_PAYMENT warning by creating a 10 EUR REMINDER fee linked to the original fee
// @Tags Import
// @Produce json
// @Security BearerAuth
// @Param id path string true "Warning ID (UUID)"
// @Success 200 {object} ResolveLateFeeResponse "Late fee created"
// @Failure 400 {object} response.ErrorBody "Invalid warning ID or warning is not a late payment"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Warning not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /import/warnings/{id}/resolve-late-fee [post]
func (h *ImportHandler) ResolveLateFee(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(w, "invalid warning ID")
		return
	}

	userCtx := middleware.GetUserFromContext(r)
	if userCtx == nil {
		response.Unauthorized(w, "not authenticated")
		return
	}

	userID, _ := uuid.Parse(userCtx.UserID)

	result, err := h.importService.ResolveWarningWithLateFee(r.Context(), id, userID)
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "warning not found")
			return
		}
		if err == service.ErrInvalidInput {
			response.BadRequest(w, "warning is not a late payment warning or is already resolved")
			return
		}
		response.InternalError(w, "failed to resolve late payment warning")
		return
	}

	resp := ResolveLateFeeResponse{
		WarningID:     result.WarningID.String(),
		LateFeeID:     result.LateFeeID.String(),
		LateFeeAmount: result.LateFeeAmount,
	}
	response.Success(w, resp)
}
