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
	PayerName     *string    `json:"payerName,omitempty" db:"payer_name" binding:"optional"`
	PayerIBAN     *string    `json:"payerIban,omitempty" db:"payer_iban" binding:"optional"`
	Description   *string    `json:"description,omitempty" db:"description" binding:"optional"`
	Amount        float64    `json:"amount" db:"amount"`
	Currency      string     `json:"currency" db:"currency"`
	ImportBatchID *uuid.UUID `json:"importBatchId,omitempty" db:"import_batch_id" binding:"optional"`
	ImportedAt    time.Time  `json:"importedAt" db:"imported_at"`
	IsHidden      bool       `json:"isHidden" db:"is_hidden"`
	HiddenAt      *time.Time `json:"hiddenAt,omitempty" db:"hidden_at" binding:"optional"`
	HiddenBy      *uuid.UUID `json:"hiddenBy,omitempty" db:"hidden_by" binding:"optional"`

	// Joined fields
	Matches []PaymentMatch `json:"matches,omitempty" db:"-" binding:"optional"`
	// MatchedAmount is the total amount of this transaction already allocated to fees.
	MatchedAmount *float64 `json:"matchedAmount,omitempty" db:"matched_amount" binding:"optional"`
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
	Amount        float64    `json:"amount" db:"amount"`
	MatchType     MatchType  `json:"matchType" db:"match_type"`
	Confidence    *float64   `json:"confidence,omitempty" db:"confidence" binding:"optional"`
	MatchedAt     time.Time  `json:"matchedAt" db:"matched_at"`
	MatchedBy     *uuid.UUID `json:"matchedBy,omitempty" db:"matched_by" binding:"optional"`

	// Joined fields
	Transaction *BankTransaction `json:"transaction,omitempty" db:"-" binding:"optional"`
	Expectation *FeeExpectation  `json:"expectation,omitempty" db:"-" binding:"optional"`
}

// ImportBatch represents a batch of imported transactions.
type ImportBatch struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	FileName         string     `json:"fileName" db:"file_name"`
	TransactionCount int        `json:"transactionCount" db:"transaction_count"`
	MatchedCount     int        `json:"matchedCount" db:"matched_count"`
	ImportedAt       time.Time  `json:"importedAt" db:"imported_at"`
	ImportedBy       uuid.UUID  `json:"importedBy" db:"imported_by"`
	ImportedByEmail  string     `json:"importedByEmail" db:"imported_by_email"`
	DateFrom         *time.Time `json:"dateFrom,omitempty" db:"date_from" binding:"optional"`
	DateTo           *time.Time `json:"dateTo,omitempty" db:"date_to" binding:"optional"`
	// ErrorCount is the number of rows that failed; Errors holds at most
	// MaxStoredImportErrors of them.
	ErrorCount int          `json:"errorCount" db:"error_count"`
	Errors     ImportErrors `json:"errors" db:"errors"`
}

// MaxStoredImportErrors caps the per-batch error details kept in the database.
const MaxStoredImportErrors = 100

// MatchSuggestion represents a suggested match between a transaction and a fee.
type MatchSuggestion struct {
	Transaction  BankTransaction  `json:"transaction"`
	Expectation  *FeeExpectation  `json:"expectation,omitempty" binding:"optional"`
	Expectations []FeeExpectation `json:"expectations,omitempty" binding:"optional"` // For combined matches (e.g., fee + reminder)
	Child        *Child           `json:"child,omitempty" binding:"optional"`
	DetectedType *FeeType         `json:"detectedType,omitempty" binding:"optional"`
	Confidence   float64          `json:"confidence"`
	MatchedBy    string           `json:"matchedBy"` // "member_number", "name", "amount", "combined"
}

// KnownIBANStatus represents the status of a known IBAN.
type KnownIBANStatus string

const (
	KnownIBANStatusTrusted     KnownIBANStatus = "trusted"
	KnownIBANStatusBlacklisted KnownIBANStatus = "blacklisted"
)

// KnownIBAN represents a known payment source (trusted or blacklisted).
type KnownIBAN struct {
	IBAN                  string          `json:"iban" db:"iban"`
	PayerName             *string         `json:"payerName,omitempty" db:"payer_name" binding:"optional"`
	Status                KnownIBANStatus `json:"status" db:"status"`
	ChildID               *uuid.UUID      `json:"childId,omitempty" db:"child_id" binding:"optional"`
	Reason                *string         `json:"reason,omitempty" db:"reason" binding:"optional"`
	OriginalTransactionID *uuid.UUID      `json:"originalTransactionId,omitempty" db:"original_transaction_id" binding:"optional"`
	OriginalDescription   *string         `json:"originalDescription,omitempty" db:"original_description" binding:"optional"`
	OriginalAmount        *float64        `json:"originalAmount,omitempty" db:"original_amount" binding:"optional"`
	CreatedAt             time.Time       `json:"createdAt" db:"created_at"`
	UpdatedAt             time.Time       `json:"updatedAt" db:"updated_at"`

	// Joined fields
	Child *Child `json:"child,omitempty" db:"-" binding:"optional"`
}

// KnownIBANSummary represents a trusted IBAN with usage stats.
type KnownIBANSummary struct {
	IBAN             string  `json:"iban" db:"iban"`
	PayerName        *string `json:"payerName,omitempty" db:"payer_name" binding:"optional"`
	TransactionCount int64   `json:"transactionCount" db:"transaction_count"`
}

// WarningType represents the type of transaction warning.
type WarningType string

const (
	WarningTypeNoMatchingFee    WarningType = "NO_MATCHING_FEE"    // Trusted IBAN but no open fee found
	WarningTypeUnexpectedAmount WarningType = "UNEXPECTED_AMOUNT"  // Amount doesn't match any expected fee
	WarningTypePartialPayment   WarningType = "PARTIAL_PAYMENT"    // Amount is less than expected
	WarningTypeOverpayment      WarningType = "OVERPAYMENT"        // Amount is more than expected
	WarningTypePossibleBulk     WarningType = "POSSIBLE_BULK"      // Amount could be multiple fees combined
	WarningTypeDuplicatePayment WarningType = "DUPLICATE_PAYMENT"  // Fee already paid, this might be duplicate
	WarningTypeLatePayment      WarningType = "LATE_PAYMENT"       // Payment received after the 15th of fee month
	WarningTypeMultipleOpenFees WarningType = "MULTIPLE_OPEN_FEES" // Multiple unpaid fees exist, manual review needed
)

// ResolutionType represents how a warning was resolved.
type ResolutionType string

const (
	ResolutionTypeDismissed    ResolutionType = "dismissed"     // Manually dismissed with a reason
	ResolutionTypeMatched      ResolutionType = "matched"       // Resolved by creating a match
	ResolutionTypeAutoResolved ResolutionType = "auto_resolved" // Automatically resolved
)

// TransactionWarning represents a warning about a suspicious or unexpected transaction.
type TransactionWarning struct {
	ID             uuid.UUID       `json:"id" db:"id"`
	TransactionID  uuid.UUID       `json:"transactionId" db:"transaction_id"`
	WarningType    WarningType     `json:"warningType" db:"warning_type"`
	Message        string          `json:"message" db:"message"`
	ExpectedAmount *float64        `json:"expectedAmount,omitempty" db:"expected_amount" binding:"optional"`
	ActualAmount   *float64        `json:"actualAmount,omitempty" db:"actual_amount" binding:"optional"`
	ChildID        *uuid.UUID      `json:"childId,omitempty" db:"child_id" binding:"optional"`
	MatchedFeeID   *uuid.UUID      `json:"matchedFeeId,omitempty" db:"matched_fee_id" binding:"optional"` // For LATE_PAYMENT: the fee that was matched
	ResolvedAt     *time.Time      `json:"resolvedAt,omitempty" db:"resolved_at" binding:"optional"`
	ResolvedBy     *uuid.UUID      `json:"resolvedBy,omitempty" db:"resolved_by" binding:"optional"`
	ResolutionType *ResolutionType `json:"resolutionType,omitempty" db:"resolution_type" binding:"optional"`
	ResolutionNote *string         `json:"resolutionNote,omitempty" db:"resolution_note" binding:"optional"`
	CreatedAt      time.Time       `json:"createdAt" db:"created_at"`

	// Joined fields
	Transaction *BankTransaction `json:"transaction,omitempty" db:"-" binding:"optional"`
	Child       *Child           `json:"child,omitempty" db:"-" binding:"optional"`
	MatchedFee  *FeeExpectation  `json:"matchedFee,omitempty" db:"-" binding:"optional"` // For LATE_PAYMENT: the fee that was matched
}

// IsResolved returns true if the warning has been resolved.
func (w *TransactionWarning) IsResolved() bool {
	return w.ResolvedAt != nil
}
