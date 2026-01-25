package testutil

import (
	"context"

	"github.com/jmoiron/sqlx"
)

// Tables to truncate in order (respecting foreign key constraints).
var tablesToTruncate = []string{
	"audit_log",
	"time_entries",
	"schedule_entries",
	"special_days",
	"group_assignments",
	"groups",
	"refresh_tokens",
	"password_reset_tokens",
	"employees",
}

// CleanupTables truncates all tables to reset state between tests.
func CleanupTables(ctx context.Context, db *sqlx.DB) error {
	// Disable triggers temporarily for faster cleanup
	if _, err := db.ExecContext(ctx, "SET session_replication_role = 'replica'"); err != nil {
		return err
	}

	for _, table := range tablesToTruncate {
		if _, err := db.ExecContext(ctx, "TRUNCATE TABLE "+table+" CASCADE"); err != nil {
			return err
		}
	}

	// Re-enable triggers
	if _, err := db.ExecContext(ctx, "SET session_replication_role = 'origin'"); err != nil {
		return err
	}

	// Reset sequences
	sequences := []string{
		"employees_id_seq",
		"groups_id_seq",
		"group_assignments_id_seq",
		"schedule_entries_id_seq",
		"time_entries_id_seq",
		"special_days_id_seq",
		"audit_log_id_seq",
		"password_reset_tokens_id_seq",
		"refresh_tokens_id_seq",
	}

	for _, seq := range sequences {
		if _, err := db.ExecContext(ctx, "ALTER SEQUENCE "+seq+" RESTART WITH 1"); err != nil {
			// Ignore errors for sequences that might not exist
			continue
		}
	}

	return nil
}

// TxContext wraps a function in a transaction that is rolled back after completion.
// Useful for tests that should not persist data.
func TxContext(ctx context.Context, db *sqlx.DB, fn func(tx *sqlx.Tx) error) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	return fn(tx)
}
