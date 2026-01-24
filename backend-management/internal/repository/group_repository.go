package repository

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"

	"github.com/knirpsenstadt/kita-apps/backend-management/internal/domain"
)

// PostgresGroupRepository is the PostgreSQL implementation of GroupRepository.
type PostgresGroupRepository struct {
	db *sqlx.DB
}

// NewPostgresGroupRepository creates a new PostgreSQL group repository.
func NewPostgresGroupRepository(db *sqlx.DB) *PostgresGroupRepository {
	return &PostgresGroupRepository{db: db}
}

// List retrieves all groups ordered by name.
func (r *PostgresGroupRepository) List(ctx context.Context) ([]domain.Group, error) {
	var groups []domain.Group
	if err := r.db.SelectContext(ctx, &groups, `
		SELECT id, name, description, color, created_at, updated_at
		FROM groups
		ORDER BY name
	`); err != nil {
		return nil, err
	}
	return groups, nil
}

// GetByID retrieves a group by ID.
func (r *PostgresGroupRepository) GetByID(ctx context.Context, id int64) (*domain.Group, error) {
	var group domain.Group
	if err := r.db.GetContext(ctx, &group, `
		SELECT id, name, description, color, created_at, updated_at
		FROM groups
		WHERE id = $1
	`, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}
	return &group, nil
}

// Create inserts a new group.
func (r *PostgresGroupRepository) Create(ctx context.Context, group *domain.Group) error {
	return r.db.QueryRowxContext(ctx, `
		INSERT INTO groups (name, description, color)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`, group.Name, group.Description, group.Color).Scan(&group.ID, &group.CreatedAt, &group.UpdatedAt)
}

// Update updates a group and returns the updated record.
func (r *PostgresGroupRepository) Update(ctx context.Context, group *domain.Group) (*domain.Group, error) {
	var updated domain.Group
	if err := r.db.GetContext(ctx, &updated, `
		UPDATE groups
		SET name = $2,
		    description = $3,
		    color = $4
		WHERE id = $1
		RETURNING id, name, description, color, created_at, updated_at
	`, group.ID, group.Name, group.Description, group.Color); err != nil {
		return nil, err
	}
	return &updated, nil
}

// Delete deletes a group by ID.
func (r *PostgresGroupRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM groups WHERE id = $1`, id)
	return err
}
