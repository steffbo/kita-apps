package handler

import (
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/knirpsenstadt/kita-apps/backend-management/internal/api/middleware"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/api/request"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/service"
)

// TimeTrackingHandler handles time tracking requests.
type TimeTrackingHandler struct {
	service *service.TimeTrackingService
}

// NewTimeTrackingHandler creates a new TimeTrackingHandler.
func NewTimeTrackingHandler(service *service.TimeTrackingService) *TimeTrackingHandler {
	return &TimeTrackingHandler{service: service}
}

type clockInRequest struct {
	Notes *string `json:"notes,omitempty"`
}

type clockOutRequest struct {
	BreakMinutes *int    `json:"breakMinutes,omitempty" validate:"omitempty,gte=0"`
	Notes        *string `json:"notes,omitempty"`
}

type createTimeEntryRequest struct {
	EmployeeID   int64   `json:"employeeId" validate:"required"`
	Date         string  `json:"date" validate:"required"`
	ClockIn      string  `json:"clockIn" validate:"required"`
	ClockOut     string  `json:"clockOut" validate:"required"`
	BreakMinutes *int    `json:"breakMinutes,omitempty" validate:"omitempty,gte=0"`
	EntryType    *string `json:"entryType,omitempty" validate:"omitempty,oneof=WORK VACATION SICK SPECIAL_LEAVE TRAINING EVENT"`
	Notes        *string `json:"notes,omitempty"`
	EditReason   *string `json:"editReason,omitempty"`
}

type updateTimeEntryRequest struct {
	ClockIn      *string `json:"clockIn,omitempty"`
	ClockOut     *string `json:"clockOut,omitempty"`
	BreakMinutes *int    `json:"breakMinutes,omitempty" validate:"omitempty,gte=0"`
	EntryType    *string `json:"entryType,omitempty" validate:"omitempty,oneof=WORK VACATION SICK SPECIAL_LEAVE TRAINING EVENT"`
	Notes        *string `json:"notes,omitempty"`
	EditReason   *string `json:"editReason,omitempty"`
}

// ClockIn handles POST /time-tracking/clock-in.
func (h *TimeTrackingHandler) ClockIn(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		response.Unauthorized(w, "Authentifizierung erforderlich")
		return
	}

	var req clockInRequest
	if err := request.DecodeJSON(r, &req); err != nil && !errors.Is(err, io.EOF) {
		response.BadRequest(w, "Ungültige Anfrage")
		return
	}

	entry, err := h.service.ClockIn(r.Context(), user.UserID, req.Notes)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.Success(w, mapTimeEntryResponse(*entry))
}

// ClockOut handles POST /time-tracking/clock-out.
func (h *TimeTrackingHandler) ClockOut(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		response.Unauthorized(w, "Authentifizierung erforderlich")
		return
	}

	var req clockOutRequest
	if err := request.DecodeJSON(r, &req); err != nil && !errors.Is(err, io.EOF) {
		response.BadRequest(w, "Ungültige Anfrage")
		return
	}

	entry, err := h.service.ClockOut(r.Context(), user.UserID, req.BreakMinutes, req.Notes)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.Success(w, mapTimeEntryResponse(*entry))
}

// Current handles GET /time-tracking/current.
func (h *TimeTrackingHandler) Current(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		response.Unauthorized(w, "Authentifizierung erforderlich")
		return
	}

	entry, err := h.service.Current(r.Context(), user.UserID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if entry == nil {
		response.NoContent(w)
		return
	}

	response.Success(w, mapTimeEntryResponse(*entry))
}

// List handles GET /time-tracking/entries.
func (h *TimeTrackingHandler) List(w http.ResponseWriter, r *http.Request) {
	startDateStr := request.GetQueryString(r, "startDate", "")
	endDateStr := request.GetQueryString(r, "endDate", "")
	if startDateStr == "" || endDateStr == "" {
		response.BadRequest(w, "startDate und endDate sind erforderlich")
		return
	}

	startDate, err := service.ParseDate(startDateStr)
	if err != nil {
		response.BadRequest(w, "Ungültiges startDate")
		return
	}
	endDate, err := service.ParseDate(endDateStr)
	if err != nil {
		response.BadRequest(w, "Ungültiges endDate")
		return
	}

	user := middleware.GetUserFromContext(r)
	if user == nil {
		response.Unauthorized(w, "Authentifizierung erforderlich")
		return
	}

	employeeID := user.UserID
	if raw := request.GetQueryString(r, "employeeId", ""); raw != "" {
		parsed, err := parseID(raw)
		if err != nil {
			response.BadRequest(w, "Ungültige employeeId")
			return
		}
		employeeID = parsed
	}

	entries, err := h.service.List(r.Context(), startDate, endDate, employeeID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	result := make([]TimeEntryResponse, 0, len(entries))
	for _, entry := range entries {
		result = append(result, mapTimeEntryResponse(entry))
	}

	response.Success(w, result)
}

// Create handles POST /time-tracking/entries.
func (h *TimeTrackingHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createTimeEntryRequest
	if validationErrors, err := request.DecodeAndValidate(r, &req); err != nil {
		response.BadRequest(w, "Ungültige Anfrage")
		return
	} else if validationErrors != nil {
		response.ValidationError(w, "Validierungsfehler", validationErrors)
		return
	}

	date, err := service.ParseDate(req.Date)
	if err != nil {
		response.BadRequest(w, "Ungültiges date")
		return
	}
	clockIn, err := service.ParseDateTime(req.ClockIn)
	if err != nil {
		response.BadRequest(w, "Ungültiges clockIn")
		return
	}
	clockOut, err := service.ParseDateTime(req.ClockOut)
	if err != nil {
		response.BadRequest(w, "Ungültiges clockOut")
		return
	}

	breakMinutes := 0
	if req.BreakMinutes != nil {
		breakMinutes = *req.BreakMinutes
	}

	entryType := domain.TimeEntryTypeWork
	if req.EntryType != nil {
		entryType = parseTimeEntryType(*req.EntryType)
	}

	entry, err := h.service.Create(r.Context(), service.CreateTimeEntryInput{
		EmployeeID:   req.EmployeeID,
		Date:         date,
		ClockIn:      clockIn,
		ClockOut:     clockOut,
		BreakMinutes: breakMinutes,
		EntryType:    entryType,
		Notes:        req.Notes,
		EditReason:   req.EditReason,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.Created(w, mapTimeEntryResponse(*entry))
}

// Update handles PUT /time-tracking/entries/{id}.
func (h *TimeTrackingHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Ungültige ID")
		return
	}

	var req updateTimeEntryRequest
	if validationErrors, err := request.DecodeAndValidate(r, &req); err != nil {
		response.BadRequest(w, "Ungültige Anfrage")
		return
	} else if validationErrors != nil {
		response.ValidationError(w, "Validierungsfehler", validationErrors)
		return
	}

	var clockIn *time.Time
	if req.ClockIn != nil {
		parsed, err := service.ParseDateTime(*req.ClockIn)
		if err != nil {
			response.BadRequest(w, "Ungültiges clockIn")
			return
		}
		clockIn = &parsed
	}

	var clockOut *time.Time
	if req.ClockOut != nil {
		parsed, err := service.ParseDateTime(*req.ClockOut)
		if err != nil {
			response.BadRequest(w, "Ungültiges clockOut")
			return
		}
		clockOut = &parsed
	}

	var entryType *domain.TimeEntryType
	if req.EntryType != nil {
		parsed := parseTimeEntryType(*req.EntryType)
		entryType = &parsed
	}

	user := middleware.GetUserFromContext(r)
	if user == nil {
		response.Unauthorized(w, "Authentifizierung erforderlich")
		return
	}

	entry, err := h.service.Update(r.Context(), id, service.UpdateTimeEntryInput{
		ClockIn:      clockIn,
		ClockOut:     clockOut,
		BreakMinutes: req.BreakMinutes,
		EntryType:    entryType,
		Notes:        req.Notes,
		EditReason:   req.EditReason,
	}, &user.UserID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.Success(w, mapTimeEntryResponse(*entry))
}

// Delete handles DELETE /time-tracking/entries/{id}.
func (h *TimeTrackingHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Ungültige ID")
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		writeServiceError(w, err)
		return
	}

	response.NoContent(w)
}

// Comparison handles GET /time-tracking/comparison.
func (h *TimeTrackingHandler) Comparison(w http.ResponseWriter, r *http.Request) {
	startDateStr := request.GetQueryString(r, "startDate", "")
	endDateStr := request.GetQueryString(r, "endDate", "")
	if startDateStr == "" || endDateStr == "" {
		response.BadRequest(w, "startDate und endDate sind erforderlich")
		return
	}

	startDate, err := service.ParseDate(startDateStr)
	if err != nil {
		response.BadRequest(w, "Ungültiges startDate")
		return
	}
	endDate, err := service.ParseDate(endDateStr)
	if err != nil {
		response.BadRequest(w, "Ungültiges endDate")
		return
	}

	user := middleware.GetUserFromContext(r)
	if user == nil {
		response.Unauthorized(w, "Authentifizierung erforderlich")
		return
	}

	employeeID := user.UserID
	if raw := request.GetQueryString(r, "employeeId", ""); raw != "" {
		parsed, err := parseID(raw)
		if err != nil {
			response.BadRequest(w, "Ungültige employeeId")
			return
		}
		employeeID = parsed
	}

	comparison, err := h.service.Comparison(r.Context(), startDate, endDate, employeeID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.Success(w, mapTimeScheduleComparisonResponse(comparison))
}

func parseTimeEntryType(value string) domain.TimeEntryType {
	switch value {
	case string(domain.TimeEntryTypeWork):
		return domain.TimeEntryTypeWork
	case string(domain.TimeEntryTypeVacation):
		return domain.TimeEntryTypeVacation
	case string(domain.TimeEntryTypeSick):
		return domain.TimeEntryTypeSick
	case string(domain.TimeEntryTypeSpecialLeave):
		return domain.TimeEntryTypeSpecialLeave
	case string(domain.TimeEntryTypeTraining):
		return domain.TimeEntryTypeTraining
	default:
		return domain.TimeEntryTypeEvent
	}
}
