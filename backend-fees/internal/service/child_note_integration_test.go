package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
)

func createTestChildNote(noteRepo repository.ChildNoteRepository, childID uuid.UUID, text string) (*domain.ChildNote, error) {
	note := &domain.ChildNote{
		ChildID: childID,
		Text:    text,
	}
	if err := noteRepo.Create(context.Background(), note); err != nil {
		return nil, err
	}
	return note, nil
}

func TestChildNoteRepo_CRUD(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	ctx := context.Background()
	childRepo := repository.NewPostgresChildRepository(testDB)
	noteRepo := repository.NewPostgresChildNoteRepository(testDB)

	child, err := createTestChild(childRepo, "CRUD")
	if err != nil {
		t.Fatalf("failed to create child: %v", err)
	}

	created, err := createTestChildNote(noteRepo, child.ID, "first note")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if created.ID == uuid.Nil {
		t.Fatalf("expected note ID to be set")
	}

	loaded, err := noteRepo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if loaded.Text != "first note" || loaded.ChildID != child.ID {
		t.Fatalf("unexpected note loaded: %+v", loaded)
	}

	loaded.Text = "edited note"
	if err := noteRepo.Update(ctx, loaded); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	updated, err := noteRepo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID after update failed: %v", err)
	}
	if updated.Text != "edited note" {
		t.Fatalf("expected updated text, got %q", updated.Text)
	}

	if err := noteRepo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if _, err := noteRepo.GetByID(ctx, created.ID); err != repository.ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}

	// Missing note is reported as ErrNotFound by GetByID, Update and Delete.
	missing := &domain.ChildNote{ID: uuid.New(), ChildID: child.ID, Text: "x"}
	if err := noteRepo.Update(ctx, missing); err != repository.ErrNotFound {
		t.Fatalf("expected ErrNotFound from Update, got %v", err)
	}
	if err := noteRepo.Delete(ctx, missing.ID); err != repository.ErrNotFound {
		t.Fatalf("expected ErrNotFound from Delete, got %v", err)
	}
}

func TestChildNoteRepo_ListByChildPaginationAndOrder(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	ctx := context.Background()
	childRepo := repository.NewPostgresChildRepository(testDB)
	noteRepo := repository.NewPostgresChildNoteRepository(testDB)

	child, err := createTestChild(childRepo, "PG1")
	if err != nil {
		t.Fatalf("failed to create child: %v", err)
	}
	other, err := createTestChild(childRepo, "PG2")
	if err != nil {
		t.Fatalf("failed to create child: %v", err)
	}

	// Create notes with distinct creation times; note a must sort last.
	noteA, err := createTestChildNote(noteRepo, child.ID, "a")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	for _, text := range []string{"b", "c", "d"} {
		if _, err := createTestChildNote(noteRepo, child.ID, text); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := createTestChildNote(noteRepo, other.ID, "other"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	notes, total, err := noteRepo.ListByChild(ctx, child.ID, 0, 2)
	if err != nil {
		t.Fatalf("ListByChild failed: %v", err)
	}
	if total != 4 {
		t.Fatalf("expected total 4, got %d", total)
	}
	if len(notes) != 2 {
		t.Fatalf("expected page size 2, got %d", len(notes))
	}
	if notes[0].Text != "d" || notes[1].Text != "c" {
		t.Fatalf("expected newest first ordering [d, c], got [%s, %s]", notes[0].Text, notes[1].Text)
	}

	// Page 2 continues with older notes, no overlap between pages.
	notes2, _, err := noteRepo.ListByChild(ctx, child.ID, 2, 2)
	if err != nil {
		t.Fatalf("ListByChild page 2 failed: %v", err)
	}
	if len(notes2) != 2 || notes2[0].Text != "b" || notes2[1].Text != "a" {
		t.Fatalf("expected second page [b, a], got %+v", notes2)
	}

	// The oldest note has the smallest created_at; re-listing is stable.
	notesAgain, _, err := noteRepo.ListByChild(ctx, child.ID, 0, 4)
	if err != nil {
		t.Fatalf("ListByChild re-run failed: %v", err)
	}
	if notesAgain[3].ID != noteA.ID {
		t.Fatalf("expected stable ordering, oldest note changed position")
	}
}

func TestChildNoteRepo_ListAllWithChildName(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	ctx := context.Background()
	childRepo := repository.NewPostgresChildRepository(testDB)
	noteRepo := repository.NewPostgresChildNoteRepository(testDB)

	child, err := createTestChild(childRepo, "GA")
	if err != nil {
		t.Fatalf("failed to create child: %v", err)
	}
	// createTestChild uses fixed names "Test Kind"; give this child a distinct name.
	childUpdate := *child
	childUpdate.FirstName = "Greta"
	if err := childRepo.Update(ctx, &childUpdate); err != nil {
		t.Fatalf("failed to update child: %v", err)
	}

	if _, err := createTestChildNote(noteRepo, child.ID, "global note"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	notes, total, err := noteRepo.ListAll(ctx, 0, 10)
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}
	if total < 1 {
		t.Fatalf("expected at least one note, got total %d", total)
	}

	var found *domain.ChildNote
	for i := range notes {
		if notes[i].ID != uuid.Nil && notes[i].Text == "global note" {
			found = &notes[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("expected the created note in the global list")
	}
	if found.ChildName == nil || *found.ChildName != "Greta Kind" {
		t.Fatalf("expected child name 'Greta Kind', got %v", found.ChildName)
	}
}

func TestChildNoteRepo_CascadeDeleteWithChild(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	ctx := context.Background()
	childRepo := repository.NewPostgresChildRepository(testDB)
	noteRepo := repository.NewPostgresChildNoteRepository(testDB)

	child, err := createTestChild(childRepo, "CD")
	if err != nil {
		t.Fatalf("failed to create child: %v", err)
	}
	note, err := createTestChildNote(noteRepo, child.ID, "orphaned on delete")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := childRepo.Delete(ctx, child.ID); err != nil {
		t.Fatalf("failed to delete child: %v", err)
	}

	if _, err := noteRepo.GetByID(ctx, note.ID); err != repository.ErrNotFound {
		t.Fatalf("expected note to be cascade-deleted with the child, got %v", err)
	}
}
