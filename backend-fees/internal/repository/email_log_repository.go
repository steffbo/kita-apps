package repository

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

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
		INSERT INTO fees.email_logs (id, sent_at, to_email, subject, body, email_type, payload, sent_by, household_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, log.ID, log.SentAt, log.ToEmail, log.Subject, log.Body, log.EmailType, log.Payload, log.SentBy, log.HouseholdID)
	return err
}

const emailLogColumns = `id, sent_at, to_email, subject, body, email_type, payload, sent_by, household_id`

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
		  AND ($3::uuid IS NULL OR household_id = $3::uuid)
	`, filter.EmailType, search, filter.HouseholdID); err != nil {
		return nil, 0, err
	}

	var logs []domain.EmailLog
	err := r.db.SelectContext(ctx, &logs, `
		SELECT `+emailLogColumns+`
		FROM fees.email_logs
		WHERE ($1::text IS NULL OR email_type = $1::text)
		  AND ($2::text = '' OR to_email ILIKE '%' || $2 || '%' OR subject ILIKE '%' || $2 || '%')
		  AND ($3::uuid IS NULL OR household_id = $3::uuid)
		ORDER BY sent_at `+sortDir+`
		LIMIT $4 OFFSET $5
	`, filter.EmailType, search, filter.HouseholdID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// BackfillHouseholdIDs maps email logs without household_id to the household
// of the fees referenced in their payload.feeIds. Logs whose fee IDs resolve
// to zero or multiple households stay unmapped. Returns the number of updated
// logs.
func (r *PostgresEmailLogRepository) BackfillHouseholdIDs(ctx context.Context) (int64, error) {
	var updated int64
	err := r.db.GetContext(ctx, &updated, `SELECT fees.backfill_email_log_households()`)
	return updated, err
}

// ListByHouseholdAndTypes returns reminder-type email logs for a household,
// newest first. Used for family contact history.
func (r *PostgresEmailLogRepository) ListByHouseholdAndTypes(ctx context.Context, householdID uuid.UUID, types []domain.EmailLogType) ([]domain.EmailLog, error) {
	var logs []domain.EmailLog
	err := r.db.SelectContext(ctx, &logs, `
		SELECT `+emailLogColumns+`
		FROM fees.email_logs
		WHERE household_id = $1
		  AND email_type = ANY($2::text[])
		ORDER BY sent_at DESC
	`, householdID, pq.Array(types))
	if err != nil {
		return nil, err
	}
	return logs, nil
}

// ListByHousehold returns the newest email logs for a household (family
// chronology), regardless of type.
func (r *PostgresEmailLogRepository) ListByHousehold(ctx context.Context, householdID uuid.UUID, limit int) ([]domain.EmailLog, error) {
	var logs []domain.EmailLog
	err := r.db.SelectContext(ctx, &logs, `
		SELECT `+emailLogColumns+`
		FROM fees.email_logs
		WHERE household_id = $1
		ORDER BY sent_at DESC
		LIMIT $2
	`, householdID, limit)
	if err != nil {
		return nil, err
	}
	return logs, nil
}
