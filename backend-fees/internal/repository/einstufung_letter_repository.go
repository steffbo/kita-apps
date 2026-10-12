package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
)

// EinstufungLetterRepository stores sent Einstufung letters with their PDF.
type EinstufungLetterRepository struct {
	db *sqlx.DB
}

// NewEinstufungLetterRepository creates the repository.
func NewEinstufungLetterRepository(db *sqlx.DB) *EinstufungLetterRepository {
	return &EinstufungLetterRepository{db: db}
}

// Create stores a sent letter.
func (r *EinstufungLetterRepository) Create(ctx context.Context, l *domain.EinstufungLetter) error {
	_, err := conn(ctx, r.db).ExecContext(ctx, `
		INSERT INTO fees.einstufung_letters (id, einstufung_id, child_id, household_id, email_log_id,
			sent_at, sent_by, recipients, subject, filename, pdf)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, l.ID, l.EinstufungID, l.ChildID, l.HouseholdID, l.EmailLogID, l.SentAt, l.SentBy,
		l.Recipients, l.Subject, l.Filename, l.PDF)
	return err
}

// ListByChild returns the child's letters without PDF data, newest first.
func (r *EinstufungLetterRepository) ListByChild(ctx context.Context, childID uuid.UUID) ([]domain.EinstufungLetter, error) {
	letters := []domain.EinstufungLetter{}
	err := conn(ctx, r.db).SelectContext(ctx, &letters, `
		SELECT l.id, l.einstufung_id, l.child_id, l.household_id, l.email_log_id, l.sent_at, l.sent_by,
			NULLIF(TRIM(COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '')), '') AS sent_by_name,
			l.recipients, l.subject, l.filename
		FROM fees.einstufung_letters l
		LEFT JOIN fees.users u ON u.id = l.sent_by
		WHERE l.child_id = $1
		ORDER BY l.sent_at DESC
	`, childID)
	return letters, err
}

// Get returns a letter including its PDF.
func (r *EinstufungLetterRepository) Get(ctx context.Context, id uuid.UUID) (*domain.EinstufungLetter, error) {
	var l domain.EinstufungLetter
	err := conn(ctx, r.db).GetContext(ctx, &l, `
		SELECT id, einstufung_id, child_id, household_id, email_log_id, sent_at, sent_by,
			recipients, subject, filename, pdf
		FROM fees.einstufung_letters WHERE id = $1
	`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &l, nil
}
