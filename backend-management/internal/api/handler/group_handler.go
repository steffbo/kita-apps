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

// GroupHandler handles group requests.
type GroupHandler struct {
	groups *service.GroupService
}

// NewGroupHandler creates a new GroupHandler.
func NewGroupHandler(groups *service.GroupService) *GroupHandler {
	return &GroupHandler{groups: groups}
}

// groupRequest contains the data for creating or updating a group.
type groupRequest struct {
	Name        string  `json:"name" validate:"required" example:"Schmetterlinge"`
	Description *string `json:"description,omitempty" example:"Gruppe für Kinder 3-4 Jahre"`
	Color       *string `json:"color,omitempty" example:"#FF6B6B"`
} //@name CreateGroupRequest

// assignmentRequest contains the data for a group assignment.
type assignmentRequest struct {
	EmployeeID     int64  `json:"employeeId" validate:"required" example:"1"`
	AssignmentType string `json:"assignmentType" validate:"required,oneof=PERMANENT SPRINGER" example:"PERMANENT"`
} //@name GroupAssignmentRequest

// List handles GET /groups.
// @Summary List all groups
// @Description Get a list of all kindergarten groups
// @Tags Groups
// @Produce json
// @Security BearerAuth
// @Success 200 {array} GroupResponse "List of groups"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /groups [get]
func (h *GroupHandler) List(w http.ResponseWriter, r *http.Request) {
	groups, err := h.groups.List(r.Context())
	if err != nil {
		response.InternalError(w, "Ein interner Fehler ist aufgetreten")
		return
	}

	result := make([]GroupResponse, 0, len(groups))
	for _, group := range groups {
		result = append(result, *mapGroupResponse(group))
	}

	response.Success(w, result)
}

// Get handles GET /groups/{id}.
// @Summary Get a group by ID
// @Description Get detailed information about a specific group including its members
// @Tags Groups
// @Produce json
// @Security BearerAuth
// @Param id path int true "Group ID"
// @Success 200 {object} GroupWithMembersResponse "Group details with members"
// @Failure 400 {object} map[string]interface{} "Invalid group ID"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Failure 404 {object} map[string]interface{} "Group not found"
// @Router /groups/{id} [get]
func (h *GroupHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Ungültige ID")
		return
	}

	group, assignments, err := h.groups.Get(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	result := GroupWithMembersResponse{GroupResponse: *mapGroupResponse(*group)}
	for _, assignment := range assignments {
		result.Members = append(result.Members, mapGroupAssignmentResponse(assignment, true))
	}

	response.Success(w, result)
}

// Create handles POST /groups.
// @Summary Create a new group
// @Description Create a new kindergarten group
// @Tags Groups
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param group body groupRequest true "Group data"
// @Success 201 {object} GroupResponse "Created group"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Router /groups [post]
func (h *GroupHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req groupRequest
	if validationErrors, err := request.DecodeAndValidate(r, &req); err != nil {
		response.BadRequest(w, "Ungültige Anfrage")
		return
	} else if validationErrors != nil {
		response.ValidationError(w, "Validierungsfehler", validationErrors)
		return
	}

	group, err := h.groups.Create(r.Context(), service.CreateGroupInput{
		Name:        req.Name,
		Description: req.Description,
		Color:       req.Color,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.Created(w, mapGroupResponse(*group))
}

// Update handles PUT /groups/{id}.
// @Summary Update a group
// @Description Update an existing group's information
// @Tags Groups
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Group ID"
// @Param group body groupRequest true "Updated group data"
// @Success 200 {object} GroupResponse "Updated group"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Failure 404 {object} map[string]interface{} "Group not found"
// @Router /groups/{id} [put]
func (h *GroupHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Ungültige ID")
		return
	}

	var req groupRequest
	if validationErrors, err := request.DecodeAndValidate(r, &req); err != nil {
		response.BadRequest(w, "Ungültige Anfrage")
		return
	} else if validationErrors != nil {
		response.ValidationError(w, "Validierungsfehler", validationErrors)
		return
	}

	group, err := h.groups.Update(r.Context(), id, service.CreateGroupInput{
		Name:        req.Name,
		Description: req.Description,
		Color:       req.Color,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.Success(w, mapGroupResponse(*group))
}

// Delete handles DELETE /groups/{id}.
// @Summary Delete a group
// @Description Delete a kindergarten group
// @Tags Groups
// @Security BearerAuth
// @Param id path int true "Group ID"
// @Success 204 "Group deleted"
// @Failure 400 {object} map[string]interface{} "Invalid group ID"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Failure 404 {object} map[string]interface{} "Group not found"
// @Router /groups/{id} [delete]
func (h *GroupHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Ungültige ID")
		return
	}

	if err := h.groups.Delete(r.Context(), id); err != nil {
		writeServiceError(w, err)
		return
	}

	response.NoContent(w)
}

// Assignments handles GET /groups/{id}/assignments.
// @Summary Get group assignments
// @Description Get all employee assignments for a group
// @Tags Groups
// @Produce json
// @Security BearerAuth
// @Param id path int true "Group ID"
// @Success 200 {array} GroupAssignmentResponse "List of employee assignments"
// @Failure 400 {object} map[string]interface{} "Invalid group ID"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Failure 404 {object} map[string]interface{} "Group not found"
// @Router /groups/{id}/assignments [get]
func (h *GroupHandler) Assignments(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Ungültige ID")
		return
	}

	assignments, err := h.groups.Assignments(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	result := make([]GroupAssignmentResponse, 0, len(assignments))
	for _, assignment := range assignments {
		result = append(result, mapGroupAssignmentResponse(assignment, true))
	}

	response.Success(w, result)
}

// UpdateAssignments handles PUT /groups/{id}/assignments.
// @Summary Update group assignments
// @Description Replace all employee assignments for a group
// @Tags Groups
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Group ID"
// @Param assignments body []assignmentRequest true "New assignments"
// @Success 200 {array} GroupAssignmentResponse "Updated assignments"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Failure 404 {object} map[string]interface{} "Group not found"
// @Router /groups/{id}/assignments [put]
func (h *GroupHandler) UpdateAssignments(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Ungültige ID")
		return
	}

	var req []assignmentRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "Ungültige Anfrage")
		return
	}

	// Validate each assignment
	for i, item := range req {
		if validationErrors := request.Validate(&item); validationErrors != nil {
			// Prefix errors with index
			indexedErrors := make(map[string]string)
			for field, errMsg := range validationErrors {
				indexedErrors[field] = errMsg
			}
			response.ValidationError(w, "Validierungsfehler bei Eintrag "+strconv.Itoa(i+1), indexedErrors)
			return
		}
	}

	inputs := make([]service.GroupAssignmentInput, 0, len(req))
	for _, item := range req {
		inputs = append(inputs, service.GroupAssignmentInput{
			EmployeeID:     item.EmployeeID,
			AssignmentType: parseAssignmentType(item.AssignmentType),
		})
	}

	assignments, err := h.groups.UpdateAssignments(r.Context(), id, inputs)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	result := make([]GroupAssignmentResponse, 0, len(assignments))
	for _, assignment := range assignments {
		result = append(result, mapGroupAssignmentResponse(assignment, true))
	}

	response.Success(w, result)
}

func parseAssignmentType(value string) domain.AssignmentType {
	switch value {
	case string(domain.AssignmentTypePermanent):
		return domain.AssignmentTypePermanent
	default:
		return domain.AssignmentTypeSpringer
	}
}
