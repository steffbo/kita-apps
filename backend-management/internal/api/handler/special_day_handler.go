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

type specialDayRequest struct {
	Date       string  `json:"date" validate:"required"`
	EndDate    *string `json:"endDate,omitempty"`
	Name       string  `json:"name" validate:"required"`
	DayType    string  `json:"dayType" validate:"required,oneof=HOLIDAY CLOSURE TEAM_DAY EVENT"`
	AffectsAll *bool   `json:"affectsAll,omitempty"`
	Notes      *string `json:"notes,omitempty"`
}

// List handles GET /special-days.
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
