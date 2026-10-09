package handler

import (
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/request"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
	"net/http"
	"time"
)

// CareHoursHistoryResponse represents one care hours history period.
type CareHoursHistoryResponse struct {
	ID             string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440010"`
	ChildID        string  `json:"childId" example:"550e8400-e29b-41d4-a716-446655440000"`
	CareHours      *int    `json:"careHours,omitempty" example:"40" binding:"optional"`
	EffectiveFrom  string  `json:"effectiveFrom" example:"2026-01-01"`
	EffectiveUntil *string `json:"effectiveUntil,omitempty" example:"2026-03-31" binding:"optional"`
	CreatedAt      string  `json:"createdAt" example:"2026-01-01T10:00:00Z"`
	UpdatedAt      string  `json:"updatedAt" example:"2026-01-01T10:00:00Z"`
} //@name CareHoursHistoryEntry

// LegalHoursHistoryResponse represents one legal hours history period.
type LegalHoursHistoryResponse struct {
	ID             string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440011"`
	ChildID        string  `json:"childId" example:"550e8400-e29b-41d4-a716-446655440000"`
	LegalHours     *int    `json:"legalHours,omitempty" example:"35" binding:"optional"`
	EffectiveFrom  string  `json:"effectiveFrom" example:"2026-01-01"`
	EffectiveUntil *string `json:"effectiveUntil,omitempty" example:"2026-03-31" binding:"optional"`
	CreatedAt      string  `json:"createdAt" example:"2026-01-01T10:00:00Z"`
	UpdatedAt      string  `json:"updatedAt" example:"2026-01-01T10:00:00Z"`
} //@name LegalHoursHistoryEntry

// CreateCareHoursHistoryRequest represents a new care hours history entry.
type CreateCareHoursHistoryRequest struct {
	CareHours *int   `json:"careHours" binding:"optional"`
	ValidFrom string `json:"validFrom" example:"2026-01-01"`
} //@name CreateCareHoursHistoryRequest

// CreateLegalHoursHistoryRequest represents a new legal hours history entry.
type CreateLegalHoursHistoryRequest struct {
	LegalHours *int   `json:"legalHours" binding:"optional"`
	ValidFrom  string `json:"validFrom" example:"2026-01-01"`
} //@name CreateLegalHoursHistoryRequest

// GetCareHoursHistory returns the care hours history for a child.
// @Summary Get child care hours history
// @Description Returns all recorded care hours periods for a child
// @Tags Children
// @Produce json
// @Security BearerAuth
// @Param id path string true "Child ID (UUID)"
// @Success 200 {array} CareHoursHistoryResponse "Care hours history"
// @Failure 400 {object} response.ErrorBody "Invalid child ID"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Child not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /children/{id}/care-hours-history [get]
func (h *ChildHandler) GetCareHoursHistory(w http.ResponseWriter, r *http.Request) {
	childID, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	history, err := h.childService.ListCareHoursHistory(r.Context(), childID)
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "child not found")
			return
		}
		response.InternalError(w, "failed to load care hours history")
		return
	}

	resp := make([]CareHoursHistoryResponse, len(history))
	for i, entry := range history {
		var effectiveUntil *string
		if entry.EffectiveUntil != nil {
			formatted := entry.EffectiveUntil.Format("2006-01-02")
			effectiveUntil = &formatted
		}
		resp[i] = CareHoursHistoryResponse{
			ID:             entry.ID.String(),
			ChildID:        entry.ChildID.String(),
			CareHours:      entry.CareHours,
			EffectiveFrom:  entry.EffectiveFrom.Format("2006-01-02"),
			EffectiveUntil: effectiveUntil,
			CreatedAt:      entry.CreatedAt.Format(time.RFC3339),
			UpdatedAt:      entry.UpdatedAt.Format(time.RFC3339),
		}
	}

	response.Success(w, resp)
}

// AddCareHoursHistory creates or updates a care hours history entry.
// @Summary Add child care hours history entry
// @Description Adds a care hours history entry effective from the given date
// @Tags Children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Child ID (UUID)"
// @Param request body CreateCareHoursHistoryRequest true "Care hours history data"
// @Success 204 "Care hours history saved"
// @Failure 400 {object} response.ErrorBody "Invalid request"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Child not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /children/{id}/care-hours-history [post]
func (h *ChildHandler) AddCareHoursHistory(w http.ResponseWriter, r *http.Request) {
	childID, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req CreateCareHoursHistoryRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if req.ValidFrom == "" {
		response.BadRequest(w, "validFrom is required")
		return
	}

	err := h.childService.AddCareHoursHistory(r.Context(), childID, service.AddCareHoursHistoryInput{
		CareHours: req.CareHours,
		ValidFrom: req.ValidFrom,
	})
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "child not found")
			return
		}
		if err == service.ErrInvalidInput {
			response.BadRequest(w, "invalid care hours history")
			return
		}
		response.InternalError(w, "failed to save care hours history")
		return
	}

	response.NoContent(w)
}

// GetLegalHoursHistory returns the legal hours history for a child.
// @Summary Get child legal hours history
// @Description Returns all recorded legal entitlement periods for a child
// @Tags Children
// @Produce json
// @Security BearerAuth
// @Param id path string true "Child ID (UUID)"
// @Success 200 {array} LegalHoursHistoryResponse "Legal hours history"
// @Failure 400 {object} response.ErrorBody "Invalid child ID"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Child not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /children/{id}/legal-hours-history [get]
func (h *ChildHandler) GetLegalHoursHistory(w http.ResponseWriter, r *http.Request) {
	childID, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	history, err := h.childService.ListLegalHoursHistory(r.Context(), childID)
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "child not found")
			return
		}
		response.InternalError(w, "failed to load legal hours history")
		return
	}

	resp := make([]LegalHoursHistoryResponse, len(history))
	for i, entry := range history {
		var effectiveUntil *string
		if entry.EffectiveUntil != nil {
			formatted := entry.EffectiveUntil.Format("2006-01-02")
			effectiveUntil = &formatted
		}
		resp[i] = LegalHoursHistoryResponse{
			ID:             entry.ID.String(),
			ChildID:        entry.ChildID.String(),
			LegalHours:     entry.LegalHours,
			EffectiveFrom:  entry.EffectiveFrom.Format("2006-01-02"),
			EffectiveUntil: effectiveUntil,
			CreatedAt:      entry.CreatedAt.Format(time.RFC3339),
			UpdatedAt:      entry.UpdatedAt.Format(time.RFC3339),
		}
	}

	response.Success(w, resp)
}

// AddLegalHoursHistory creates or updates a legal hours history entry.
// @Summary Add child legal hours history entry
// @Description Adds a legal entitlement history entry effective from the given date
// @Tags Children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Child ID (UUID)"
// @Param request body CreateLegalHoursHistoryRequest true "Legal hours history data"
// @Success 204 "Legal hours history saved"
// @Failure 400 {object} response.ErrorBody "Invalid request"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Child not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /children/{id}/legal-hours-history [post]
func (h *ChildHandler) AddLegalHoursHistory(w http.ResponseWriter, r *http.Request) {
	childID, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req CreateLegalHoursHistoryRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if req.ValidFrom == "" {
		response.BadRequest(w, "validFrom is required")
		return
	}

	err := h.childService.AddLegalHoursHistory(r.Context(), childID, service.AddLegalHoursHistoryInput{
		LegalHours: req.LegalHours,
		ValidFrom:  req.ValidFrom,
	})
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "child not found")
			return
		}
		if err == service.ErrInvalidInput {
			response.BadRequest(w, "invalid legal hours history")
			return
		}
		response.InternalError(w, "failed to save legal hours history")
		return
	}

	response.NoContent(w)
}
