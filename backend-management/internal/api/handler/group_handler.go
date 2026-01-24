package handler

import (
	"net/http"

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

type groupRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Color       *string `json:"color,omitempty"`
}

type assignmentRequest struct {
	EmployeeID     int64  `json:"employeeId"`
	AssignmentType string `json:"assignmentType"`
}

// List handles GET /groups.
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
func (h *GroupHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req groupRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "Ungültige Anfrage")
		return
	}
	if req.Name == "" {
		response.BadRequest(w, "Name ist erforderlich")
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
func (h *GroupHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Ungültige ID")
		return
	}

	var req groupRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "Ungültige Anfrage")
		return
	}
	if req.Name == "" {
		response.BadRequest(w, "Name ist erforderlich")
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

	inputs := make([]service.GroupAssignmentInput, 0, len(req))
	for _, item := range req {
		assignmentType, ok := parseAssignmentType(item.AssignmentType)
		if !ok {
			response.BadRequest(w, "Ungültiger Assignment-Typ")
			return
		}
		inputs = append(inputs, service.GroupAssignmentInput{
			EmployeeID:     item.EmployeeID,
			AssignmentType: assignmentType,
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

func parseAssignmentType(value string) (domain.AssignmentType, bool) {
	switch value {
	case string(domain.AssignmentTypePermanent):
		return domain.AssignmentTypePermanent, true
	case string(domain.AssignmentTypeSpringer):
		return domain.AssignmentTypeSpringer, true
	default:
		return "", false
	}
}
