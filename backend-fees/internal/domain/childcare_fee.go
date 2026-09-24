package domain

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// CareHourSteps are the weekly care hours the fee tables have columns for.
var CareHourSteps = [6]int{30, 35, 40, 45, 50, 55}

// FeeTableRow is one income bracket of a fee table.
type FeeTableRow struct {
	MinIncome float64    `json:"minIncome"`
	Rates     [6]float64 `json:"rates"` // monthly rates for CareHourSteps
} //@name FeeTableRow

// FeeScheduleConfig holds all amounts of one version of the fee regulation
// (Elternbeitragsordnung). Brackets are evaluated as:
//
//	income <= FreeIncomeLimit                     → free
//	FreeIncomeLimit < income <= EntlastungLimit   → EntlastungTable (no sibling discount)
//	income > EntlastungIncomeLimit                → SatzungTable (sibling discount)
//
// The last SatzungTable row is the highest rate; the average of all SatzungTable
// rows is the foster family rate.
type FeeScheduleConfig struct {
	FreeIncomeLimit       float64       `json:"freeIncomeLimit"`
	EntlastungIncomeLimit float64       `json:"entlastungIncomeLimit"`
	EntlastungTable       []FeeTableRow `json:"entlastungTable"`
	SatzungTable          []FeeTableRow `json:"satzungTable"`
	// KindergartenTable is the Satzung table for children from their 3rd birthday.
	// Reference only: kindergarten care is free under the Elternbeitragsentlastungsgesetz,
	// so the calculation does not use it.
	KindergartenTable []FeeTableRow `json:"kindergartenTable,omitempty"`
	// SiblingDiscountFactors[i] applies to i+1 children; more children use the last factor.
	SiblingDiscountFactors []float64 `json:"siblingDiscountFactors"`
	// From this many children on, the childcare fee is waived.
	SiblingsFreeThreshold int     `json:"siblingsFreeThreshold"`
	MonthlyFoodFee        float64 `json:"monthlyFoodFee"`
	AnnualMembershipFee   float64 `json:"annualMembershipFee"`
} //@name FeeScheduleConfig

// FeeSchedule is a version of the fee regulation, valid from ValidFrom until the
// next version starts.
type FeeSchedule struct {
	ID        uuid.UUID         `json:"id" db:"id"`
	ValidFrom time.Time         `json:"validFrom" db:"valid_from"`
	Name      string            `json:"name" db:"name"`
	Config    FeeScheduleConfig `json:"config" db:"config"`
	CreatedAt time.Time         `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time         `json:"updatedAt" db:"updated_at"`
} //@name FeeSchedule

// ErrNoFeeSchedule is returned when no fee schedule covers a date.
var ErrNoFeeSchedule = errors.New("no fee schedule valid at this date")

// FeeSchedules is a list of fee schedule versions.
type FeeSchedules []FeeSchedule

// At returns the version valid at date (the latest one starting on or before it).
func (s FeeSchedules) At(date time.Time) (*FeeSchedule, error) {
	day := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	var found *FeeSchedule
	for i := range s {
		if !s[i].ValidFrom.After(day) && (found == nil || s[i].ValidFrom.After(found.ValidFrom)) {
			found = &s[i]
		}
	}
	if found == nil {
		return nil, fmt.Errorf("%w: %s", ErrNoFeeSchedule, day.Format("2006-01-02"))
	}
	return found, nil
}

// Validate checks that the configuration is complete and consistent.
func (c FeeScheduleConfig) Validate() error {
	var problems []string
	if c.FreeIncomeLimit < 0 {
		problems = append(problems, "Beitragsfrei-Grenze darf nicht negativ sein")
	}
	if c.EntlastungIncomeLimit < c.FreeIncomeLimit {
		problems = append(problems, "Entlastungs-Grenze muss mindestens so hoch wie die Beitragsfrei-Grenze sein")
	}
	problems = append(problems, validateFeeTable("Entlastungstabelle", c.EntlastungTable)...)
	problems = append(problems, validateFeeTable("Satzungstabelle", c.SatzungTable)...)
	if len(c.KindergartenTable) > 0 {
		problems = append(problems, validateFeeTable("Kindergartentabelle", c.KindergartenTable)...)
	}
	if len(c.SiblingDiscountFactors) == 0 {
		problems = append(problems, "Geschwisterermäßigung braucht mindestens einen Faktor")
	}
	for i, f := range c.SiblingDiscountFactors {
		if f <= 0 || f > 1 {
			problems = append(problems, fmt.Sprintf("Geschwisterfaktor für %d Kinder muss zwischen 0 und 1 liegen", i+1))
		}
	}
	if c.SiblingsFreeThreshold < 1 {
		problems = append(problems, "Beitragsfreiheit ab Kinderzahl muss mindestens 1 sein")
	}
	if c.MonthlyFoodFee < 0 || c.AnnualMembershipFee < 0 {
		problems = append(problems, "Essensgeld und Mitgliedsbeitrag dürfen nicht negativ sein")
	}
	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}

func validateFeeTable(name string, table []FeeTableRow) []string {
	if len(table) == 0 {
		return []string{name + " braucht mindestens eine Zeile"}
	}
	var problems []string
	for i, row := range table {
		if i > 0 && row.MinIncome <= table[i-1].MinIncome {
			problems = append(problems, fmt.Sprintf("%s: Einkommensgrenzen müssen aufsteigend sein (Zeile %d)", name, i+1))
		}
		for _, rate := range row.Rates {
			if rate < 0 {
				problems = append(problems, fmt.Sprintf("%s: negative Beträge in Zeile %d", name, i+1))
				break
			}
		}
	}
	return problems
}

// Normalize sorts the tables by income so lookups work regardless of input order.
func (c *FeeScheduleConfig) Normalize() {
	sort.SliceStable(c.EntlastungTable, func(i, j int) bool { return c.EntlastungTable[i].MinIncome < c.EntlastungTable[j].MinIncome })
	sort.SliceStable(c.SatzungTable, func(i, j int) bool { return c.SatzungTable[i].MinIncome < c.SatzungTable[j].MinIncome })
	sort.SliceStable(c.KindergartenTable, func(i, j int) bool {
		return c.KindergartenTable[i].MinIncome < c.KindergartenTable[j].MinIncome
	})
}

// ChildAgeType represents whether a child is in Krippe or Kindergarten.
type ChildAgeType string

const (
	ChildAgeTypeKrippe       ChildAgeType = "krippe"       // Under 3 years
	ChildAgeTypeKindergarten ChildAgeType = "kindergarten" // 3 years and older
)

// ChildcareFeeInput represents the input for childcare fee calculation.
type ChildcareFeeInput struct {
	ChildAgeType  ChildAgeType `json:"childAgeType"`  // "krippe" or "kindergarten"
	NetIncome     float64      `json:"netIncome"`     // Annual net household income
	SiblingsCount int          `json:"siblingsCount"` // Number of children in household (including this child)
	CareHours     int          `json:"careHours"`     // Weekly care hours (30, 35, 40, 45, 50, or 55)
	HighestRate   bool         `json:"highestRate"`   // Voluntarily pay highest rate (no income check)
	FosterFamily  bool         `json:"fosterFamily"`  // Foster family: fee is average of all Satzung rates
}

// ChildcareFeeResult represents the result of childcare fee calculation.
type ChildcareFeeResult struct {
	Fee             float64  `json:"fee"`             // Final fee after discounts
	BaseFee         float64  `json:"baseFee"`         // Base fee before discounts
	Rule            string   `json:"rule"`            // Rule/bracket applied
	DiscountFactor  float64  `json:"discountFactor"`  // Sibling discount factor (1.0 = no discount)
	DiscountPercent int      `json:"discountPercent"` // Discount as percentage
	ShowEntlastung  bool     `json:"showEntlastung"`  // Show link to Entlastung info
	Notes           []string `json:"notes"`           // Additional explanatory notes
}

// CalculateChildcareFee calculates the monthly childcare fee (Platzgeld) based on
// income, care hours, number of siblings and child age type.
func (c FeeScheduleConfig) CalculateChildcareFee(input ChildcareFeeInput) *ChildcareFeeResult {
	// Default values
	if input.SiblingsCount < 1 {
		input.SiblingsCount = 1
	}
	if input.CareHours == 0 {
		input.CareHours = 30
	}

	// Kindergarten (>= 3 years) is free in Brandenburg
	if input.ChildAgeType == ChildAgeTypeKindergarten {
		return &ChildcareFeeResult{
			Rule:           "Beitragsfrei (ab 3 Jahren)",
			DiscountFactor: 1.0,
			Notes:          []string{"Die Betreuung im Kindergartenalter ist in Brandenburg beitragsfrei."},
		}
	}

	// Krippe (< 3 years)

	// Foster family: average of all Satzung rates for the care hours (no sibling discount)
	if input.FosterFamily {
		avgFee := c.averageSatzungRate(input.CareHours)
		return &ChildcareFeeResult{
			Fee:            roundToTwoDecimals(avgFee),
			BaseFee:        avgFee,
			Rule:           "Pflegefamilie (Durchschnittsbeitrag)",
			DiscountFactor: 1.0,
			Notes:          []string{"Beitrag ist der Durchschnitt aller Sätze für die entsprechende Betreuungszeit."},
		}
	}

	// Many children: free
	if input.SiblingsCount >= c.SiblingsFreeThreshold {
		return &ChildcareFeeResult{
			Rule:           fmt.Sprintf("Beitragsfrei (≥ %d Kinder)", c.SiblingsFreeThreshold),
			DiscountFactor: 1.0,
			Notes:          []string{fmt.Sprintf("Bei %d oder mehr unterhaltsberechtigten Kindern entfällt der Elternbeitrag.", c.SiblingsFreeThreshold)},
		}
	}

	// Highest rate voluntarily chosen (no income check, but sibling discount applies)
	if input.HighestRate {
		lastRow := c.SatzungTable[len(c.SatzungTable)-1]
		return c.satzungResult(lastRow.Rates[hoursToIndex(input.CareHours)], input.SiblingsCount, "Höchstsatz (Satzung U3)")
	}

	if input.NetIncome <= c.FreeIncomeLimit {
		return &ChildcareFeeResult{
			Rule:           fmt.Sprintf("Beitragsfrei (Einkommen ≤ %s EUR)", formatGermanAmount(c.FreeIncomeLimit)),
			DiscountFactor: 1.0,
			ShowEntlastung: true,
			Notes:          []string{"Gemäß Elternbeitragsentlastungsgesetz."},
		}
	}

	// Entlastung bracket (no sibling discount)
	if input.NetIncome <= c.EntlastungIncomeLimit {
		baseFee := findRateInTable(c.EntlastungTable, input.NetIncome, input.CareHours)
		return &ChildcareFeeResult{
			Fee:            baseFee,
			BaseFee:        baseFee,
			Rule:           "Reduzierter Beitrag (Entlastung U3)",
			DiscountFactor: 1.0,
			ShowEntlastung: true,
			Notes: []string{
				"Kein zusätzlicher Geschwisterrabatt in diesem Einkommensbereich.",
				"Rechtsgrundlage: Elternbeitragsentlastungsgesetz.",
			},
		}
	}

	// Satzung bracket (sibling discount applies)
	baseFee := findRateInTable(c.SatzungTable, input.NetIncome, input.CareHours)
	return c.satzungResult(baseFee, input.SiblingsCount, "Regulärer Beitrag (Satzung U3)")
}

func (c FeeScheduleConfig) satzungResult(baseFee float64, siblingsCount int, rule string) *ChildcareFeeResult {
	discountFactor := c.siblingDiscountFactor(siblingsCount)
	notes := []string{}
	if siblingsCount > 1 && discountFactor < 1.0 {
		notes = append(notes, "Geschwisterermäßigung berücksichtigt.")
	}
	return &ChildcareFeeResult{
		Fee:             roundToTwoDecimals(baseFee * discountFactor),
		BaseFee:         baseFee,
		Rule:            rule,
		DiscountFactor:  discountFactor,
		DiscountPercent: int(math.Round((1 - discountFactor) * 100)),
		Notes:           notes,
	}
}

// siblingDiscountFactor returns the factor for the number of children; counts
// beyond the configured factors use the last one.
func (c FeeScheduleConfig) siblingDiscountFactor(siblingsCount int) float64 {
	if len(c.SiblingDiscountFactors) == 0 || siblingsCount < 1 {
		return 1.0
	}
	if siblingsCount > len(c.SiblingDiscountFactors) {
		siblingsCount = len(c.SiblingDiscountFactors)
	}
	return c.SiblingDiscountFactors[siblingsCount-1]
}

// averageSatzungRate is the average of all Satzung rates for the care hours
// (foster family rate).
func (c FeeScheduleConfig) averageSatzungRate(hours int) float64 {
	if len(c.SatzungTable) == 0 {
		return 0
	}
	idx := hoursToIndex(hours)
	var sum float64
	for _, row := range c.SatzungTable {
		sum += row.Rates[idx]
	}
	return sum / float64(len(c.SatzungTable))
}

// hoursToIndex maps care hours (30, 35, 40, 45, 50, 55) to a table column (0-5).
func hoursToIndex(hours int) int {
	idx := (hours - 30) / 5
	if idx < 0 {
		return 0
	}
	if idx > 5 {
		return 5
	}
	return idx
}

// findRateInTable returns the rate of the highest bracket whose MinIncome <= income.
func findRateInTable(table []FeeTableRow, income float64, hours int) float64 {
	idx := hoursToIndex(hours)
	for i := len(table) - 1; i >= 0; i-- {
		if income >= table[i].MinIncome {
			return table[i].Rates[idx]
		}
	}
	return 0
}

func roundToTwoDecimals(val float64) float64 {
	return float64(int(val*100+0.5)) / 100
}

// formatGermanAmount formats 35000 as "35.000" and 35000.5 as "35.000,50".
func formatGermanAmount(v float64) string {
	cents := int64(math.Round(v * 100))
	whole := strconv.FormatInt(cents/100, 10)
	var b strings.Builder
	for i, r := range whole {
		if i > 0 && (len(whole)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}
	if frac := cents % 100; frac != 0 {
		fmt.Fprintf(&b, ",%02d", frac)
	}
	return b.String()
}

// Scan implements the sql.Scanner interface for reading JSONB from PostgreSQL.
func (c *FeeScheduleConfig) Scan(src interface{}) error {
	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into FeeScheduleConfig", src)
	}
	return json.Unmarshal(data, c)
}

// Value implements the driver.Valuer interface for writing JSONB to PostgreSQL.
func (c FeeScheduleConfig) Value() (driver.Value, error) {
	return json.Marshal(c)
}
