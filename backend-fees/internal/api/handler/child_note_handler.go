package handler

import (
	"net/http"
	"time"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/request"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

// ChildNoteHandler handles child note requests.
type ChildNoteHandler struct {
	noteService *service.ChildNoteService
}

// NewChildNoteHandler creates a new child note handler.
func NewChildNoteHandler(noteService *service.ChildNoteService) *ChildNoteHandler {
	return &ChildNoteHandler{noteService: noteService}
}

// ChildNoteResponse represents a single child note.
type ChildNoteResponse struct {
	ID        string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440012"`
	ChildID   string  `json:"childId" example:"550e8400-e29b-41d4-a716-446655440000"`
	ChildName *string `json:"childName,omitempty" example:"Emma Müller"`
	Text      string  `json:"text" example:"Muss früher abgeholt werden am 12.09."`
	CreatedAt string  `json:"createdAt" example:"2026-09-01T10:00:00Z"`
	UpdatedAt string  `json:"updatedAt" example:"2026-09-01T10:00:00Z"`
} //@name ChildNote

// ChildNoteListResponse represents a paginated list of child notes.
// @Description Paginated list of child notes as returned by the note endpoints
type ChildNoteListResponse struct {
	Data       []ChildNoteResponse `json:"data"`
	Total      int64               `json:"total" example:"100"`
	Page       int                 `json:"page" example:"1"`
	PerPage    int                 `json:"perPage" example:"20"`
	TotalPages int                 `json:"totalPages" example:"5"`
} //@name ChildNoteList

// CreateChildNoteRequest represents a request to create a child note.
// @Description Request body for creating a new child note
type CreateChildNoteRequest struct {
	Text string `json:"text" example:"Muss früher abgeholt werden am 12.09."`
} //@name CreateChildNoteRequest

// UpdateChildNoteRequest represents a request to update a child note.
// @Description Request body for updating a child note
type UpdateChildNoteRequest struct {
	Text string `json:"text" example:"Muss früher abgeholt werden am 12.09."`
} //@name UpdateChildNoteRequest

// CreateChildNote creates a new note for a child.
// @Summary Create child note
// @Description Adds a new note to a child
// @Tags Notes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Child ID (UUID)"
// @Param request body CreateChildNoteRequest true "Note data"
// @Success 201 {object} ChildNoteResponse "Note created"
// @Failure 400 {object} response.ErrorBody "Invalid request"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Child not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /children/{id}/notes [post]
func (h *ChildNoteHandler) Create(w http.ResponseWriter, r *http.Request) {
	childID, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req CreateChildNoteRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	note, err := h.noteService.Create(r.Context(), childID, req.Text)
	if err != nil {
		if err == service.ErrInvalidInput {
			response.BadRequest(w, "text is required")
			return
		}
		if err == service.ErrNotFound {
			response.NotFound(w, "child not found")
			return
		}
		response.InternalError(w, "failed to create note")
		return
	}

	response.Created(w, toChildNoteResponse(*note))
}

// ListByChild returns the notes for a child.
// @Summary List notes for a child
// @Description Returns a paginated list of notes for a child, newest first
// @Tags Notes
// @Produce json
// @Security BearerAuth
// @Param id path string true "Child ID (UUID)"
// @Param page query int false "Page number" default(1)
// @Param perPage query int false "Items per page" default(20)
// @Success 200 {object} ChildNoteListResponse "Paginated notes for the child"
// @Failure 400 {object} response.ErrorBody "Invalid child ID"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Child not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /children/{id}/notes [get]
func (h *ChildNoteHandler) ListByChild(w http.ResponseWriter, r *http.Request) {
	childID, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	pagination := request.GetPagination(r)
	notes, total, err := h.noteService.ListByChild(r.Context(), childID, pagination.Offset, pagination.PerPage)
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "child not found")
			return
		}
		response.InternalError(w, "failed to list notes")
		return
	}

	response.Paginated(w, toChildNoteResponses(notes), total, pagination.Page, pagination.PerPage)
}

// Update updates a note of a child.
// @Summary Update child note
// @Description Updates the text of an existing note of a child
// @Tags Notes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Child ID (UUID)"
// @Param noteId path string true "Note ID (UUID)"
// @Param request body UpdateChildNoteRequest true "Updated note data"
// @Success 200 {object} ChildNoteResponse "Note updated"
// @Failure 400 {object} response.ErrorBody "Invalid request"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Note not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /children/{id}/notes/{noteId} [put]
func (h *ChildNoteHandler) Update(w http.ResponseWriter, r *http.Request) {
	childID, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	noteID, ok := parseUUIDParam(w, r, "noteId")
	if !ok {
		return
	}

	var req UpdateChildNoteRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	note, err := h.noteService.Update(r.Context(), childID, noteID, req.Text)
	if err != nil {
		if err == service.ErrInvalidInput {
			response.BadRequest(w, "text is required")
			return
		}
		if err == service.ErrNotFound {
			response.NotFound(w, "note not found")
			return
		}
		response.InternalError(w, "failed to update note")
		return
	}

	response.Success(w, toChildNoteResponse(*note))
}

// Delete removes a note of a child.
// @Summary Delete child note
// @Description Deletes an existing note of a child
// @Tags Notes
// @Security BearerAuth
// @Param id path string true "Child ID (UUID)"
// @Param noteId path string true "Note ID (UUID)"
// @Success 204 "Note deleted"
// @Failure 400 {object} response.ErrorBody "Invalid request"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 404 {object} response.ErrorBody "Note not found"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /children/{id}/notes/{noteId} [delete]
func (h *ChildNoteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	childID, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	noteID, ok := parseUUIDParam(w, r, "noteId")
	if !ok {
		return
	}

	err := h.noteService.Delete(r.Context(), childID, noteID)
	if err != nil {
		if err == service.ErrNotFound {
			response.NotFound(w, "note not found")
			return
		}
		response.InternalError(w, "failed to delete note")
		return
	}

	response.NoContent(w)
}

// ListAll returns all notes across children.
// @Summary List all notes
// @Description Returns a paginated list of all notes across children, newest first
// @Tags Notes
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param perPage query int false "Items per page" default(20)
// @Success 200 {object} ChildNoteListResponse "Paginated list of all notes"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /notes [get]
func (h *ChildNoteHandler) ListAll(w http.ResponseWriter, r *http.Request) {
	pagination := request.GetPagination(r)
	notes, total, err := h.noteService.ListAll(r.Context(), pagination.Offset, pagination.PerPage)
	if err != nil {
		response.InternalError(w, "failed to list notes")
		return
	}

	response.Paginated(w, toChildNoteResponses(notes), total, pagination.Page, pagination.PerPage)
}

func toChildNoteResponse(note domain.ChildNote) ChildNoteResponse {
	var childName *string
	if note.ChildName != nil {
		name := *note.ChildName
		childName = &name
	}
	return ChildNoteResponse{
		ID:        note.ID.String(),
		ChildID:   note.ChildID.String(),
		ChildName: childName,
		Text:      note.Text,
		CreatedAt: note.CreatedAt.Format(time.RFC3339),
		UpdatedAt: note.UpdatedAt.Format(time.RFC3339),
	}
}

func toChildNoteResponses(notes []domain.ChildNote) []ChildNoteResponse {
	resp := make([]ChildNoteResponse, len(notes))
	for i, note := range notes {
		resp[i] = toChildNoteResponse(note)
	}
	return resp
}
