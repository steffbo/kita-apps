package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/knirpsenstadt/kita-apps/backend-management/internal/api/middleware"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/api/request"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/service"
)

// ScheduleHandler handles schedule requests.
type ScheduleHandler struct {
	schedules *service.ScheduleService
}

// NewScheduleHandler creates a new ScheduleHandler.
func NewScheduleHandler(schedules *service.ScheduleService) *ScheduleHandler {
	return &ScheduleHandler{schedules: schedules}
}

// createScheduleEntryRequest contains the data for creating a schedule entry.
type createScheduleEntryRequest struct {
	EmployeeID         int64                         `json:"employeeId" validate:"required" example:"1"`
	Date               string                        `json:"date" validate:"required" example:"2024-03-15"`
	StartTime          *string                       `json:"startTime,omitempty" example:"08:00:00"`
	EndTime            *string                       `json:"endTime,omitempty" example:"16:00:00"`
	BreakMinutes       *int                          `json:"breakMinutes,omitempty" validate:"omitempty,gte=0" example:"30"`
	GroupID            *int64                        `json:"groupId,omitempty" example:"1"`
	EntryType          *string                       `json:"entryType,omitempty" validate:"omitempty,oneof=WORK VACATION SICK CHILD_SICK RECOVERY_DAY SPECIAL_LEAVE TRAINING EVENT" example:"WORK"`
	ShiftKind          *string                       `json:"shiftKind,omitempty" validate:"omitempty,oneof=EARLY LATE MANUAL" example:"EARLY"`
	Notes              *string                       `json:"notes,omitempty" example:"Frühdienst"`
	Segments           []scheduleEntrySegmentRequest `json:"segments,omitempty"`
	OverrideBlockedDay *bool                         `json:"overrideBlockedDay,omitempty" example:"false"`
} //@name CreateScheduleEntryRequest

// updateScheduleEntryRequest contains the data for updating a schedule entry.
type updateScheduleEntryRequest struct {
	Date               *string                        `json:"date,omitempty" example:"2024-03-15"`
	StartTime          *string                        `json:"startTime,omitempty" example:"08:00:00"`
	EndTime            *string                        `json:"endTime,omitempty" example:"16:00:00"`
	BreakMinutes       *int                           `json:"breakMinutes,omitempty" validate:"omitempty,gte=0" example:"30"`
	GroupID            *int64                         `json:"groupId,omitempty" example:"1"`
	EntryType          *string                        `json:"entryType,omitempty" validate:"omitempty,oneof=WORK VACATION SICK CHILD_SICK RECOVERY_DAY SPECIAL_LEAVE TRAINING EVENT" example:"WORK"`
	ShiftKind          *string                        `json:"shiftKind,omitempty" validate:"omitempty,oneof=EARLY LATE MANUAL" example:"EARLY"`
	Notes              *string                        `json:"notes,omitempty" example:"Frühdienst"`
	Segments           *[]scheduleEntrySegmentRequest `json:"segments,omitempty"`
	OverrideBlockedDay *bool                          `json:"overrideBlockedDay,omitempty" example:"false"`
} //@name UpdateScheduleEntryRequest

type scheduleEntrySegmentRequest struct {
	GroupID   int64   `json:"groupId" validate:"required" example:"1"`
	StartTime string  `json:"startTime" validate:"required" example:"08:00:00"`
	EndTime   string  `json:"endTime" validate:"required" example:"12:00:00"`
	Notes     *string `json:"notes,omitempty" example:"bis 12 Igel"`
	SortOrder int     `json:"sortOrder,omitempty" example:"1"`
} //@name ScheduleEntrySegmentRequest

type createScheduleRequestRequest struct {
	EmployeeID  *int64  `json:"employeeId,omitempty" example:"1"`
	Date        string  `json:"date" validate:"required" example:"2026-04-29"`
	StartTime   *string `json:"startTime,omitempty" example:"10:00:00"`
	EndTime     *string `json:"endTime,omitempty" example:"11:00:00"`
	RequestType *string `json:"requestType,omitempty" validate:"omitempty,oneof=WISH APPOINTMENT" example:"APPOINTMENT"`
	Text        string  `json:"text" validate:"required" example:"Arzttermin"`
} //@name CreateScheduleRequestRequest

type updateScheduleRequestRequest struct {
	Date        *string `json:"date,omitempty" example:"2026-04-29"`
	StartTime   *string `json:"startTime,omitempty" example:"10:00:00"`
	EndTime     *string `json:"endTime,omitempty" example:"11:00:00"`
	RequestType *string `json:"requestType,omitempty" validate:"omitempty,oneof=WISH APPOINTMENT" example:"APPOINTMENT"`
	Text        *string `json:"text,omitempty" example:"Arzttermin"`
	Status      *string `json:"status,omitempty" validate:"omitempty,oneof=OPEN DONE" example:"DONE"`
} //@name UpdateScheduleRequestRequest

type timeSuggestionRequest struct {
	EmployeeID int64   `json:"employeeId" validate:"required" example:"1"`
	Date       string  `json:"date" validate:"required" example:"2026-05-04"`
	ShiftKind  string  `json:"shiftKind" validate:"required,oneof=EARLY LATE MANUAL" example:"EARLY"`
	StartTime  *string `json:"startTime,omitempty" example:"07:00:00"`
} //@name TimeSuggestionRequest

type TimeSuggestionResponse struct {
	StartTime      *string `json:"startTime,omitempty"`
	EndTime        *string `json:"endTime,omitempty"`
	BreakMinutes   int     `json:"breakMinutes"`
	PlannedMinutes int     `json:"plannedMinutes"`
	IsBlocked      bool    `json:"isBlocked"`
	ContractID     *int64  `json:"contractId,omitempty"`
} //@name TimeSuggestion

// List handles GET /schedule.
// @Summary List schedule entries
// @Description Get schedule entries for a date range, optionally filtered by employee or group
// @Tags Schedule
// @Produce json
// @Security BearerAuth
// @Param startDate query string true "Start date (YYYY-MM-DD)"
// @Param endDate query string true "End date (YYYY-MM-DD)"
// @Param employeeId query int false "Filter by employee ID"
// @Param groupId query int false "Filter by group ID"
// @Success 200 {array} ScheduleEntryResponse "List of schedule entries"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Router /schedule [get]
func (h *ScheduleHandler) List(w http.ResponseWriter, r *http.Request) {
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

	var employeeID *int64
	if raw := request.GetQueryString(r, "employeeId", ""); raw != "" {
		parsed, err := parseID(raw)
		if err != nil {
			response.BadRequest(w, "Ungültige employeeId")
			return
		}
		employeeID = &parsed
	}
	var groupID *int64
	if raw := request.GetQueryString(r, "groupId", ""); raw != "" {
		parsed, err := parseID(raw)
		if err != nil {
			response.BadRequest(w, "Ungültige groupId")
			return
		}
		groupID = &parsed
	}

	entries, err := h.schedules.List(r.Context(), startDate, endDate, employeeID, groupID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	result := make([]ScheduleEntryResponse, 0, len(entries))
	for _, entry := range entries {
		result = append(result, mapScheduleEntryResponse(entry))
	}

	response.Success(w, result)
}

// Week handles GET /schedule/week.
// @Summary Get week schedule
// @Description Get the complete schedule for a week with all employees and groups
// @Tags Schedule
// @Produce json
// @Security BearerAuth
// @Param weekStart query string true "Week start date (YYYY-MM-DD, should be Monday)"
// @Success 200 {object} WeekScheduleResponse "Week schedule"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Router /schedule/week [get]
func (h *ScheduleHandler) Week(w http.ResponseWriter, r *http.Request) {
	weekStartStr := request.GetQueryString(r, "weekStart", "")
	if weekStartStr == "" {
		response.BadRequest(w, "weekStart ist erforderlich")
		return
	}

	weekStart, err := service.ParseDate(weekStartStr)
	if err != nil {
		response.BadRequest(w, "Ungültiges weekStart")
		return
	}

	weekSchedule, err := h.schedules.Week(r.Context(), weekStart)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	responseDays := make([]DayScheduleResponse, 0, len(weekSchedule.Days))
	for _, day := range weekSchedule.Days {
		entries := make([]ScheduleEntryResponse, 0, len(day.Entries))
		for _, entry := range day.Entries {
			entries = append(entries, mapScheduleEntryResponse(entry))
		}

		byGroup := make(map[string][]ScheduleEntryResponse)
		for groupID, groupEntries := range day.ByGroup {
			mapped := make([]ScheduleEntryResponse, 0, len(groupEntries))
			for _, entry := range groupEntries {
				mapped = append(mapped, mapScheduleEntryResponse(entry))
			}
			byGroup[groupID] = mapped
		}

		responseDays = append(responseDays, DayScheduleResponse{
			Date:        day.Date.Format(dateLayout),
			DayOfWeek:   strings.ToUpper(day.DayOfWeek.String()),
			IsHoliday:   day.IsHoliday,
			HolidayName: day.HolidayName,
			Entries:     entries,
			ByGroup:     byGroup,
		})
	}

	specialDays := make([]SpecialDayResponse, 0, len(weekSchedule.SpecialDays))
	for _, day := range weekSchedule.SpecialDays {
		specialDays = append(specialDays, mapSpecialDayResponse(day))
	}

	response.Success(w, WeekScheduleResponse{
		WeekStart:   weekSchedule.WeekStart.Format(dateLayout),
		WeekEnd:     weekSchedule.WeekEnd.Format(dateLayout),
		Days:        responseDays,
		SpecialDays: specialDays,
		Requests:    mapScheduleRequestResponses(weekSchedule.Requests),
	})
}

// TimeSuggestion handles POST /schedule/time-suggestion.
// @Summary Calculate schedule time suggestion
// @Description Calculate suggested start and end times for a schedule entry
// @Tags Schedule
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body timeSuggestionRequest true "Time suggestion request"
// @Success 200 {object} TimeSuggestionResponse "Time suggestion"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Router /schedule/time-suggestion [post]
func (h *ScheduleHandler) TimeSuggestion(w http.ResponseWriter, r *http.Request) {
	var req timeSuggestionRequest
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
	var startTime *time.Time
	if req.StartTime != nil {
		parsed, err := service.ParseTime(*req.StartTime)
		if err != nil {
			response.BadRequest(w, "Ungültiges startTime")
			return
		}
		startTime = &parsed
	}

	suggestion, err := h.schedules.SuggestTimes(r.Context(), service.TimeSuggestionInput{
		EmployeeID: req.EmployeeID,
		Date:       date,
		ShiftKind:  parseShiftKind(req.ShiftKind),
		StartTime:  startTime,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.Success(w, mapTimeSuggestionResponse(suggestion))
}

// Create handles POST /schedule.
// @Summary Create a schedule entry
// @Description Create a new schedule entry for an employee
// @Tags Schedule
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param entry body createScheduleEntryRequest true "Schedule entry data"
// @Success 201 {object} ScheduleEntryResponse "Created schedule entry"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Router /schedule [post]
func (h *ScheduleHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createScheduleEntryRequest
	if validationErrors, err := request.DecodeAndValidate(r, &req); err != nil {
		response.BadRequest(w, "Ungültige Anfrage")
		return
	} else if validationErrors != nil {
		response.ValidationError(w, "Validierungsfehler", validationErrors)
		return
	}

	input, errMsg := parseScheduleEntryInput(req)
	if errMsg != "" {
		response.BadRequest(w, errMsg)
		return
	}

	entry, err := h.schedules.Create(r.Context(), *input)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.Created(w, mapScheduleEntryResponse(*entry))
}

// BulkCreate handles POST /schedule/bulk.
// @Summary Create multiple schedule entries
// @Description Create multiple schedule entries at once
// @Tags Schedule
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param entries body []createScheduleEntryRequest true "Schedule entries data"
// @Success 201 {array} ScheduleEntryResponse "Created schedule entries"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Router /schedule/bulk [post]
func (h *ScheduleHandler) BulkCreate(w http.ResponseWriter, r *http.Request) {
	var req []createScheduleEntryRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "Ungültige Anfrage")
		return
	}

	inputs := make([]service.CreateScheduleEntryInput, 0, len(req))
	for i, item := range req {
		// Validate each item
		if validationErrors := request.Validate(&item); validationErrors != nil {
			response.ValidationError(w, "Validierungsfehler bei Eintrag "+strconv.Itoa(i+1), validationErrors)
			return
		}

		input, errMsg := parseScheduleEntryInput(item)
		if errMsg != "" {
			response.BadRequest(w, errMsg+" bei Eintrag "+strconv.Itoa(i+1))
			return
		}
		inputs = append(inputs, *input)
	}

	entries, err := h.schedules.BulkCreate(r.Context(), inputs)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	result := make([]ScheduleEntryResponse, 0, len(entries))
	for _, entry := range entries {
		result = append(result, mapScheduleEntryResponse(entry))
	}

	response.Created(w, result)
}

// Update handles PUT /schedule/{id}.
// @Summary Update a schedule entry
// @Description Update an existing schedule entry
// @Tags Schedule
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Schedule entry ID"
// @Param entry body updateScheduleEntryRequest true "Updated schedule entry data"
// @Success 200 {object} ScheduleEntryResponse "Updated schedule entry"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Failure 404 {object} map[string]interface{} "Schedule entry not found"
// @Router /schedule/{id} [put]
func (h *ScheduleHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Ungültige ID")
		return
	}

	var req updateScheduleEntryRequest
	if validationErrors, err := request.DecodeAndValidate(r, &req); err != nil {
		response.BadRequest(w, "Ungültige Anfrage")
		return
	} else if validationErrors != nil {
		response.ValidationError(w, "Validierungsfehler", validationErrors)
		return
	}

	var date *time.Time
	if req.Date != nil {
		parsed, err := service.ParseDate(*req.Date)
		if err != nil {
			response.BadRequest(w, "Ungültiges date")
			return
		}
		date = &parsed
	}

	var startTime *time.Time
	if req.StartTime != nil {
		parsed, err := service.ParseTime(*req.StartTime)
		if err != nil {
			response.BadRequest(w, "Ungültiges startTime")
			return
		}
		startTime = &parsed
	}

	var endTime *time.Time
	if req.EndTime != nil {
		parsed, err := service.ParseTime(*req.EndTime)
		if err != nil {
			response.BadRequest(w, "Ungültiges endTime")
			return
		}
		endTime = &parsed
	}

	var entryType *domain.ScheduleEntryType
	if req.EntryType != nil {
		parsed := parseScheduleEntryType(*req.EntryType)
		entryType = &parsed
	}
	var shiftKind *domain.ShiftKind
	if req.ShiftKind != nil {
		parsed := parseShiftKind(*req.ShiftKind)
		shiftKind = &parsed
	}
	overrideBlockedDay := false
	if req.OverrideBlockedDay != nil {
		overrideBlockedDay = *req.OverrideBlockedDay
	}
	var segments *[]domain.ScheduleEntrySegment
	if req.Segments != nil {
		parsed, errMsg := parseScheduleEntrySegments(*req.Segments)
		if errMsg != "" {
			response.BadRequest(w, errMsg)
			return
		}
		segments = &parsed
	}

	entry, err := h.schedules.Update(r.Context(), id, service.UpdateScheduleEntryInput{
		Date:               date,
		StartTime:          startTime,
		EndTime:            endTime,
		BreakMinutes:       req.BreakMinutes,
		GroupID:            req.GroupID,
		EntryType:          entryType,
		ShiftKind:          shiftKind,
		Notes:              req.Notes,
		Segments:           segments,
		OverrideBlockedDay: overrideBlockedDay,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.Success(w, mapScheduleEntryResponse(*entry))
}

// ListRequests handles GET /schedule/requests.
// @Summary List schedule requests
// @Description Get non-working wish and appointment requests for a date range
// @Tags Schedule
// @Produce json
// @Security BearerAuth
// @Param startDate query string true "Start date (YYYY-MM-DD)"
// @Param endDate query string true "End date (YYYY-MM-DD)"
// @Param employeeId query int false "Filter by employee ID"
// @Success 200 {array} ScheduleRequestResponse "List of schedule requests"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Router /schedule/requests [get]
func (h *ScheduleHandler) ListRequests(w http.ResponseWriter, r *http.Request) {
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

	var employeeID *int64
	if raw := request.GetQueryString(r, "employeeId", ""); raw != "" {
		parsed, err := parseID(raw)
		if err != nil {
			response.BadRequest(w, "Ungültige employeeId")
			return
		}
		employeeID = &parsed
	}
	if user.Role != string(domain.EmployeeRoleAdmin) {
		employeeID = &user.UserID
	}

	requests, err := h.schedules.ListRequests(r.Context(), startDate, endDate, employeeID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.Success(w, mapScheduleRequestResponses(requests))
}

// CreateRequest handles POST /schedule/requests.
// @Summary Create a schedule request
// @Description Create a non-working wish or appointment request
// @Tags Schedule
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body createScheduleRequestRequest true "Schedule request data"
// @Success 201 {object} ScheduleRequestResponse "Created schedule request"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Failure 403 {object} map[string]interface{} "Forbidden"
// @Router /schedule/requests [post]
func (h *ScheduleHandler) CreateRequest(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		response.Unauthorized(w, "Authentifizierung erforderlich")
		return
	}

	var req createScheduleRequestRequest
	if validationErrors, err := request.DecodeAndValidate(r, &req); err != nil {
		response.BadRequest(w, "Ungültige Anfrage")
		return
	} else if validationErrors != nil {
		response.ValidationError(w, "Validierungsfehler", validationErrors)
		return
	}

	input, errMsg := parseCreateScheduleRequestInput(req, user.UserID)
	if errMsg != "" {
		response.BadRequest(w, errMsg)
		return
	}
	item, err := h.schedules.CreateRequest(r.Context(), *input, user.UserID, user.Role == string(domain.EmployeeRoleAdmin))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.Created(w, mapScheduleRequestResponse(*item))
}

// UpdateRequest handles PUT /schedule/requests/{id}.
// @Summary Update a schedule request
// @Description Update a non-working wish or appointment request
// @Tags Schedule
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Schedule request ID"
// @Param request body updateScheduleRequestRequest true "Updated schedule request data"
// @Success 200 {object} ScheduleRequestResponse "Updated schedule request"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Failure 403 {object} map[string]interface{} "Forbidden"
// @Failure 404 {object} map[string]interface{} "Schedule request not found"
// @Router /schedule/requests/{id} [put]
func (h *ScheduleHandler) UpdateRequest(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		response.Unauthorized(w, "Authentifizierung erforderlich")
		return
	}
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Ungültige ID")
		return
	}

	var req updateScheduleRequestRequest
	if validationErrors, err := request.DecodeAndValidate(r, &req); err != nil {
		response.BadRequest(w, "Ungültige Anfrage")
		return
	} else if validationErrors != nil {
		response.ValidationError(w, "Validierungsfehler", validationErrors)
		return
	}

	input, errMsg := parseUpdateScheduleRequestInput(req)
	if errMsg != "" {
		response.BadRequest(w, errMsg)
		return
	}
	item, err := h.schedules.UpdateRequest(r.Context(), id, *input, user.UserID, user.Role == string(domain.EmployeeRoleAdmin))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.Success(w, mapScheduleRequestResponse(*item))
}

// DeleteRequest handles DELETE /schedule/requests/{id}.
// @Summary Delete a schedule request
// @Description Delete a non-working wish or appointment request
// @Tags Schedule
// @Security BearerAuth
// @Param id path int true "Schedule request ID"
// @Success 204 "Schedule request deleted"
// @Failure 400 {object} map[string]interface{} "Invalid schedule request ID"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Failure 403 {object} map[string]interface{} "Forbidden"
// @Failure 404 {object} map[string]interface{} "Schedule request not found"
// @Router /schedule/requests/{id} [delete]
func (h *ScheduleHandler) DeleteRequest(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		response.Unauthorized(w, "Authentifizierung erforderlich")
		return
	}
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Ungültige ID")
		return
	}

	if err := h.schedules.DeleteRequest(r.Context(), id, user.UserID, user.Role == string(domain.EmployeeRoleAdmin)); err != nil {
		writeServiceError(w, err)
		return
	}
	response.NoContent(w)
}

func parseShiftKind(value string) domain.ShiftKind {
	switch value {
	case string(domain.ShiftKindEarly):
		return domain.ShiftKindEarly
	case string(domain.ShiftKindLate):
		return domain.ShiftKindLate
	default:
		return domain.ShiftKindManual
	}
}

// Delete handles DELETE /schedule/{id}.
// @Summary Delete a schedule entry
// @Description Delete a schedule entry
// @Tags Schedule
// @Security BearerAuth
// @Param id path int true "Schedule entry ID"
// @Success 204 "Schedule entry deleted"
// @Failure 400 {object} map[string]interface{} "Invalid schedule entry ID"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Failure 404 {object} map[string]interface{} "Schedule entry not found"
// @Router /schedule/{id} [delete]
func (h *ScheduleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Ungültige ID")
		return
	}

	if err := h.schedules.Delete(r.Context(), id); err != nil {
		writeServiceError(w, err)
		return
	}

	response.NoContent(w)
}

func parseScheduleEntryType(value string) domain.ScheduleEntryType {
	switch value {
	case string(domain.ScheduleEntryTypeWork):
		return domain.ScheduleEntryTypeWork
	case string(domain.ScheduleEntryTypeVacation):
		return domain.ScheduleEntryTypeVacation
	case string(domain.ScheduleEntryTypeSick):
		return domain.ScheduleEntryTypeSick
	case string(domain.ScheduleEntryTypeChildSick):
		return domain.ScheduleEntryTypeChildSick
	case string(domain.ScheduleEntryTypeRecoveryDay):
		return domain.ScheduleEntryTypeRecoveryDay
	case string(domain.ScheduleEntryTypeSpecialLeave):
		return domain.ScheduleEntryTypeSpecialLeave
	case string(domain.ScheduleEntryTypeTraining):
		return domain.ScheduleEntryTypeTraining
	default:
		return domain.ScheduleEntryTypeEvent
	}
}

// parseScheduleEntryInput converts a createScheduleEntryRequest to service.CreateScheduleEntryInput.
// Returns the input and an error message if parsing fails.
func parseScheduleEntryInput(req createScheduleEntryRequest) (*service.CreateScheduleEntryInput, string) {
	date, err := service.ParseDate(req.Date)
	if err != nil {
		return nil, "Ungültiges date"
	}

	var startTime *time.Time
	if req.StartTime != nil {
		parsed, err := service.ParseTime(*req.StartTime)
		if err != nil {
			return nil, "Ungültiges startTime"
		}
		startTime = &parsed
	}

	var endTime *time.Time
	if req.EndTime != nil {
		parsed, err := service.ParseTime(*req.EndTime)
		if err != nil {
			return nil, "Ungültiges endTime"
		}
		endTime = &parsed
	}

	breakMinutes := 0
	if req.BreakMinutes != nil {
		breakMinutes = *req.BreakMinutes
	}

	entryType := domain.ScheduleEntryTypeWork
	if req.EntryType != nil {
		entryType = parseScheduleEntryType(*req.EntryType)
	}
	shiftKind := domain.ShiftKindManual
	if req.ShiftKind != nil {
		shiftKind = parseShiftKind(*req.ShiftKind)
	}
	overrideBlockedDay := false
	if req.OverrideBlockedDay != nil {
		overrideBlockedDay = *req.OverrideBlockedDay
	}
	segments, errMsg := parseScheduleEntrySegments(req.Segments)
	if errMsg != "" {
		return nil, errMsg
	}

	return &service.CreateScheduleEntryInput{
		EmployeeID:         req.EmployeeID,
		Date:               date,
		StartTime:          startTime,
		EndTime:            endTime,
		BreakMinutes:       breakMinutes,
		GroupID:            req.GroupID,
		EntryType:          entryType,
		ShiftKind:          shiftKind,
		Notes:              req.Notes,
		Segments:           segments,
		OverrideBlockedDay: overrideBlockedDay,
	}, ""
}

func parseScheduleEntrySegments(req []scheduleEntrySegmentRequest) ([]domain.ScheduleEntrySegment, string) {
	segments := make([]domain.ScheduleEntrySegment, 0, len(req))
	for _, item := range req {
		startTime, err := service.ParseTime(item.StartTime)
		if err != nil {
			return nil, "Ungültiges Segment startTime"
		}
		endTime, err := service.ParseTime(item.EndTime)
		if err != nil {
			return nil, "Ungültiges Segment endTime"
		}
		segments = append(segments, domain.ScheduleEntrySegment{
			GroupID:   item.GroupID,
			StartTime: startTime,
			EndTime:   endTime,
			Notes:     item.Notes,
			SortOrder: item.SortOrder,
		})
	}
	return segments, ""
}

func parseCreateScheduleRequestInput(req createScheduleRequestRequest, fallbackEmployeeID int64) (*service.CreateScheduleRequestInput, string) {
	date, err := service.ParseDate(req.Date)
	if err != nil {
		return nil, "Ungültiges date"
	}
	startTime, errMsg := parseOptionalTime(req.StartTime, "startTime")
	if errMsg != "" {
		return nil, errMsg
	}
	endTime, errMsg := parseOptionalTime(req.EndTime, "endTime")
	if errMsg != "" {
		return nil, errMsg
	}

	employeeID := fallbackEmployeeID
	if req.EmployeeID != nil {
		employeeID = *req.EmployeeID
	}
	requestType := domain.ScheduleRequestTypeWish
	if req.RequestType != nil {
		requestType = parseScheduleRequestType(*req.RequestType)
	}

	return &service.CreateScheduleRequestInput{
		EmployeeID:  employeeID,
		Date:        date,
		StartTime:   startTime,
		EndTime:     endTime,
		RequestType: requestType,
		Text:        strings.TrimSpace(req.Text),
	}, ""
}

func parseUpdateScheduleRequestInput(req updateScheduleRequestRequest) (*service.UpdateScheduleRequestInput, string) {
	var date *time.Time
	if req.Date != nil {
		parsed, err := service.ParseDate(*req.Date)
		if err != nil {
			return nil, "Ungültiges date"
		}
		date = &parsed
	}
	startTime, errMsg := parseOptionalTime(req.StartTime, "startTime")
	if errMsg != "" {
		return nil, errMsg
	}
	endTime, errMsg := parseOptionalTime(req.EndTime, "endTime")
	if errMsg != "" {
		return nil, errMsg
	}

	var requestType *domain.ScheduleRequestType
	if req.RequestType != nil {
		parsed := parseScheduleRequestType(*req.RequestType)
		requestType = &parsed
	}
	var status *domain.ScheduleRequestStatus
	if req.Status != nil {
		parsed := parseScheduleRequestStatus(*req.Status)
		status = &parsed
	}
	var text *string
	if req.Text != nil {
		trimmed := strings.TrimSpace(*req.Text)
		text = &trimmed
	}

	return &service.UpdateScheduleRequestInput{
		Date:        date,
		StartTime:   startTime,
		EndTime:     endTime,
		RequestType: requestType,
		Text:        text,
		Status:      status,
	}, ""
}

func parseOptionalTime(value *string, field string) (*time.Time, string) {
	if value == nil {
		return nil, ""
	}
	parsed, err := service.ParseTime(*value)
	if err != nil {
		return nil, "Ungültiges " + field
	}
	return &parsed, ""
}

func parseScheduleRequestType(value string) domain.ScheduleRequestType {
	switch value {
	case string(domain.ScheduleRequestTypeAppointment):
		return domain.ScheduleRequestTypeAppointment
	default:
		return domain.ScheduleRequestTypeWish
	}
}

func parseScheduleRequestStatus(value string) domain.ScheduleRequestStatus {
	switch value {
	case string(domain.ScheduleRequestStatusDone):
		return domain.ScheduleRequestStatusDone
	default:
		return domain.ScheduleRequestStatusOpen
	}
}

func mapScheduleRequestResponses(requests []domain.ScheduleRequest) []ScheduleRequestResponse {
	result := make([]ScheduleRequestResponse, 0, len(requests))
	for _, item := range requests {
		result = append(result, mapScheduleRequestResponse(item))
	}
	return result
}

func mapTimeSuggestionResponse(suggestion *service.TimeSuggestion) TimeSuggestionResponse {
	response := TimeSuggestionResponse{
		BreakMinutes:   suggestion.BreakMinutes,
		PlannedMinutes: suggestion.PlannedMinutes,
		IsBlocked:      suggestion.IsBlocked,
		ContractID:     suggestion.ContractID,
	}
	if suggestion.StartTime != nil {
		formatted := suggestion.StartTime.Format(timeLayoutSecs)
		response.StartTime = &formatted
	}
	if suggestion.EndTime != nil {
		formatted := suggestion.EndTime.Format(timeLayoutSecs)
		response.EndTime = &formatted
	}
	return response
}
