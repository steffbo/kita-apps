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

// clockInRequest contains optional notes for clocking in.
type clockInRequest struct {
	Notes *string `json:"notes,omitempty" example:"Frühdienst übernommen"`
} //@name ClockInRequest

// clockOutRequest contains the data for clocking out.
type clockOutRequest struct {
	BreakMinutes *int    `json:"breakMinutes,omitempty" validate:"omitempty,gte=0" example:"30"`
	Notes        *string `json:"notes,omitempty" example:"Überstunden wegen Krankheitsvertretung"`
} //@name ClockOutRequest

// createTimeEntryRequest contains the data for creating a time entry.
type createTimeEntryRequest struct {
	EmployeeID   int64   `json:"employeeId" validate:"required" example:"1"`
	Date         string  `json:"date" validate:"required" example:"2024-03-15"`
	ClockIn      string  `json:"clockIn" validate:"required" example:"2024-03-15T08:00:00Z"`
	ClockOut     string  `json:"clockOut" validate:"required" example:"2024-03-15T16:30:00Z"`
	BreakMinutes *int    `json:"breakMinutes,omitempty" validate:"omitempty,gte=0" example:"30"`
	EntryType    *string `json:"entryType,omitempty" validate:"omitempty,oneof=WORK VACATION SICK SPECIAL_LEAVE TRAINING EVENT" example:"WORK"`
	Notes        *string `json:"notes,omitempty" example:"Normale Arbeitszeit"`
	EditReason   *string `json:"editReason,omitempty" example:"Nachträgliche Korrektur"`
} //@name CreateTimeEntryRequest

// updateTimeEntryRequest contains the data for updating a time entry.
type updateTimeEntryRequest struct {
	ClockIn      *string `json:"clockIn,omitempty" example:"2024-03-15T08:00:00Z"`
	ClockOut     *string `json:"clockOut,omitempty" example:"2024-03-15T16:30:00Z"`
	BreakMinutes *int    `json:"breakMinutes,omitempty" validate:"omitempty,gte=0" example:"30"`
	EntryType    *string `json:"entryType,omitempty" validate:"omitempty,oneof=WORK VACATION SICK SPECIAL_LEAVE TRAINING EVENT" example:"WORK"`
	Notes        *string `json:"notes,omitempty" example:"Korrigierte Arbeitszeit"`
	EditReason   *string `json:"editReason,omitempty" example:"Fehlerhafte Stempelung korrigiert"`
} //@name UpdateTimeEntryRequest

// ClockIn handles POST /time-tracking/clock-in.
// @Summary Clock in
// @Description Record the current user's clock-in time
// @Tags Time Tracking
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param clockIn body clockInRequest false "Optional notes"
// @Success 200 {object} TimeEntryResponse "Time entry with clock-in recorded"
// @Failure 400 {object} map[string]interface{} "Invalid request or already clocked in"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Router /time-tracking/clock-in [post]
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
// @Summary Clock out
// @Description Record the current user's clock-out time
// @Tags Time Tracking
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param clockOut body clockOutRequest false "Break minutes and optional notes"
// @Success 200 {object} TimeEntryResponse "Completed time entry"
// @Failure 400 {object} map[string]interface{} "Invalid request or not clocked in"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Router /time-tracking/clock-out [post]
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
// @Summary Get current time entry
// @Description Get the current user's active (not clocked out) time entry if any
// @Tags Time Tracking
// @Produce json
// @Security BearerAuth
// @Success 200 {object} TimeEntryResponse "Current active time entry"
// @Success 204 "No active time entry"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Router /time-tracking/current [get]
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
// @Summary List time entries
// @Description Get time entries for a date range
// @Tags Time Tracking
// @Produce json
// @Security BearerAuth
// @Param startDate query string true "Start date (YYYY-MM-DD)"
// @Param endDate query string true "End date (YYYY-MM-DD)"
// @Param employeeId query int false "Filter by employee ID (defaults to current user)"
// @Success 200 {array} TimeEntryResponse "List of time entries"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Router /time-tracking/entries [get]
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
// @Summary Create a time entry
// @Description Manually create a time entry (admin function)
// @Tags Time Tracking
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param entry body createTimeEntryRequest true "Time entry data"
// @Success 201 {object} TimeEntryResponse "Created time entry"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Router /time-tracking/entries [post]
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
// @Summary Update a time entry
// @Description Update an existing time entry
// @Tags Time Tracking
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Time entry ID"
// @Param entry body updateTimeEntryRequest true "Updated time entry data"
// @Success 200 {object} TimeEntryResponse "Updated time entry"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Failure 404 {object} map[string]interface{} "Time entry not found"
// @Router /time-tracking/entries/{id} [put]
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
// @Summary Delete a time entry
// @Description Delete a time entry
// @Tags Time Tracking
// @Security BearerAuth
// @Param id path int true "Time entry ID"
// @Success 204 "Time entry deleted"
// @Failure 400 {object} map[string]interface{} "Invalid time entry ID"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Failure 404 {object} map[string]interface{} "Time entry not found"
// @Router /time-tracking/entries/{id} [delete]
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
// @Summary Compare time entries with schedule
// @Description Compare actual time entries with scheduled shifts for a date range
// @Tags Time Tracking
// @Produce json
// @Security BearerAuth
// @Param startDate query string true "Start date (YYYY-MM-DD)"
// @Param endDate query string true "End date (YYYY-MM-DD)"
// @Param employeeId query int false "Filter by employee ID (defaults to current user)"
// @Success 200 {object} TimeScheduleComparisonResponse "Time vs schedule comparison"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Router /time-tracking/comparison [get]
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
