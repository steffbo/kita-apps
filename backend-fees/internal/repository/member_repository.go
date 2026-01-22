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

// PostgresMemberRepository is the PostgreSQL implementation of MemberRepository.
type PostgresMemberRepository struct {
	db *sqlx.DB
}

// NewPostgresMemberRepository creates a new PostgreSQL member repository.
func NewPostgresMemberRepository(db *sqlx.DB) *PostgresMemberRepository {
	return &PostgresMemberRepository{db: db}
}

// List retrieves members with optional search filtering and sorting.
func (r *PostgresMemberRepository) List(ctx context.Context, activeOnly bool, search string, sortBy string, sortDir string, offset, limit int) ([]domain.Member, int64, error) {
	var members []domain.Member
	var total int64

	baseQuery := `FROM fees.members WHERE 1=1`
	args := make([]interface{}, 0)
	argIdx := 1

	if activeOnly {
		baseQuery += fmt.Sprintf(" AND is_active = $%d", argIdx)
		args = append(args, true)
		argIdx++
	}

	if search != "" {
		baseQuery += fmt.Sprintf(" AND (first_name ILIKE $%d OR last_name ILIKE $%d OR email ILIKE $%d OR member_number ILIKE $%d)", argIdx, argIdx, argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}

	// Count total
	countQuery := "SELECT COUNT(*) " + baseQuery
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	// Get sort order
	orderClause := getMemberSortOrder(sortBy, sortDir)

	// Fetch with pagination
	selectQuery := fmt.Sprintf(`
		SELECT id, member_number, first_name, last_name, email, phone,
		       street, street_no, postal_code, city, household_id,
		       membership_start, membership_end, is_active, created_at, updated_at
		%s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, baseQuery, orderClause, argIdx, argIdx+1)
	args = append(args, limit, offset)

	err = r.db.SelectContext(ctx, &members, selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	return members, total, nil
}

// getMemberSortOrder returns a safe ORDER BY clause for members.
func getMemberSortOrder(sortBy, sortDir string) string {
	dir := "ASC"
	if sortDir == "desc" {
		dir = "DESC"
	}

	allowedColumns := map[string]string{
		"name":         "last_name",
		"memberNumber": "member_number",
		"email":        "email",
	}

	if col, ok := allowedColumns[sortBy]; ok {
		if sortBy == "name" {
			return fmt.Sprintf("last_name %s, first_name %s", dir, dir)
		}
		return fmt.Sprintf("%s %s NULLS LAST", col, dir)
	}

	// Default sort by name
	return fmt.Sprintf("last_name %s, first_name %s", dir, dir)
}

// GetByID retrieves a member by ID.
func (r *PostgresMemberRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Member, error) {
	var member domain.Member
	err := r.db.GetContext(ctx, &member, `
		SELECT id, member_number, first_name, last_name, email, phone,
		       street, street_no, postal_code, city, household_id,
		       membership_start, membership_end, is_active, created_at, updated_at
		FROM fees.members
		WHERE id = $1
	`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("member not found")
		}
		return nil, err
	}
	return &member, nil
}

// GetByMemberNumber retrieves a member by member number.
func (r *PostgresMemberRepository) GetByMemberNumber(ctx context.Context, memberNumber string) (*domain.Member, error) {
	var member domain.Member
	err := r.db.GetContext(ctx, &member, `
		SELECT id, member_number, first_name, last_name, email, phone,
		       street, street_no, postal_code, city, household_id,
		       membership_start, membership_end, is_active, created_at, updated_at
		FROM fees.members
		WHERE member_number = $1
	`, memberNumber)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("member not found")
		}
		return nil, err
	}
	return &member, nil
}

// Create creates a new member.
func (r *PostgresMemberRepository) Create(ctx context.Context, member *domain.Member) error {
	if member.ID == uuid.Nil {
		member.ID = uuid.New()
	}
	now := time.Now()
	member.CreatedAt = now
	member.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO fees.members (id, member_number, first_name, last_name, email, phone,
		                          street, street_no, postal_code, city, household_id,
		                          membership_start, membership_end, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`, member.ID, member.MemberNumber, member.FirstName, member.LastName, member.Email, member.Phone,
		member.Street, member.StreetNo, member.PostalCode, member.City, member.HouseholdID,
		member.MembershipStart, member.MembershipEnd, member.IsActive, member.CreatedAt, member.UpdatedAt)
	return err
}

// Update updates an existing member.
func (r *PostgresMemberRepository) Update(ctx context.Context, member *domain.Member) error {
	member.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE fees.members
		SET member_number = $2, first_name = $3, last_name = $4, email = $5, phone = $6,
		    street = $7, street_no = $8, postal_code = $9, city = $10, household_id = $11,
		    membership_start = $12, membership_end = $13, is_active = $14, updated_at = $15
		WHERE id = $1
	`, member.ID, member.MemberNumber, member.FirstName, member.LastName, member.Email, member.Phone,
		member.Street, member.StreetNo, member.PostalCode, member.City, member.HouseholdID,
		member.MembershipStart, member.MembershipEnd, member.IsActive, member.UpdatedAt)
	return err
}

// Delete deletes a member.
func (r *PostgresMemberRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM fees.members WHERE id = $1`, id)
	return err
}

// ListActiveAt retrieves all members that are active at a given date.
func (r *PostgresMemberRepository) ListActiveAt(ctx context.Context, date time.Time) ([]domain.Member, error) {
	var members []domain.Member
	err := r.db.SelectContext(ctx, &members, `
		SELECT id, member_number, first_name, last_name, email, phone,
		       street, street_no, postal_code, city, household_id,
		       membership_start, membership_end, is_active, created_at, updated_at
		FROM fees.members
		WHERE is_active = true
		  AND membership_start <= $1
		  AND (membership_end IS NULL OR membership_end >= $1)
		ORDER BY last_name, first_name
	`, date)
	if err != nil {
		return nil, err
	}
	return members, nil
}

// GetNextMemberNumber generates the next available member number.
func (r *PostgresMemberRepository) GetNextMemberNumber(ctx context.Context) (string, error) {
	var maxNum sql.NullInt64
	err := r.db.GetContext(ctx, &maxNum, `
		SELECT MAX(CAST(SUBSTRING(member_number FROM 2) AS INTEGER))
		FROM fees.members
		WHERE member_number ~ '^M[0-9]+$'
	`)
	if err != nil {
		return "", err
	}

	nextNum := int64(1)
	if maxNum.Valid {
		nextNum = maxNum.Int64 + 1
	}

	return fmt.Sprintf("M%04d", nextNum), nil
}
