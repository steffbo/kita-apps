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

// createEmployeeRequest contains the data for creating a new employee.
type createEmployeeRequest struct {
	Email               string  `json:"email" validate:"required,email" example:"max.mustermann@knirpsenstadt.de"`
	FirstName           string  `json:"firstName" validate:"required" example:"Max"`
	LastName            string  `json:"lastName" validate:"required" example:"Mustermann"`
	Role                *string `json:"role,omitempty" validate:"omitempty,oneof=ADMIN EMPLOYEE" example:"EMPLOYEE"`
	WeeklyHours         float64 `json:"weeklyHours" validate:"required,gt=0" example:"40"`
	VacationDaysPerYear *int    `json:"vacationDaysPerYear,omitempty" validate:"omitempty,gte=0" example:"30"`
	PrimaryGroupID      *int64  `json:"primaryGroupId,omitempty" example:"1"`
} //@name CreateEmployeeRequest

// updateEmployeeRequest contains the data for updating an employee.
type updateEmployeeRequest struct {
	Email                 *string  `json:"email,omitempty" validate:"omitempty,email" example:"max.mustermann@knirpsenstadt.de"`
	FirstName             *string  `json:"firstName,omitempty" example:"Max"`
	LastName              *string  `json:"lastName,omitempty" example:"Mustermann"`
	Role                  *string  `json:"role,omitempty" validate:"omitempty,oneof=ADMIN EMPLOYEE" example:"EMPLOYEE"`
	WeeklyHours           *float64 `json:"weeklyHours,omitempty" validate:"omitempty,gt=0" example:"40"`
	VacationDaysPerYear   *int     `json:"vacationDaysPerYear,omitempty" validate:"omitempty,gte=0" example:"30"`
	RemainingVacationDays *float64 `json:"remainingVacationDays,omitempty" example:"25.5"`
	OvertimeBalance       *float64 `json:"overtimeBalance,omitempty" example:"10.25"`
	Active                *bool    `json:"active,omitempty" example:"true"`
	PrimaryGroupID        *int64   `json:"primaryGroupId,omitempty" example:"1"`
} //@name UpdateEmployeeRequest

// List handles GET /employees.
// @Summary List all employees
// @Description Get a list of all employees, optionally including inactive ones
// @Tags Employees
// @Produce json
// @Security BearerAuth
// @Param includeInactive query bool false "Include inactive employees" default(false)
// @Success 200 {array} EmployeeResponse "List of employees"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /employees [get]
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
// @Summary Get an employee by ID
// @Description Get detailed information about a specific employee
// @Tags Employees
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Success 200 {object} EmployeeResponse "Employee details"
// @Failure 400 {object} map[string]interface{} "Invalid employee ID"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Failure 404 {object} map[string]interface{} "Employee not found"
// @Router /employees/{id} [get]
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
// @Summary Create a new employee
// @Description Create a new employee account
// @Tags Employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param employee body createEmployeeRequest true "Employee data"
// @Success 201 {object} EmployeeResponse "Created employee"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Failure 409 {object} map[string]interface{} "Email already exists"
// @Router /employees [post]
func (h *EmployeeHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createEmployeeRequest
	if validationErrors, err := request.DecodeAndValidate(r, &req); err != nil {
		response.BadRequest(w, "Ungültige Anfrage")
		return
	} else if validationErrors != nil {
		response.ValidationError(w, "Validierungsfehler", validationErrors)
		return
	}

	role := domain.EmployeeRoleEmployee
	if req.Role != nil {
		role = parseEmployeeRole(*req.Role)
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
// @Summary Update an employee
// @Description Update an existing employee's information
// @Tags Employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Param employee body updateEmployeeRequest true "Updated employee data"
// @Success 200 {object} EmployeeResponse "Updated employee"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Failure 404 {object} map[string]interface{} "Employee not found"
// @Router /employees/{id} [put]
func (h *EmployeeHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Ungültige ID")
		return
	}

	var req updateEmployeeRequest
	if validationErrors, err := request.DecodeAndValidate(r, &req); err != nil {
		response.BadRequest(w, "Ungültige Anfrage")
		return
	} else if validationErrors != nil {
		response.ValidationError(w, "Validierungsfehler", validationErrors)
		return
	}

	var role *domain.EmployeeRole
	if req.Role != nil {
		parsed := parseEmployeeRole(*req.Role)
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
// @Summary Delete an employee
// @Description Delete an employee (soft delete - sets inactive)
// @Tags Employees
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Success 204 "Employee deleted"
// @Failure 400 {object} map[string]interface{} "Invalid employee ID"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Failure 404 {object} map[string]interface{} "Employee not found"
// @Router /employees/{id} [delete]
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
// @Summary Reset employee password
// @Description Reset an employee's password and send them an email with the new password
// @Tags Employees
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Success 200 {object} map[string]string "Password reset email sent"
// @Failure 400 {object} map[string]interface{} "Invalid employee ID"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Failure 404 {object} map[string]interface{} "Employee not found"
// @Router /employees/{id}/reset-password [post]
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
// @Summary Get employee group assignments
// @Description Get all group assignments for an employee
// @Tags Employees
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Success 200 {array} GroupAssignmentResponse "List of group assignments"
// @Failure 400 {object} map[string]interface{} "Invalid employee ID"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Failure 404 {object} map[string]interface{} "Employee not found"
// @Router /employees/{id}/assignments [get]
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

func parseEmployeeRole(value string) domain.EmployeeRole {
	switch value {
	case string(domain.EmployeeRoleAdmin):
		return domain.EmployeeRoleAdmin
	default:
		return domain.EmployeeRoleEmployee
	}
}
