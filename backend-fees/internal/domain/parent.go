package domain

import (
	"time"

	"github.com/google/uuid"
)

// Parent represents a parent or guardian of a child.
type Parent struct {
	ID                    uuid.UUID  `json:"id" db:"id"`
	FirstName             string     `json:"firstName" db:"first_name"`
	LastName              string     `json:"lastName" db:"last_name"`
	BirthDate             *time.Time `json:"birthDate,omitempty" db:"birth_date"`
	Email                 *string    `json:"email,omitempty" db:"email"`
	Phone                 *string    `json:"phone,omitempty" db:"phone"`
	Street                *string    `json:"street,omitempty" db:"street"`
	HouseNumber           *string    `json:"houseNumber,omitempty" db:"house_number"`
	PostalCode            *string    `json:"postalCode,omitempty" db:"postal_code"`
	City                  *string    `json:"city,omitempty" db:"city"`
	AnnualHouseholdIncome *float64   `json:"annualHouseholdIncome,omitempty" db:"annual_household_income"`
	CreatedAt             time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt             time.Time  `json:"updatedAt" db:"updated_at"`
	Children              []Child    `json:"children,omitempty" db:"-"`
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
