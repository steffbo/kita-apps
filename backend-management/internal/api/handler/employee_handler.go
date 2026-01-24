package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/knirpsenstadt/kita-apps/backend-management/internal/api/request"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/service"
)

// EmployeeHandler handles employee requests.
type EmployeeHandler struct {
	employees *service.EmployeeService
}

// NewEmployeeHandler creates a new EmployeeHandler.
func NewEmployeeHandler(employees *service.EmployeeService) *EmployeeHandler {
	return &EmployeeHandler{employees: employees}
}

type createEmployeeRequest struct {
	Email               string  `json:"email"`
	FirstName           string  `json:"firstName"`
	LastName            string  `json:"lastName"`
	Role                *string `json:"role,omitempty"`
	WeeklyHours         float64 `json:"weeklyHours"`
	VacationDaysPerYear *int    `json:"vacationDaysPerYear,omitempty"`
	PrimaryGroupID      *int64  `json:"primaryGroupId,omitempty"`
}

type updateEmployeeRequest struct {
	Email                 *string  `json:"email,omitempty"`
	FirstName             *string  `json:"firstName,omitempty"`
	LastName              *string  `json:"lastName,omitempty"`
	Role                  *string  `json:"role,omitempty"`
	WeeklyHours           *float64 `json:"weeklyHours,omitempty"`
	VacationDaysPerYear   *int     `json:"vacationDaysPerYear,omitempty"`
	RemainingVacationDays *float64 `json:"remainingVacationDays,omitempty"`
	OvertimeBalance       *float64 `json:"overtimeBalance,omitempty"`
	Active                *bool    `json:"active,omitempty"`
	PrimaryGroupID        *int64   `json:"primaryGroupId,omitempty"`
}

// List handles GET /employees.
func (h *EmployeeHandler) List(w http.ResponseWriter, r *http.Request) {
	includeInactive := request.GetQueryBool(r, "includeInactive")
	activeOnly := true
	if includeInactive != nil {
		activeOnly = !*includeInactive
	}

	employees, err := h.employees.List(r.Context(), activeOnly)
	if err != nil {
		response.InternalError(w, "Ein interner Fehler ist aufgetreten")
		return
	}

	result := make([]EmployeeResponse, 0, len(employees))
	for _, emp := range employees {
		result = append(result, mapEmployeeResponse(emp.Employee, emp.PrimaryGroup, emp.PrimaryGroupID))
	}

	response.Success(w, result)
}

// Get handles GET /employees/{id}.
func (h *EmployeeHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Ungültige ID")
		return
	}

	employee, err := h.employees.Get(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.Success(w, mapEmployeeResponse(employee.Employee, employee.PrimaryGroup, employee.PrimaryGroupID))
}

// Create handles POST /employees.
func (h *EmployeeHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createEmployeeRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "Ungültige Anfrage")
		return
	}

	if req.Email == "" || req.FirstName == "" || req.LastName == "" || req.WeeklyHours == 0 {
		response.BadRequest(w, "Pflichtfelder fehlen")
		return
	}

	role := domain.EmployeeRoleEmployee
	if req.Role != nil {
		parsed, ok := parseEmployeeRole(*req.Role)
		if !ok {
			response.BadRequest(w, "Ungültige Rolle")
			return
		}
		role = parsed
	}

	vacDays := 30
	if req.VacationDaysPerYear != nil {
		vacDays = *req.VacationDaysPerYear
	}

	employee, err := h.employees.Create(r.Context(), service.CreateEmployeeInput{
		Email:               req.Email,
		FirstName:           req.FirstName,
		LastName:            req.LastName,
		Role:                role,
		WeeklyHours:         req.WeeklyHours,
		VacationDaysPerYear: vacDays,
		PrimaryGroupID:      req.PrimaryGroupID,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.Created(w, mapEmployeeResponse(employee.Employee, employee.PrimaryGroup, employee.PrimaryGroupID))
}

// Update handles PUT /employees/{id}.
func (h *EmployeeHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Ungültige ID")
		return
	}

	var req updateEmployeeRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "Ungültige Anfrage")
		return
	}

	var role *domain.EmployeeRole
	if req.Role != nil {
		parsed, ok := parseEmployeeRole(*req.Role)
		if !ok {
			response.BadRequest(w, "Ungültige Rolle")
			return
		}
		role = &parsed
	}

	employee, err := h.employees.Update(r.Context(), id, service.UpdateEmployeeInput{
		Email:                 req.Email,
		FirstName:             req.FirstName,
		LastName:              req.LastName,
		Role:                  role,
		WeeklyHours:           req.WeeklyHours,
		VacationDaysPerYear:   req.VacationDaysPerYear,
		RemainingVacationDays: req.RemainingVacationDays,
		OvertimeBalance:       req.OvertimeBalance,
		Active:                req.Active,
		PrimaryGroupID:        req.PrimaryGroupID,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.Success(w, mapEmployeeResponse(employee.Employee, employee.PrimaryGroup, employee.PrimaryGroupID))
}

// Delete handles DELETE /employees/{id}.
func (h *EmployeeHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Ungültige ID")
		return
	}

	if err := h.employees.Delete(r.Context(), id); err != nil {
		writeServiceError(w, err)
		return
	}

	response.NoContent(w)
}

// ResetPassword handles POST /employees/{id}/reset-password.
func (h *EmployeeHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Ungültige ID")
		return
	}

	if _, err := h.employees.ResetPassword(r.Context(), id); err != nil {
		writeServiceError(w, err)
		return
	}

	response.Success(w, map[string]string{"message": "Passwort wurde zurückgesetzt. Eine E-Mail mit dem neuen Passwort wurde versendet."})
}

// Assignments handles GET /employees/{id}/assignments.
func (h *EmployeeHandler) Assignments(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Ungültige ID")
		return
	}

	assignments, err := h.employees.Assignments(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	result := make([]GroupAssignmentResponse, 0, len(assignments))
	for _, assignment := range assignments {
		result = append(result, mapGroupAssignmentResponse(assignment, false))
	}

	response.Success(w, result)
}

func parseID(raw string) (int64, error) {
	return strconv.ParseInt(raw, 10, 64)
}

func parseEmployeeRole(value string) (domain.EmployeeRole, bool) {
	switch value {
	case string(domain.EmployeeRoleAdmin):
		return domain.EmployeeRoleAdmin, true
	case string(domain.EmployeeRoleEmployee):
		return domain.EmployeeRoleEmployee, true
	default:
		return "", false
	}
}
