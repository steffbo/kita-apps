package repository

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
)

// PostgresSettingsRepository is the PostgreSQL implementation of SettingsRepository.
type PostgresSettingsRepository struct {
	db *sqlx.DB
}

// NewPostgresSettingsRepository creates a new settings repository.
func NewPostgresSettingsRepository(db *sqlx.DB) *PostgresSettingsRepository {
	return &PostgresSettingsRepository{db: db}
}

// Get retrieves a setting by key.
func (r *PostgresSettingsRepository) Get(ctx context.Context, key string) (*domain.AppSetting, error) {
	var setting domain.AppSetting
	err := r.db.GetContext(ctx, &setting, `
		SELECT key, value, updated_at
		FROM fees.app_settings
		WHERE key = $1
	`, key)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &setting, nil
}

// Upsert inserts or updates a setting.
func (r *PostgresSettingsRepository) Upsert(ctx context.Context, setting *domain.AppSetting) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO fees.app_settings (key, value, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (key)
		DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()
	`, setting.Key, setting.Value)
	return err
}
