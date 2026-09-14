package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

// stubChildNoteRepo is an in-memory ChildNoteRepository for service unit tests.
type stubChildNoteRepo struct {
	notes map[uuid.UUID]*domain.ChildNote
}

func newStubChildNoteRepo() *stubChildNoteRepo {
	return &stubChildNoteRepo{notes: make(map[uuid.UUID]*domain.ChildNote)}
}

func (r *stubChildNoteRepo) Create(_ context.Context, note *domain.ChildNote) error {
	note.ID = uuid.New()
	note.CreatedAt = time.Now()
	note.UpdatedAt = note.CreatedAt
	r.notes[note.ID] = note
	return nil
}

func (r *stubChildNoteRepo) Update(_ context.Context, note *domain.ChildNote) error {
	if _, ok := r.notes[note.ID]; !ok {
		return repository.ErrNotFound
	}
	note.UpdatedAt = time.Now()
	r.notes[note.ID] = note
	return nil
}

func (r *stubChildNoteRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := r.notes[id]; !ok {
		return repository.ErrNotFound
	}
	delete(r.notes, id)
	return nil
}

func (r *stubChildNoteRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.ChildNote, error) {
	note, ok := r.notes[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return note, nil
}

func (r *stubChildNoteRepo) ListByChild(_ context.Context, childID uuid.UUID, _, _ int) ([]domain.ChildNote, int64, error) {
	var result []domain.ChildNote
	for _, note := range r.notes {
		if note.ChildID == childID {
			result = append(result, *note)
		}
	}
	return result, int64(len(result)), nil
}

func (r *stubChildNoteRepo) ListAll(_ context.Context, _, _ int) ([]domain.ChildNote, int64, error) {
	var result []domain.ChildNote
	for _, note := range r.notes {
		result = append(result, *note)
	}
	return result, int64(len(result)), nil
}

// stubChildLookup is a minimal ChildLookup for service unit tests.
type stubChildLookup struct {
	existing map[uuid.UUID]bool
}

func (s *stubChildLookup) GetByID(_ context.Context, id uuid.UUID) (*domain.Child, error) {
	if s.existing[id] {
		return &domain.Child{ID: id}, nil
	}
	return nil, repository.ErrNotFound
}

func newNoteServiceTestSetup() (*service.ChildNoteService, *stubChildNoteRepo, uuid.UUID) {
	repo := newStubChildNoteRepo()
	childID := uuid.New()
	childSvc := &stubChildLookup{existing: map[uuid.UUID]bool{childID: true}}
	return service.NewChildNoteService(repo, childSvc), repo, childID
}

func TestChildNoteService_CreateForUnknownChild(t *testing.T) {
	svc, _, _ := newNoteServiceTestSetup()

	_, err := svc.Create(context.Background(), uuid.New(), "note")
	if !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestChildNoteService_CreateRejectsEmptyAndWhitespaceText(t *testing.T) {
	svc, _, childID := newNoteServiceTestSetup()

	for _, text := range []string{"", "   ", "\t\n"} {
		if _, err := svc.Create(context.Background(), childID, text); !errors.Is(err, service.ErrInvalidInput) {
			t.Fatalf("expected ErrInvalidInput for %q, got %v", text, err)
		}
	}
}

func TestChildNoteService_CreateAndUpdateSuccess(t *testing.T) {
	svc, repo, childID := newNoteServiceTestSetup()

	created, err := svc.Create(context.Background(), childID, "  first note  ")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if created.Text != "first note" {
		t.Fatalf("expected trimmed text, got %q", created.Text)
	}

	updated, err := svc.Update(context.Background(), childID, created.ID, "  updated note  ")
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Text != "updated note" {
		t.Fatalf("expected updated text, got %q", updated.Text)
	}
	if updated.UpdatedAt.Before(updated.CreatedAt) {
		t.Fatalf("expected UpdatedAt to be >= CreatedAt after update")
	}

	if len(repo.notes) != 1 {
		t.Fatalf("expected one note, got %d", len(repo.notes))
	}
}

func TestChildNoteService_UpdateAndDeleteUnknownNote(t *testing.T) {
	svc, _, childID := newNoteServiceTestSetup()

	if _, err := svc.Update(context.Background(), childID, uuid.New(), "text"); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for unknown note, got %v", err)
	}
	if err := svc.Delete(context.Background(), childID, uuid.New()); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for unknown note, got %v", err)
	}
}

func TestChildNoteService_UpdateAndDeleteWithWrongChild(t *testing.T) {
	svc, _, childID := newNoteServiceTestSetup()

	created, err := svc.Create(context.Background(), childID, "note")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	otherChild := uuid.New()
	if _, err := svc.Update(context.Background(), otherChild, created.ID, "text"); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for wrong child, got %v", err)
	}
	if err := svc.Delete(context.Background(), otherChild, created.ID); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for wrong child, got %v", err)
	}
}

func TestChildNoteService_UpdateAndDeleteWithoutUser(t *testing.T) {
	// Notes are not owned by a user: every authenticated caller may edit and
	// delete any note. There is deliberately no user identity involved.
	svc, _, childID := newNoteServiceTestSetup()

	created, err := svc.Create(context.Background(), childID, "shared note")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if _, err := svc.Update(context.Background(), childID, created.ID, "edited"); err != nil {
		t.Fatalf("Update without user identity failed: %v", err)
	}
	if err := svc.Delete(context.Background(), childID, created.ID); err != nil {
		t.Fatalf("Delete without user identity failed: %v", err)
	}
}
