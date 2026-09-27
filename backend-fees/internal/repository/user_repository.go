package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/util"
)

// PostgresUserRepository is the PostgreSQL implementation of UserRepository.
type PostgresUserRepository struct {
	db *sqlx.DB
}

// NewPostgresUserRepository creates a new PostgreSQL user repository.
func NewPostgresUserRepository(db *sqlx.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

const userColumns = `u.id, u.email, u.password_hash, u.first_name, u.last_name, u.role,
    u.parent_id, NULLIF(CONCAT_WS(' ', p.first_name, p.last_name), '') AS parent_name,
    u.is_active, u.created_at, u.updated_at`

// List returns all users, admins first, then by email.
func (r *PostgresUserRepository) List(ctx context.Context) ([]domain.User, error) {
	var users []domain.User
	err := conn(ctx, r.db).SelectContext(ctx, &users, `
		SELECT `+userColumns+`
		FROM fees.users u LEFT JOIN fees.parents p ON p.id = u.parent_id
		ORDER BY u.role = 'ADMIN' DESC, LOWER(u.email)
	`)
	return users, err
}

// GetByID returns the user or ErrNotFound.
func (r *PostgresUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var user domain.User
	err := conn(ctx, r.db).GetContext(ctx, &user, `SELECT `+userColumns+`
        FROM fees.users u LEFT JOIN fees.parents p ON p.id=u.parent_id WHERE u.id = $1`, id)
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
	err := conn(ctx, r.db).GetContext(ctx, &user, `SELECT `+userColumns+`
        FROM fees.users u LEFT JOIN fees.parents p ON p.id=u.parent_id
        WHERE LOWER(u.email) = LOWER($1)`, email)
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
	now := util.Now()
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	user.CreatedAt, user.UpdatedAt = now, now
	_, err := conn(ctx, r.db).ExecContext(ctx, `
		INSERT INTO fees.users (id, email, password_hash, first_name, last_name, role, parent_id,
            is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, user.ID, user.Email, user.PasswordHash, user.FirstName, user.LastName, user.Role,
		user.ParentID, user.IsActive, user.CreatedAt, user.UpdatedAt)
	return mapUniqueViolation(err)
}

// Update writes profile, role and active flag; a taken email yields ErrDuplicate.
func (r *PostgresUserRepository) Update(ctx context.Context, user *domain.User) error {
	user.UpdatedAt = util.Now()
	result, err := conn(ctx, r.db).ExecContext(ctx, `
		UPDATE fees.users
		SET email = $2, first_name = $3, last_name = $4, role = $5, parent_id = $6,
            is_active = $7, updated_at = $8
		WHERE id = $1
	`, user.ID, user.Email, user.FirstName, user.LastName, user.Role,
		user.ParentID, user.IsActive, user.UpdatedAt)
	if err != nil {
		return mapUniqueViolation(err)
	}
	return requireRow(result)
}

func (r *PostgresUserRepository) FindParentMatches(ctx context.Context, email string) ([]uuid.UUID, error) {
	ids := []uuid.UUID{}
	err := conn(ctx, r.db).SelectContext(ctx, &ids,
		`SELECT id FROM fees.parents WHERE LOWER(email)=LOWER($1) ORDER BY id`, email)
	return ids, err
}

func (r *PostgresUserRepository) ParentLinkedToOther(
	ctx context.Context, parentID, userID uuid.UUID,
) (bool, error) {
	var exists bool
	err := conn(ctx, r.db).GetContext(ctx, &exists,
		`SELECT EXISTS(SELECT 1 FROM fees.users WHERE parent_id=$1 AND id<>$2)`, parentID, userID)
	return exists, err
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

func (r *PostgresUserRepository) UpdateLinkedUser(ctx context.Context, user *domain.User, actorID uuid.UUID) error {
	return NewParentAccountRepository(r.db).UpdateLinkedUser(ctx, user, actorID)
}
