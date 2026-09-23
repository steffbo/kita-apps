package domain_test

import "github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"

// testFeeScheduleConfig mirrors the seed of migration 000034 (the values that
// were hard-coded before fee schedules became configurable).
func testFeeScheduleConfig() domain.FeeScheduleConfig {
	return domain.FeeScheduleConfig{
		FreeIncomeLimit:       35000.00,
		EntlastungIncomeLimit: 55000.00,
		EntlastungTable: []domain.FeeTableRow{
			{MinIncome: 35000.01, Rates: [6]float64{48.00, 54.00, 60.00, 66.00, 72.00, 78.00}},
			{MinIncome: 40000.01, Rates: [6]float64{80.00, 90.00, 100.00, 110.00, 120.00, 130.00}},
			{MinIncome: 45000.01, Rates: [6]float64{120.00, 135.00, 150.00, 165.00, 180.00, 195.00}},
			{MinIncome: 50000.01, Rates: [6]float64{168.00, 189.00, 210.00, 231.00, 252.00, 273.00}},
		},
		SatzungTable: []domain.FeeTableRow{
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
		},
		SiblingDiscountFactors: []float64{1.00, 0.90, 0.80, 0.65, 0.45, 0.25},
		SiblingsFreeThreshold:  7,
		MonthlyFoodFee:         45.40,
		AnnualMembershipFee:    30.00,
	}
}
