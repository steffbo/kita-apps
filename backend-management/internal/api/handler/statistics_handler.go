package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/knirpsenstadt/kita-apps/backend-management/internal/api/request"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/service"
)

// StatisticsHandler handles statistics requests.
type StatisticsHandler struct {
	service *service.StatisticsService
}

// NewStatisticsHandler creates a new StatisticsHandler.
func NewStatisticsHandler(service *service.StatisticsService) *StatisticsHandler {
	return &StatisticsHandler{service: service}
}

// Overview handles GET /statistics/overview.
func (h *StatisticsHandler) Overview(w http.ResponseWriter, r *http.Request) {
	monthStr := request.GetQueryString(r, "month", "")
	if monthStr == "" {
		response.BadRequest(w, "month ist erforderlich")
		return
	}

	month, err := service.ParseDate(monthStr)
	if err != nil {
		response.BadRequest(w, "Ungültiges month")
		return
	}

	stats, err := h.service.Overview(r.Context(), month)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.Success(w, mapOverviewResponse(stats))
}

// Employee handles GET /statistics/employee/{id}.
func (h *StatisticsHandler) Employee(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Ungültige ID")
		return
	}

	monthStr := request.GetQueryString(r, "month", "")
	if monthStr == "" {
		response.BadRequest(w, "month ist erforderlich")
		return
	}

	month, err := service.ParseDate(monthStr)
	if err != nil {
		response.BadRequest(w, "Ungültiges month")
		return
	}

	stats, err := h.service.EmployeeStats(r.Context(), id, month)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.Success(w, mapEmployeeStatsResponse(stats))
}

// Weekly handles GET /statistics/weekly.
func (h *StatisticsHandler) Weekly(w http.ResponseWriter, r *http.Request) {
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

	stats, err := h.service.Weekly(r.Context(), weekStart)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.Success(w, mapWeeklyStatsResponse(stats))
}

// ExportTimesheet handles GET /export/timesheet.
func (h *StatisticsHandler) ExportTimesheet(w http.ResponseWriter, r *http.Request) {
	response.NoContent(w)
}

// ExportSchedule handles GET /export/schedule.
func (h *StatisticsHandler) ExportSchedule(w http.ResponseWriter, r *http.Request) {
	response.NoContent(w)
}
