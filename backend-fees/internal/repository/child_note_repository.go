package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
)

const childNoteColumns = `id, child_id, text, created_at, updated_at`

// PostgresChildNoteRepository is the PostgreSQL implementation of ChildNoteRepository.
type PostgresChildNoteRepository struct {
	db *sqlx.DB
}

// NewPostgresChildNoteRepository creates a new PostgreSQL child note repository.
func NewPostgresChildNoteRepository(db *sqlx.DB) *PostgresChildNoteRepository {
	return &PostgresChildNoteRepository{db: db}
}

// Create creates a new child note.
func (r *PostgresChildNoteRepository) Create(ctx context.Context, note *domain.ChildNote) error {
	if note.ID == uuid.Nil {
		note.ID = uuid.New()
	}
	now := time.Now()
	note.CreatedAt = now
	note.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO fees.child_notes (id, child_id, text, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`, note.ID, note.ChildID, note.Text, note.CreatedAt, note.UpdatedAt)
	return err
}

// Update updates the text of an existing child note.
func (r *PostgresChildNoteRepository) Update(ctx context.Context, note *domain.ChildNote) error {
	note.UpdatedAt = time.Now()
	result, err := r.db.ExecContext(ctx, `
		UPDATE fees.child_notes SET text = $2, updated_at = $3 WHERE id = $1
	`, note.ID, note.Text, note.UpdatedAt)
	if err != nil {
		return err
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete removes a child note.
func (r *PostgresChildNoteRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM fees.child_notes WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return ErrNotFound
	}
	return nil
}

// GetByID retrieves a child note by ID.
func (r *PostgresChildNoteRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.ChildNote, error) {
	var note domain.ChildNote
	err := r.db.GetContext(ctx, &note, fmt.Sprintf(`
		SELECT %s FROM fees.child_notes WHERE id = $1
	`, childNoteColumns), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &note, nil
}

// ListByChild retrieves notes for a child with pagination, newest first.
func (r *PostgresChildNoteRepository) ListByChild(ctx context.Context, childID uuid.UUID, offset, limit int) ([]domain.ChildNote, int64, error) {
	var total int64
	err := r.db.GetContext(ctx, &total, `
		SELECT COUNT(*) FROM fees.child_notes WHERE child_id = $1
	`, childID)
	if err != nil {
		return nil, 0, err
	}

	var results []domain.ChildNote
	err = r.db.SelectContext(ctx, &results, fmt.Sprintf(`
		SELECT %s FROM fees.child_notes
		WHERE child_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT $2 OFFSET $3
	`, childNoteColumns), childID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return results, total, nil
}

// ListAll retrieves all notes with pagination, newest first, including the child name.
func (r *PostgresChildNoteRepository) ListAll(ctx context.Context, offset, limit int) ([]domain.ChildNote, int64, error) {
	var total int64
	err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM fees.child_notes`)
	if err != nil {
		return nil, 0, err
	}

	var results []struct {
		domain.ChildNote
		FirstName *string `db:"first_name"`
		LastName  *string `db:"last_name"`
	}
	err = r.db.SelectContext(ctx, &results, `
		SELECT n.id, n.child_id, n.text, n.created_at, n.updated_at,
		       c.first_name, c.last_name
		FROM fees.child_notes n
		JOIN fees.children c ON c.id = n.child_id
		ORDER BY n.created_at DESC, n.id DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	notes := make([]domain.ChildNote, len(results))
	for i, row := range results {
		notes[i] = row.ChildNote
		if row.FirstName != nil && row.LastName != nil {
			name := fmt.Sprintf("%s %s", *row.FirstName, *row.LastName)
			notes[i].ChildName = &name
		}
	}

	return notes, total, nil
}
