package repository

import (
	"context"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
)

// PostgresEmailLogRepository is the PostgreSQL implementation of EmailLogRepository.
type PostgresEmailLogRepository struct {
	db *sqlx.DB
}

// NewPostgresEmailLogRepository creates a new email log repository.
func NewPostgresEmailLogRepository(db *sqlx.DB) *PostgresEmailLogRepository {
	return &PostgresEmailLogRepository{db: db}
}

// Create creates a new email log entry.
func (r *PostgresEmailLogRepository) Create(ctx context.Context, log *domain.EmailLog) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO fees.email_logs (id, sent_at, to_email, subject, body, email_type, payload, sent_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, log.ID, log.SentAt, log.ToEmail, log.Subject, log.Body, log.EmailType, log.Payload, log.SentBy)
	return err
}

// List returns a filtered, sorted and paginated list of email logs.
func (r *PostgresEmailLogRepository) List(ctx context.Context, offset, limit int, filter EmailLogFilter) ([]domain.EmailLog, int64, error) {
	sortDir := strings.ToUpper(strings.TrimSpace(filter.SortDir))
	if sortDir != "ASC" && sortDir != "DESC" {
		sortDir = "DESC"
	}
	search := strings.TrimSpace(filter.Search)

	var total int64
	if err := r.db.GetContext(ctx, &total, `
		SELECT COUNT(*)
		FROM fees.email_logs
		WHERE ($1::text IS NULL OR email_type = $1::text)
		  AND ($2::text = '' OR to_email ILIKE '%' || $2 || '%' OR subject ILIKE '%' || $2 || '%')
	`, filter.EmailType, search); err != nil {
		return nil, 0, err
	}

	var logs []domain.EmailLog
	err := r.db.SelectContext(ctx, &logs, `
		SELECT id, sent_at, to_email, subject, body, email_type, payload, sent_by
		FROM fees.email_logs
		WHERE ($1::text IS NULL OR email_type = $1::text)
		  AND ($2::text = '' OR to_email ILIKE '%' || $2 || '%' OR subject ILIKE '%' || $2 || '%')
		ORDER BY sent_at `+sortDir+`
		LIMIT $3 OFFSET $4
	`, filter.EmailType, search, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}
