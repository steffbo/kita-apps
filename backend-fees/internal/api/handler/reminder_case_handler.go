package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/middleware"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/request"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

// ReminderCaseConflictResponse is the 409 body for stale preview states.
// @Description Selected fees are no longer open, are foreign to the household, or received reminder fees after the preview
type ReminderCaseConflictResponse struct {
	Message string   `json:"message" example:"selected fees are no longer open"`
	FeeIDs  []string `json:"feeIds,omitempty"`
}

// ReminderCaseRequestDTO is the shared request body for preview and send.
// @Description Stage selection, fee IDs and content overrides; the deadline is computed server-side as runDate + 7 days
type ReminderCaseRequestDTO struct {
	Stage       string   `json:"stage" example:"initial" enums:"initial,final"`
	RunDate     string   `json:"runDate,omitempty" example:"2026-09-15"`
	FeeIDs      []string `json:"feeIds"`
	IncludeQR   *bool    `json:"includeQR,omitempty" example:"true"`
	Subject     string   `json:"subject,omitempty"`
	Body        string   `json:"body,omitempty"`
	PreviewedAt string   `json:"previewedAt,omitempty" example:"2026-09-15T10:00:00Z"`
} //@name ReminderCaseRequest

// GetReminderCases handles GET /fees/reminder-cases
// @Summary List family reminder cases
// @Description Returns open fees grouped by household with workflow status; scope filters to actionable families
// @Tags Fees
// @Produce json
// @Security BearerAuth
// @Param asOf query string false "Reference date (YYYY-MM-DD, defaults to today)"
// @Param scope query string false "Scope" Enums(actionable, all) default(actionable)
// @Success 200 {object} service.ReminderCasesResult "Family reminder cases"
// @Failure 400 {object} response.ErrorBody "Invalid request"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /fees/reminder-cases [get]
func (h *FeeHandler) GetReminderCases(w http.ResponseWriter, r *http.Request) {
	asOf := time.Now()
	if asOfStr := request.GetQueryString(r, "asOf", ""); asOfStr != "" {
		parsed, err := time.Parse("2006-01-02", asOfStr)
		if err != nil {
			response.BadRequest(w, "invalid asOf format (expected YYYY-MM-DD)")
			return
		}
		asOf = parsed
	}

	scope := strings.ToLower(strings.TrimSpace(request.GetQueryString(r, "scope", service.ReminderCasesScopeActionable)))
	if scope == "" {
		scope = service.ReminderCasesScopeActionable
	}

	result, err := h.reminderService.ListReminderCases(r.Context(), asOf, scope)
	if err != nil {
		if err == service.ErrInvalidInput {
			response.BadRequest(w, "invalid scope (expected actionable, all)")
			return
		}
		response.InternalError(w, "failed to list reminder cases")
		return
	}

	response.Success(w, result)
}

// PreviewReminderCase handles POST /fees/reminder-cases/{householdId}/preview
// @Summary Preview a family reminder email
// @Description Builds the final mail content, QR data, planned reminder fees, recommendation and warnings without side effects
// @Tags Fees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param householdId path string true "Household UUID"
// @Param request body ReminderCaseRequestDTO true "Preview parameters"
// @Success 200 {object} service.ReminderCasePreview "Email preview"
// @Failure 400 {object} response.ErrorBody "Invalid request"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 409 {object} ReminderCaseConflictResponse "Stale preview state"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /fees/reminder-cases/{householdId}/preview [post]
func (h *FeeHandler) PreviewReminderCase(w http.ResponseWriter, r *http.Request) {
	if !h.ensureReminderService(w) {
		return
	}
	householdID, ok := parseHouseholdIDPath(w, r)
	if !ok {
		return
	}

	req, ok := decodeReminderCaseRequest(w, r)
	if !ok {
		return
	}

	preview, err := h.reminderService.PreviewReminderCase(r.Context(), householdID, req)
	if err != nil {
		writeReminderCaseError(w, err)
		return
	}

	response.Success(w, preview)
}

// SendReminderCase handles POST /fees/reminder-cases/{householdId}/send
// @Summary Send a family reminder email
// @Description Re-validates the preview state, creates planned reminder fees, sends the email and writes the log; 409 when state changed
// @Tags Fees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param householdId path string true "Household UUID"
// @Param request body ReminderCaseRequestDTO true "Send parameters"
// @Success 200 {object} service.ReminderCaseSendResult "Send result"
// @Failure 400 {object} response.ErrorBody "Invalid request"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 409 {object} ReminderCaseConflictResponse "Stale preview state"
// @Failure 503 {object} response.ErrorBody "Email service disabled"
// @Router /fees/reminder-cases/{householdId}/send [post]
func (h *FeeHandler) SendReminderCase(w http.ResponseWriter, r *http.Request) {
	if !h.ensureReminderService(w) {
		return
	}
	userCtx := middleware.GetUserFromContext(r)
	if userCtx == nil {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}
	householdID, ok := parseHouseholdIDPath(w, r)
	if !ok {
		return
	}

	req, ok := decodeReminderCaseRequest(w, r)
	if !ok {
		return
	}

	var sentBy *uuid.UUID
	if userCtx.UserID != "" {
		if parsed, err := uuid.Parse(userCtx.UserID); err == nil {
			sentBy = &parsed
		}
	}

	result, err := h.reminderService.SendReminderCase(r.Context(), householdID, req, sentBy)
	if err != nil {
		writeReminderCaseError(w, err)
		return
	}

	response.Success(w, result)
}

func (h *FeeHandler) ensureReminderService(w http.ResponseWriter) bool {
	if h.reminderService == nil {
		response.InternalError(w, "reminder service not configured")
		return false
	}
	return true
}

func parseHouseholdIDPath(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	householdID, err := uuid.Parse(chi.URLParam(r, "householdId"))
	if err != nil {
		response.BadRequest(w, "invalid householdId format")
		return uuid.Nil, false
	}
	return householdID, true
}

func decodeReminderCaseRequest(w http.ResponseWriter, r *http.Request) (*service.ReminderCaseRequest, bool) {
	var dto ReminderCaseRequestDTO
	if err := request.DecodeJSON(r, &dto); err != nil {
		response.BadRequest(w, "invalid request body")
		return nil, false
	}

	stage, err := service.ParseReminderStage(dto.Stage)
	if err != nil || (stage != service.ReminderStageInitial && stage != service.ReminderStageFinal) {
		response.BadRequest(w, "invalid stage (expected initial, final)")
		return nil, false
	}

	req := &service.ReminderCaseRequest{Stage: stage}

	if strings.TrimSpace(dto.RunDate) != "" {
		parsed, err := time.Parse("2006-01-02", dto.RunDate)
		if err != nil {
			response.BadRequest(w, "invalid runDate format (expected YYYY-MM-DD)")
			return nil, false
		}
		req.RunDate = parsed
	}

	if len(dto.FeeIDs) == 0 {
		response.BadRequest(w, "feeIds must not be empty")
		return nil, false
	}
	feeIDs := make([]uuid.UUID, 0, len(dto.FeeIDs))
	for _, raw := range dto.FeeIDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			response.BadRequest(w, "invalid feeId format")
			return nil, false
		}
		feeIDs = append(feeIDs, id)
	}
	req.FeeIDs = feeIDs
	req.IncludeQR = dto.IncludeQR
	req.Subject = dto.Subject
	req.Body = dto.Body

	if strings.TrimSpace(dto.PreviewedAt) != "" {
		parsed, err := time.Parse(time.RFC3339, dto.PreviewedAt)
		if err != nil {
			response.BadRequest(w, "invalid previewedAt format (expected RFC3339)")
			return nil, false
		}
		req.PreviewedAt = &parsed
	}

	return req, true
}

func writeReminderCaseError(w http.ResponseWriter, err error) {
	var conflict *service.CaseConflictError
	if errors.As(err, &conflict) {
		feeIDs := make([]string, 0, len(conflict.FeeIDs))
		for _, id := range conflict.FeeIDs {
			feeIDs = append(feeIDs, id.String())
		}
		response.JSON(w, http.StatusConflict, ReminderCaseConflictResponse{
			Message: conflict.Reason,
			FeeIDs:  feeIDs,
		})
		return
	}
	switch err {
	case service.ErrInvalidInput:
		response.BadRequest(w, "invalid request")
	case service.ErrEmailDisabled:
		response.Error(w, http.StatusServiceUnavailable, "email service disabled")
	case repository.ErrNotFound:
		response.NotFound(w, "household not found")
	default:
		response.InternalError(w, "failed to process reminder case")
	}
}
