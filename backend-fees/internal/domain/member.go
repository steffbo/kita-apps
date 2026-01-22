package domain

import (
	"time"

	"github.com/google/uuid"
)

// Member represents a Verein member who pays annual membership fees.
// Members can exist independently of having children in the Kita.
// Parents can optionally be linked to a Member record.
type Member struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	MemberNumber    string     `json:"memberNumber" db:"member_number"`
	FirstName       string     `json:"firstName" db:"first_name"`
	LastName        string     `json:"lastName" db:"last_name"`
	Email           *string    `json:"email,omitempty" db:"email"`
	Phone           *string    `json:"phone,omitempty" db:"phone"`
	Street          *string    `json:"street,omitempty" db:"street"`
	StreetNo        *string    `json:"streetNo,omitempty" db:"street_no"`
	PostalCode      *string    `json:"postalCode,omitempty" db:"postal_code"`
	City            *string    `json:"city,omitempty" db:"city"`
	HouseholdID     *uuid.UUID `json:"householdId,omitempty" db:"household_id"`
	MembershipStart time.Time  `json:"membershipStart" db:"membership_start"`
	MembershipEnd   *time.Time `json:"membershipEnd,omitempty" db:"membership_end"`
	IsActive        bool       `json:"isActive" db:"is_active"`
	CreatedAt       time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt       time.Time  `json:"updatedAt" db:"updated_at"`
	// Loaded relations (not stored in DB)
	Household *Household `json:"household,omitempty" db:"-"`
}

// FullName returns the member's full name.
func (m *Member) FullName() string {
	return m.FirstName + " " + m.LastName
}

// IsActiveAt checks if the member is active at a given date.
func (m *Member) IsActiveAt(date time.Time) bool {
	if !m.IsActive {
		return false
	}
	if date.Before(m.MembershipStart) {
		return false
	}
	if m.MembershipEnd != nil && date.After(*m.MembershipEnd) {
		return false
	}
	return true
}
