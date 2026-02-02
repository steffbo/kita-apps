package repository

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
)

// PostgresBankingConfigRepository implements BankingConfigRepository using PostgreSQL.
type PostgresBankingConfigRepository struct {
	db *sqlx.DB
}

// NewPostgresBankingConfigRepository creates a new PostgreSQL banking config repository.
func NewPostgresBankingConfigRepository(db *sqlx.DB) *PostgresBankingConfigRepository {
	return &PostgresBankingConfigRepository{db: db}
}

// Get retrieves the banking configuration (only one config is supported).
func (r *PostgresBankingConfigRepository) Get(ctx context.Context) (*domain.BankingConfig, error) {
	var config domain.BankingConfig
	err := r.db.GetContext(ctx, &config, `
		SELECT id, bank_name, bank_blz, user_id, account_number, encrypted_pin, fints_url, tan_method, product_id,
		       last_sync_at, sync_enabled, created_at, updated_at
		FROM fees.banking_configs
		ORDER BY created_at DESC
		LIMIT 1
	`)
	if err != nil {
		return nil, ErrNotFound
	}
	return &config, nil
}

// Create creates a new banking configuration.
func (r *PostgresBankingConfigRepository) Create(ctx context.Context, config *domain.BankingConfig) error {
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO fees.banking_configs (id, bank_name, bank_blz, user_id, account_number, encrypted_pin, fints_url, tan_method, product_id, sync_enabled)
		VALUES (:id, :bank_name, :bank_blz, :user_id, :account_number, :encrypted_pin, :fints_url, :tan_method, :product_id, :sync_enabled)
	`, config)
	return err
}

// Update updates the banking configuration.
func (r *PostgresBankingConfigRepository) Update(ctx context.Context, config *domain.BankingConfig) error {
	_, err := r.db.NamedExecContext(ctx, `
		UPDATE fees.banking_configs
		SET bank_name = :bank_name,
		    bank_blz = :bank_blz,
		    user_id = :user_id,
		    account_number = :account_number,
		    encrypted_pin = :encrypted_pin,
		    fints_url = :fints_url,
		    tan_method = :tan_method,
		    product_id = :product_id,
		    sync_enabled = :sync_enabled,
		    updated_at = NOW()
		WHERE id = :id
	`, config)
	return err
}

// Delete removes the banking configuration.
func (r *PostgresBankingConfigRepository) Delete(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM fees.banking_configs`)
	return err
}

// UpdateLastSync updates the last sync timestamp.
func (r *PostgresBankingConfigRepository) UpdateLastSync(ctx context.Context, syncTime time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE fees.banking_configs
		SET last_sync_at = $1,
		    updated_at = NOW()
	`, syncTime)
	return err
}
