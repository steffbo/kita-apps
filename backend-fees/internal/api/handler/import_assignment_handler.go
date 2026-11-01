package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/middleware"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/request"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

// AssignTransactionRequest gives a transaction a purpose instead of a fee.
// @Description Request body for assigning a transaction to a purpose
type AssignTransactionRequest struct {
	Kind domain.AssignmentKind `json:"kind" enums:"DONATION" example:"DONATION"`
	Note string                `json:"note" example:"Spende Sommerfest"`
	// HouseholdID credits the donation to a household (optional).
	HouseholdID *uuid.UUID `json:"householdId,omitempty" swaggertype:"string" binding:"optional"`
} //@name AssignTransactionRequest

// AssignTransaction handles POST /import/transactions/{id}/assign
// @Summary Assign a transaction to a purpose
// @Description Marks an unassigned incoming transaction as donation. Undo with the unmatch action.
// @Tags Import
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Transaction ID (UUID)"
// @Param request body AssignTransactionRequest true "Purpose"
// @Success 204 "Assigned"
// @Failure 400 {object} response.ErrorBody "Invalid kind, outgoing transaction or already assigned"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Transaction or household not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /import/transactions/{id}/assign [post]
func (h *ImportHandler) AssignTransaction(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid transaction ID")
		return
	}
	var req AssignTransactionRequest
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

	switch err := h.importService.AssignTransaction(r.Context(), id, domain.AssignmentInput{Kind: req.Kind, Note: req.Note, HouseholdID: req.HouseholdID}, userID); {
	case err == nil:
		response.NoContent(w)
	case errors.Is(err, service.ErrNotFound):
		response.NotFound(w, "transaction or household not found")
	case errors.Is(err, service.ErrInvalidInput):
		response.BadRequest(w, "Zahlung ist nicht zuordenbar: nur offene Eingänge, ohne Beitrag oder Ratenzahlung")
	default:
		response.InternalError(w, "failed to assign transaction")
	}
}

// UpdateAssignment handles PUT /import/transactions/{id}/assign
// @Summary Change note and household of an assignment
// @Description Replaces note and household of an existing donation assignment
// @Tags Import
// @Accept json
// @Security BearerAuth
// @Param id path string true "Transaction ID (UUID)"
// @Param request body AssignTransactionRequest true "Purpose"
// @Success 204 "Updated"
// @Failure 400 {object} response.ErrorBody "Invalid request"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Assignment or household not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /import/transactions/{id}/assign [put]
func (h *ImportHandler) UpdateAssignment(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid transaction ID")
		return
	}
	var req AssignTransactionRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	switch err := h.importService.UpdateAssignment(r.Context(), id, domain.AssignmentInput{Kind: req.Kind, Note: req.Note, HouseholdID: req.HouseholdID}); {
	case err == nil:
		response.NoContent(w)
	case errors.Is(err, service.ErrNotFound):
		response.NotFound(w, "assignment or household not found")
	case errors.Is(err, service.ErrInvalidInput):
		response.BadRequest(w, "invalid assignment")
	default:
		response.InternalError(w, "failed to update assignment")
	}
}

// Donations handles GET /import/donations
// @Summary List donations grouped by year
// @Description Transactions assigned as donation, grouped by booking year with sums, newest first
// @Tags Import
// @Produce json
// @Security BearerAuth
// @Param householdId query string false "Only donations credited to this household (UUID)"
// @Success 200 {object} service.DonationReport "Donations"
// @Failure 400 {object} response.ErrorBody "Invalid household ID"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /import/donations [get]
func (h *ImportHandler) Donations(w http.ResponseWriter, r *http.Request) {
	var householdID *uuid.UUID
	if raw := r.URL.Query().Get("householdId"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			response.BadRequest(w, "invalid household ID")
			return
		}
		householdID = &id
	}
	report, err := h.importService.GetDonations(r.Context(), householdID)
	if err != nil {
		response.InternalError(w, "failed to get donations")
		return
	}
	response.Success(w, report)
}
