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
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/request"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/util"
)

type ParentAccountHandler struct{ svc *service.ParentAccountService }

func NewParentAccountHandler(accounts *repository.ParentAccountRepository,
	work *service.ParentWorkService) *ParentAccountHandler {
	return &ParentAccountHandler{svc: service.NewParentAccountService(accounts, work)}
}
func parentAccountError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrNotFound):
		response.NotFound(w, "Nicht gefunden")
	case errors.Is(err, service.ErrConflict):
		response.Conflict(w, strings.TrimPrefix(err.Error(), "conflict: "))
	case errors.Is(err, service.ErrInvalidInput):
		response.BadRequest(w, strings.TrimPrefix(err.Error(), "invalid input: "))
	default:
		response.InternalError(w, "Anfrage konnte nicht verarbeitet werden")
	}
}
func ownID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) { return currentUserID(w, r) }
func pathID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Ungültige ID")
		return uuid.Nil, false
	}
	return id, true
}
func ownYear(w http.ResponseWriter, r *http.Request, defaultYear int) (int, bool) {
	raw := r.URL.Query().Get("year")
	if raw == "" {
		return defaultYear, true
	}
	year, err := strconv.Atoi(raw)
	if err != nil || year < 1900 || year > 9998 {
		response.BadRequest(w, "Ungültiges Jahr")
		return 0, false
	}
	return year, true
}

type ownParent struct {
	ID         uuid.UUID `json:"id"`
	FirstName  string    `json:"firstName"`
	LastName   string    `json:"lastName"`
	Email      *string   `json:"email" binding:"optional"`
	Phone      *string   `json:"phone" binding:"optional"`
	Street     *string   `json:"street" binding:"optional"`
	StreetNo   *string   `json:"streetNo" binding:"optional"`
	PostalCode *string   `json:"postalCode" binding:"optional"`
	City       *string   `json:"city" binding:"optional"`
}
type ownHousehold struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}
type ownChild struct {
	ID           uuid.UUID  `json:"id"`
	FirstName    string     `json:"firstName"`
	LastName     string     `json:"lastName"`
	BirthDate    time.Time  `json:"birthDate"`
	EntryDate    time.Time  `json:"entryDate"`
	ExitDate     *time.Time `json:"exitDate" binding:"optional"`
	CareHours    *int       `json:"careHours" binding:"optional"`
	MemberNumber string     `json:"memberNumber"`
	Street       *string    `json:"street" binding:"optional"`
	StreetNo     *string    `json:"streetNo" binding:"optional"`
	PostalCode   *string    `json:"postalCode" binding:"optional"`
	City         *string    `json:"city" binding:"optional"`
	LegalHours   *int       `json:"legalHours" binding:"optional"`
}
type ownOverview struct {
	Parent       ownParent     `json:"parent"`
	Household    *ownHousehold `json:"household" binding:"optional"`
	OtherParents []ownParent   `json:"otherParents"`
	Children     []ownChild    `json:"children"`
}
type ownFees struct {
	Items     []repository.ParentFeeRow `json:"items"`
	OpenTotal float64                   `json:"openTotal"`
	PaidTotal float64                   `json:"paidTotal"`
}
type ownWorkEntry struct {
	ID              uuid.UUID `json:"id"`
	WorkDate        time.Time `json:"workDate"`
	DurationMinutes int       `json:"durationMinutes"`
	Occasion        string    `json:"occasion"`
	MemberName      *string   `json:"memberName" binding:"optional"`
	ChildName       *string   `json:"childName" binding:"optional"`
	Status          string    `json:"status"`
	Source          string    `json:"source"`
	RejectReason    *string   `json:"rejectReason" binding:"optional"`
	VoidReason      *string   `json:"voidReason" binding:"optional"`
}
type ownWork struct {
	RequiredMinutes    int            `json:"requiredMinutes"`
	DoneMinutes        int            `json:"doneMinutes"`
	OpenMinutes        int            `json:"openMinutes"`
	CarryInMinutes     int            `json:"carryInMinutes"`
	CarryOutMinutes    int            `json:"carryOutMinutes"`
	MissingAmountCents int            `json:"missingAmountCents"`
	Exempt             bool           `json:"exempt"`
	ExemptText         *string        `json:"exemptText" binding:"optional"`
	Entries            []ownWorkEntry `json:"entries"`
}
type ownWorkRequest struct {
	WorkDate        string  `json:"workDate"`
	DurationMinutes int     `json:"durationMinutes"`
	Occasion        string  `json:"occasion"`
	MemberName      *string `json:"memberName" binding:"optional"`
	ChildName       *string `json:"childName" binding:"optional"`
}
type ownContactRequest struct {
	Email      *string `json:"email" binding:"optional"`
	Phone      *string `json:"phone" binding:"optional"`
	Street     *string `json:"street" binding:"optional"`
	StreetNo   *string `json:"streetNo" binding:"optional"`
	PostalCode *string `json:"postalCode" binding:"optional"`
	City       *string `json:"city" binding:"optional"`
}

func ownParentFrom(p domain.Parent) ownParent {
	return ownParent{p.ID, p.FirstName, p.LastName, p.Email, p.Phone, p.Street, p.StreetNo, p.PostalCode,
		p.City}
}
func ownChildFrom(v domain.Child) ownChild {
	return ownChild{v.ID, v.FirstName, v.LastName, v.BirthDate, v.EntryDate, v.ExitDate, v.CareHours,
		v.MemberNumber, v.Street, v.StreetNo, v.PostalCode, v.City, v.LegalHours}
}
func ownWorkEntryFrom(v *domain.ParentWorkEntry) ownWorkEntry {
	return ownWorkEntry{v.ID, v.WorkDate, v.DurationMinutes, v.Occasion, v.MemberName, v.ChildName,
		v.Status, v.Source, v.RejectReason, v.VoidReason}
}

// Me handles GET /me.
// @Summary Eigene Familiendaten
// @Tags Parent account
// @Security BearerAuth
// @Success 200 {object} ownOverview
// @Router /me [get]
func (h *ParentAccountHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := ownID(w, r)
	if !ok {
		return
	}
	account, others, children, err := h.svc.Overview(r.Context(), userID)
	if err != nil {
		parentAccountError(w, err)
		return
	}
	out := ownOverview{Parent: ownParentFrom(account.Parent), OtherParents: []ownParent{},
		Children: []ownChild{}}
	if account.Household != nil {
		out.Household = &ownHousehold{account.Household.ID, account.Household.Name}
	}
	for _, p := range others {
		out.OtherParents = append(out.OtherParents, ownParentFrom(p))
	}
	for _, c := range children {
		out.Children = append(out.Children, ownChildFrom(c))
	}
	response.Success(w, out)
}

// Fees handles GET /me/fees.
// @Summary Eigene Beiträge
// @Tags Parent account
// @Security BearerAuth
// @Param year query int false "Kalenderjahr"
// @Success 200 {object} ownFees
// @Router /me/fees [get]
func (h *ParentAccountHandler) Fees(w http.ResponseWriter, r *http.Request) {
	id, ok := ownID(w, r)
	if !ok {
		return
	}
	year, ok := ownYear(w, r, util.Now().Year())
	if !ok {
		return
	}
	rows, open, paid, err := h.svc.Fees(r.Context(), id, year)
	if err != nil {
		parentAccountError(w, err)
		return
	}
	response.Success(w, ownFees{rows, open, paid})
}

// ParentWork handles GET /me/parent-work.
// @Summary Eigene Elternstunden
// @Tags Parent account
// @Security BearerAuth
// @Param year query int false "Kita-Jahr"
// @Success 200 {object} ownWork
// @Router /me/parent-work [get]
func (h *ParentAccountHandler) ParentWork(w http.ResponseWriter, r *http.Request) {
	id, ok := ownID(w, r)
	if !ok {
		return
	}
	year, ok := ownYear(w, r, domain.ParentWorkKitaYear(util.Today()))
	if !ok {
		return
	}
	detail, err := h.svc.Work(r.Context(), id, year)
	if err != nil {
		parentAccountError(w, err)
		return
	}
	out := ownWork{detail.RequiredMinutes, detail.DoneMinutes, detail.OpenMinutes, detail.CarryInMinutes,
		detail.CarryOutMinutes, detail.MissingAmountCents, detail.ExemptReason != nil, nil,
		[]ownWorkEntry{}}
	if out.Exempt {
		label := "Befreit"
		out.ExemptText = &label
	}
	for i := range detail.Entries {
		out.Entries = append(out.Entries, ownWorkEntryFrom(&detail.Entries[i]))
	}
	response.Success(w, out)
}

// SubmitEntry handles POST /me/parent-work/entries.
// @Summary Elternstunden melden
// @Tags Parent account
// @Security BearerAuth
// @Param request body ownWorkRequest true "Meldung"
// @Success 201 {object} ownWorkEntry
// @Router /me/parent-work/entries [post]
func (h *ParentAccountHandler) SubmitEntry(w http.ResponseWriter, r *http.Request) {
	id, ok := ownID(w, r)
	if !ok {
		return
	}
	var req ownWorkRequest
	if request.DecodeJSON(r, &req) != nil {
		response.BadRequest(w, "Ungültige Anfrage")
		return
	}
	date, err := time.Parse("2006-01-02", req.WorkDate)
	if err != nil {
		response.BadRequest(w, "Ungültiges Arbeitsdatum")
		return
	}
	v, err := h.svc.Submit(r.Context(), id, domain.ParentWorkEntry{WorkDate: date,
		DurationMinutes: req.DurationMinutes, Occasion: req.Occasion, MemberName: req.MemberName,
		ChildName: req.ChildName})
	if err != nil {
		parentAccountError(w, err)
		return
	}
	response.Created(w, ownWorkEntryFrom(v))
}

// WithdrawEntry handles POST /me/parent-work/entries/{id}/withdraw.
// @Summary Eigene Meldung zurückziehen
// @Tags Parent account
// @Security BearerAuth
// @Param id path string true "Eintrag"
// @Success 200 {object} ownWorkEntry
// @Router /me/parent-work/entries/{id}/withdraw [post]
func (h *ParentAccountHandler) WithdrawEntry(w http.ResponseWriter, r *http.Request) {
	userID, ok := ownID(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	v, err := h.svc.Withdraw(r.Context(), userID, id)
	if err != nil {
		parentAccountError(w, err)
		return
	}
	response.Success(w, ownWorkEntryFrom(v))
}

// Contact handles PUT /me/contact.
// @Summary Eigene Kontaktdaten ändern
// @Tags Parent account
// @Security BearerAuth
// @Param request body ownContactRequest true "Kontaktdaten"
// @Success 200 {object} ownParent
// @Router /me/contact [put]
func (h *ParentAccountHandler) Contact(w http.ResponseWriter, r *http.Request) {
	id, ok := ownID(w, r)
	if !ok {
		return
	}
	values, ok := contactValues(w, r)
	if !ok {
		return
	}
	p, err := h.svc.Contact(r.Context(), id, values)
	if err != nil {
		parentAccountError(w, err)
		return
	}
	response.Success(w, ownParentFrom(*p))
}

// ParentContact handles PUT /me/parents/{id}/contact.
// @Summary Kontaktdaten eines Elternteils der eigenen Familie ändern
// @Tags Parent account
// @Security BearerAuth
// @Param id path string true "Elternteil"
// @Param request body ownContactRequest true "Kontaktdaten"
// @Success 200 {object} ownParent
// @Router /me/parents/{id}/contact [put]
func (h *ParentAccountHandler) ParentContact(w http.ResponseWriter, r *http.Request) {
	userID, ok := ownID(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	values, ok := contactValues(w, r)
	if !ok {
		return
	}
	p, err := h.svc.ParentContact(r.Context(), userID, id, values)
	if err != nil {
		parentAccountError(w, err)
		return
	}
	response.Success(w, ownParentFrom(*p))
}

// contactValues keeps omitted fields apart from explicit nulls.
func contactValues(w http.ResponseWriter, r *http.Request) (map[string]*string, bool) {
	var raw map[string]json.RawMessage
	if request.DecodeJSON(r, &raw) != nil {
		response.BadRequest(w, "Ungültige Anfrage")
		return nil, false
	}
	values := map[string]*string{}
	for key, value := range raw {
		var decoded *string
		if json.Unmarshal(value, &decoded) != nil {
			response.BadRequest(w, "Ungültiges Kontaktfeld")
			return nil, false
		}
		values[key] = decoded
	}
	return values, true
}

// UpdateChild handles PUT /me/children/{id}.
// @Summary Eigene Kinderdaten ändern
// @Tags Parent account
// @Security BearerAuth
// @Param id path string true "Kind"
// @Param request body service.OwnChildInput true "Kinderdaten"
// @Success 200 {object} ownChild
// @Router /me/children/{id} [put]
func (h *ParentAccountHandler) UpdateChild(w http.ResponseWriter, r *http.Request) {
	userID, ok := ownID(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in service.OwnChildInput
	if request.DecodeJSON(r, &in) != nil {
		response.BadRequest(w, "Ungültige Anfrage")
		return
	}
	v, err := h.svc.UpdateChild(r.Context(), userID, id, in)
	if err != nil {
		parentAccountError(w, err)
		return
	}
	response.Success(w, ownChildFrom(*v))
}

// CreateReport handles POST /me/reports.
// @Summary Fehler melden
// @Tags Parent account
// @Security BearerAuth
// @Param request body service.ReportInput true "Meldung"
// @Success 201 {object} repository.ParentReport
// @Router /me/reports [post]
func (h *ParentAccountHandler) CreateReport(w http.ResponseWriter, r *http.Request) {
	id, ok := ownID(w, r)
	if !ok {
		return
	}
	var in service.ReportInput
	if request.DecodeJSON(r, &in) != nil {
		response.BadRequest(w, "Ungültige Anfrage")
		return
	}
	v, err := h.svc.CreateReport(r.Context(), id, in)
	if err != nil {
		parentAccountError(w, err)
		return
	}
	response.Created(w, v)
}

// OwnReports handles GET /me/reports.
// @Summary Eigene Fehlermeldungen
// @Tags Parent account
// @Security BearerAuth
// @Success 200 {array} repository.ParentReport
// @Router /me/reports [get]
func (h *ParentAccountHandler) OwnReports(w http.ResponseWriter, r *http.Request) {
	id, ok := ownID(w, r)
	if !ok {
		return
	}
	rows, err := h.svc.OwnReports(r.Context(), id)
	if err != nil {
		parentAccountError(w, err)
		return
	}
	response.Success(w, rows)
}

// StaffReports handles GET /parent-reports.
// @Summary Fehlermeldungen der Eltern
// @Tags Parent account
// @Security BearerAuth
// @Param status query string false "OPEN, DONE oder ALL"
// @Success 200 {array} repository.ParentReport
// @Router /parent-reports [get]
func (h *ParentAccountHandler) StaffReports(w http.ResponseWriter, r *http.Request) {
	rows, err := h.svc.StaffReports(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		parentAccountError(w, err)
		return
	}
	response.Success(w, rows)
}

type resolveReportRequest struct {
	Response string `json:"response" binding:"optional"`
}

// ResolveReport handles POST /parent-reports/{id}/resolve.
// @Summary Fehlermeldung erledigen, optional mit Antwort an die Eltern
// @Tags Parent account
// @Security BearerAuth
// @Param id path string true "Meldung"
// @Param request body resolveReportRequest false "Antwort"
// @Success 200 {object} repository.ParentReport
// @Router /parent-reports/{id}/resolve [post]
func (h *ParentAccountHandler) ResolveReport(w http.ResponseWriter, r *http.Request) {
	userID, ok := ownID(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var req resolveReportRequest
	if err := request.DecodeJSON(r, &req); err != nil && !errors.Is(err, io.EOF) {
		response.BadRequest(w, "Ungültige Anfrage")
		return
	}
	v, err := h.svc.ResolveReport(r.Context(), id, userID, req.Response)
	if err != nil {
		parentAccountError(w, err)
		return
	}
	response.Success(w, v)
}

// Activity handles GET /activity.
// @Summary Letzte Eltern-Aktivitäten
// @Tags Parent account
// @Security BearerAuth
// @Param limit query int false "Anzahl (1..100)"
// @Success 200 {array} repository.Activity
// @Router /activity [get]
func (h *ParentAccountHandler) Activity(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		var err error
		limit, err = strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > 100 {
			response.BadRequest(w, "Limit muss zwischen 1 und 100 liegen")
			return
		}
	}
	rows, err := h.svc.Activity(r.Context(), limit)
	if err != nil {
		parentAccountError(w, err)
		return
	}
	response.Success(w, rows)
}

// ParentChanges handles GET /parents/{id}/changes.
// @Summary Datenänderungen eines Elternteils
// @Tags Parent account
// @Security BearerAuth
// @Param id path string true "Elternteil"
// @Success 200 {array} repository.DataChange
// @Router /parents/{id}/changes [get]
func (h *ParentAccountHandler) ParentChanges(w http.ResponseWriter, r *http.Request) {
	h.changes(w, r, "PARENT")
}

// ChildChanges handles GET /children/{id}/changes.
// @Summary Datenänderungen eines Kindes
// @Tags Parent account
// @Security BearerAuth
// @Param id path string true "Kind"
// @Success 200 {array} repository.DataChange
// @Router /children/{id}/changes [get]
func (h *ParentAccountHandler) ChildChanges(w http.ResponseWriter, r *http.Request) {
	h.changes(w, r, "CHILD")
}
func (h *ParentAccountHandler) changes(w http.ResponseWriter, r *http.Request, kind string) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	rows, err := h.svc.Changes(r.Context(), kind, id)
	if err != nil {
		parentAccountError(w, err)
		return
	}
	response.Success(w, rows)
}
