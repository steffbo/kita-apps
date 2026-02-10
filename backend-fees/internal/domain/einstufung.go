package domain

import (
	"time"

	"github.com/google/uuid"
)

// Einstufung represents a yearly fee classification for a child.
// It is created when parents submit income proofs and the Elternarbeitsamt calculates
// the monthly fees. The resulting document (Sheet 1) is sent to parents as a signed PDF.
type Einstufung struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	ChildID     uuid.UUID  `json:"childId" db:"child_id"`
	HouseholdID uuid.UUID  `json:"householdId" db:"household_id"`
	Year        int        `json:"year" db:"year"`           // Classification year (e.g. 2026)
	ValidFrom   time.Time  `json:"validFrom" db:"valid_from"` // Start date of this classification

	// Income calculation details (stored as JSONB for audit trail)
	IncomeCalculation HouseholdIncomeCalculation `json:"incomeCalculation" db:"income_calculation"`

	// Computed results
	AnnualNetIncome float64 `json:"annualNetIncome" db:"annual_net_income"` // Fee-relevant household income (Gesamt Jahresnettoeinkommen)

	// Classification parameters
	HighestRateVoluntary bool         `json:"highestRateVoluntary" db:"highest_rate_voluntary"` // Freiwillige Anerkennung des Höchstsatzes
	CareHoursPerWeek     int          `json:"careHoursPerWeek" db:"care_hours_per_week"`        // Betreuungszeit in Wochenstunden (30,35,40,45,50,55)
	CareType             ChildAgeType `json:"careType" db:"care_type"`                          // krippe / kindergarten
	ChildrenCount        int          `json:"childrenCount" db:"children_count"`                 // Anzahl unterhaltspflichtiger Kinder

	// Resulting fees
	MonthlyChildcareFee float64 `json:"monthlyChildcareFee" db:"monthly_childcare_fee"` // Platzgeld (monatlich)
	MonthlyFoodFee      float64 `json:"monthlyFoodFee" db:"monthly_food_fee"`           // Essengeld (monatlich, 45.40)
	AnnualMembershipFee float64 `json:"annualMembershipFee" db:"annual_membership_fee"` // Vereinsbeitrag (jährlich, 30.00)

	// Fee calculation details (the bracket/rule applied)
	FeeRule         string  `json:"feeRule" db:"fee_rule"`                           // Applied rule (e.g. "Entlastung", "Satzung", "beitragsfrei")
	DiscountPercent int     `json:"discountPercent" db:"discount_percent"`           // Sibling discount percentage
	DiscountFactor  float64 `json:"discountFactor" db:"discount_factor"`             // Sibling discount factor (1.0 = no discount)
	BaseFee         float64 `json:"baseFee" db:"base_fee"`                           // Fee before sibling discount

	Notes     string    `json:"notes,omitempty" db:"notes"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`

	// Loaded relations
	Child     *Child     `json:"child,omitempty" db:"-"`
	Household *Household `json:"household,omitempty" db:"-"`
}

// EinstufungMonthRow represents one month in the Einstufung letter table.
// This is computed on-the-fly for PDF/letter generation, not stored in DB.
type EinstufungMonthRow struct {
	Month             time.Month `json:"month"`
	Year              int        `json:"year"`
	CareHoursPerWeek  int        `json:"careHoursPerWeek"`
	CareType          string     `json:"careType"`          // "Krippe" or "Kindergarten"
	ChildcareFee      float64    `json:"childcareFee"`      // Beitrag für Kinderbetreuung
	FoodFee           float64    `json:"foodFee"`           // Essengeld
	MembershipFee     float64    `json:"membershipFee"`     // Vereinsbeitrag (only in first month)
}

// GenerateMonthlyTable generates the monthly fee breakdown for the Einstufung letter.
// It produces rows from validFrom until the end of the year (or exitDate if earlier).
func (e *Einstufung) GenerateMonthlyTable(exitDate *time.Time) []EinstufungMonthRow {
	var rows []EinstufungMonthRow

	startMonth := e.ValidFrom.Month()
	startYear := e.ValidFrom.Year()
	endMonth := time.December
	endYear := startYear

	// If exit date is within this year, stop there
	if exitDate != nil && exitDate.Year() == startYear && exitDate.Month() <= time.December {
		endMonth = exitDate.Month()
	}

	for m := startMonth; m <= endMonth; m++ {
		row := EinstufungMonthRow{
			Month:            m,
			Year:             endYear,
			CareHoursPerWeek: e.CareHoursPerWeek,
			CareType:         formatCareType(e.CareType),
			ChildcareFee:     e.MonthlyChildcareFee,
			FoodFee:          e.MonthlyFoodFee,
		}

		// Membership fee only in the first month
		if m == startMonth {
			row.MembershipFee = e.AnnualMembershipFee
		}

		rows = append(rows, row)
	}

	return rows
}

func formatCareType(ct ChildAgeType) string {
	switch ct {
	case ChildAgeTypeKrippe:
		return "Krippe"
	case ChildAgeTypeKindergarten:
		return "Kindergarten"
	default:
		return string(ct)
	}
}
