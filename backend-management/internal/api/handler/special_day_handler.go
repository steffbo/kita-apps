package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/knirpsenstadt/kita-apps/backend-management/internal/api/request"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/service"
)

// SpecialDayHandler handles special day requests.
type SpecialDayHandler struct {
	service *service.SpecialDayService
}

// NewSpecialDayHandler creates a new SpecialDayHandler.
func NewSpecialDayHandler(service *service.SpecialDayService) *SpecialDayHandler {
	return &SpecialDayHandler{service: service}
}

// specialDayRequest contains the data for creating or updating a special day.
type specialDayRequest struct {
	Date       string  `json:"date" validate:"required" example:"2024-12-25"`
	EndDate    *string `json:"endDate,omitempty" example:"2024-12-26"`
	Name       string  `json:"name" validate:"required" example:"Weihnachten"`
	DayType    string  `json:"dayType" validate:"required,oneof=HOLIDAY CLOSURE TEAM_DAY EVENT" example:"HOLIDAY"`
	AffectsAll *bool   `json:"affectsAll,omitempty" example:"true"`
	Notes      *string `json:"notes,omitempty" example:"Gesetzlicher Feiertag"`
} //@name CreateSpecialDayRequest

// List handles GET /special-days.
// @Summary List special days
// @Description Get all special days (holidays, closures, events) for a year
// @Tags Special Days
// @Produce json
// @Security BearerAuth
// @Param year query int true "Year"
// @Param includeHolidays query bool false "Include public holidays" default(true)
// @Success 200 {array} SpecialDayResponse "List of special days"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Router /special-days [get]
func (h *SpecialDayHandler) List(w http.ResponseWriter, r *http.Request) {
	yearStr := request.GetQueryString(r, "year", "")
	if yearStr == "" {
		response.BadRequest(w, "year ist erforderlich")
		return
	}
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		response.BadRequest(w, "Ungültiges year")
		return
	}

	includeHolidays := true
	if flag := request.GetQueryBool(r, "includeHolidays"); flag != nil {
		includeHolidays = *flag
	}

	days, err := h.service.List(r.Context(), year, includeHolidays)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	result := make([]SpecialDayResponse, 0, len(days))
	for _, day := range days {
		result = append(result, mapSpecialDayResponse(day))
	}

	response.Success(w, result)
}

// Holidays handles GET /special-days/holidays/{year}.
// @Summary List public holidays
// @Description Get all public holidays for a year (auto-generated based on German holidays)
// @Tags Special Days
// @Produce json
// @Security BearerAuth
// @Param year path int true "Year"
// @Success 200 {array} SpecialDayResponse "List of public holidays"
// @Failure 400 {object} map[string]interface{} "Invalid year"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Router /special-days/holidays/{year} [get]
func (h *SpecialDayHandler) Holidays(w http.ResponseWriter, r *http.Request) {
	yearStr := chi.URLParam(r, "year")
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		response.BadRequest(w, "Ungültiges year")
		return
	}

	days, err := h.service.Holidays(r.Context(), year)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	result := make([]SpecialDayResponse, 0, len(days))
	for _, day := range days {
		result = append(result, mapSpecialDayResponse(day))
	}

	response.Success(w, result)
}

// Create handles POST /special-days.
// @Summary Create a special day
// @Description Create a new special day (closure, team day, or event)
// @Tags Special Days
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param specialDay body specialDayRequest true "Special day data"
// @Success 201 {object} SpecialDayResponse "Created special day"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Failure 409 {object} map[string]interface{} "Conflict - date already exists"
// @Router /special-days [post]
func (h *SpecialDayHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req specialDayRequest
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

	var endDate *time.Time
	if req.EndDate != nil {
		parsed, err := service.ParseDate(*req.EndDate)
		if err != nil {
			response.BadRequest(w, "Ungültiges endDate")
			return
		}
		endDate = &parsed
	}

	affectsAll := true
	if req.AffectsAll != nil {
		affectsAll = *req.AffectsAll
	}

	day, err := h.service.Create(r.Context(), service.CreateSpecialDayInput{
		Date:       date,
		EndDate:    endDate,
		Name:       req.Name,
		DayType:    parseSpecialDayType(req.DayType),
		AffectsAll: affectsAll,
		Notes:      req.Notes,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.Created(w, mapSpecialDayResponse(*day))
}

// Update handles PUT /special-days/{id}.
// @Summary Update a special day
// @Description Update an existing special day
// @Tags Special Days
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Special day ID"
// @Param specialDay body specialDayRequest true "Special day data"
// @Success 200 {object} SpecialDayResponse "Updated special day"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Failure 404 {object} map[string]interface{} "Special day not found"
// @Router /special-days/{id} [put]
func (h *SpecialDayHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Ungültige ID")
		return
	}

	var req specialDayRequest
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

	var endDate *time.Time
	if req.EndDate != nil {
		parsed, err := service.ParseDate(*req.EndDate)
		if err != nil {
			response.BadRequest(w, "Ungültiges endDate")
			return
		}
		endDate = &parsed
	}

	affectsAll := true
	if req.AffectsAll != nil {
		affectsAll = *req.AffectsAll
	}

	day, err := h.service.Update(r.Context(), id, service.CreateSpecialDayInput{
		Date:       date,
		EndDate:    endDate,
		Name:       req.Name,
		DayType:    parseSpecialDayType(req.DayType),
		AffectsAll: affectsAll,
		Notes:      req.Notes,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.Success(w, mapSpecialDayResponse(*day))
}

// Delete handles DELETE /special-days/{id}.
// @Summary Delete a special day
// @Description Delete an existing special day
// @Tags Special Days
// @Produce json
// @Security BearerAuth
// @Param id path int true "Special day ID"
// @Success 204 "Special day deleted"
// @Failure 400 {object} map[string]interface{} "Invalid ID"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Failure 404 {object} map[string]interface{} "Special day not found"
// @Router /special-days/{id} [delete]
func (h *SpecialDayHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

func parseSpecialDayType(value string) domain.SpecialDayType {
	switch value {
	case string(domain.SpecialDayTypeHoliday):
		return domain.SpecialDayTypeHoliday
	case string(domain.SpecialDayTypeClosure):
		return domain.SpecialDayTypeClosure
	case string(domain.SpecialDayTypeTeamDay):
		return domain.SpecialDayTypeTeamDay
	default:
		return domain.SpecialDayTypeEvent
	}
}
