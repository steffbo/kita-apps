package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/middleware"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/request"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/util"
)

// ParentWorkHandler exposes parent work operations over HTTP.
type ParentWorkHandler struct{ svc *service.ParentWorkService }

// NewParentWorkHandler creates the HTTP handler.
func NewParentWorkHandler(svc *service.ParentWorkService) *ParentWorkHandler {
	return &ParentWorkHandler{svc: svc}
}

type parentWorkRuleRequest struct {
	ValidFrom            string `json:"validFrom" example:"2027-08-01"`
	HoursPerChildMinutes int    `json:"hoursPerChildMinutes"`
	MissingHourRateCents int    `json:"missingHourRateCents"`
	MaxCarryOverMinutes  int    `json:"maxCarryOverMinutes"`
}

type parentWorkEntryRequest struct {
	HouseholdID     uuid.UUID `json:"householdId"`
	WorkDate        string    `json:"workDate" example:"2026-09-01"`
	DurationMinutes int       `json:"durationMinutes"`
	Occasion        string    `json:"occasion"`
	MemberName      *string   `json:"memberName,omitempty" binding:"optional"`
	ChildName       *string   `json:"childName,omitempty" binding:"optional"`
	Status          string    `json:"status" enums:"SUBMITTED,APPROVED,REJECTED"`
}

type parentWorkVoidRequest struct {
	Reason string `json:"reason"`
}

type parentWorkOverrideRequest struct {
	KitaYear        int    `json:"kitaYear"`
	RequiredMinutes int    `json:"requiredMinutes"`
	Reason          string `json:"reason"`
}

type boardTermRequest struct {
	MemberID  uuid.UUID `json:"memberId"`
	Office    string    `json:"office"`
	StartDate string    `json:"startDate" example:"2026-08-01"`
	EndDate   *string   `json:"endDate,omitempty" binding:"optional"`
	Note      *string   `json:"note,omitempty" binding:"optional"`
}

func parentWorkYear(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := r.URL.Query().Get("kitaYear")
	if raw == "" {
		return domain.ParentWorkKitaYear(util.Today()), true
	}
	year, err := strconv.Atoi(raw)
	if err != nil || year < 1900 || year > 9998 {
		response.BadRequest(w, "Ungültiges Kita-Jahr")
		return 0, false
	}
	return year, true
}

func parentWorkID(w http.ResponseWriter, r *http.Request, key string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, key))
	if err != nil {
		response.BadRequest(w, "Ungültige ID")
		return uuid.Nil, false
	}
	return id, true
}

func parentWorkDate(raw string) (time.Time, error) { return time.Parse("2006-01-02", raw) }

func parentWorkUser(r *http.Request) uuid.UUID {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		return uuid.Nil
	}
	id, _ := uuid.Parse(user.UserID)
	return id
}

func parentWorkError(w http.ResponseWriter, err error) {
	msg := err.Error()
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		response.BadRequest(w, strings.TrimPrefix(msg, service.ErrInvalidInput.Error()+": "))
	case errors.Is(err, service.ErrConflict):
		response.Conflict(w, strings.TrimPrefix(msg, service.ErrConflict.Error()+": "))
	case errors.Is(err, service.ErrNotFound):
		response.NotFound(w, "Datensatz nicht gefunden")
	default:
		response.InternalError(w, "Elternstunden konnten nicht verarbeitet werden")
	}
}

func decodeParentWork(w http.ResponseWriter, r *http.Request, v interface{}) bool {
	if err := request.DecodeJSON(r, v); err != nil {
		response.BadRequest(w, "Ungültige Anfrage")
		return false
	}
	return true
}

// Overview handles GET /parent-work/overview.
// @Summary Elternstundenübersicht
// @Tags parent-work
// @Produce json
// @Security BearerAuth
// @Param kitaYear query int false "Kita-Jahr"
// @Success 200 {object} service.ParentWorkOverview
// @Failure 400 {object} response.ErrorBody
// @Router /parent-work/overview [get]
func (h *ParentWorkHandler) Overview(w http.ResponseWriter, r *http.Request) {
	year, ok := parentWorkYear(w, r)
	if !ok {
		return
	}
	v, err := h.svc.Overview(r.Context(), year)
	if err != nil {
		parentWorkError(w, err)
		return
	}
	response.Success(w, v)
}

// Households handles GET /parent-work/households.
// @Summary Familienauswahl für Elternstunden
// @Tags parent-work
// @Produce json
// @Security BearerAuth
// @Param search query string false "Suche"
// @Success 200 {array} service.ParentWorkHouseholdOption
// @Router /parent-work/households [get]
func (h *ParentWorkHandler) Households(w http.ResponseWriter, r *http.Request) {
	v, err := h.svc.Households(r.Context(), r.URL.Query().Get("search"))
	if err != nil {
		parentWorkError(w, err)
		return
	}
	response.Success(w, v)
}

// Household handles GET /parent-work/households/{id}.
// @Summary Elternstunden einer Familie
// @Tags parent-work
// @Produce json
// @Security BearerAuth
// @Param id path string true "Familie"
// @Param kitaYear query int false "Kita-Jahr"
// @Success 200 {object} service.ParentWorkDetail
// @Router /parent-work/households/{id} [get]
func (h *ParentWorkHandler) Household(w http.ResponseWriter, r *http.Request) {
	id, ok := parentWorkID(w, r, "id")
	if !ok {
		return
	}
	year, ok := parentWorkYear(w, r)
	if !ok {
		return
	}
	v, err := h.svc.Detail(r.Context(), id, year)
	if err != nil {
		parentWorkError(w, err)
		return
	}
	response.Success(w, v)
}

// CreateEntry handles POST /parent-work/entries.
// @Summary Elternstunden erfassen
// @Tags parent-work
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body parentWorkEntryRequest true "Eintrag"
// @Success 201 {object} domain.ParentWorkEntry
// @Router /parent-work/entries [post]
func (h *ParentWorkHandler) CreateEntry(w http.ResponseWriter, r *http.Request) {
	h.saveEntry(w, r, uuid.Nil)
}

// UpdateEntry handles PUT /parent-work/entries/{id}.
// @Summary Elternstunden bearbeiten
// @Tags parent-work
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Eintrag"
// @Param request body parentWorkEntryRequest true "Eintrag"
// @Success 200 {object} domain.ParentWorkEntry
// @Router /parent-work/entries/{id} [put]
func (h *ParentWorkHandler) UpdateEntry(w http.ResponseWriter, r *http.Request) {
	id, ok := parentWorkID(w, r, "id")
	if ok {
		h.saveEntry(w, r, id)
	}
}

func (h *ParentWorkHandler) saveEntry(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	var req parentWorkEntryRequest
	if !decodeParentWork(w, r, &req) {
		return
	}
	if req.WorkDate == "" {
		response.BadRequest(w, "Datum fehlt")
		return
	}
	date, err := parentWorkDate(req.WorkDate)
	if err != nil {
		response.BadRequest(w, "Ungültiges Arbeitsdatum")
		return
	}
	status := req.Status
	if status == "" && id == uuid.Nil {
		status = domain.ParentWorkStatusApproved
	}
	v, err := h.svc.SaveEntry(r.Context(), id, domain.ParentWorkEntry{
		HouseholdID: req.HouseholdID, WorkDate: date, DurationMinutes: req.DurationMinutes,
		Occasion: req.Occasion, MemberName: req.MemberName, ChildName: req.ChildName, Status: status,
	}, parentWorkUser(r))
	if err != nil {
		parentWorkError(w, err)
		return
	}
	if id == uuid.Nil {
		response.Created(w, v)
	} else {
		response.Success(w, v)
	}
}

// VoidEntry handles POST /parent-work/entries/{id}/void.
// @Summary Elternstunden stornieren
// @Tags parent-work
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Eintrag"
// @Param request body parentWorkVoidRequest true "Stornogrund"
// @Success 200 {object} domain.ParentWorkEntry
// @Router /parent-work/entries/{id}/void [post]
func (h *ParentWorkHandler) VoidEntry(w http.ResponseWriter, r *http.Request) {
	id, ok := parentWorkID(w, r, "id")
	if !ok {
		return
	}
	var req parentWorkVoidRequest
	if !decodeParentWork(w, r, &req) {
		return
	}
	v, err := h.svc.VoidEntry(r.Context(), id, req.Reason, parentWorkUser(r))
	if err != nil {
		parentWorkError(w, err)
		return
	}
	response.Success(w, v)
}

// SaveOverride handles PUT /parent-work/households/{id}/override.
// @Summary Manuelles Soll setzen
// @Tags parent-work
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Familie"
// @Param request body parentWorkOverrideRequest true "Soll"
// @Success 200 {object} domain.ParentWorkOverride
// @Router /parent-work/households/{id}/override [put]
func (h *ParentWorkHandler) SaveOverride(w http.ResponseWriter, r *http.Request) {
	id, ok := parentWorkID(w, r, "id")
	if !ok {
		return
	}
	var req parentWorkOverrideRequest
	if !decodeParentWork(w, r, &req) {
		return
	}
	v := domain.ParentWorkOverride{HouseholdID: id, KitaYear: req.KitaYear,
		RequiredMinutes: req.RequiredMinutes, Reason: req.Reason}
	saved, err := h.svc.SaveOverride(r.Context(), v, parentWorkUser(r))
	if err != nil {
		parentWorkError(w, err)
		return
	}
	response.Success(w, saved)
}

// DeleteOverride handles DELETE /parent-work/households/{id}/override.
// @Summary Manuelles Soll entfernen
// @Tags parent-work
// @Security BearerAuth
// @Param id path string true "Familie"
// @Param kitaYear query int true "Kita-Jahr"
// @Success 204
// @Router /parent-work/households/{id}/override [delete]
func (h *ParentWorkHandler) DeleteOverride(w http.ResponseWriter, r *http.Request) {
	id, ok := parentWorkID(w, r, "id")
	if !ok {
		return
	}
	year, err := strconv.Atoi(r.URL.Query().Get("kitaYear"))
	if err != nil || year < 1900 || year > 9998 {
		response.BadRequest(w, "Ungültiges Kita-Jahr")
		return
	}
	if err := h.svc.DeleteOverride(r.Context(), id, year); err != nil {
		parentWorkError(w, err)
		return
	}
	response.NoContent(w)
}

// Terms handles GET /parent-work/board-terms.
// @Summary Vorstandsämter auflisten
// @Tags parent-work
// @Produce json
// @Security BearerAuth
// @Success 200 {array} domain.BoardTerm
// @Router /parent-work/board-terms [get]
func (h *ParentWorkHandler) Terms(w http.ResponseWriter, r *http.Request) {
	v, err := h.svc.Terms(r.Context())
	if err != nil {
		parentWorkError(w, err)
		return
	}
	response.Success(w, v)
}

// CreateTerm handles POST /parent-work/board-terms.
// @Summary Vorstandsamt anlegen
// @Tags parent-work
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body boardTermRequest true "Amt"
// @Success 201 {object} domain.BoardTerm
// @Router /parent-work/board-terms [post]
func (h *ParentWorkHandler) CreateTerm(w http.ResponseWriter, r *http.Request) {
	h.saveTerm(w, r, uuid.Nil)
}

// UpdateTerm handles PUT /parent-work/board-terms/{id}.
// @Summary Vorstandsamt bearbeiten
// @Tags parent-work
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Amt"
// @Param request body boardTermRequest true "Amt"
// @Success 200 {object} domain.BoardTerm
// @Router /parent-work/board-terms/{id} [put]
func (h *ParentWorkHandler) UpdateTerm(w http.ResponseWriter, r *http.Request) {
	id, ok := parentWorkID(w, r, "id")
	if ok {
		h.saveTerm(w, r, id)
	}
}

func (h *ParentWorkHandler) saveTerm(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	var req boardTermRequest
	if !decodeParentWork(w, r, &req) {
		return
	}
	start, err := parentWorkDate(req.StartDate)
	if err != nil {
		response.BadRequest(w, "Ungültiges Startdatum")
		return
	}
	var end *time.Time
	if req.EndDate != nil && *req.EndDate != "" {
		d, err := parentWorkDate(*req.EndDate)
		if err != nil {
			response.BadRequest(w, "Ungültiges Enddatum")
			return
		}
		end = &d
	}
	v, err := h.svc.SaveTerm(r.Context(), id, domain.BoardTerm{MemberID: req.MemberID,
		Office: req.Office, StartDate: start, EndDate: end, Note: req.Note})
	if err != nil {
		parentWorkError(w, err)
		return
	}
	if id == uuid.Nil {
		response.Created(w, v)
	} else {
		response.Success(w, v)
	}
}

// DeleteTerm handles DELETE /parent-work/board-terms/{id}.
// @Summary Vorstandsamt löschen
// @Tags parent-work
// @Security BearerAuth
// @Param id path string true "Amt"
// @Success 204
// @Router /parent-work/board-terms/{id} [delete]
func (h *ParentWorkHandler) DeleteTerm(w http.ResponseWriter, r *http.Request) {
	id, ok := parentWorkID(w, r, "id")
	if !ok {
		return
	}
	if err := h.svc.DeleteTerm(r.Context(), id); err != nil {
		parentWorkError(w, err)
		return
	}
	response.NoContent(w)
}

// ParseImport parses an uploaded CSV file for the parent-work import.
// @Summary Elternstunden-CSV einlesen
// @Tags parent-work
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file true "CSV-Datei"
// @Success 200 {object} service.ParentWorkImportParseResult
// @Failure 400 {object} response.ErrorBody
// @Router /parent-work/import/parse [post]
func (h *ParentWorkHandler) ParseImport(w http.ResponseWriter, r *http.Request) {
	if !parseUpload(w, r) {
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		response.BadRequest(w, "Keine Datei hochgeladen")
		return
	}
	defer file.Close()
	v, err := h.svc.ParseImport(file)
	if err != nil {
		response.BadRequest(w, "CSV-Datei konnte nicht gelesen werden")
		return
	}
	response.Success(w, v)
}

// PreviewImport returns parsed values, household matches, and validation errors.
// @Summary Elternstunden-Import vorschauen
// @Tags parent-work
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param preview body service.ParentWorkImportPreviewRequest true "Spaltenzuordnung und CSV-Zeilen"
// @Success 200 {array} service.ParentWorkImportRow
// @Failure 400 {object} response.ErrorBody
// @Router /parent-work/import/preview [post]
func (h *ParentWorkHandler) PreviewImport(w http.ResponseWriter, r *http.Request) {
	var req service.ParentWorkImportPreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Ungültige Anfrage")
		return
	}
	v, err := h.svc.PreviewImport(r.Context(), req)
	if err != nil {
		parentWorkError(w, err)
		return
	}
	response.Success(w, v)
}

// ExecuteImport atomically creates approved entries from selected import rows.
// @Summary Elternstunden-Import ausführen
// @Tags parent-work
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param execute body service.ParentWorkImportExecuteRequest true "Zu importierende Zeilen"
// @Success 200 {object} service.ParentWorkImportExecuteResult
// @Failure 400 {object} response.ErrorBody
// @Router /parent-work/import/execute [post]
func (h *ParentWorkHandler) ExecuteImport(w http.ResponseWriter, r *http.Request) {
	var req service.ParentWorkImportExecuteRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 2<<20)).Decode(&req); err != nil {
		response.BadRequest(w, "Ungültige Anfrage")
		return
	}
	v, err := h.svc.ExecuteImport(r.Context(), req, parentWorkUser(r))
	if err != nil {
		parentWorkError(w, err)
		return
	}
	response.Success(w, v)
}

// Rules handles GET /parent-work/rules.
// @Summary Elternstundenregelwerke auflisten
// @Tags parent-work
// @Produce json
// @Security BearerAuth
// @Success 200 {array} domain.ParentWorkRule
// @Router /parent-work/rules [get]
func (h *ParentWorkHandler) Rules(w http.ResponseWriter, r *http.Request) {
	v, err := h.svc.Rules(r.Context())
	if err != nil {
		parentWorkError(w, err)
		return
	}
	response.Success(w, v)
}

// CreateRule handles POST /parent-work/rules.
// @Summary Elternstundenregelwerk anlegen
// @Tags parent-work
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body parentWorkRuleRequest true "Regelwerk"
// @Success 201 {object} domain.ParentWorkRule
// @Router /parent-work/rules [post]
func (h *ParentWorkHandler) CreateRule(w http.ResponseWriter, r *http.Request) {
	h.saveRule(w, r, uuid.Nil)
}

// UpdateRule handles PUT /parent-work/rules/{id}.
// @Summary Elternstundenregelwerk bearbeiten
// @Tags parent-work
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Regelwerk"
// @Param request body parentWorkRuleRequest true "Regelwerk"
// @Success 200 {object} domain.ParentWorkRule
// @Router /parent-work/rules/{id} [put]
func (h *ParentWorkHandler) UpdateRule(w http.ResponseWriter, r *http.Request) {
	id, ok := parentWorkID(w, r, "id")
	if ok {
		h.saveRule(w, r, id)
	}
}

func (h *ParentWorkHandler) saveRule(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	var req parentWorkRuleRequest
	if !decodeParentWork(w, r, &req) {
		return
	}
	from, err := parentWorkDate(req.ValidFrom)
	if err != nil {
		response.BadRequest(w, "Ungültiges Gültig-ab-Datum")
		return
	}
	v, err := h.svc.SaveRule(r.Context(), id, domain.ParentWorkRule{ValidFrom: from,
		HoursPerChildMinutes: req.HoursPerChildMinutes, MissingHourRateCents: req.MissingHourRateCents,
		MaxCarryOverMinutes: req.MaxCarryOverMinutes})
	if err != nil {
		parentWorkError(w, err)
		return
	}
	if id == uuid.Nil {
		response.Created(w, v)
	} else {
		response.Success(w, v)
	}
}
