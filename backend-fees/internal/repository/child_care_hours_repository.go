package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/util"
	"time"
)

type careHoursHistoryRow struct {
	ID             uuid.UUID  `db:"id"`
	ChildID        uuid.UUID  `db:"child_id"`
	CareHours      *int       `db:"care_hours"`
	EffectiveFrom  time.Time  `db:"effective_from"`
	EffectiveUntil *time.Time `db:"effective_until"`
	CreatedAt      time.Time  `db:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at"`
}

type legalHoursHistoryRow struct {
	ID             uuid.UUID  `db:"id"`
	ChildID        uuid.UUID  `db:"child_id"`
	LegalHours     *int       `db:"legal_hours"`
	EffectiveFrom  time.Time  `db:"effective_from"`
	EffectiveUntil *time.Time `db:"effective_until"`
	CreatedAt      time.Time  `db:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at"`
}

// ListCareHoursHistory returns the care hours history for a child.
func (r *PostgresChildRepository) ListCareHoursHistory(ctx context.Context, childID uuid.UUID) ([]domain.ChildCareHoursHistory, error) {
	var rows []domain.ChildCareHoursHistory
	err := conn(ctx, r.db).SelectContext(ctx, &rows, `
		SELECT id, child_id, care_hours, effective_from, effective_until, created_at, updated_at
		FROM fees.child_care_hours_history
		WHERE child_id = $1
		ORDER BY effective_from DESC, created_at DESC
	`, childID)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// UpsertCareHoursHistory creates or updates a care hours period for a child.
func (r *PostgresChildRepository) UpsertCareHoursHistory(ctx context.Context, childID uuid.UUID, careHours *int, validFrom time.Time) error {
	tx, err := beginTx(ctx, r.db)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := r.upsertCareHoursHistoryTx(ctx, tx, childID, careHours, validFrom); err != nil {
		return err
	}

	return tx.Commit()
}

// ListLegalHoursHistory returns the legal hours history for a child.
func (r *PostgresChildRepository) ListLegalHoursHistory(ctx context.Context, childID uuid.UUID) ([]domain.ChildLegalHoursHistory, error) {
	var rows []domain.ChildLegalHoursHistory
	err := conn(ctx, r.db).SelectContext(ctx, &rows, `
		SELECT id, child_id, legal_hours, effective_from, effective_until, created_at, updated_at
		FROM fees.child_legal_hours_history
		WHERE child_id = $1
		ORDER BY effective_from DESC, created_at DESC
	`, childID)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// UpsertLegalHoursHistory creates or updates a legal hours period for a child.
func (r *PostgresChildRepository) UpsertLegalHoursHistory(ctx context.Context, childID uuid.UUID, legalHours *int, validFrom time.Time) error {
	tx, err := beginTx(ctx, r.db)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := r.upsertLegalHoursHistoryTx(ctx, tx, childID, legalHours, validFrom, nil); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *PostgresChildRepository) upsertCareHoursHistoryTx(ctx context.Context, tx querier, childID uuid.UUID, careHours *int, validFrom time.Time) error {
	var rows []careHoursHistoryRow
	err := tx.SelectContext(ctx, &rows, `
		SELECT id, child_id, care_hours, effective_from, effective_until, created_at, updated_at
		FROM fees.child_care_hours_history
		WHERE child_id = $1
		ORDER BY effective_from ASC, created_at ASC
		FOR UPDATE
	`, childID)
	if err != nil {
		return err
	}

	newPeriod := careHoursHistoryRow{
		ID:            uuid.New(),
		ChildID:       childID,
		CareHours:     cloneNullableInt(careHours),
		EffectiveFrom: truncateDate(validFrom),
	}

	insertAt := len(rows)
	updatedExisting := false

	for i := range rows {
		rows[i].EffectiveFrom = truncateDate(rows[i].EffectiveFrom)
		if rows[i].EffectiveUntil != nil {
			until := truncateDate(*rows[i].EffectiveUntil)
			rows[i].EffectiveUntil = &until
		}

		if rows[i].EffectiveFrom.Equal(newPeriod.EffectiveFrom) {
			rows[i].CareHours = cloneNullableInt(careHours)
			updatedExisting = true
			insertAt = i
			break
		}

		if rows[i].EffectiveFrom.After(newPeriod.EffectiveFrom) {
			insertAt = i
			break
		}
	}

	if !updatedExisting {
		containing := -1
		for i, row := range rows {
			if !row.EffectiveFrom.After(newPeriod.EffectiveFrom) && (row.EffectiveUntil == nil || !row.EffectiveUntil.Before(newPeriod.EffectiveFrom)) {
				containing = i
				break
			}
		}

		if containing >= 0 {
			row := rows[containing]
			if row.EffectiveFrom.Equal(newPeriod.EffectiveFrom) {
				rows[containing].CareHours = cloneNullableInt(careHours)
			} else {
				until := newPeriod.EffectiveFrom.AddDate(0, 0, -1)
				rows[containing].EffectiveUntil = &until
				newPeriod.EffectiveUntil = row.EffectiveUntil
				insertAt = containing + 1
				rows = append(rows[:insertAt], append([]careHoursHistoryRow{newPeriod}, rows[insertAt:]...)...)
			}
		} else {
			if insertAt < len(rows) {
				until := rows[insertAt].EffectiveFrom.AddDate(0, 0, -1)
				newPeriod.EffectiveUntil = &until
			}
			rows = append(rows, careHoursHistoryRow{})
			copy(rows[insertAt+1:], rows[insertAt:])
			rows[insertAt] = newPeriod
		}
	}

	rows = normalizeCareHoursHistoryRows(rows)
	if _, err := tx.ExecContext(ctx, `DELETE FROM fees.child_care_hours_history WHERE child_id = $1`, childID); err != nil {
		return err
	}

	now := util.Now()
	for _, row := range rows {
		id := row.ID
		if id == uuid.Nil {
			id = uuid.New()
		}
		_, err := tx.ExecContext(ctx, `
			INSERT INTO fees.child_care_hours_history (
				id, child_id, care_hours, effective_from, effective_until, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, id, childID, row.CareHours, row.EffectiveFrom, row.EffectiveUntil, now, now)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *PostgresChildRepository) upsertLegalHoursHistoryTx(ctx context.Context, tx querier, childID uuid.UUID, legalHours *int, validFrom time.Time, validUntil *time.Time) error {
	var rows []legalHoursHistoryRow
	err := tx.SelectContext(ctx, &rows, `
		SELECT id, child_id, legal_hours, effective_from, effective_until, created_at, updated_at
		FROM fees.child_legal_hours_history
		WHERE child_id = $1
		ORDER BY effective_from ASC, created_at ASC
		FOR UPDATE
	`, childID)
	if err != nil {
		return err
	}

	newPeriod := legalHoursHistoryRow{
		ID:            uuid.New(),
		ChildID:       childID,
		LegalHours:    cloneNullableInt(legalHours),
		EffectiveFrom: truncateDate(validFrom),
	}
	if validUntil != nil {
		until := truncateDate(*validUntil)
		newPeriod.EffectiveUntil = &until
	}

	insertAt := len(rows)
	updatedExisting := false

	for i := range rows {
		rows[i].EffectiveFrom = truncateDate(rows[i].EffectiveFrom)
		if rows[i].EffectiveUntil != nil {
			until := truncateDate(*rows[i].EffectiveUntil)
			rows[i].EffectiveUntil = &until
		}

		if rows[i].EffectiveFrom.Equal(newPeriod.EffectiveFrom) {
			rows[i].LegalHours = cloneNullableInt(legalHours)
			rows[i].EffectiveUntil = cloneNullableTime(newPeriod.EffectiveUntil)
			updatedExisting = true
			insertAt = i
			break
		}

		if rows[i].EffectiveFrom.After(newPeriod.EffectiveFrom) {
			insertAt = i
			break
		}
	}

	if !updatedExisting {
		containing := -1
		for i, row := range rows {
			if !row.EffectiveFrom.After(newPeriod.EffectiveFrom) && (row.EffectiveUntil == nil || !row.EffectiveUntil.Before(newPeriod.EffectiveFrom)) {
				containing = i
				break
			}
		}

		if containing >= 0 {
			row := rows[containing]
			if row.EffectiveFrom.Equal(newPeriod.EffectiveFrom) {
				rows[containing].LegalHours = cloneNullableInt(legalHours)
				rows[containing].EffectiveUntil = cloneNullableTime(newPeriod.EffectiveUntil)
			} else {
				originalUntil := cloneNullableTime(row.EffectiveUntil)
				until := newPeriod.EffectiveFrom.AddDate(0, 0, -1)
				rows[containing].EffectiveUntil = &until
				newPeriod.EffectiveUntil = minNullableDate(newPeriod.EffectiveUntil, originalUntil)
				insertAt = containing + 1
				rows = append(rows[:insertAt], append([]legalHoursHistoryRow{newPeriod}, rows[insertAt:]...)...)
			}
		} else {
			if insertAt < len(rows) {
				nextUntil := rows[insertAt].EffectiveFrom.AddDate(0, 0, -1)
				newPeriod.EffectiveUntil = minNullableDate(newPeriod.EffectiveUntil, &nextUntil)
			}
			rows = append(rows, legalHoursHistoryRow{})
			copy(rows[insertAt+1:], rows[insertAt:])
			rows[insertAt] = newPeriod
		}
	}

	rows = normalizeLegalHoursHistoryRows(rows)
	if _, err := tx.ExecContext(ctx, `DELETE FROM fees.child_legal_hours_history WHERE child_id = $1`, childID); err != nil {
		return err
	}

	now := util.Now()
	for _, row := range rows {
		id := row.ID
		if id == uuid.Nil {
			id = uuid.New()
		}
		_, err := tx.ExecContext(ctx, `
			INSERT INTO fees.child_legal_hours_history (
				id, child_id, legal_hours, effective_from, effective_until, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, id, childID, row.LegalHours, row.EffectiveFrom, row.EffectiveUntil, now, now)
		if err != nil {
			return err
		}
	}

	return nil
}

func currentCareHoursTx(ctx context.Context, tx querier, childID uuid.UUID, at time.Time) (*int, error) {
	var raw sql.NullInt64
	err := tx.GetContext(ctx, &raw, `
		SELECT care_hours
		FROM fees.child_care_hours_history
		WHERE child_id = $1
		  AND effective_from <= $2
		  AND (effective_until IS NULL OR effective_until >= $2)
		ORDER BY effective_from DESC, created_at DESC
		LIMIT 1
	`, childID, truncateDate(at))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !raw.Valid {
		return nil, nil
	}
	value := int(raw.Int64)
	return &value, nil
}

func currentLegalHoursTx(ctx context.Context, tx querier, childID uuid.UUID, at time.Time) (*int, *time.Time, error) {
	type currentLegalRow struct {
		LegalHours     sql.NullInt64 `db:"legal_hours"`
		EffectiveUntil *time.Time    `db:"effective_until"`
	}
	var current currentLegalRow
	err := tx.GetContext(ctx, &current, `
		SELECT legal_hours, effective_until
		FROM fees.child_legal_hours_history
		WHERE child_id = $1
		  AND effective_from <= $2
		  AND (effective_until IS NULL OR effective_until >= $2)
		ORDER BY effective_from DESC, created_at DESC
		LIMIT 1
	`, childID, truncateDate(at))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}

	var legalHours *int
	if current.LegalHours.Valid {
		value := int(current.LegalHours.Int64)
		legalHours = &value
	}
	return legalHours, current.EffectiveUntil, nil
}

func normalizeCareHoursHistoryRows(rows []careHoursHistoryRow) []careHoursHistoryRow {
	if len(rows) == 0 {
		return rows
	}

	normalized := make([]careHoursHistoryRow, 0, len(rows))
	for _, row := range rows {
		if row.EffectiveUntil != nil && row.EffectiveUntil.Before(row.EffectiveFrom) {
			continue
		}

		if len(normalized) == 0 {
			normalized = append(normalized, row)
			continue
		}

		prev := &normalized[len(normalized)-1]
		if nullableIntEqual(prev.CareHours, row.CareHours) && prev.EffectiveUntil != nil && prev.EffectiveUntil.AddDate(0, 0, 1).Equal(row.EffectiveFrom) {
			prev.EffectiveUntil = row.EffectiveUntil
			continue
		}

		normalized = append(normalized, row)
	}

	return normalized
}

func normalizeLegalHoursHistoryRows(rows []legalHoursHistoryRow) []legalHoursHistoryRow {
	if len(rows) == 0 {
		return rows
	}

	normalized := make([]legalHoursHistoryRow, 0, len(rows))
	for _, row := range rows {
		if row.EffectiveUntil != nil && row.EffectiveUntil.Before(row.EffectiveFrom) {
			continue
		}
		if len(normalized) == 0 {
			normalized = append(normalized, row)
			continue
		}

		prev := &normalized[len(normalized)-1]
		if nullableIntEqual(prev.LegalHours, row.LegalHours) && prev.EffectiveUntil != nil && prev.EffectiveUntil.AddDate(0, 0, 1).Equal(row.EffectiveFrom) {
			prev.EffectiveUntil = row.EffectiveUntil
			continue
		}

		if prev.EffectiveUntil == nil || prev.EffectiveUntil.After(row.EffectiveFrom.AddDate(0, 0, -1)) {
			until := row.EffectiveFrom.AddDate(0, 0, -1)
			prev.EffectiveUntil = &until
		}

		normalized = append(normalized, row)
	}

	return normalized
}

// loadHoursBreakdown aggregates children by the hours that were effective at the Stichtag.
// The history tables are the only source; children without a period covering the Stichtag
// are grouped as unknown (NULL).
func (r *PostgresChildRepository) loadHoursBreakdown(ctx context.Context, dest interface{}, stichtag time.Time, historyTable, historyColumn string) error {
	u3Threshold := stichtag.AddDate(-3, 0, 0)

	query := fmt.Sprintf(`
		SELECT
			history_match.value AS %s,
			COUNT(*) AS count,
			COUNT(*) FILTER (WHERE c.birth_date > $2) AS u3_count,
			COUNT(*) FILTER (WHERE c.birth_date <= $2) AS ue3_count
		FROM fees.children c
		LEFT JOIN LATERAL (
			SELECT %s AS value
			FROM %s h
			WHERE h.child_id = c.id
			  AND h.effective_from <= $1
			  AND (h.effective_until IS NULL OR h.effective_until >= $1)
			ORDER BY h.effective_from DESC, h.created_at DESC
			LIMIT 1
		) history_match ON TRUE
		WHERE c.entry_date <= $1
		  AND (c.exit_date IS NULL OR c.exit_date >= $1)
		GROUP BY history_match.value
		ORDER BY history_match.value ASC NULLS LAST
	`, historyColumn, historyColumn, historyTable)

	return conn(ctx, r.db).SelectContext(ctx, dest, query, stichtag, u3Threshold)
}

// truncateDate normalizes a timestamp to the UTC midnight of its calendar date. History
// periods are calendar dates; storing them at UTC midnight keeps comparisons in Go and in
// PostgreSQL (DATE columns, UTC session) consistent regardless of the caller's location.
func truncateDate(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func cloneNullableInt(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func nullableIntEqual(a, b *int) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func nullableTimeEqual(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(*b)
}

func cloneNullableTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func minNullableDate(a, b *time.Time) *time.Time {
	if a == nil {
		return cloneNullableTime(b)
	}
	if b == nil {
		return cloneNullableTime(a)
	}
	if a.Before(*b) {
		return cloneNullableTime(a)
	}
	return cloneNullableTime(b)
}
