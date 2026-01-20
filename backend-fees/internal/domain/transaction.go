package domain

import (
	"time"

	"github.com/google/uuid"
)

// BankTransaction represents an imported bank transaction.
type BankTransaction struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	BookingDate   time.Time  `json:"bookingDate" db:"booking_date"`
	ValueDate     time.Time  `json:"valueDate" db:"value_date"`
	PayerName     *string    `json:"payerName,omitempty" db:"payer_name"`
	PayerIBAN     *string    `json:"payerIban,omitempty" db:"payer_iban"`
	Description   *string    `json:"description,omitempty" db:"description"`
	Amount        float64    `json:"amount" db:"amount"`
	Currency      string     `json:"currency" db:"currency"`
	ImportBatchID *uuid.UUID `json:"importBatchId,omitempty" db:"import_batch_id"`
	ImportedAt    time.Time  `json:"importedAt" db:"imported_at"`

	// Joined fields
	Matches []PaymentMatch `json:"matches,omitempty" db:"-"`
}

// IsIncoming returns true if the transaction is an incoming payment.
func (t *BankTransaction) IsIncoming() bool {
	return t.Amount > 0
}

// MatchType represents how a payment was matched.
type MatchType string

const (
	MatchTypeAuto   MatchType = "AUTO"
	MatchTypeManual MatchType = "MANUAL"
)

// PaymentMatch represents a match between a transaction and a fee expectation.
type PaymentMatch struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	TransactionID uuid.UUID  `json:"transactionId" db:"transaction_id"`
	ExpectationID uuid.UUID  `json:"expectationId" db:"expectation_id"`
	MatchType     MatchType  `json:"matchType" db:"match_type"`
	Confidence    *float64   `json:"confidence,omitempty" db:"confidence"`
	MatchedAt     time.Time  `json:"matchedAt" db:"matched_at"`
	MatchedBy     *uuid.UUID `json:"matchedBy,omitempty" db:"matched_by"`

	// Joined fields
	Transaction *BankTransaction `json:"transaction,omitempty" db:"-"`
	Expectation *FeeExpectation  `json:"expectation,omitempty" db:"-"`
}

// ImportBatch represents a batch of imported transactions.
type ImportBatch struct {
	ID               uuid.UUID `json:"id" db:"id"`
	FileName         string    `json:"fileName" db:"file_name"`
	TransactionCount int       `json:"transactionCount" db:"transaction_count"`
	MatchedCount     int       `json:"matchedCount" db:"matched_count"`
	ImportedAt       time.Time `json:"importedAt" db:"imported_at"`
	ImportedBy       uuid.UUID `json:"importedBy" db:"imported_by"`
}

// MatchSuggestion represents a suggested match between a transaction and a fee.
type MatchSuggestion struct {
	Transaction   BankTransaction `json:"transaction"`
	Expectation   *FeeExpectation `json:"expectation,omitempty"`
	Child         *Child          `json:"child,omitempty"`
	DetectedType  *FeeType        `json:"detectedType,omitempty"`
	Confidence    float64         `json:"confidence"`
	MatchedBy     string          `json:"matchedBy"` // "member_number", "name", "amount"
}
