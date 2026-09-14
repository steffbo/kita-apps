package service

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
)

// ChildLookup is the minimal child access the note service needs.
type ChildLookup interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Child, error)
}

// ChildNoteService handles child note business logic.
type ChildNoteService struct {
	noteRepo repository.ChildNoteRepository
	childSvc ChildLookup
}

// NewChildNoteService creates a new child note service.
func NewChildNoteService(noteRepo repository.ChildNoteRepository, childSvc ChildLookup) *ChildNoteService {
	return &ChildNoteService{noteRepo: noteRepo, childSvc: childSvc}
}

// Create creates a new note for the given child.
func (s *ChildNoteService) Create(ctx context.Context, childID uuid.UUID, text string) (*domain.ChildNote, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, ErrInvalidInput
	}

	if _, err := s.childSvc.GetByID(ctx, childID); err != nil {
		return nil, ErrNotFound
	}

	note := &domain.ChildNote{
		ChildID: childID,
		Text:    text,
	}
	if err := s.noteRepo.Create(ctx, note); err != nil {
		return nil, err
	}
	return note, nil
}

// Update updates an existing note that must belong to the given child.
func (s *ChildNoteService) Update(ctx context.Context, childID, noteID uuid.UUID, text string) (*domain.ChildNote, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, ErrInvalidInput
	}

	note, err := s.noteRepo.GetByID(ctx, noteID)
	if err != nil {
		return nil, ErrNotFound
	}
	if note.ChildID != childID {
		return nil, ErrNotFound
	}

	note.Text = text
	if err := s.noteRepo.Update(ctx, note); err != nil {
		return nil, err
	}
	return note, nil
}

// Delete removes an existing note that must belong to the given child.
func (s *ChildNoteService) Delete(ctx context.Context, childID, noteID uuid.UUID) error {
	note, err := s.noteRepo.GetByID(ctx, noteID)
	if err != nil {
		return ErrNotFound
	}
	if note.ChildID != childID {
		return ErrNotFound
	}

	return s.noteRepo.Delete(ctx, noteID)
}

// ListByChild returns notes for a child, newest first.
func (s *ChildNoteService) ListByChild(ctx context.Context, childID uuid.UUID, offset, limit int) ([]domain.ChildNote, int64, error) {
	if _, err := s.childSvc.GetByID(ctx, childID); err != nil {
		return nil, 0, ErrNotFound
	}

	return s.noteRepo.ListByChild(ctx, childID, offset, limit)
}

// ListAll returns all notes across children, newest first.
func (s *ChildNoteService) ListAll(ctx context.Context, offset, limit int) ([]domain.ChildNote, int64, error) {
	return s.noteRepo.ListAll(ctx, offset, limit)
}
