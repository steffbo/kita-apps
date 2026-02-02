package domain

import (
	"time"

	"github.com/google/uuid"
)

// BankingConfig represents the bank account configuration for FinTS synchronization.
type BankingConfig struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	BankName      string     `json:"bankName" db:"bank_name"`
	BankBLZ       string     `json:"bankBlz" db:"bank_blz"`
	UserID        string     `json:"userId" db:"user_id"`
	AccountNumber string     `json:"accountNumber" db:"account_number"`
	EncryptedPIN  string     `json:"-" db:"encrypted_pin"` // Never expose in JSON
	FinTSURL      string     `json:"fintsUrl" db:"fints_url"`
	TANMethod     string     `json:"tanMethod" db:"tan_method"`
	ProductID     string     `json:"productId" db:"product_id"`
	LastSyncAt    *time.Time `json:"lastSyncAt,omitempty" db:"last_sync_at"`
	SyncEnabled   bool       `json:"syncEnabled" db:"sync_enabled"`
	CreatedAt     time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt     time.Time  `json:"updatedAt" db:"updated_at"`
}

// IsConfigured returns true if the config has all required fields set.
func (c *BankingConfig) IsConfigured() bool {
	return c.BankBLZ != "" && c.UserID != "" && c.EncryptedPIN != "" && c.FinTSURL != ""
}

// SyncStatus represents the current synchronization status.
type SyncStatus struct {
	LastSyncAt        *time.Time `json:"lastSyncAt,omitempty"`
	LastSyncError     *string    `json:"lastSyncError,omitempty"`
	TransactionsCount int64      `json:"transactionsCount"`
	IsConfigured      bool       `json:"isConfigured"`
	SyncEnabled       bool       `json:"syncEnabled"`
}
