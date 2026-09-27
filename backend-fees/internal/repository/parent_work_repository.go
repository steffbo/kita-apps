package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/util"
)

// ParentWorkHousehold is the minimal household row needed for work accounts.
type ParentWorkHousehold struct {
	ID   uuid.UUID `json:"id" db:"id"`
	Name string    `json:"name" db:"name"`
}

// ParentWorkSnapshot contains source rows for annual account calculations.
type ParentWorkSnapshot struct {
	Households         []ParentWorkHousehold
	Children           []domain.ParentWorkChild
	UnassignedChildren []domain.ParentWorkUnassignedChild
	Members            []domain.ParentWorkMember
	Parents            []domain.ParentWorkParent
	Terms              []domain.BoardTerm
	Entries            []domain.ParentWorkEntry
	Overrides          []domain.ParentWorkOverride
	Rules              []domain.ParentWorkRule
}

// PostgresParentWorkRepository reads and writes parent work records in PostgreSQL.
type PostgresParentWorkRepository struct{ db *sqlx.DB }

// NewPostgresParentWorkRepository creates a parent work repository.
func NewPostgresParentWorkRepository(db *sqlx.DB) *PostgresParentWorkRepository {
	return &PostgresParentWorkRepository{db: db}
}

// Snapshot loads source records for a date range.
func (r *PostgresParentWorkRepository) Snapshot(
	ctx context.Context, from, until time.Time,
) (*ParentWorkSnapshot, error) {
	c := conn(ctx, r.db)
	s := &ParentWorkSnapshot{}
	queries := []struct {
		dest  interface{}
		query string
		args  []interface{}
	}{
		{&s.Households, `SELECT id, name FROM fees.households ORDER BY name, id`, nil},
		{&s.Children, `SELECT id, household_id, first_name || ' ' || last_name AS name,
            first_name, last_name, entry_date, exit_date FROM fees.children WHERE household_id IS NOT NULL
            AND entry_date <= $2 AND (exit_date IS NULL OR exit_date >= $1)`,
			[]interface{}{from, until}},
		{&s.UnassignedChildren, `SELECT id, first_name || ' ' || last_name AS name,
            entry_date, exit_date FROM fees.children WHERE household_id IS NULL
            AND entry_date <= $2 AND (exit_date IS NULL OR exit_date >= $1)`,
			[]interface{}{from, until}},
		{&s.Members, `SELECT DISTINCT m.id, h.id AS household_id,
            m.first_name || ' ' || m.last_name AS name FROM fees.members m
            JOIN fees.households h ON h.id = m.household_id
            OR h.id IN (SELECT p.household_id FROM fees.parents p WHERE p.member_id = m.id)
            ORDER BY name, m.id, household_id`, nil},
		{&s.Parents, `SELECT household_id, first_name || ' ' || last_name AS name
            FROM fees.parents WHERE household_id IS NOT NULL`, nil},
		{&s.Terms, `SELECT bt.id, bt.member_id, m.first_name || ' ' || m.last_name AS member_name,
            h.id AS household_id, h.name AS household_name, bt.office, bt.start_date, bt.end_date,
            bt.note, bt.created_at, bt.updated_at FROM fees.board_terms bt
            JOIN fees.members m ON m.id = bt.member_id
            LEFT JOIN fees.households h ON h.id = COALESCE(m.household_id,
                (SELECT p.household_id FROM fees.parents p WHERE p.member_id = m.id
                 AND p.household_id IS NOT NULL ORDER BY p.id LIMIT 1))
            WHERE bt.start_date <= $2 AND (bt.end_date IS NULL OR bt.end_date >= $1)`,
			[]interface{}{from, until}},
		{&s.Entries, `SELECT * FROM fees.parent_work_entries
            WHERE work_date >= $1 AND work_date <= $2`, []interface{}{from, until}},
		{&s.Overrides, `SELECT * FROM fees.parent_work_overrides
            WHERE kita_year >= $1 AND kita_year <= $2`,
			[]interface{}{domain.ParentWorkKitaYear(from), domain.ParentWorkKitaYear(until)}},
		{&s.Rules, `SELECT * FROM fees.parent_work_rules ORDER BY valid_from`, nil},
	}
	for _, q := range queries {
		if err := c.SelectContext(ctx, q.dest, q.query, q.args...); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// ListRules lists stored rule versions.
func (r *PostgresParentWorkRepository) ListRules(ctx context.Context) ([]domain.ParentWorkRule, error) {
	var rules []domain.ParentWorkRule
	err := conn(ctx, r.db).SelectContext(ctx, &rules,
		`SELECT * FROM fees.parent_work_rules ORDER BY valid_from`)
	return rules, err
}

// GetRule loads a rule version by ID.
func (r *PostgresParentWorkRepository) GetRule(
	ctx context.Context, id uuid.UUID,
) (*domain.ParentWorkRule, error) {
	var v domain.ParentWorkRule
	err := conn(ctx, r.db).GetContext(ctx, &v, `SELECT * FROM fees.parent_work_rules WHERE id=$1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &v, err
}

// SaveRule creates or updates a rule version.
func (r *PostgresParentWorkRepository) SaveRule(ctx context.Context, v *domain.ParentWorkRule) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	now := util.Now()
	v.UpdatedAt = now
	if v.CreatedAt.IsZero() {
		v.CreatedAt = now
		_, err := conn(ctx, r.db).ExecContext(ctx, `INSERT INTO fees.parent_work_rules
            (id, valid_from, hours_per_child_minutes, missing_hour_rate_cents,
             max_carry_over_minutes, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			v.ID, v.ValidFrom, v.HoursPerChildMinutes, v.MissingHourRateCents,
			v.MaxCarryOverMinutes, v.CreatedAt, v.UpdatedAt)
		return mapUniqueViolation(err)
	}
	result, err := conn(ctx, r.db).ExecContext(ctx, `UPDATE fees.parent_work_rules SET
        valid_from=$2, hours_per_child_minutes=$3, missing_hour_rate_cents=$4,
        max_carry_over_minutes=$5, updated_at=$6 WHERE id=$1`,
		v.ID, v.ValidFrom, v.HoursPerChildMinutes, v.MissingHourRateCents, v.MaxCarryOverMinutes, now)
	if err != nil {
		return mapUniqueViolation(err)
	}
	return requireRow(result)
}

// GetEntry loads a work entry by ID.
func (r *PostgresParentWorkRepository) GetEntry(
	ctx context.Context, id uuid.UUID,
) (*domain.ParentWorkEntry, error) {
	var v domain.ParentWorkEntry
	err := conn(ctx, r.db).GetContext(ctx, &v, `SELECT * FROM fees.parent_work_entries WHERE id=$1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &v, err
}

// SaveEntry creates or updates a work entry.
func (r *PostgresParentWorkRepository) SaveEntry(ctx context.Context, v *domain.ParentWorkEntry) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	now := util.Now()
	v.UpdatedAt = now
	if v.CreatedAt.IsZero() {
		v.CreatedAt = now
		_, err := conn(ctx, r.db).ExecContext(ctx, `INSERT INTO fees.parent_work_entries
            (id,household_id,work_date,duration_minutes,occasion,member_name,child_name,
             status,void_reason,reject_reason,source,created_by,updated_by,created_at,updated_at)
            VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
			v.ID, v.HouseholdID, v.WorkDate, v.DurationMinutes, v.Occasion, v.MemberName, v.ChildName,
			v.Status, v.VoidReason, v.RejectReason, v.Source, v.CreatedBy, v.UpdatedBy, v.CreatedAt,
			v.UpdatedAt)
		return err
	}
	result, err := conn(ctx, r.db).ExecContext(ctx, `UPDATE fees.parent_work_entries SET
        household_id=$2,work_date=$3,duration_minutes=$4,occasion=$5,member_name=$6,
        child_name=$7,status=$8,void_reason=$9,reject_reason=$10,updated_by=$11,updated_at=$12 WHERE id=$1`,
		v.ID, v.HouseholdID, v.WorkDate, v.DurationMinutes, v.Occasion, v.MemberName, v.ChildName,
		v.Status, v.VoidReason, v.RejectReason, v.UpdatedBy, now)
	if err != nil {
		return err
	}
	return requireRow(result)
}

// GetTerm loads a board appointment by ID.
func (r *PostgresParentWorkRepository) GetTerm(ctx context.Context, id uuid.UUID) (*domain.BoardTerm, error) {
	var v domain.BoardTerm
	err := conn(ctx, r.db).GetContext(ctx, &v, `SELECT bt.*, m.first_name || ' ' || m.last_name
        AS member_name, h.id AS household_id, h.name AS household_name FROM fees.board_terms bt
        JOIN fees.members m ON m.id=bt.member_id LEFT JOIN fees.households h ON h.id=COALESCE(m.household_id,
        (SELECT p.household_id FROM fees.parents p WHERE p.member_id=m.id
         AND p.household_id IS NOT NULL ORDER BY p.id LIMIT 1))
        WHERE bt.id=$1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &v, err
}

// ListTerms lists stored board appointments.
func (r *PostgresParentWorkRepository) ListTerms(ctx context.Context) ([]domain.BoardTerm, error) {
	var v []domain.BoardTerm
	err := conn(ctx, r.db).SelectContext(ctx, &v, `SELECT bt.*, m.first_name || ' ' || m.last_name
        AS member_name, h.id AS household_id, h.name AS household_name FROM fees.board_terms bt
        JOIN fees.members m ON m.id=bt.member_id LEFT JOIN fees.households h ON h.id=COALESCE(m.household_id,
        (SELECT p.household_id FROM fees.parents p WHERE p.member_id=m.id
         AND p.household_id IS NOT NULL ORDER BY p.id LIMIT 1))
        ORDER BY bt.start_date DESC, bt.id`)
	return v, err
}

// SaveTerm creates or updates a board appointment.
func (r *PostgresParentWorkRepository) SaveTerm(ctx context.Context, v *domain.BoardTerm) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	now := util.Now()
	v.UpdatedAt = now
	if v.CreatedAt.IsZero() {
		v.CreatedAt = now
		_, err := conn(ctx, r.db).ExecContext(ctx, `INSERT INTO fees.board_terms
            (id,member_id,office,start_date,end_date,note,created_at,updated_at)
            VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, v.ID, v.MemberID, v.Office, v.StartDate,
			v.EndDate, v.Note, now, now)
		return err
	}
	result, err := conn(ctx, r.db).ExecContext(ctx, `UPDATE fees.board_terms SET member_id=$2,
        office=$3,start_date=$4,end_date=$5,note=$6,updated_at=$7 WHERE id=$1`,
		v.ID, v.MemberID, v.Office, v.StartDate, v.EndDate, v.Note, now)
	if err != nil {
		return err
	}
	return requireRow(result)
}

// DeleteTerm removes a board appointment.
func (r *PostgresParentWorkRepository) DeleteTerm(ctx context.Context, id uuid.UUID) error {
	result, err := conn(ctx, r.db).ExecContext(ctx, `DELETE FROM fees.board_terms WHERE id=$1`, id)
	if err != nil {
		return err
	}
	return requireRow(result)
}

// MemberExists checks whether a member exists.
func (r *PostgresParentWorkRepository) MemberExists(ctx context.Context, id uuid.UUID) (bool, error) {
	var exists bool
	err := conn(ctx, r.db).GetContext(ctx, &exists, `SELECT EXISTS(SELECT 1 FROM fees.members WHERE id=$1)`, id)
	return exists, err
}

// HouseholdExists checks whether a household exists.
func (r *PostgresParentWorkRepository) HouseholdExists(ctx context.Context, id uuid.UUID) (bool, error) {
	var exists bool
	err := conn(ctx, r.db).GetContext(ctx, &exists,
		`SELECT EXISTS(SELECT 1 FROM fees.households WHERE id=$1)`, id)
	return exists, err
}

// SaveOverride sets a household annual requirement.
func (r *PostgresParentWorkRepository) SaveOverride(ctx context.Context, v *domain.ParentWorkOverride) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	now := util.Now()
	v.UpdatedAt = now
	err := conn(ctx, r.db).GetContext(ctx, v, `INSERT INTO fees.parent_work_overrides
        (id,household_id,kita_year,required_minutes,reason,created_by,created_at,updated_at)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$7) ON CONFLICT (household_id,kita_year)
        DO UPDATE SET required_minutes=EXCLUDED.required_minutes,reason=EXCLUDED.reason,
        updated_at=EXCLUDED.updated_at RETURNING *`, v.ID, v.HouseholdID, v.KitaYear, v.RequiredMinutes,
		v.Reason, v.CreatedBy, now)
	return err
}

// DeleteOverride removes a household annual requirement.
func (r *PostgresParentWorkRepository) DeleteOverride(ctx context.Context, id uuid.UUID, year int) error {
	result, err := conn(ctx, r.db).ExecContext(ctx, `DELETE FROM fees.parent_work_overrides
        WHERE household_id=$1 AND kita_year=$2`, id, year)
	if err != nil {
		return err
	}
	return requireRow(result)
}
