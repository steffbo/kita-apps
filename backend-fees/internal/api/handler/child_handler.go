package handler

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/request"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

// ChildHandler handles child-related requests.
type ChildHandler struct {
	childService    *service.ChildService
	feeService      *service.FeeService
	coverageService *service.CoverageService
	feeRepo         repository.FeeRepository
	matchRepo       repository.MatchRepository
	transactionRepo repository.TransactionRepository
}

// NewChildHandler creates a new child handler.
func NewChildHandler(childService *service.ChildService, feeService *service.FeeService, coverageService *service.CoverageService, feeRepo repository.FeeRepository, matchRepo repository.MatchRepository, transactionRepo repository.TransactionRepository) *ChildHandler {
	return &ChildHandler{
		childService:    childService,
		feeService:      feeService,
		coverageService: coverageService,
		feeRepo:         feeRepo,
		matchRepo:       matchRepo,
		transactionRepo: transactionRepo,
	}
}

// ChildResponse represents a child in API responses.
// @Description Child information
type ChildResponse struct {
	ID              string           `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	HouseholdID     *string          `json:"householdId,omitempty" example:"550e8400-e29b-41d4-a716-446655440001"`
	MemberNumber    string           `json:"memberNumber" example:"K-2024-001"`
	FirstName       string           `json:"firstName" example:"Emma"`
	LastName        string           `json:"lastName" example:"Müller"`
	BirthDate       string           `json:"birthDate" example:"2020-06-15"`
	EntryDate       string           `json:"entryDate" example:"2023-08-01"`
	ExitDate        *string          `json:"exitDate,omitempty" example:"2026-07-31"`
	Street          *string          `json:"street,omitempty" example:"Hauptstraße"`
	StreetNo        *string          `json:"streetNo,omitempty" example:"42"`
	PostalCode      *string          `json:"postalCode,omitempty" example:"14467"`
	City            *string          `json:"city,omitempty" example:"Potsdam"`
	LegalHours      *int             `json:"legalHours,omitempty" example:"35"`
	LegalHoursUntil *string          `json:"legalHoursUntil,omitempty" example:"2024-12-31"`
	CareHours       *int             `json:"careHours,omitempty" example:"40"`
	IsActive        bool             `json:"isActive" example:"true"`
	CreatedAt       string           `json:"createdAt" example:"2023-08-01T10:00:00Z"`
	UpdatedAt       string           `json:"updatedAt" example:"2023-08-01T10:00:00Z"`
	Parents         []ParentResponse `json:"parents,omitempty"`
	Household       interface{}      `json:"household,omitempty"`
} //@name Child

// ChildListResponse represents a paginated list of children.
// @Description Paginated list of children
type ChildListResponse struct {
	Data       []ChildResponse `json:"data"`
	Total      int64           `json:"total" example:"100"`
	Page       int             `json:"page" example:"1"`
	PerPage    int             `json:"perPage" example:"20"`
	TotalPages int             `json:"totalPages" example:"5"`
} //@name ChildList

// NextMemberNumberResponse represents the next available member number.
// @Description Next available member number
type NextMemberNumberResponse struct {
	MemberNumber string `json:"memberNumber" example:"12002"`
} //@name NextMemberNumberResponse

// ParentResponse represents a parent in API responses (summary).
// @Description Parent information (summary)
type ParentResponse struct {
	ID        string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440002"`
	FirstName string  `json:"firstName" example:"Thomas"`
	LastName  string  `json:"lastName" example:"Müller"`
	Email     *string `json:"email,omitempty" example:"thomas.mueller@example.com"`
	Phone     *string `json:"phone,omitempty" example:"+49 331 12345"`
} //@name ParentSummary

// CreateChildRequest represents a request to create a child.
// @Description Request body for creating a new child
type CreateChildRequest struct {
	MemberNumber    string  `json:"memberNumber" example:"K-2024-001"`
	FirstName       string  `json:"firstName" example:"Emma"`
	LastName        string  `json:"lastName" example:"Müller"`
	BirthDate       string  `json:"birthDate" example:"2020-06-15"`
	EntryDate       string  `json:"entryDate" example:"2023-08-01"`
	ExitDate        *string `json:"exitDate,omitempty" example:"2026-07-31"`
	Street          *string `json:"street,omitempty" example:"Hauptstraße"`
	StreetNo        *string `json:"streetNo,omitempty" example:"42"`
	PostalCode      *string `json:"postalCode,omitempty" example:"14467"`
	City            *string `json:"city,omitempty" example:"Potsdam"`
	LegalHours      *int    `json:"legalHours,omitempty" example:"35"`
	LegalHoursUntil *string `json:"legalHoursUntil,omitempty" example:"2024-12-31"`
	CareHours       *int    `json:"careHours,omitempty" example:"40"`
} //@name CreateChildRequest

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
// @Success 201 {object} ChildResponse "Child created successfully"
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
// @Success 200 {object} ChildResponse "Child found"
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
	FirstName       *string `json:"firstName,omitempty" example:"Emma"`
	LastName        *string `json:"lastName,omitempty" example:"Müller"`
	BirthDate       *string `json:"birthDate,omitempty" example:"2020-06-15"`
	EntryDate       *string `json:"entryDate,omitempty" example:"2023-08-01"`
	ExitDate        *string `json:"exitDate,omitempty" example:"2026-07-31"`
	Street          *string `json:"street,omitempty" example:"Hauptstraße"`
	StreetNo        *string `json:"streetNo,omitempty" example:"42"`
	PostalCode      *string `json:"postalCode,omitempty" example:"14467"`
	City            *string `json:"city,omitempty" example:"Potsdam"`
	LegalHours      *int    `json:"legalHours,omitempty" example:"35"`
	LegalHoursUntil *string `json:"legalHoursUntil,omitempty" example:"2024-12-31"`
	CareHours       *int    `json:"careHours,omitempty" example:"40"`
	IsActive        *bool   `json:"isActive,omitempty" example:"true"`
} //@name UpdateChildRequest

// Update updates a child
// @Summary Update child
// @Description Update child information
// @Tags Children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Child ID (UUID)"
// @Param request body UpdateChildRequest true "Updated child data"
// @Success 200 {object} ChildResponse "Child updated"
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
		response.InternalError(w, "failed to update child")
		return
	}

	response.Success(w, child)
}

// Delete deactivates a child
// @Summary Delete child
// @Description Soft-delete (deactivate) a child record
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

	if err := h.childService.Deactivate(r.Context(), id); err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "child not found")
			return
		}
		response.InternalError(w, "failed to deactivate child")
		return
	}

	response.NoContent(w)
}

// LinkParentRequest represents a request to link a parent to a child.
// @Description Request body for linking a parent to a child
type LinkParentRequest struct {
	ParentID  string `json:"parentId" example:"550e8400-e29b-41d4-a716-446655440000"`
	IsPrimary bool   `json:"isPrimary" example:"true"`
} //@name LinkParentRequest

// LinkParent links a parent to a child
// @Summary Link parent to child
// @Description Create a relationship between a parent and a child
// @Tags Children
// @Accept json
// @Security BearerAuth
// @Param id path string true "Child ID (UUID)"
// @Param request body LinkParentRequest true "Parent link data"
// @Success 204 "Parent linked"
// @Failure 400 {object} response.ErrorBody "Invalid request"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Child or parent not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /children/{id}/parents [post]
func (h *ChildHandler) LinkParent(w http.ResponseWriter, r *http.Request) {
	childID, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req LinkParentRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	parentID, err := uuid.Parse(req.ParentID)
	if err != nil {
		response.BadRequest(w, "invalid parent ID")
		return
	}

	if err := h.childService.LinkParent(r.Context(), childID, parentID, req.IsPrimary); err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "child or parent not found")
			return
		}
		response.InternalError(w, "failed to link parent")
		return
	}

	response.NoContent(w)
}

// UnlinkParent removes the link between a parent and a child
// @Summary Unlink parent from child
// @Description Remove the relationship between a parent and a child
// @Tags Children
// @Security BearerAuth
// @Param id path string true "Child ID (UUID)"
// @Param parentId path string true "Parent ID (UUID)"
// @Success 204 "Parent unlinked"
// @Failure 400 {object} response.ErrorBody "Invalid ID"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /children/{id}/parents/{parentId} [delete]
func (h *ChildHandler) UnlinkParent(w http.ResponseWriter, r *http.Request) {
	childID, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	parentID, ok := parseUUIDParam(w, r, "parentId")
	if !ok {
		return
	}

	if err := h.childService.UnlinkParent(r.Context(), childID, parentID); err != nil {
		response.InternalError(w, "failed to unlink parent")
		return
	}

	response.NoContent(w)
}

// LedgerEntryResponse represents a single entry in the payment ledger.
// @Description Ledger entry for a child
type LedgerEntryResponse struct {
	ID          string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Date        string  `json:"date" example:"2024-01-05"`
	Type        string  `json:"type" example:"fee" enums:"fee,payment"`
	Description string  `json:"description" example:"Essensgeld Januar 2024"`
	FeeType     string  `json:"feeType,omitempty" example:"FOOD"`
	Year        int     `json:"year,omitempty" example:"2024"`
	Month       *int    `json:"month,omitempty" example:"1"`
	Debit       float64 `json:"debit" example:"45.40"`
	Credit      float64 `json:"credit" example:"0"`
	Balance     float64 `json:"balance" example:"45.40"`
	IsPaid      bool    `json:"isPaid,omitempty" example:"false"`
	PaidAt      *string `json:"paidAt,omitempty" example:"2024-01-10"`
} //@name LedgerEntry

// LedgerSummaryResponse provides totals for the ledger.
// @Description Summary totals for the ledger
type LedgerSummaryResponse struct {
	TotalFees      float64 `json:"totalFees" example:"500.00"`
	TotalPaid      float64 `json:"totalPaid" example:"400.00"`
	TotalOpen      float64 `json:"totalOpen" example:"100.00"`
	OpenFeesCount  int     `json:"openFeesCount" example:"2"`
	PaidFeesCount  int     `json:"paidFeesCount" example:"8"`
	TotalFeesCount int     `json:"totalFeesCount" example:"10"`
} //@name LedgerSummary

// ChildLedgerResponse represents the complete payment ledger for a child.
// @Description Payment ledger for a child
type ChildLedgerResponse struct {
	ChildID string                `json:"childId" example:"550e8400-e29b-41d4-a716-446655440000"`
	Child   interface{}           `json:"child,omitempty"`
	Entries []LedgerEntryResponse `json:"entries"`
	Summary LedgerSummaryResponse `json:"summary"`
} //@name ChildLedger

// GetLedger returns the payment ledger for a child
// @Summary Get child payment ledger
// @Description Retrieve the payment ledger showing all fees and payments for a child
// @Tags Children
// @Produce json
// @Security BearerAuth
// @Param id path string true "Child ID (UUID)"
// @Param year query int false "Filter by year"
// @Success 200 {object} ChildLedgerResponse "Payment ledger"
// @Failure 400 {object} response.ErrorBody "Invalid child ID"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Child not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /children/{id}/ledger [get]
func (h *ChildHandler) GetLedger(w http.ResponseWriter, r *http.Request) {
	childID, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	year := request.GetQueryIntOptional(r, "year")

	ledger, err := h.feeService.GetChildLedger(r.Context(), childID, year)
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "child not found")
			return
		}
		response.InternalError(w, "failed to get ledger")
		return
	}

	response.Success(w, ledger)
}

// FeeCoverageResponse represents monthly fee coverage.
// @Description Monthly fee coverage with transaction details
type FeeCoverageResponse struct {
	Year          int                          `json:"year" example:"2024"`
	Month         int                          `json:"month" example:"3"`
	ExpectedTotal float64                      `json:"expectedTotal" example:"110.00"`
	ReceivedTotal float64                      `json:"receivedTotal" example:"110.00"`
	Balance       float64                      `json:"balance" example:"0.00"`
	Status        string                       `json:"status" example:"COVERED" enums:"UNPAID,PARTIAL,COVERED,OVERPAID"`
	Transactions  []CoveredTransactionResponse `json:"transactions"`
}

// CoveredTransactionResponse represents a transaction covering a fee period.
type CoveredTransactionResponse struct {
	TransactionID  string  `json:"transactionId" example:"550e8400-e29b-41d4-a716-446655440000"`
	Amount         float64 `json:"amount" example:"66.00"`
	BookingDate    string  `json:"bookingDate" example:"2024-03-05"`
	Description    *string `json:"description,omitempty" example:"Platzgeld März"`
	IsForThisMonth bool    `json:"isForThisMonth" example:"true"`
}

// GetTimeline returns a month-by-month fee coverage timeline for a child.
// @Summary Get child fee timeline
// @Description Returns monthly fee coverage showing which months are paid/unpaid based on transaction dates
// @Tags Children
// @Produce json
// @Security BearerAuth
// @Param id path string true "Child ID (UUID)"
// @Param year query int false "Year (defaults to current year)"
// @Success 200 {array} FeeCoverageResponse "Monthly coverage timeline"
// @Failure 400 {object} response.ErrorBody "Invalid child ID"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Child not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /children/{id}/timeline [get]
func (h *ChildHandler) GetTimeline(w http.ResponseWriter, r *http.Request) {
	childID, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	year := request.GetQueryIntOptional(r, "year")
	if year == nil {
		response.BadRequest(w, "year is required")
		return
	}

	timeline, err := h.coverageService.GetChildTimeline(r.Context(), childID, *year)
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "child not found")
			return
		}
		response.InternalError(w, "failed to get timeline")
		return
	}

	// Convert to response format
	var resp []FeeCoverageResponse
	for _, c := range timeline {
		coverage := FeeCoverageResponse{
			Year:          c.Year,
			Month:         c.Month,
			ExpectedTotal: c.ExpectedTotal,
			ReceivedTotal: c.ReceivedTotal,
			Balance:       c.Balance,
			Status:        string(c.Status),
		}

		for _, tx := range c.Transactions {
			coverage.Transactions = append(coverage.Transactions, CoveredTransactionResponse{
				TransactionID:  tx.TransactionID.String(),
				Amount:         tx.Amount,
				BookingDate:    tx.BookingDate.Format("2006-01-02"),
				Description:    tx.Description,
				IsForThisMonth: tx.IsForThisMonth,
			})
		}

		resp = append(resp, coverage)
	}

	response.Success(w, resp)
}
