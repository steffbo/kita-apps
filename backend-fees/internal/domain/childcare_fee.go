package domain

// ChildcareFeeLimits defines income limits for fee calculation.
var ChildcareFeeLimits = struct {
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

// ChildcareFeeMeta defines metadata constants.
var ChildcareFeeMeta = struct {
	KigaAgeThreshold       int
	MaxSiblingsForDiscount int
	SiblingsFreeThreshold  int
}{
	KigaAgeThreshold:       3,
	MaxSiblingsForDiscount: 6,
	SiblingsFreeThreshold:  7,
}

// FeeTableRow represents a row in the fee table.
type FeeTableRow struct {
	MinIncome float64
	Rates     [6]float64 // Rates for 30, 35, 40, 45, 50, 55 hours
}

// FeeTableKrippeSatzung is the fee table for U3 children (regular/Satzung bracket).
// Income > 55,000 EUR or income >= 20,000.01 when voluntarily choosing highest rate.
var FeeTableKrippeSatzung = []FeeTableRow{
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

// FeeTableKrippeEntlastung is the fee table for U3 children (Entlastung bracket).
// Income between 35,000.01 and 55,000.00 EUR (no sibling discount in this bracket).
var FeeTableKrippeEntlastung = []FeeTableRow{
	{MinIncome: 35000.01, Rates: [6]float64{48.00, 54.00, 60.00, 66.00, 72.00, 78.00}},
	{MinIncome: 40000.01, Rates: [6]float64{80.00, 90.00, 100.00, 110.00, 120.00, 130.00}},
	{MinIncome: 45000.01, Rates: [6]float64{120.00, 135.00, 150.00, 165.00, 180.00, 195.00}},
	{MinIncome: 50000.01, Rates: [6]float64{168.00, 189.00, 210.00, 231.00, 252.00, 273.00}},
}

// SiblingDiscount maps number of children to discount factor.
// 1 child = 100%, 2 children = 90%, etc.
var SiblingDiscount = map[int]float64{
	1: 1.00,
	2: 0.90,
	3: 0.80,
	4: 0.65,
	5: 0.45,
	6: 0.25,
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
