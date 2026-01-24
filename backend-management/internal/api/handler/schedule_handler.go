package handler

import (
	"net/http"
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
	EmployeeID   int64   `json:"employeeId"`
	Date         string  `json:"date"`
	StartTime    *string `json:"startTime,omitempty"`
	EndTime      *string `json:"endTime,omitempty"`
	BreakMinutes *int    `json:"breakMinutes,omitempty"`
	GroupID      *int64  `json:"groupId,omitempty"`
	EntryType    *string `json:"entryType,omitempty"`
	Notes        *string `json:"notes,omitempty"`
}

type updateScheduleEntryRequest struct {
	Date         *string `json:"date,omitempty"`
	StartTime    *string `json:"startTime,omitempty"`
	EndTime      *string `json:"endTime,omitempty"`
	BreakMinutes *int    `json:"breakMinutes,omitempty"`
	GroupID      *int64  `json:"groupId,omitempty"`
	EntryType    *string `json:"entryType,omitempty"`
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
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "Ungültige Anfrage")
		return
	}
	if req.EmployeeID == 0 || req.Date == "" {
		response.BadRequest(w, "employeeId und date sind erforderlich")
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

	var endTime *time.Time
	if req.EndTime != nil {
		parsed, err := service.ParseTime(*req.EndTime)
		if err != nil {
			response.BadRequest(w, "Ungültiges endTime")
			return
		}
		endTime = &parsed
	}

	breakMinutes := 0
	if req.BreakMinutes != nil {
		breakMinutes = *req.BreakMinutes
	}

	entryType := domain.ScheduleEntryTypeWork
	if req.EntryType != nil {
		parsed, ok := parseScheduleEntryType(*req.EntryType)
		if !ok {
			response.BadRequest(w, "Ungültiger entryType")
			return
		}
		entryType = parsed
	}

	entry, err := h.schedules.Create(r.Context(), service.CreateScheduleEntryInput{
		EmployeeID:   req.EmployeeID,
		Date:         date,
		StartTime:    startTime,
		EndTime:      endTime,
		BreakMinutes: breakMinutes,
		GroupID:      req.GroupID,
		EntryType:    entryType,
		Notes:        req.Notes,
	})
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
	for _, item := range req {
		if item.EmployeeID == 0 || item.Date == "" {
			response.BadRequest(w, "employeeId und date sind erforderlich")
			return
		}

		date, err := service.ParseDate(item.Date)
		if err != nil {
			response.BadRequest(w, "Ungültiges date")
			return
		}

		var startTime *time.Time
		if item.StartTime != nil {
			parsed, err := service.ParseTime(*item.StartTime)
			if err != nil {
				response.BadRequest(w, "Ungültiges startTime")
				return
			}
			startTime = &parsed
		}

		var endTime *time.Time
		if item.EndTime != nil {
			parsed, err := service.ParseTime(*item.EndTime)
			if err != nil {
				response.BadRequest(w, "Ungültiges endTime")
				return
			}
			endTime = &parsed
		}

		breakMinutes := 0
		if item.BreakMinutes != nil {
			breakMinutes = *item.BreakMinutes
		}

		entryType := domain.ScheduleEntryTypeWork
		if item.EntryType != nil {
			parsed, ok := parseScheduleEntryType(*item.EntryType)
			if !ok {
				response.BadRequest(w, "Ungültiger entryType")
				return
			}
			entryType = parsed
		}

		inputs = append(inputs, service.CreateScheduleEntryInput{
			EmployeeID:   item.EmployeeID,
			Date:         date,
			StartTime:    startTime,
			EndTime:      endTime,
			BreakMinutes: breakMinutes,
			GroupID:      item.GroupID,
			EntryType:    entryType,
			Notes:        item.Notes,
		})
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
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "Ungültige Anfrage")
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
		parsed, ok := parseScheduleEntryType(*req.EntryType)
		if !ok {
			response.BadRequest(w, "Ungültiger entryType")
			return
		}
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

func parseScheduleEntryType(value string) (domain.ScheduleEntryType, bool) {
	switch value {
	case string(domain.ScheduleEntryTypeWork):
		return domain.ScheduleEntryTypeWork, true
	case string(domain.ScheduleEntryTypeVacation):
		return domain.ScheduleEntryTypeVacation, true
	case string(domain.ScheduleEntryTypeSick):
		return domain.ScheduleEntryTypeSick, true
	case string(domain.ScheduleEntryTypeSpecialLeave):
		return domain.ScheduleEntryTypeSpecialLeave, true
	case string(domain.ScheduleEntryTypeTraining):
		return domain.ScheduleEntryTypeTraining, true
	case string(domain.ScheduleEntryTypeEvent):
		return domain.ScheduleEntryTypeEvent, true
	default:
		return "", false
	}
}
