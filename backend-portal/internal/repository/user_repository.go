package repository

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/knirpsenstadt/kita-apps/backend-portal/internal/domain"
)

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

type userRow struct {
	ID           uuid.UUID      `db:"id"`
	Email        string         `db:"email"`
	PasswordHash sql.NullString `db:"password_hash"`
	FirstName    sql.NullString `db:"first_name"`
	LastName     sql.NullString `db:"last_name"`
	Status       string         `db:"status"`
	Roles        pq.StringArray `db:"roles"`
	LastLoginAt  sql.NullTime   `db:"last_login_at"`
	CreatedAt    time.Time      `db:"created_at"`
	UpdatedAt    time.Time      `db:"updated_at"`
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var row userRow
	err := r.db.GetContext(ctx, &row, `
		SELECT u.id, u.email, u.password_hash, u.first_name, u.last_name, u.status,
		       COALESCE(array_agg(ur.role::text ORDER BY ur.role::text) FILTER (WHERE ur.role IS NOT NULL), '{}') AS roles,
		       u.last_login_at, u.created_at, u.updated_at
		FROM portal.users u
		LEFT JOIN portal.user_roles ur ON ur.user_id = u.id
		WHERE LOWER(u.email) = LOWER($1)
		GROUP BY u.id
	`, strings.TrimSpace(email))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return mapUserRow(row), nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var row userRow
	err := r.db.GetContext(ctx, &row, `
		SELECT u.id, u.email, u.password_hash, u.first_name, u.last_name, u.status,
		       COALESCE(array_agg(ur.role::text ORDER BY ur.role::text) FILTER (WHERE ur.role IS NOT NULL), '{}') AS roles,
		       u.last_login_at, u.created_at, u.updated_at
		FROM portal.users u
		LEFT JOIN portal.user_roles ur ON ur.user_id = u.id
		WHERE u.id = $1
		GROUP BY u.id
	`, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return mapUserRow(row), nil
}

func (r *UserRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE portal.users
		SET last_login_at = NOW()
		WHERE id = $1
	`, id)
	return err
}

func (r *UserRepository) EnsureBootstrapAdmin(ctx context.Context, email, passwordHash string) (*domain.User, error) {
	email = strings.TrimSpace(email)
	if email == "" || passwordHash == "" {
		return nil, nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var id uuid.UUID
	err = tx.QueryRowxContext(ctx, `
		INSERT INTO portal.users (email, password_hash, status)
		VALUES ($1, $2, 'ACTIVE')
		ON CONFLICT (LOWER(email)) DO UPDATE
		SET password_hash = EXCLUDED.password_hash,
		    status = 'ACTIVE'
		RETURNING id
	`, email, passwordHash).Scan(&id)
	if err != nil {
		return nil, err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO portal.user_roles (user_id, role)
		VALUES ($1, 'ADMIN')
		ON CONFLICT (user_id, role) DO NOTHING
	`, id)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return r.GetByID(ctx, id)
}

func mapUserRow(row userRow) *domain.User {
	roles := make([]domain.UserRole, 0, len(row.Roles))
	for _, role := range row.Roles {
		roles = append(roles, domain.UserRole(role))
	}

	var passwordHash *string
	if row.PasswordHash.Valid {
		passwordHash = &row.PasswordHash.String
	}

	var firstName *string
	if row.FirstName.Valid {
		firstName = &row.FirstName.String
	}

	var lastName *string
	if row.LastName.Valid {
		lastName = &row.LastName.String
	}

	var lastLoginAt *time.Time
	if row.LastLoginAt.Valid {
		lastLoginAt = &row.LastLoginAt.Time
	}

	return &domain.User{
		ID:           row.ID,
		Email:        row.Email,
		PasswordHash: passwordHash,
		FirstName:    firstName,
		LastName:     lastName,
		Status:       domain.UserStatus(row.Status),
		Roles:        roles,
		LastLoginAt:  lastLoginAt,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
}
