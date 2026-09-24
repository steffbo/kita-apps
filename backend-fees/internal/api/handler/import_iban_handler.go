package handler

import (
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/request"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

// IBANListResponse represents a paginated list of IBANs
// @Description Paginated IBAN list as returned by the blacklist/trusted endpoints
type IBANListResponse struct {
	Data       []domain.KnownIBAN `json:"data"`
	Total      int                `json:"total" example:"5"`
	Page       int                `json:"page" example:"1"`
	PerPage    int                `json:"perPage" example:"20"`
	TotalPages int                `json:"totalPages" example:"1"`
} //@name IBANList

// ChildTrustedIBANsResponse represents trusted IBANs for a child.
// @Description Trusted IBANs with usage counts
type ChildTrustedIBANsResponse struct {
	IBAN             string  `json:"iban" example:"DE89370400440532013000"`
	PayerName        *string `json:"payerName,omitempty" example:"Max Mustermann" binding:"optional"`
	TransactionCount int64   `json:"transactionCount" example:"4"`
} //@name ChildTrustedIBANsResponse

// ChildTrustedIBANs handles GET /import/trusted/child/{id}
// @Summary Get trusted IBANs for a child
// @Description Returns trusted IBANs linked to the child with transaction counts
// @Tags Import
// @Produce json
// @Security BearerAuth
// @Param id path string true "Child ID (UUID)"
// @Success 200 {array} ChildTrustedIBANsResponse "Trusted IBANs"
// @Failure 400 {object} response.ErrorBody "Invalid child ID"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /import/trusted/child/{id} [get]
func (h *ImportHandler) ChildTrustedIBANs(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	childID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(w, "invalid child ID")
		return
	}

	result, err := h.importService.GetTrustedIBANsForChild(r.Context(), childID)
	if err != nil {
		response.InternalError(w, "failed to load trusted ibans")
		return
	}

	response.Success(w, result)
}

// GetBlacklist handles GET /import/blacklist
// @Summary Get blacklisted IBANs
// @Description Get a paginated list of IBANs that are ignored during import
// @Tags Import
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param perPage query int false "Items per page" default(20)
// @Success 200 {object} IBANListResponse "Blacklisted IBANs"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /import/blacklist [get]
func (h *ImportHandler) GetBlacklist(w http.ResponseWriter, r *http.Request) {
	pagination := request.GetPagination(r)

	ibans, total, err := h.importService.GetBlacklist(r.Context(), pagination.Offset, pagination.PerPage)
	if err != nil {
		response.InternalError(w, "failed to get blacklist")
		return
	}

	response.Paginated(w, ibans, total, pagination.Page, pagination.PerPage)
}

// RemoveFromBlacklist handles DELETE /import/blacklist/{iban}
// @Summary Remove IBAN from blacklist
// @Description Remove an IBAN from the blacklist so transactions from it will be processed again
// @Tags Import
// @Security BearerAuth
// @Param iban path string true "IBAN (URL-encoded)"
// @Success 204 "IBAN removed from blacklist"
// @Failure 400 {object} response.ErrorBody "Invalid IBAN or IBAN is not blacklisted"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "IBAN not found in blacklist"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /import/blacklist/{iban} [delete]
func (h *ImportHandler) RemoveFromBlacklist(w http.ResponseWriter, r *http.Request) {
	iban, err := url.PathUnescape(chi.URLParam(r, "iban"))
	if err != nil || iban == "" {
		response.BadRequest(w, "invalid IBAN")
		return
	}

	err = h.importService.RemoveFromBlacklist(r.Context(), iban)
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "IBAN not found in blacklist")
			return
		}
		if err == service.ErrInvalidInput {
			response.BadRequest(w, "IBAN is not blacklisted")
			return
		}
		response.InternalError(w, "failed to remove from blacklist")
		return
	}

	response.NoContent(w)
}

// GetTrustedIBANs handles GET /import/trusted
// @Summary Get trusted IBANs
// @Description Get a paginated list of trusted IBANs that are auto-matched to specific children
// @Tags Import
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param perPage query int false "Items per page" default(20)
// @Success 200 {object} IBANListResponse "Trusted IBANs"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /import/trusted [get]
func (h *ImportHandler) GetTrustedIBANs(w http.ResponseWriter, r *http.Request) {
	pagination := request.GetPagination(r)

	ibans, total, err := h.importService.GetTrustedIBANs(r.Context(), pagination.Offset, pagination.PerPage)
	if err != nil {
		response.InternalError(w, "failed to get trusted IBANs")
		return
	}

	response.Paginated(w, ibans, total, pagination.Page, pagination.PerPage)
}

// LinkIBANRequest represents a request to link an IBAN to a child.
// @Description Request body for linking an IBAN to a child
type LinkIBANRequest struct {
	ChildID string `json:"childId" example:"550e8400-e29b-41d4-a716-446655440000"`
} //@name LinkIBANRequest

// LinkIBANToChild handles POST /import/trusted/{iban}/link
// @Summary Link IBAN to child
// @Description Link a trusted IBAN to a specific child for automatic matching
// @Tags Import
// @Accept json
// @Security BearerAuth
// @Param iban path string true "IBAN (URL-encoded)"
// @Param link body LinkIBANRequest true "Link data"
// @Success 204 "IBAN linked to child"
// @Failure 400 {object} response.ErrorBody "Invalid IBAN, child ID, or IBAN is not trusted"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "IBAN or child not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /import/trusted/{iban}/link [post]
func (h *ImportHandler) LinkIBANToChild(w http.ResponseWriter, r *http.Request) {
	iban, err := url.PathUnescape(chi.URLParam(r, "iban"))
	if err != nil || iban == "" {
		response.BadRequest(w, "invalid IBAN")
		return
	}

	var req LinkIBANRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	childID, err := uuid.Parse(req.ChildID)
	if err != nil {
		response.BadRequest(w, "invalid child ID")
		return
	}

	err = h.importService.LinkIBANToChild(r.Context(), iban, childID)
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "IBAN or child not found")
			return
		}
		if err == service.ErrInvalidInput {
			response.BadRequest(w, "IBAN is not trusted")
			return
		}
		response.InternalError(w, "failed to link IBAN to child")
		return
	}

	response.NoContent(w)
}

// UnlinkIBANFromChild handles DELETE /import/trusted/{iban}/link
// @Summary Unlink IBAN from child
// @Description Remove the link between a trusted IBAN and a child
// @Tags Import
// @Security BearerAuth
// @Param iban path string true "IBAN (URL-encoded)"
// @Success 204 "IBAN unlinked from child"
// @Failure 400 {object} response.ErrorBody "Invalid IBAN"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "IBAN not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /import/trusted/{iban}/link [delete]
func (h *ImportHandler) UnlinkIBANFromChild(w http.ResponseWriter, r *http.Request) {
	iban, err := url.PathUnescape(chi.URLParam(r, "iban"))
	if err != nil || iban == "" {
		response.BadRequest(w, "invalid IBAN")
		return
	}

	err = h.importService.UnlinkIBANFromChild(r.Context(), iban)
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "IBAN not found")
			return
		}
		response.InternalError(w, "failed to unlink IBAN from child")
		return
	}

	response.NoContent(w)
}
