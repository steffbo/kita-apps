package domain

import (
	"time"

	"github.com/google/uuid"
)

// IncomeStatus represents why we do or don't have income information for a family.
type IncomeStatus string

const (
	// IncomeStatusUnknown - no status set yet (legacy data or new record)
	IncomeStatusUnknown IncomeStatus = ""
	// IncomeStatusProvided - income was provided and should be used for fee calculation
	IncomeStatusProvided IncomeStatus = "PROVIDED"
	// IncomeStatusMaxAccepted - family accepted Höchstsatz, no income needed
	IncomeStatusMaxAccepted IncomeStatus = "MAX_ACCEPTED"
	// IncomeStatusPending - waiting for documents to calculate income
	IncomeStatusPending IncomeStatus = "PENDING"
	// IncomeStatusNotRequired - child was >3y when joining, income not required
	IncomeStatusNotRequired IncomeStatus = "NOT_REQUIRED"
	// IncomeStatusHistoric - child is now >3y, income kept for historic reference only
	IncomeStatusHistoric IncomeStatus = "HISTORIC"
)

// Parent represents a parent or guardian of a child.
type Parent struct {
	ID                    uuid.UUID    `json:"id" db:"id"`
	FirstName             string       `json:"firstName" db:"first_name"`
	LastName              string       `json:"lastName" db:"last_name"`
	BirthDate             *time.Time   `json:"birthDate,omitempty" db:"birth_date"`
	Email                 *string      `json:"email,omitempty" db:"email"`
	Phone                 *string      `json:"phone,omitempty" db:"phone"`
	Street                *string      `json:"street,omitempty" db:"street"`
	StreetNo              *string      `json:"streetNo,omitempty" db:"street_no"`
	PostalCode            *string      `json:"postalCode,omitempty" db:"postal_code"`
	City                  *string      `json:"city,omitempty" db:"city"`
	AnnualHouseholdIncome *float64     `json:"annualHouseholdIncome,omitempty" db:"annual_household_income"`
	IncomeStatus          IncomeStatus `json:"incomeStatus" db:"income_status"`
	CreatedAt             time.Time    `json:"createdAt" db:"created_at"`
	UpdatedAt             time.Time    `json:"updatedAt" db:"updated_at"`
	Children              []Child      `json:"children,omitempty" db:"-"`
}

// FullName returns the parent's full name.
func (p *Parent) FullName() string {
	return p.FirstName + " " + p.LastName
}

// ChildParent represents the relationship between a child and a parent.
type ChildParent struct {
	ChildID   uuid.UUID `json:"childId" db:"child_id"`
	ParentID  uuid.UUID `json:"parentId" db:"parent_id"`
	IsPrimary bool      `json:"isPrimary" db:"is_primary"`
}
