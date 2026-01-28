package domain

import (
	"time"

	"github.com/google/uuid"
)

// Household represents a family unit containing parents and children.
// Income data is stored at the household level since it applies to the entire family.
type Household struct {
	ID                    uuid.UUID    `json:"id" db:"id"`
	Name                  string       `json:"name" db:"name"` // e.g. "Familie Müller" - auto-generated or manual
	AnnualHouseholdIncome *float64     `json:"annualHouseholdIncome,omitempty" db:"annual_household_income"`
	IncomeStatus          IncomeStatus `json:"incomeStatus" db:"income_status"`
	ChildrenCountForFees  *int         `json:"childrenCountForFees,omitempty" db:"children_count_for_fees"` // Override for sibling discount calculation
	CreatedAt             time.Time    `json:"createdAt" db:"created_at"`
	UpdatedAt             time.Time    `json:"updatedAt" db:"updated_at"`
	// Loaded relations (not stored in DB)
	Parents  []Parent `json:"parents,omitempty" db:"-"`
	Children []Child  `json:"children,omitempty" db:"-"`
}
