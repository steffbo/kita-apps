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
// @Summary Get overview statistics
// @Description Get monthly overview statistics for all employees
// @Tags Statistics
// @Produce json
// @Security BearerAuth
// @Param month query string true "Month (YYYY-MM-DD, any day in the month)"
// @Success 200 {object} OverviewStatisticsResponse "Overview statistics"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Router /statistics/overview [get]
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
// @Summary Get employee statistics
// @Description Get monthly statistics for a specific employee
// @Tags Statistics
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Param month query string true "Month (YYYY-MM-DD, any day in the month)"
// @Success 200 {object} EmployeeStatisticsResponse "Employee statistics"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Failure 404 {object} map[string]interface{} "Employee not found"
// @Router /statistics/employee/{id} [get]
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
// @Summary Get weekly statistics
// @Description Get weekly statistics by employee and group
// @Tags Statistics
// @Produce json
// @Security BearerAuth
// @Param weekStart query string true "Week start date (YYYY-MM-DD, must be a Monday)"
// @Success 200 {object} WeeklyStatisticsResponse "Weekly statistics"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Router /statistics/weekly [get]
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
// @Summary Export timesheet
// @Description Export timesheet data (placeholder, not yet implemented)
// @Tags Export
// @Produce json
// @Security BearerAuth
// @Success 204 "No content - not yet implemented"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Router /export/timesheet [get]
func (h *StatisticsHandler) ExportTimesheet(w http.ResponseWriter, r *http.Request) {
	response.NoContent(w)
}

// ExportSchedule handles GET /export/schedule.
// @Summary Export schedule
// @Description Export schedule data (placeholder, not yet implemented)
// @Tags Export
// @Produce json
// @Security BearerAuth
// @Success 204 "No content - not yet implemented"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Router /export/schedule [get]
func (h *StatisticsHandler) ExportSchedule(w http.ResponseWriter, r *http.Request) {
	response.NoContent(w)
}
