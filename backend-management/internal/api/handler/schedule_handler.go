package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

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

type createScheduleEntryRequest struct {
	EmployeeID   int64   `json:"employeeId" validate:"required"`
	Date         string  `json:"date" validate:"required"`
	StartTime    *string `json:"startTime,omitempty"`
	EndTime      *string `json:"endTime,omitempty"`
	BreakMinutes *int    `json:"breakMinutes,omitempty" validate:"omitempty,gte=0"`
	GroupID      *int64  `json:"groupId,omitempty"`
	EntryType    *string `json:"entryType,omitempty" validate:"omitempty,oneof=WORK VACATION SICK SPECIAL_LEAVE TRAINING EVENT"`
	Notes        *string `json:"notes,omitempty"`
}

type updateScheduleEntryRequest struct {
	Date         *string `json:"date,omitempty"`
	StartTime    *string `json:"startTime,omitempty"`
	EndTime      *string `json:"endTime,omitempty"`
	BreakMinutes *int    `json:"breakMinutes,omitempty" validate:"omitempty,gte=0"`
	GroupID      *int64  `json:"groupId,omitempty"`
	EntryType    *string `json:"entryType,omitempty" validate:"omitempty,oneof=WORK VACATION SICK SPECIAL_LEAVE TRAINING EVENT"`
	Notes        *string `json:"notes,omitempty"`
}

// List handles GET /schedule.
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
	})
}

// Create handles POST /schedule.
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

	entry, err := h.schedules.Update(r.Context(), id, service.UpdateScheduleEntryInput{
		Date:         date,
		StartTime:    startTime,
		EndTime:      endTime,
		BreakMinutes: req.BreakMinutes,
		GroupID:      req.GroupID,
		EntryType:    entryType,
		Notes:        req.Notes,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.Success(w, mapScheduleEntryResponse(*entry))
}

// Delete handles DELETE /schedule/{id}.
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

	return &service.CreateScheduleEntryInput{
		EmployeeID:   req.EmployeeID,
		Date:         date,
		StartTime:    startTime,
		EndTime:      endTime,
		BreakMinutes: breakMinutes,
		GroupID:      req.GroupID,
		EntryType:    entryType,
		Notes:        req.Notes,
	}, ""
}
