package service

// LegacyCalculateChildcareFee is exported for the service_test package.
//
// Frozen copy of the childcare fee calculation and tables as they were
// hard-coded before fee schedules became configurable (2026-09). It is the
// reference that the schedule-based calculation must reproduce exactly.
// Do not change it.

import (
	"math"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
)

// legacyChildcareFeeLimits defines income limits for fee calculation.
var legacyChildcareFeeLimits = struct {
	MinIncomeFreeU3       float64 // Income <= this is free
	MinIncomeEntlastungU3 float64 // Start of "Entlastung" bracket
	MaxIncomeEntlastungU3 float64 // End of "Entlastung" bracket
	MinIncomeSatzungU3    float64 // Start of "Satzung" bracket
}{
	MinIncomeFreeU3:       35000.00,
	MinIncomeEntlastungU3: 35000.01,
	MaxIncomeEntlastungU3: 55000.00,
	MinIncomeSatzungU3:    55000.01,
}

// legacyChildcareFeeMeta defines metadata constants.
var legacyChildcareFeeMeta = struct {
	KigaAgeThreshold       int
	MaxSiblingsForDiscount int
	SiblingsFreeThreshold  int
}{
	KigaAgeThreshold:       3,
	MaxSiblingsForDiscount: 6,
	SiblingsFreeThreshold:  7,
}

// legacyFeeTableRow represents a row in the fee table.
type legacyFeeTableRow struct {
	MinIncome float64
	Rates     [6]float64 // Rates for 30, 35, 40, 45, 50, 55 hours
}

// legacyFeeTableKrippeSatzung is the fee table for U3 children (regular/Satzung bracket).
// Income > 55,000 EUR or income >= 20,000.01 when voluntarily choosing highest rate.
var legacyFeeTableKrippeSatzung = []legacyFeeTableRow{
	{MinIncome: 20000.01, Rates: [6]float64{55.52, 62.46, 69.40, 76.34, 83.28, 90.22}},
	{MinIncome: 22000.00, Rates: [6]float64{77.73, 87.44, 97.16, 106.88, 116.59, 126.31}},
	{MinIncome: 25000.00, Rates: [6]float64{107.25, 120.66, 134.07, 147.48, 160.88, 174.29}},
	{MinIncome: 28000.00, Rates: [6]float64{141.32, 158.99, 176.65, 194.32, 211.99, 229.65}},
	{MinIncome: 31000.00, Rates: [6]float64{156.47, 176.02, 195.58, 215.14, 234.70, 254.26}},
	{MinIncome: 34000.00, Rates: [6]float64{171.61, 193.06, 214.51, 235.96, 257.41, 278.86}},
	{MinIncome: 37000.00, Rates: [6]float64{186.75, 210.09, 233.44, 256.78, 280.12, 303.47}},
	{MinIncome: 40000.00, Rates: [6]float64{201.89, 227.13, 252.36, 277.60, 302.84, 328.07}},
	{MinIncome: 43000.00, Rates: [6]float64{217.03, 244.16, 271.29, 298.42, 325.55, 352.68}},
	{MinIncome: 46000.00, Rates: [6]float64{232.17, 261.20, 290.22, 319.24, 348.26, 377.28}},
	{MinIncome: 49000.00, Rates: [6]float64{247.32, 278.23, 309.15, 340.06, 370.97, 401.89}},
	{MinIncome: 52000.00, Rates: [6]float64{262.46, 295.27, 328.07, 360.88, 393.69, 426.49}},
	{MinIncome: 55000.01, Rates: [6]float64{277.60, 312.30, 347.00, 381.70, 416.40, 451.10}},
}

// legacyFeeTableKrippeEntlastung is the fee table for U3 children (Entlastung bracket).
// Income between 35,000.01 and 55,000.00 EUR (no sibling discount in this bracket).
var legacyFeeTableKrippeEntlastung = []legacyFeeTableRow{
	{MinIncome: 35000.01, Rates: [6]float64{48.00, 54.00, 60.00, 66.00, 72.00, 78.00}},
	{MinIncome: 40000.01, Rates: [6]float64{80.00, 90.00, 100.00, 110.00, 120.00, 130.00}},
	{MinIncome: 45000.01, Rates: [6]float64{120.00, 135.00, 150.00, 165.00, 180.00, 195.00}},
	{MinIncome: 50000.01, Rates: [6]float64{168.00, 189.00, 210.00, 231.00, 252.00, 273.00}},
}

// legacylegacySiblingDiscount maps number of children to discount factor.
// 1 child = 100%, 2 children = 90%, etc.
var legacySiblingDiscount = map[int]float64{
	1: 1.00,
	2: 0.90,
	3: 0.80,
	4: 0.65,
	5: 0.45,
	6: 0.25,
}

func LegacyCalculateChildcareFee(input domain.ChildcareFeeInput) *domain.ChildcareFeeResult {
	limits := legacyChildcareFeeLimits
	meta := legacyChildcareFeeMeta

	// Default values
	if input.SiblingsCount < 1 {
		input.SiblingsCount = 1
	}
	if input.CareHours == 0 {
		input.CareHours = 30
	}

	// Kindergarten (>= 3 years) is free in Brandenburg
	if input.ChildAgeType == domain.ChildAgeTypeKindergarten {
		return &domain.ChildcareFeeResult{
			Fee:             0,
			BaseFee:         0,
			Rule:            "Beitragsfrei (ab 3 Jahren)",
			DiscountFactor:  1.0,
			DiscountPercent: 0,
			ShowEntlastung:  false,
			Notes:           []string{"Die Betreuung im Kindergartenalter ist in Brandenburg beitragsfrei."},
		}
	}

	// Krippe (< 3 years)

	// Foster family: average of all Satzung rates for the care hours (no sibling discount)
	if input.FosterFamily {
		avgFee := legacyCalculateAverageSatzungRate(input.CareHours)
		return &domain.ChildcareFeeResult{
			Fee:             legacyRoundToTwoDecimals(avgFee),
			BaseFee:         avgFee,
			Rule:            "Pflegefamilie (Durchschnittsbeitrag)",
			DiscountFactor:  1.0,
			DiscountPercent: 0,
			ShowEntlastung:  false,
			Notes:           []string{"Beitrag ist der Durchschnitt aller Sätze für die entsprechende Betreuungszeit."},
		}
	}

	// 7+ children: free
	if input.SiblingsCount >= meta.SiblingsFreeThreshold {
		return &domain.ChildcareFeeResult{
			Fee:             0,
			BaseFee:         0,
			Rule:            "Beitragsfrei (≥ 7 Kinder)",
			DiscountFactor:  1.0,
			DiscountPercent: 0,
			ShowEntlastung:  false,
			Notes:           []string{"Bei 7 oder mehr unterhaltsberechtigten Kindern entfällt der Elternbeitrag."},
		}
	}

	// Highest rate voluntarily chosen (no income check, but sibling discount applies)
	if input.HighestRate {
		lastRow := legacyFeeTableKrippeSatzung[len(legacyFeeTableKrippeSatzung)-1]
		baseFee := legacyFindRate(lastRow.Rates[:], input.CareHours)
		discountFactor := legacyGetSiblingDiscountFactor(input.SiblingsCount, meta.MaxSiblingsForDiscount)
		fee := baseFee * discountFactor
		discountPercent := int(math.Round((1 - discountFactor) * 100))

		notes := []string{}
		if input.SiblingsCount > 1 && discountFactor < 1.0 {
			notes = append(notes, "Geschwisterermäßigung berücksichtigt.")
		}

		return &domain.ChildcareFeeResult{
			Fee:             legacyRoundToTwoDecimals(fee),
			BaseFee:         baseFee,
			Rule:            "Höchstsatz (Satzung U3)",
			DiscountFactor:  discountFactor,
			DiscountPercent: discountPercent,
			ShowEntlastung:  false,
			Notes:           notes,
		}
	}

	// Income <= 35,000: free
	if input.NetIncome <= limits.MinIncomeFreeU3 {
		return &domain.ChildcareFeeResult{
			Fee:             0,
			BaseFee:         0,
			Rule:            "Beitragsfrei (Einkommen ≤ 35.000 EUR)",
			DiscountFactor:  1.0,
			DiscountPercent: 0,
			ShowEntlastung:  true,
			Notes:           []string{"Gemäß Elternbeitragsentlastungsgesetz."},
		}
	}

	// Entlastung bracket: 35,000.01 - 55,000.00 (no sibling discount)
	if input.NetIncome >= limits.MinIncomeEntlastungU3 && input.NetIncome <= limits.MaxIncomeEntlastungU3 {
		baseFee := legacyFindRateInTable(legacyFeeTableKrippeEntlastung, input.NetIncome, input.CareHours)
		return &domain.ChildcareFeeResult{
			Fee:             baseFee,
			BaseFee:         baseFee,
			Rule:            "Reduzierter Beitrag (Entlastung U3)",
			DiscountFactor:  1.0,
			DiscountPercent: 0,
			ShowEntlastung:  true,
			Notes: []string{
				"Kein zusätzlicher Geschwisterrabatt in diesem Einkommensbereich.",
				"Rechtsgrundlage: Elternbeitragsentlastungsgesetz.",
			},
		}
	}

	// Satzung bracket: >= 55,000.01 (sibling discount applies)
	if input.NetIncome >= limits.MinIncomeSatzungU3 {
		baseFee := legacyFindRateInTable(legacyFeeTableKrippeSatzung, input.NetIncome, input.CareHours)
		discountFactor := legacyGetSiblingDiscountFactor(input.SiblingsCount, meta.MaxSiblingsForDiscount)
		fee := baseFee * discountFactor
		discountPercent := int(math.Round((1 - discountFactor) * 100))

		notes := []string{}
		if input.SiblingsCount > 1 && discountFactor < 1.0 {
			notes = append(notes, "Geschwisterermäßigung berücksichtigt.")
		}

		return &domain.ChildcareFeeResult{
			Fee:             legacyRoundToTwoDecimals(fee),
			BaseFee:         baseFee,
			Rule:            "Regulärer Beitrag (Satzung U3)",
			DiscountFactor:  discountFactor,
			DiscountPercent: discountPercent,
			ShowEntlastung:  false,
			Notes:           notes,
		}
	}

	// Fallback (should not occur, covered by <= 35k)
	return &domain.ChildcareFeeResult{
		Fee:             0,
		BaseFee:         0,
		Rule:            "Beitragsfrei (Einkommen U3 < 35k)",
		DiscountFactor:  1.0,
		DiscountPercent: 0,
		ShowEntlastung:  true,
		Notes:           []string{},
	}
}

// hoursToIndex maps care hours (30, 35, 40, 45, 50, 55) to array index (0-5).
func legacyHoursToIndex(hours int) int {
	idx := (hours - 30) / 5
	if idx < 0 {
		return 0
	}
	if idx > 5 {
		return 5
	}
	return idx
}

// findRate finds the rate for the given hours from a rates array.
func legacyFindRate(rates []float64, hours int) float64 {
	idx := legacyHoursToIndex(hours)
	if idx >= 0 && idx < len(rates) {
		return rates[idx]
	}
	return 0
}

// findRateInTable finds the appropriate rate from a fee table based on income and hours.
func legacyFindRateInTable(table []legacyFeeTableRow, income float64, hours int) float64 {
	idx := legacyHoursToIndex(hours)

	// Find last bracket where income >= minIncome
	for i := len(table) - 1; i >= 0; i-- {
		if income >= table[i].MinIncome {
			return table[i].Rates[idx]
		}
	}

	return 0
}

// calculateAverageSatzungRate calculates the average of all Satzung rates for the given care hours.
// Used for foster family fee calculation.
func legacyCalculateAverageSatzungRate(hours int) float64 {
	idx := legacyHoursToIndex(hours)
	table := legacyFeeTableKrippeSatzung

	var sum float64
	for _, row := range table {
		sum += row.Rates[idx]
	}

	if len(table) == 0 {
		return 0
	}

	return sum / float64(len(table))
}

// getSiblingDiscountFactor returns the discount factor based on number of siblings.
func legacyGetSiblingDiscountFactor(siblingsCount, maxForDiscount int) float64 {
	if siblingsCount > maxForDiscount {
		siblingsCount = maxForDiscount
	}
	if factor, ok := legacySiblingDiscount[siblingsCount]; ok {
		return factor
	}
	return 1.0
}

// roundToTwoDecimals rounds a float to two decimal places.
func legacyRoundToTwoDecimals(val float64) float64 {
	return float64(int(val*100+0.5)) / 100
}
