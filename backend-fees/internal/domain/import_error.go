package domain

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// ImportError describes a bank CSV row that could not be imported or fully
// processed (e.g. saving failed or the automatic match could not be written).
type ImportError struct {
	BookingDate *time.Time `json:"bookingDate,omitempty"`
	PayerName   *string    `json:"payerName,omitempty"`
	Amount      float64    `json:"amount"`
	Message     string     `json:"message"`
} //@name ImportError

// NewImportError builds an ImportError for the given transaction.
func NewImportError(tx BankTransaction, message string) ImportError {
	bookingDate := tx.BookingDate
	return ImportError{
		BookingDate: &bookingDate,
		PayerName:   tx.PayerName,
		Amount:      tx.Amount,
		Message:     message,
	}
}

// ImportErrors is stored as JSONB on fees.import_batches.errors.
type ImportErrors []ImportError

// Scan implements the sql.Scanner interface for reading JSONB from PostgreSQL.
func (e *ImportErrors) Scan(src interface{}) error {
	if src == nil {
		*e = ImportErrors{}
		return nil
	}
	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into ImportErrors", src)
	}
	return json.Unmarshal(data, e)
}

// Value implements the driver.Valuer interface for writing JSONB to PostgreSQL.
func (e ImportErrors) Value() (driver.Value, error) {
	if e == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(e)
}
