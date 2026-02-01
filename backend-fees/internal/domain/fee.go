package domain

import (
	"time"

	"github.com/google/uuid"
)

// FeeType represents the type of fee.
type FeeType string

const (
	FeeTypeMembership FeeType = "MEMBERSHIP" // Vereinsbeitrag (yearly, 30 EUR)
	FeeTypeFood       FeeType = "FOOD"       // Essensgeld (monthly, 45.40 EUR)
	FeeTypeChildcare  FeeType = "CHILDCARE"  // Platzgeld (monthly, variable, for U3)
	FeeTypeReminder   FeeType = "REMINDER"   // Mahngebühr (10 EUR, manually triggered)
)

// FeeAmounts defines the standard fee amounts.
const (
	MembershipFeeAmount         = 30.00
	FoodFeeAmount               = 45.40
	DefaultChildcareFee         = 100.00 // Default until income-based calculation is implemented
	ReminderFeeAmount           = 10.00  // Mahngebühr for Food/Childcare
	MembershipReminderFeeAmount = 5.00   // Mahngebühr for Membership
)

// FeeExpectation represents an expected fee payment.
type FeeExpectation struct {
	ID                 uuid.UUID  `json:"id" db:"id"`
	ChildID            uuid.UUID  `json:"childId" db:"child_id"`
	FeeType            FeeType    `json:"feeType" db:"fee_type"`
	Year               int        `json:"year" db:"year"`
	Month              *int       `json:"month,omitempty" db:"month"` // nil for yearly fees
	Amount             float64    `json:"amount" db:"amount"`
	DueDate            time.Time  `json:"dueDate" db:"due_date"`
	CreatedAt          time.Time  `json:"createdAt" db:"created_at"`
	ReminderForID      *uuid.UUID `json:"reminderForId,omitempty" db:"reminder_for_id"`          // For REMINDER type: links to the original fee
	ReconciliationYear *int       `json:"reconciliationYear,omitempty" db:"reconciliation_year"` // For Kalendarjahresabrechnung: the year this Nachzahlung is for

	// Joined fields
	Child     *Child        `json:"child,omitempty" db:"-"`
	IsPaid    bool          `json:"isPaid" db:"-"`
	PaidAt    *time.Time    `json:"paidAt,omitempty" db:"-"`
	MatchedBy *PaymentMatch `json:"matchedBy,omitempty" db:"-"`
}

// FeeStatus represents the payment status of a fee.
type FeeStatus string

const (
	FeeStatusOpen    FeeStatus = "OPEN"
	FeeStatusPaid    FeeStatus = "PAID"
	FeeStatusOverdue FeeStatus = "OVERDUE"
)

// Status returns the current status of the fee expectation.
func (f *FeeExpectation) Status(atDate time.Time) FeeStatus {
	if f.IsPaid {
		return FeeStatusPaid
	}
	if atDate.After(f.DueDate) {
		return FeeStatusOverdue
	}
	return FeeStatusOpen
}

// FeeOverview represents a summary of fees for reporting.
type FeeOverview struct {
	TotalOpen            int            `json:"totalOpen"`
	TotalPaid            int            `json:"totalPaid"`
	TotalOverdue         int            `json:"totalOverdue"`
	AmountOpen           float64        `json:"amountOpen"`
	AmountPaid           float64        `json:"amountPaid"`
	AmountOverdue        float64        `json:"amountOverdue"`
	ByMonth              []MonthSummary `json:"byMonth"`
	ChildrenWithOpenFees int            `json:"childrenWithOpenFees"`
}

// MonthSummary represents fee summary for a specific month.
type MonthSummary struct {
	Year       int     `json:"year"`
	Month      int     `json:"month"`
	OpenCount  int     `json:"openCount"`
	PaidCount  int     `json:"paidCount"`
	OpenAmount float64 `json:"openAmount"`
	PaidAmount float64 `json:"paidAmount"`
}
