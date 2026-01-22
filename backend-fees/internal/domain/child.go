package domain

import (
	"time"

	"github.com/google/uuid"
)

// Child represents a child enrolled in the Kita.
type Child struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	MemberNumber    string     `json:"memberNumber" db:"member_number"`
	FirstName       string     `json:"firstName" db:"first_name"`
	LastName        string     `json:"lastName" db:"last_name"`
	BirthDate       time.Time  `json:"birthDate" db:"birth_date"`
	EntryDate       time.Time  `json:"entryDate" db:"entry_date"`
	ExitDate        *time.Time `json:"exitDate,omitempty" db:"exit_date"`
	Street          *string    `json:"street,omitempty" db:"street"`
	StreetNo        *string    `json:"streetNo,omitempty" db:"street_no"`
	PostalCode      *string    `json:"postalCode,omitempty" db:"postal_code"`
	City            *string    `json:"city,omitempty" db:"city"`
	LegalHours      *int       `json:"legalHours,omitempty" db:"legal_hours"`
	LegalHoursUntil *time.Time `json:"legalHoursUntil,omitempty" db:"legal_hours_until"`
	CareHours       *int       `json:"careHours,omitempty" db:"care_hours"`
	IsActive        bool       `json:"isActive" db:"is_active"`
	CreatedAt       time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt       time.Time  `json:"updatedAt" db:"updated_at"`
	Parents         []Parent   `json:"parents,omitempty" db:"-"`
}

// IsUnderThree checks if the child is under 3 years old at the given date.
func (c *Child) IsUnderThree(atDate time.Time) bool {
	thirdBirthday := c.BirthDate.AddDate(3, 0, 0)
	return atDate.Before(thirdBirthday)
}

// Age returns the child's age in years at the given date.
func (c *Child) Age(atDate time.Time) int {
	years := atDate.Year() - c.BirthDate.Year()
	if atDate.YearDay() < c.BirthDate.YearDay() {
		years--
	}
	return years
}

// FullName returns the child's full name.
func (c *Child) FullName() string {
	return c.FirstName + " " + c.LastName
}
