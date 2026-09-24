package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
)

// PostgresUserRepository is the PostgreSQL implementation of UserRepository.
type PostgresUserRepository struct {
	db *sqlx.DB
}

// NewPostgresUserRepository creates a new PostgreSQL user repository.
func NewPostgresUserRepository(db *sqlx.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

const userColumns = `id, email, password_hash, first_name, last_name, role, is_active, created_at, updated_at`

// List returns all users, admins first, then by email.
func (r *PostgresUserRepository) List(ctx context.Context) ([]domain.User, error) {
	var users []domain.User
	err := conn(ctx, r.db).SelectContext(ctx, &users, `
		SELECT `+userColumns+`
		FROM fees.users
		ORDER BY role = 'ADMIN' DESC, LOWER(email)
	`)
	return users, err
}

// GetByID returns the user or ErrNotFound.
func (r *PostgresUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var user domain.User
	err := conn(ctx, r.db).GetContext(ctx, &user, `SELECT `+userColumns+` FROM fees.users WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByEmail returns the user with that email (case-insensitive) or ErrNotFound.
func (r *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	err := conn(ctx, r.db).GetContext(ctx, &user, `SELECT `+userColumns+` FROM fees.users WHERE LOWER(email) = LOWER($1)`, email)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Create inserts the user; a taken email yields ErrDuplicate.
func (r *PostgresUserRepository) Create(ctx context.Context, user *domain.User) error {
	now := time.Now()
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	user.CreatedAt, user.UpdatedAt = now, now
	_, err := conn(ctx, r.db).ExecContext(ctx, `
		INSERT INTO fees.users (id, email, password_hash, first_name, last_name, role, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, user.ID, user.Email, user.PasswordHash, user.FirstName, user.LastName, user.Role, user.IsActive, user.CreatedAt, user.UpdatedAt)
	return mapUniqueViolation(err)
}

// Update writes profile, role and active flag; a taken email yields ErrDuplicate.
func (r *PostgresUserRepository) Update(ctx context.Context, user *domain.User) error {
	user.UpdatedAt = time.Now()
	result, err := conn(ctx, r.db).ExecContext(ctx, `
		UPDATE fees.users
		SET email = $2, first_name = $3, last_name = $4, role = $5, is_active = $6, updated_at = $7
		WHERE id = $1
	`, user.ID, user.Email, user.FirstName, user.LastName, user.Role, user.IsActive, user.UpdatedAt)
	if err != nil {
		return mapUniqueViolation(err)
	}
	return requireRow(result)
}

// UpdatePassword replaces the password hash.
func (r *PostgresUserRepository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	result, err := conn(ctx, r.db).ExecContext(ctx, `
		UPDATE fees.users SET password_hash = $2, updated_at = NOW() WHERE id = $1
	`, id, passwordHash)
	if err != nil {
		return err
	}
	return requireRow(result)
}
