package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/request"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

// ParentHandler handles parent-related requests.
type ParentHandler struct {
	parentService *service.ParentService
}

// NewParentHandler creates a new parent handler.
func NewParentHandler(parentService *service.ParentService) *ParentHandler {
	return &ParentHandler{parentService: parentService}
}

// ParentDetailResponse represents a parent in API responses.
// @Description Parent information with relationships
type ParentDetailResponse struct {
	ID                    string      `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	HouseholdID           *string     `json:"householdId,omitempty" example:"550e8400-e29b-41d4-a716-446655440001"`
	MemberID              *string     `json:"memberId,omitempty" example:"550e8400-e29b-41d4-a716-446655440002"`
	FirstName             string      `json:"firstName" example:"Thomas"`
	LastName              string      `json:"lastName" example:"Müller"`
	BirthDate             *string     `json:"birthDate,omitempty" example:"1985-03-15"`
	Email                 *string     `json:"email,omitempty" example:"thomas.mueller@example.com"`
	Phone                 *string     `json:"phone,omitempty" example:"+49 331 12345"`
	Street                *string     `json:"street,omitempty" example:"Hauptstraße"`
	StreetNo              *string     `json:"streetNo,omitempty" example:"42"`
	PostalCode            *string     `json:"postalCode,omitempty" example:"14467"`
	City                  *string     `json:"city,omitempty" example:"Potsdam"`
	AnnualHouseholdIncome *float64    `json:"annualHouseholdIncome,omitempty" example:"65000.00"`
	IncomeStatus          *string     `json:"incomeStatus,omitempty" example:"PROVIDED" enums:"PROVIDED,MAX_ACCEPTED,PENDING,NOT_REQUIRED,HISTORIC,FOSTER_FAMILY"`
	CreatedAt             string      `json:"createdAt" example:"2023-01-15T10:00:00Z"`
	UpdatedAt             string      `json:"updatedAt" example:"2023-01-15T10:00:00Z"`
	Children              interface{} `json:"children,omitempty"`
	Household             interface{} `json:"household,omitempty"`
	Member                interface{} `json:"member,omitempty"`
} //@name Parent

// ParentListResponse represents a paginated list of parents.
// @Description Paginated list of parents
type ParentListResponse struct {
	Data       []ParentDetailResponse `json:"data"`
	Total      int64                  `json:"total" example:"50"`
	Page       int                    `json:"page" example:"1"`
	PerPage    int                    `json:"perPage" example:"20"`
	TotalPages int                    `json:"totalPages" example:"3"`
} //@name ParentList

// CreateParentRequest represents a request to create a parent.
// @Description Request body for creating a new parent
type CreateParentRequest struct {
	FirstName             string   `json:"firstName" example:"Thomas"`
	LastName              string   `json:"lastName" example:"Müller"`
	BirthDate             *string  `json:"birthDate,omitempty" example:"1985-03-15"`
	Email                 *string  `json:"email,omitempty" example:"thomas.mueller@example.com"`
	Phone                 *string  `json:"phone,omitempty" example:"+49 331 12345"`
	Street                *string  `json:"street,omitempty" example:"Hauptstraße"`
	StreetNo              *string  `json:"streetNo,omitempty" example:"42"`
	PostalCode            *string  `json:"postalCode,omitempty" example:"14467"`
	City                  *string  `json:"city,omitempty" example:"Potsdam"`
	AnnualHouseholdIncome *float64 `json:"annualHouseholdIncome,omitempty" example:"65000.00"`
	IncomeStatus          *string  `json:"incomeStatus,omitempty" example:"PROVIDED" enums:"PROVIDED,MAX_ACCEPTED,PENDING,NOT_REQUIRED,HISTORIC,FOSTER_FAMILY"`
} //@name CreateParentRequest

// List returns all parents with pagination
// @Summary List all parents
// @Description Get a paginated list of parents with optional search and sorting
// @Tags Parents
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param perPage query int false "Items per page" default(20)
// @Param search query string false "Search by name or email"
// @Param sortBy query string false "Sort field (name, email)" default(name)
// @Param sortDir query string false "Sort direction (asc, desc)" default(asc)
// @Success 200 {object} ParentListResponse "Paginated list of parents"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /parents [get]
func (h *ParentHandler) List(w http.ResponseWriter, r *http.Request) {
	pagination := request.GetPagination(r)
	search := request.GetQueryString(r, "search", "")
	sortBy := request.GetQueryString(r, "sortBy", "name")
	sortDir := request.GetQueryString(r, "sortDir", "asc")

	parents, total, err := h.parentService.List(r.Context(), search, sortBy, sortDir, pagination.Offset, pagination.PerPage)
	if err != nil {
		response.InternalError(w, "failed to list parents")
		return
	}

	response.Paginated(w, parents, total, pagination.Page, pagination.PerPage)
}

// Create creates a new parent
// @Summary Create new parent
// @Description Register a new parent in the system
// @Tags Parents
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateParentRequest true "Parent data"
// @Success 201 {object} ParentDetailResponse "Parent created successfully"
// @Failure 400 {object} response.ErrorBody "Invalid request body"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /parents [post]
func (h *ParentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateParentRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if req.FirstName == "" || req.LastName == "" {
		response.BadRequest(w, "firstName and lastName are required")
		return
	}

	parent, err := h.parentService.Create(r.Context(), service.CreateParentInput{
		FirstName:             req.FirstName,
		LastName:              req.LastName,
		BirthDate:             req.BirthDate,
		Email:                 req.Email,
		Phone:                 req.Phone,
		Street:                req.Street,
		StreetNo:              req.StreetNo,
		PostalCode:            req.PostalCode,
		City:                  req.City,
		AnnualHouseholdIncome: req.AnnualHouseholdIncome,
		IncomeStatus:          req.IncomeStatus,
	})
	if err != nil {
		response.InternalError(w, "failed to create parent")
		return
	}

	response.Created(w, parent)
}

// Get returns a parent by ID
// @Summary Get parent by ID
// @Description Retrieve detailed information about a specific parent
// @Tags Parents
// @Produce json
// @Security BearerAuth
// @Param id path string true "Parent ID (UUID)"
// @Success 200 {object} ParentDetailResponse "Parent found"
// @Failure 400 {object} response.ErrorBody "Invalid parent ID"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Parent not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /parents/{id} [get]
func (h *ParentHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid parent ID")
		return
	}

	parent, err := h.parentService.GetByID(r.Context(), id)
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "parent not found")
			return
		}
		response.InternalError(w, "failed to get parent")
		return
	}

	response.Success(w, parent)
}

// UpdateParentRequest represents a request to update a parent.
// @Description Request body for updating a parent
type UpdateParentRequest struct {
	FirstName             *string  `json:"firstName,omitempty" example:"Thomas"`
	LastName              *string  `json:"lastName,omitempty" example:"Müller"`
	BirthDate             *string  `json:"birthDate,omitempty" example:"1985-03-15"`
	Email                 *string  `json:"email,omitempty" example:"thomas.mueller@example.com"`
	Phone                 *string  `json:"phone,omitempty" example:"+49 331 12345"`
	Street                *string  `json:"street,omitempty" example:"Hauptstraße"`
	StreetNo              *string  `json:"streetNo,omitempty" example:"42"`
	PostalCode            *string  `json:"postalCode,omitempty" example:"14467"`
	City                  *string  `json:"city,omitempty" example:"Potsdam"`
	AnnualHouseholdIncome *float64 `json:"annualHouseholdIncome,omitempty" example:"65000.00"`
	IncomeStatus          *string  `json:"incomeStatus,omitempty" example:"PROVIDED" enums:"PROVIDED,MAX_ACCEPTED,PENDING,NOT_REQUIRED,HISTORIC,FOSTER_FAMILY"`
} //@name UpdateParentRequest

// Update updates a parent
// @Summary Update parent
// @Description Update parent information
// @Tags Parents
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Parent ID (UUID)"
// @Param request body UpdateParentRequest true "Updated parent data"
// @Success 200 {object} ParentDetailResponse "Parent updated"
// @Failure 400 {object} response.ErrorBody "Invalid request"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Parent not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /parents/{id} [put]
func (h *ParentHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid parent ID")
		return
	}

	var req UpdateParentRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	parent, err := h.parentService.Update(r.Context(), id, service.UpdateParentInput{
		FirstName:             req.FirstName,
		LastName:              req.LastName,
		BirthDate:             req.BirthDate,
		Email:                 req.Email,
		Phone:                 req.Phone,
		Street:                req.Street,
		StreetNo:              req.StreetNo,
		PostalCode:            req.PostalCode,
		City:                  req.City,
		AnnualHouseholdIncome: req.AnnualHouseholdIncome,
		IncomeStatus:          req.IncomeStatus,
	})
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "parent not found")
			return
		}
		response.InternalError(w, "failed to update parent")
		return
	}

	response.Success(w, parent)
}

// Delete deletes a parent
// @Summary Delete parent
// @Description Remove a parent from the system
// @Tags Parents
// @Security BearerAuth
// @Param id path string true "Parent ID (UUID)"
// @Success 204 "Parent deleted"
// @Failure 400 {object} response.ErrorBody "Invalid parent ID"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Parent not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /parents/{id} [delete]
func (h *ParentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid parent ID")
		return
	}

	if err := h.parentService.Delete(r.Context(), id); err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "parent not found")
			return
		}
		response.InternalError(w, "failed to delete parent")
		return
	}

	response.NoContent(w)
}

// CreateMemberFromParentRequest represents a request to create a member from a parent.
// @Description Request body for creating a member from a parent
type CreateMemberFromParentRequest struct {
	MembershipStart string `json:"membershipStart" example:"2023-08-01"`
} //@name CreateMemberFromParentRequest

// CreateMember creates a new member from parent data
// @Summary Create member for parent
// @Description Create a new club member from the parent's data and link them
// @Tags Parents
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Parent ID (UUID)"
// @Param request body CreateMemberFromParentRequest false "Membership start date (defaults to oldest child's entry date)"
// @Success 201 {object} ParentDetailResponse "Member created and linked"
// @Failure 400 {object} response.ErrorBody "Invalid request"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Parent not found"
// @Failure 409 {object} response.ErrorBody "Parent already linked to a member"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /parents/{id}/member [post]
func (h *ParentHandler) CreateMember(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid parent ID")
		return
	}

	var req CreateMemberFromParentRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	// Parse membership start date - if empty, service will use oldest child's entry date
	var membershipStart time.Time
	if req.MembershipStart != "" {
		membershipStart, err = time.Parse("2006-01-02", req.MembershipStart)
		if err != nil {
			response.BadRequest(w, "invalid membershipStart date format (expected YYYY-MM-DD)")
			return
		}
	}
	// If empty, membershipStart stays zero and service will auto-detect from children

	parent, err := h.parentService.CreateMemberFromParent(r.Context(), id, membershipStart)
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "parent not found")
			return
		}
		if err == service.ErrConflict {
			response.Conflict(w, "parent is already linked to a member")
			return
		}
		response.InternalError(w, "failed to create member")
		return
	}

	response.Created(w, parent)
}

// UnlinkMember removes the member link from a parent
// @Summary Unlink member from parent
// @Description Remove the member association from a parent (does not delete the member)
// @Tags Parents
// @Produce json
// @Security BearerAuth
// @Param id path string true "Parent ID (UUID)"
// @Success 200 {object} ParentDetailResponse "Member unlinked"
// @Failure 400 {object} response.ErrorBody "Invalid parent ID"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Parent not found or not linked to a member"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /parents/{id}/member [delete]
func (h *ParentHandler) UnlinkMember(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid parent ID")
		return
	}

	parent, err := h.parentService.UnlinkMember(r.Context(), id)
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "parent not found or not linked to a member")
			return
		}
		response.InternalError(w, "failed to unlink member")
		return
	}

	response.Success(w, parent)
}
