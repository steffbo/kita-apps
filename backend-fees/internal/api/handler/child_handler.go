package handler

import (
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/request"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
	"net/http"
)

// ChildHandler handles child-related requests.
type ChildHandler struct {
	childService    *service.ChildService
	feeService      *service.FeeService
	coverageService *service.CoverageService
}

// NewChildHandler creates a new child handler.
func NewChildHandler(childService *service.ChildService, feeService *service.FeeService, coverageService *service.CoverageService) *ChildHandler {
	return &ChildHandler{
		childService:    childService,
		feeService:      feeService,
		coverageService: coverageService,
	}
}

// ChildListResponse represents a paginated list of children.
// @Description Paginated list of children as returned by the child endpoints
type ChildListResponse struct {
	Data       []domain.Child `json:"data"`
	Total      int64          `json:"total" example:"100"`
	Page       int            `json:"page" example:"1"`
	PerPage    int            `json:"perPage" example:"20"`
	TotalPages int            `json:"totalPages" example:"5"`
}

// NextMemberNumberResponse represents the next available member number.
// @Description Next available member number
type NextMemberNumberResponse struct {
	MemberNumber string `json:"memberNumber" example:"12002"`
}

// CreateChildRequest represents a request to create a child.
// @Description Request body for creating a new child
type CreateChildRequest struct {
	MemberNumber    string  `json:"memberNumber" example:"K-2024-001"`
	FirstName       string  `json:"firstName" example:"Emma"`
	LastName        string  `json:"lastName" example:"Müller"`
	BirthDate       string  `json:"birthDate" example:"2020-06-15"`
	EntryDate       string  `json:"entryDate" example:"2023-08-01"`
	ExitDate        *string `json:"exitDate,omitempty" example:"2026-07-31" binding:"optional"`
	Street          *string `json:"street,omitempty" example:"Hauptstraße" binding:"optional"`
	StreetNo        *string `json:"streetNo,omitempty" example:"42" binding:"optional"`
	PostalCode      *string `json:"postalCode,omitempty" example:"14467" binding:"optional"`
	City            *string `json:"city,omitempty" example:"Potsdam" binding:"optional"`
	LegalHours      *int    `json:"legalHours,omitempty" example:"35" binding:"optional"`
	LegalHoursUntil *string `json:"legalHoursUntil,omitempty" example:"2024-12-31" binding:"optional"`
	CareHours       *int    `json:"careHours,omitempty" example:"40" binding:"optional"`
}

// List returns all children with pagination and filtering
// @Summary List all children
// @Description Get a paginated list of children with optional filters
// @Tags Children
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param perPage query int false "Items per page" default(20)
// @Param active query bool false "Filter by active status"
// @Param u3Only query bool false "Filter for children under 3"
// @Param hasWarnings query bool false "Filter for children with warnings"
// @Param hasOpenFees query bool false "Filter for children with open fees"
// @Param search query string false "Search by name or member number"
// @Param sortBy query string false "Sort field (name, birthDate, entryDate)" default(name)
// @Param sortDir query string false "Sort direction (asc, desc)" default(asc)
// @Success 200 {object} ChildListResponse "Paginated list of children"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /children [get]
func (h *ChildHandler) List(w http.ResponseWriter, r *http.Request) {
	pagination := request.GetPagination(r)
	activeOnly := request.GetQueryBool(r, "active")
	u3Only := request.GetQueryBool(r, "u3Only")
	hasWarnings := request.GetQueryBool(r, "hasWarnings")
	hasOpenFees := request.GetQueryBool(r, "hasOpenFees")
	search := request.GetQueryString(r, "search", "")
	sortBy := request.GetQueryString(r, "sortBy", "name")
	sortDir := request.GetQueryString(r, "sortDir", "asc")

	filter := service.ChildFilter{
		ActiveOnly:  activeOnly != nil && *activeOnly,
		U3Only:      u3Only != nil && *u3Only,
		HasWarnings: hasWarnings != nil && *hasWarnings,
		HasOpenFees: hasOpenFees != nil && *hasOpenFees,
		Search:      search,
		SortBy:      sortBy,
		SortDir:     sortDir,
	}

	children, total, err := h.childService.List(r.Context(), filter, pagination.Offset, pagination.PerPage)
	if err != nil {
		response.InternalError(w, "failed to list children")
		return
	}

	response.Paginated(w, children, total, pagination.Page, pagination.PerPage)
}

// NextMemberNumber returns the next available numeric member number.
// @Summary Get next available member number
// @Description Returns the next available numeric member number for a new child
// @Tags Children
// @Produce json
// @Security BearerAuth
// @Success 200 {object} NextMemberNumberResponse "Next member number"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /children/next-member-number [get]
func (h *ChildHandler) NextMemberNumber(w http.ResponseWriter, r *http.Request) {
	memberNumber, err := h.childService.GetNextMemberNumber(r.Context())
	if err != nil {
		response.InternalError(w, "failed to get next member number")
		return
	}

	response.Success(w, NextMemberNumberResponse{MemberNumber: memberNumber})
}

// Create creates a new child
// @Summary Create new child
// @Description Register a new child in the system
// @Tags Children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateChildRequest true "Child data"
// @Success 201 {object} domain.Child "Child created successfully"
// @Failure 400 {object} response.ErrorBody "Invalid request body"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 409 {object} response.ErrorBody "Member number already exists"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /children [post]
func (h *ChildHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateChildRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if req.MemberNumber == "" || req.FirstName == "" || req.LastName == "" || req.BirthDate == "" || req.EntryDate == "" {
		response.BadRequest(w, "memberNumber, firstName, lastName, birthDate and entryDate are required")
		return
	}

	child, err := h.childService.Create(r.Context(), service.CreateChildInput{
		MemberNumber:    req.MemberNumber,
		FirstName:       req.FirstName,
		LastName:        req.LastName,
		BirthDate:       req.BirthDate,
		EntryDate:       req.EntryDate,
		ExitDate:        req.ExitDate,
		Street:          req.Street,
		StreetNo:        req.StreetNo,
		PostalCode:      req.PostalCode,
		City:            req.City,
		LegalHours:      req.LegalHours,
		LegalHoursUntil: req.LegalHoursUntil,
		CareHours:       req.CareHours,
	})
	if err != nil {
		if err == service.ErrDuplicateMemberNumber {
			response.Conflict(w, "member number already exists")
			return
		}
		if err == service.ErrInvalidInput {
			response.BadRequest(w, "invalid child data")
			return
		}
		response.InternalError(w, "failed to create child")
		return
	}

	response.Created(w, child)
}

// Get returns a child by ID
// @Summary Get child by ID
// @Description Retrieve detailed information about a specific child
// @Tags Children
// @Produce json
// @Security BearerAuth
// @Param id path string true "Child ID (UUID)"
// @Success 200 {object} domain.Child "Child found"
// @Failure 400 {object} response.ErrorBody "Invalid child ID"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Child not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /children/{id} [get]
func (h *ChildHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	child, err := h.childService.GetByID(r.Context(), id)
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "child not found")
			return
		}
		response.InternalError(w, "failed to get child")
		return
	}

	response.Success(w, child)
}

// UpdateChildRequest represents a request to update a child.
// @Description Request body for updating a child
type UpdateChildRequest struct {
	FirstName       *string `json:"firstName,omitempty" example:"Emma" binding:"optional"`
	LastName        *string `json:"lastName,omitempty" example:"Müller" binding:"optional"`
	BirthDate       *string `json:"birthDate,omitempty" example:"2020-06-15" binding:"optional"`
	EntryDate       *string `json:"entryDate,omitempty" example:"2023-08-01" binding:"optional"`
	ExitDate        *string `json:"exitDate,omitempty" example:"2026-07-31" binding:"optional"`
	Street          *string `json:"street,omitempty" example:"Hauptstraße" binding:"optional"`
	StreetNo        *string `json:"streetNo,omitempty" example:"42" binding:"optional"`
	PostalCode      *string `json:"postalCode,omitempty" example:"14467" binding:"optional"`
	City            *string `json:"city,omitempty" example:"Potsdam" binding:"optional"`
	LegalHours      *int    `json:"legalHours,omitempty" example:"35" binding:"optional"`
	LegalHoursUntil *string `json:"legalHoursUntil,omitempty" example:"2024-12-31" binding:"optional"`
	CareHours       *int    `json:"careHours,omitempty" example:"40" binding:"optional"`
	IsActive        *bool   `json:"isActive,omitempty" example:"true" binding:"optional"`
}

// Update updates a child
// @Summary Update child
// @Description Update child information
// @Tags Children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Child ID (UUID)"
// @Param request body UpdateChildRequest true "Updated child data"
// @Success 200 {object} domain.Child "Child updated"
// @Failure 400 {object} response.ErrorBody "Invalid request"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Child not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /children/{id} [put]
func (h *ChildHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req UpdateChildRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	child, err := h.childService.Update(r.Context(), id, service.UpdateChildInput{
		FirstName:       req.FirstName,
		LastName:        req.LastName,
		BirthDate:       req.BirthDate,
		EntryDate:       req.EntryDate,
		ExitDate:        req.ExitDate,
		Street:          req.Street,
		StreetNo:        req.StreetNo,
		PostalCode:      req.PostalCode,
		City:            req.City,
		LegalHours:      req.LegalHours,
		LegalHoursUntil: req.LegalHoursUntil,
		CareHours:       req.CareHours,
		IsActive:        req.IsActive,
	})
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "child not found")
			return
		}
		if err == service.ErrInvalidInput {
			response.BadRequest(w, "invalid child data")
			return
		}
		response.InternalError(w, "failed to update child")
		return
	}

	response.Success(w, child)
}

// Delete removes a child
// @Summary Delete child
// @Description Permanently delete a child record
// @Tags Children
// @Security BearerAuth
// @Param id path string true "Child ID (UUID)"
// @Success 204 "Child deleted"
// @Failure 400 {object} response.ErrorBody "Invalid child ID"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Child not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /children/{id} [delete]
func (h *ChildHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	if err := h.childService.Delete(r.Context(), id); err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "child not found")
			return
		}
		response.InternalError(w, "failed to delete child")
		return
	}

	response.NoContent(w)
}
