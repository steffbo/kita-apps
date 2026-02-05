package domain

import "time"

// AppSetting represents a key/value application setting.
type AppSetting struct {
	Key       string    `json:"key" db:"key"`
	Value     string    `json:"value" db:"value"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}
