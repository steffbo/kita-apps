package domain

import (
	"time"

	"github.com/google/uuid"
)

// Child represents a child enrolled in the Kita.
type Child struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	HouseholdID     *uuid.UUID `json:"householdId,omitempty" db:"household_id"`
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
	// Computed fields (not stored in DB, populated by queries)
	OpenFeesCount *int64 `json:"openFeesCount,omitempty" db:"open_fees_count"`
	// Loaded relations (not stored in DB)
	Parents   []Parent   `json:"parents,omitempty" db:"-"`
	Household *Household `json:"household,omitempty" db:"-"`
}

// IsUnderThree checks if the child is under 3 years old at the given date.
func (c *Child) IsUnderThree(atDate time.Time) bool {
	thirdBirthday := c.BirthDate.AddDate(3, 0, 0)
	return atDate.Before(thirdBirthday)
}

// IsUnderThreeForEntireMonth checks if the child remains under 3 for the entire month.
// Returns false if the child completes their 3rd year of life during or before the given month,
// meaning no childcare fee should be charged for that month.
//
// Important: Per German law (§ 188 Abs. 2 BGB), the 3rd year of life is completed on the day
// BEFORE the 3rd birthday, not on the birthday itself. This means:
// - A child born on October 1st completes their 3rd year on September 30th
// - Therefore September is already fee-free (beitragsfrei)
func (c *Child) IsUnderThreeForEntireMonth(year int, month time.Month) bool {
	// The day the child completes 3 years of life (day before 3rd birthday)
	// Per § 188 Abs. 2 BGB, age is completed at the END of the day before the birthday
	thirdBirthday := c.BirthDate.AddDate(3, 0, 0)
	completesThirdYear := thirdBirthday.AddDate(0, 0, -1)

	// Check if the child completes their 3rd year within or before this month
	if completesThirdYear.Year() < year {
		return false
	}
	if completesThirdYear.Year() == year && completesThirdYear.Month() <= month {
		return false
	}
	return true
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
