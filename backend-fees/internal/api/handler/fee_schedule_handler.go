package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/request"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/util"
)

// FeeScheduleHandler handles the versioned fee regulation (Beitragsordnung).
type FeeScheduleHandler struct {
	scheduleService *service.FeeScheduleService
}

// NewFeeScheduleHandler creates a new fee schedule handler.
func NewFeeScheduleHandler(scheduleService *service.FeeScheduleService) *FeeScheduleHandler {
	return &FeeScheduleHandler{scheduleService: scheduleService}
}

// Fee schedule version status values.
const (
	FeeScheduleStatusPast    = "past"
	FeeScheduleStatusActive  = "active"
	FeeScheduleStatusPlanned = "planned"
)

// FeeScheduleResponse is one version of the fee regulation.
// @Description Version of the fee regulation (Elternbeitragsordnung)
type FeeScheduleResponse struct {
	ID        string `json:"id"`
	ValidFrom string `json:"validFrom" example:"2027-01-01"`
	// ValidUntil is the day before the next version starts; absent for the latest version.
	ValidUntil *string                  `json:"validUntil,omitempty" example:"2027-07-31"`
	Name       string                   `json:"name"`
	Config     domain.FeeScheduleConfig `json:"config"`
	// Status is past, active or planned.
	Status string `json:"status" enums:"past,active,planned"`
	// Editable is true for planned versions only.
	Editable  bool   `json:"editable"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
} //@name FeeScheduleVersion

// FeeScheduleRequest creates or updates a planned version.
// @Description Planned version of the fee regulation
type FeeScheduleRequest struct {
	ValidFrom string                   `json:"validFrom" example:"2027-01-01"` // first of a future month
	Name      string                   `json:"name" example:"Elternbeitragsordnung 2027"`
	Config    domain.FeeScheduleConfig `json:"config"`
} //@name FeeScheduleRequest

// List handles GET /fee-schedules
// @Summary List fee schedule versions
// @Description Returns all versions of the fee regulation, oldest first
// @Tags FeeSchedules
// @Produce json
// @Security BearerAuth
// @Success 200 {array} FeeScheduleResponse "Versions"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /fee-schedules [get]
func (h *FeeScheduleHandler) List(w http.ResponseWriter, r *http.Request) {
	schedules, err := h.scheduleService.List(r.Context())
	if err != nil {
		response.InternalError(w, "failed to load fee schedules")
		return
	}
	response.Success(w, toFeeScheduleResponses(schedules, h.scheduleService))
}

// Create handles POST /fee-schedules
// @Summary Create planned fee schedule version
// @Description Adds a version that starts on the first of a future month
// @Tags FeeSchedules
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body FeeScheduleRequest true "Version"
// @Success 201 {object} FeeScheduleResponse "Created version"
// @Failure 400 {object} response.ErrorBody "Invalid version"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 409 {object} response.ErrorBody "A version already starts on this date"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /fee-schedules [post]
func (h *FeeScheduleHandler) Create(w http.ResponseWriter, r *http.Request) {
	input, ok := decodeFeeScheduleRequest(w, r)
	if !ok {
		return
	}
	created, err := h.scheduleService.Create(r.Context(), input)
	if err != nil {
		writeFeeScheduleError(w, err)
		return
	}
	h.respondWithVersion(w, r, created.ID.String(), http.StatusCreated)
}

// Update handles PUT /fee-schedules/{id}
// @Summary Update planned fee schedule version
// @Description Changes a version that has not started yet
// @Tags FeeSchedules
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Version ID (UUID)"
// @Param request body FeeScheduleRequest true "Version"
// @Success 200 {object} FeeScheduleResponse "Updated version"
// @Failure 400 {object} response.ErrorBody "Invalid version"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Version not found"
// @Failure 409 {object} response.ErrorBody "Version already in effect or date taken"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /fee-schedules/{id} [put]
func (h *FeeScheduleHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	input, ok := decodeFeeScheduleRequest(w, r)
	if !ok {
		return
	}
	if _, err := h.scheduleService.Update(r.Context(), id, input); err != nil {
		writeFeeScheduleError(w, err)
		return
	}
	h.respondWithVersion(w, r, id.String(), http.StatusOK)
}

// Delete handles DELETE /fee-schedules/{id}
// @Summary Delete planned fee schedule version
// @Tags FeeSchedules
// @Security BearerAuth
// @Param id path string true "Version ID (UUID)"
// @Success 204 "Deleted"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Version not found"
// @Failure 409 {object} response.ErrorBody "Version already in effect"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /fee-schedules/{id} [delete]
func (h *FeeScheduleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	if err := h.scheduleService.Delete(r.Context(), id); err != nil {
		writeFeeScheduleError(w, err)
		return
	}
	response.NoContent(w)
}

// respondWithVersion returns the version with status/validUntil, which depend on its neighbours.
func (h *FeeScheduleHandler) respondWithVersion(w http.ResponseWriter, r *http.Request, id string, status int) {
	schedules, err := h.scheduleService.List(r.Context())
	if err != nil {
		response.InternalError(w, "failed to load fee schedules")
		return
	}
	for _, v := range toFeeScheduleResponses(schedules, h.scheduleService) {
		if v.ID == id {
			response.JSON(w, status, v)
			return
		}
	}
	response.NotFound(w, "fee schedule not found")
}

func decodeFeeScheduleRequest(w http.ResponseWriter, r *http.Request) (service.FeeScheduleInput, bool) {
	var req FeeScheduleRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return service.FeeScheduleInput{}, false
	}
	validFrom, err := time.Parse("2006-01-02", req.ValidFrom)
	if err != nil {
		response.BadRequest(w, "invalid validFrom (expected YYYY-MM-DD)")
		return service.FeeScheduleInput{}, false
	}
	return service.FeeScheduleInput{ValidFrom: validFrom, Name: req.Name, Config: req.Config}, true
}

func writeFeeScheduleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		response.BadRequest(w, strings.TrimPrefix(err.Error(), service.ErrInvalidInput.Error()+": "))
	case errors.Is(err, service.ErrConflict):
		response.Conflict(w, strings.TrimPrefix(err.Error(), service.ErrConflict.Error()+": "))
	case errors.Is(err, service.ErrNotFound):
		response.NotFound(w, "fee schedule not found")
	default:
		response.InternalError(w, "failed to save fee schedule")
	}
}

func toFeeScheduleResponses(schedules domain.FeeSchedules, svc *service.FeeScheduleService) []FeeScheduleResponse {
	today := util.Today()
	active, _ := schedules.At(today)
	result := make([]FeeScheduleResponse, 0, len(schedules))
	for i, s := range schedules {
		status := FeeScheduleStatusPast
		switch {
		case svc.IsEditable(s):
			status = FeeScheduleStatusPlanned
		case active != nil && active.ID == s.ID:
			status = FeeScheduleStatusActive
		}
		var validUntil *string
		if i+1 < len(schedules) {
			until := schedules[i+1].ValidFrom.AddDate(0, 0, -1).Format("2006-01-02")
			validUntil = &until
		}
		result = append(result, FeeScheduleResponse{
			ID:         s.ID.String(),
			ValidFrom:  s.ValidFrom.Format("2006-01-02"),
			ValidUntil: validUntil,
			Name:       s.Name,
			Config:     s.Config,
			Status:     status,
			Editable:   status == FeeScheduleStatusPlanned,
			CreatedAt:  s.CreatedAt.Format(time.RFC3339),
			UpdatedAt:  s.UpdatedAt.Format(time.RFC3339),
		})
	}
	return result
}
