package service

import (
	"math"
	"testing"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
)

func TestCalculateChildcareFee(t *testing.T) {
	s := &FeeService{} // No dependencies needed for fee calculation

	tests := []struct {
		name            string
		input           domain.ChildcareFeeInput
		expectedFee     float64
		expectedBaseFee float64
		expectedRule    string
		tolerance       float64 // Allow small floating point differences
	}{
		// Kindergarten is always free
		{
			name: "Kindergarten is always free regardless of income",
			input: domain.ChildcareFeeInput{
				ChildAgeType:  domain.ChildAgeTypeKindergarten,
				NetIncome:     100000,
				SiblingsCount: 1,
				CareHours:     45,
			},
			expectedFee:     0,
			expectedBaseFee: 0,
			expectedRule:    "Beitragsfrei (ab 3 Jahren)",
			tolerance:       0.01,
		},

		// 7+ children is always free
		{
			name: "7 or more children is free",
			input: domain.ChildcareFeeInput{
				ChildAgeType:  domain.ChildAgeTypeKrippe,
				NetIncome:     80000,
				SiblingsCount: 7,
				CareHours:     45,
			},
			expectedFee:     0,
			expectedBaseFee: 0,
			expectedRule:    "Beitragsfrei (≥ 7 Kinder)",
			tolerance:       0.01,
		},
		{
			name: "8 children is also free",
			input: domain.ChildcareFeeInput{
				ChildAgeType:  domain.ChildAgeTypeKrippe,
				NetIncome:     100000,
				SiblingsCount: 8,
				CareHours:     55,
			},
			expectedFee:     0,
			expectedBaseFee: 0,
			expectedRule:    "Beitragsfrei (≥ 7 Kinder)",
			tolerance:       0.01,
		},

		// Income <= 35,000 is free
		{
			name: "Income 35000 is free",
			input: domain.ChildcareFeeInput{
				ChildAgeType:  domain.ChildAgeTypeKrippe,
				NetIncome:     35000,
				SiblingsCount: 1,
				CareHours:     45,
			},
			expectedFee:     0,
			expectedBaseFee: 0,
			expectedRule:    "Beitragsfrei (Einkommen ≤ 35.000 EUR)",
			tolerance:       0.01,
		},
		{
			name: "Income 30000 is free",
			input: domain.ChildcareFeeInput{
				ChildAgeType:  domain.ChildAgeTypeKrippe,
				NetIncome:     30000,
				SiblingsCount: 1,
				CareHours:     45,
			},
			expectedFee:     0,
			expectedBaseFee: 0,
			expectedRule:    "Beitragsfrei (Einkommen ≤ 35.000 EUR)",
			tolerance:       0.01,
		},
		{
			name: "Income 0 is free",
			input: domain.ChildcareFeeInput{
				ChildAgeType:  domain.ChildAgeTypeKrippe,
				NetIncome:     0,
				SiblingsCount: 1,
				CareHours:     45,
			},
			expectedFee:     0,
			expectedBaseFee: 0,
			expectedRule:    "Beitragsfrei (Einkommen ≤ 35.000 EUR)",
			tolerance:       0.01,
		},

		// Entlastung bracket: 35,000.01 - 55,000 (no sibling discount)
		{
			name: "Entlastung bracket - 40000 income, 45h",
			input: domain.ChildcareFeeInput{
				ChildAgeType:  domain.ChildAgeTypeKrippe,
				NetIncome:     40000,
				SiblingsCount: 1,
				CareHours:     45,
			},
			expectedFee:     66, // From feeTableKrippeEntlastung: 35000.01 rates[3] = 66
			expectedBaseFee: 66,
			expectedRule:    "Reduzierter Beitrag (Entlastung U3)",
			tolerance:       0.01,
		},
		{
			name: "Entlastung bracket - 45000 income, 30h",
			input: domain.ChildcareFeeInput{
				ChildAgeType:  domain.ChildAgeTypeKrippe,
				NetIncome:     45000,
				SiblingsCount: 1,
				CareHours:     30,
			},
			expectedFee:     80, // From feeTableKrippeEntlastung: 40000.01 rates[0] = 80
			expectedBaseFee: 80,
			expectedRule:    "Reduzierter Beitrag (Entlastung U3)",
			tolerance:       0.01,
		},
		{
			name: "Entlastung bracket - no sibling discount even with multiple children",
			input: domain.ChildcareFeeInput{
				ChildAgeType:  domain.ChildAgeTypeKrippe,
				NetIncome:     50000,
				SiblingsCount: 3,
				CareHours:     45,
			},
			expectedFee:     165, // 45000.01 rates[3] = 165, no sibling discount
			expectedBaseFee: 165,
			expectedRule:    "Reduzierter Beitrag (Entlastung U3)",
			tolerance:       0.01,
		},
		{
			name: "Entlastung bracket - 55000 income (upper boundary)",
			input: domain.ChildcareFeeInput{
				ChildAgeType:  domain.ChildAgeTypeKrippe,
				NetIncome:     55000,
				SiblingsCount: 1,
				CareHours:     45,
			},
			expectedFee:     231, // 50000.01 rates[3] = 231
			expectedBaseFee: 231,
			expectedRule:    "Reduzierter Beitrag (Entlastung U3)",
			tolerance:       0.01,
		},

		// Satzung bracket: >= 55,000.01 (with sibling discount)
		{
			name: "Satzung bracket - 60000 income, 1 child",
			input: domain.ChildcareFeeInput{
				ChildAgeType:  domain.ChildAgeTypeKrippe,
				NetIncome:     60000,
				SiblingsCount: 1,
				CareHours:     45,
			},
			expectedFee:     381.70, // 55000.01 rates[3] = 381.70
			expectedBaseFee: 381.70,
			expectedRule:    "Regulärer Beitrag (Satzung U3)",
			tolerance:       0.01,
		},
		{
			name: "Satzung bracket - 60000 income, 2 children (10% discount)",
			input: domain.ChildcareFeeInput{
				ChildAgeType:  domain.ChildAgeTypeKrippe,
				NetIncome:     60000,
				SiblingsCount: 2,
				CareHours:     45,
			},
			expectedFee:     343.53, // 381.70 * 0.9 = 343.53
			expectedBaseFee: 381.70,
			expectedRule:    "Regulärer Beitrag (Satzung U3)",
			tolerance:       0.01,
		},
		{
			name: "Satzung bracket - 60000 income, 3 children (20% discount)",
			input: domain.ChildcareFeeInput{
				ChildAgeType:  domain.ChildAgeTypeKrippe,
				NetIncome:     60000,
				SiblingsCount: 3,
				CareHours:     45,
			},
			expectedFee:     305.36, // 381.70 * 0.8 = 305.36
			expectedBaseFee: 381.70,
			expectedRule:    "Regulärer Beitrag (Satzung U3)",
			tolerance:       0.01,
		},
		{
			name: "Satzung bracket - 60000 income, 4 children (35% discount)",
			input: domain.ChildcareFeeInput{
				ChildAgeType:  domain.ChildAgeTypeKrippe,
				NetIncome:     60000,
				SiblingsCount: 4,
				CareHours:     45,
			},
			expectedFee:     248.11, // 381.70 * 0.65 = 248.105
			expectedBaseFee: 381.70,
			expectedRule:    "Regulärer Beitrag (Satzung U3)",
			tolerance:       0.02,
		},
		{
			name: "Satzung bracket - 60000 income, 5 children (55% discount)",
			input: domain.ChildcareFeeInput{
				ChildAgeType:  domain.ChildAgeTypeKrippe,
				NetIncome:     60000,
				SiblingsCount: 5,
				CareHours:     45,
			},
			expectedFee:     171.77, // 381.70 * 0.45 = 171.765
			expectedBaseFee: 381.70,
			expectedRule:    "Regulärer Beitrag (Satzung U3)",
			tolerance:       0.02,
		},
		{
			name: "Satzung bracket - 60000 income, 6 children (75% discount)",
			input: domain.ChildcareFeeInput{
				ChildAgeType:  domain.ChildAgeTypeKrippe,
				NetIncome:     60000,
				SiblingsCount: 6,
				CareHours:     45,
			},
			expectedFee:     95.43, // 381.70 * 0.25 = 95.425
			expectedBaseFee: 381.70,
			expectedRule:    "Regulärer Beitrag (Satzung U3)",
			tolerance:       0.02,
		},

		// Highest rate option
		{
			name: "Highest rate - 1 child, 45h",
			input: domain.ChildcareFeeInput{
				ChildAgeType:  domain.ChildAgeTypeKrippe,
				NetIncome:     0, // Income ignored when highest rate selected
				SiblingsCount: 1,
				CareHours:     45,
				HighestRate:   true,
			},
			expectedFee:     381.70, // Last row of Satzung table, rates[3]
			expectedBaseFee: 381.70,
			expectedRule:    "Höchstsatz (Satzung U3)",
			tolerance:       0.01,
		},
		{
			name: "Highest rate - 2 children, 55h (with sibling discount)",
			input: domain.ChildcareFeeInput{
				ChildAgeType:  domain.ChildAgeTypeKrippe,
				NetIncome:     0,
				SiblingsCount: 2,
				CareHours:     55,
				HighestRate:   true,
			},
			expectedFee:     405.99, // 451.10 * 0.9 = 405.99
			expectedBaseFee: 451.10,
			expectedRule:    "Höchstsatz (Satzung U3)",
			tolerance:       0.01,
		},
		{
			name: "Highest rate - 30h",
			input: domain.ChildcareFeeInput{
				ChildAgeType:  domain.ChildAgeTypeKrippe,
				NetIncome:     0,
				SiblingsCount: 1,
				CareHours:     30,
				HighestRate:   true,
			},
			expectedFee:     277.60, // Last row of Satzung table, rates[0]
			expectedBaseFee: 277.60,
			expectedRule:    "Höchstsatz (Satzung U3)",
			tolerance:       0.01,
		},

		// Different care hours
		{
			name: "30 hours care",
			input: domain.ChildcareFeeInput{
				ChildAgeType:  domain.ChildAgeTypeKrippe,
				NetIncome:     60000,
				SiblingsCount: 1,
				CareHours:     30,
			},
			expectedFee:     277.60,
			expectedBaseFee: 277.60,
			expectedRule:    "Regulärer Beitrag (Satzung U3)",
			tolerance:       0.01,
		},
		{
			name: "35 hours care",
			input: domain.ChildcareFeeInput{
				ChildAgeType:  domain.ChildAgeTypeKrippe,
				NetIncome:     60000,
				SiblingsCount: 1,
				CareHours:     35,
			},
			expectedFee:     312.30,
			expectedBaseFee: 312.30,
			expectedRule:    "Regulärer Beitrag (Satzung U3)",
			tolerance:       0.01,
		},
		{
			name: "40 hours care",
			input: domain.ChildcareFeeInput{
				ChildAgeType:  domain.ChildAgeTypeKrippe,
				NetIncome:     60000,
				SiblingsCount: 1,
				CareHours:     40,
			},
			expectedFee:     347.00,
			expectedBaseFee: 347.00,
			expectedRule:    "Regulärer Beitrag (Satzung U3)",
			tolerance:       0.01,
		},
		{
			name: "50 hours care",
			input: domain.ChildcareFeeInput{
				ChildAgeType:  domain.ChildAgeTypeKrippe,
				NetIncome:     60000,
				SiblingsCount: 1,
				CareHours:     50,
			},
			expectedFee:     416.40,
			expectedBaseFee: 416.40,
			expectedRule:    "Regulärer Beitrag (Satzung U3)",
			tolerance:       0.01,
		},
		{
			name: "55 hours care",
			input: domain.ChildcareFeeInput{
				ChildAgeType:  domain.ChildAgeTypeKrippe,
				NetIncome:     60000,
				SiblingsCount: 1,
				CareHours:     55,
			},
			expectedFee:     451.10,
			expectedBaseFee: 451.10,
			expectedRule:    "Regulärer Beitrag (Satzung U3)",
			tolerance:       0.01,
		},

		// Boundary tests for Satzung income brackets
		{
			name: "Satzung - income 22000 bracket",
			input: domain.ChildcareFeeInput{
				ChildAgeType:  domain.ChildAgeTypeKrippe,
				NetIncome:     56000,
				SiblingsCount: 1,
				CareHours:     45,
				HighestRate:   true, // Use highest rate to test Satzung table
			},
			expectedFee:     381.70,
			expectedBaseFee: 381.70,
			expectedRule:    "Höchstsatz (Satzung U3)",
			tolerance:       0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.CalculateChildcareFee(tt.input)

			if math.Abs(result.Fee-tt.expectedFee) > tt.tolerance {
				t.Errorf("Fee: got %v, want %v (tolerance %v)", result.Fee, tt.expectedFee, tt.tolerance)
			}

			if math.Abs(result.BaseFee-tt.expectedBaseFee) > tt.tolerance {
				t.Errorf("BaseFee: got %v, want %v (tolerance %v)", result.BaseFee, tt.expectedBaseFee, tt.tolerance)
			}

			if result.Rule != tt.expectedRule {
				t.Errorf("Rule: got %q, want %q", result.Rule, tt.expectedRule)
			}
		})
	}
}

func TestCalculateChildcareFee_DiscountFactors(t *testing.T) {
	s := &FeeService{}

	tests := []struct {
		siblingsCount           int
		expectedDiscountFactor  float64
		expectedDiscountPercent int
	}{
		{1, 1.0, 0},
		{2, 0.9, 10},
		{3, 0.8, 20},
		{4, 0.65, 35},
		{5, 0.45, 55},
		{6, 0.25, 75},
	}

	for _, tt := range tests {
		t.Run("siblings_"+string(rune('0'+tt.siblingsCount)), func(t *testing.T) {
			result := s.CalculateChildcareFee(domain.ChildcareFeeInput{
				ChildAgeType:  domain.ChildAgeTypeKrippe,
				NetIncome:     60000, // Satzung bracket
				SiblingsCount: tt.siblingsCount,
				CareHours:     45,
			})

			if math.Abs(result.DiscountFactor-tt.expectedDiscountFactor) > 0.001 {
				t.Errorf("DiscountFactor: got %v, want %v", result.DiscountFactor, tt.expectedDiscountFactor)
			}

			// Allow 1% tolerance for rounding
			if result.DiscountPercent != tt.expectedDiscountPercent && result.DiscountPercent != tt.expectedDiscountPercent-1 {
				t.Errorf("DiscountPercent: got %v, want %v", result.DiscountPercent, tt.expectedDiscountPercent)
			}
		})
	}
}

func TestCalculateChildcareFee_ShowEntlastung(t *testing.T) {
	s := &FeeService{}

	tests := []struct {
		name           string
		input          domain.ChildcareFeeInput
		showEntlastung bool
	}{
		{
			name: "Free due to low income - show Entlastung link",
			input: domain.ChildcareFeeInput{
				ChildAgeType:  domain.ChildAgeTypeKrippe,
				NetIncome:     30000,
				SiblingsCount: 1,
				CareHours:     45,
			},
			showEntlastung: true,
		},
		{
			name: "Entlastung bracket - show Entlastung link",
			input: domain.ChildcareFeeInput{
				ChildAgeType:  domain.ChildAgeTypeKrippe,
				NetIncome:     45000,
				SiblingsCount: 1,
				CareHours:     45,
			},
			showEntlastung: true,
		},
		{
			name: "Satzung bracket - don't show Entlastung link",
			input: domain.ChildcareFeeInput{
				ChildAgeType:  domain.ChildAgeTypeKrippe,
				NetIncome:     60000,
				SiblingsCount: 1,
				CareHours:     45,
			},
			showEntlastung: false,
		},
		{
			name: "Highest rate - don't show Entlastung link",
			input: domain.ChildcareFeeInput{
				ChildAgeType:  domain.ChildAgeTypeKrippe,
				NetIncome:     0,
				SiblingsCount: 1,
				CareHours:     45,
				HighestRate:   true,
			},
			showEntlastung: false,
		},
		{
			name: "Kindergarten - don't show Entlastung link",
			input: domain.ChildcareFeeInput{
				ChildAgeType:  domain.ChildAgeTypeKindergarten,
				NetIncome:     0,
				SiblingsCount: 1,
				CareHours:     45,
			},
			showEntlastung: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.CalculateChildcareFee(tt.input)

			if result.ShowEntlastung != tt.showEntlastung {
				t.Errorf("ShowEntlastung: got %v, want %v", result.ShowEntlastung, tt.showEntlastung)
			}
		})
	}
}

func TestCalculateChildcareFee_DefaultValues(t *testing.T) {
	s := &FeeService{}

	// Test with zero/default values
	result := s.CalculateChildcareFee(domain.ChildcareFeeInput{
		ChildAgeType:  domain.ChildAgeTypeKrippe,
		NetIncome:     0,
		SiblingsCount: 0, // Should default to 1
		CareHours:     0, // Should default to 30
	})

	// With 0 income, should be free
	if result.Fee != 0 {
		t.Errorf("Fee with 0 income should be 0, got %v", result.Fee)
	}
}

func TestHoursToIndex(t *testing.T) {
	tests := []struct {
		hours    int
		expected int
	}{
		{30, 0},
		{35, 1},
		{40, 2},
		{45, 3},
		{50, 4},
		{55, 5},
		{25, 0}, // Below minimum, clamp to 0
		{60, 5}, // Above maximum, clamp to 5
		{32, 0}, // Rounds down
		{38, 1}, // Rounds to nearest
	}

	for _, tt := range tests {
		t.Run("hours_"+string(rune('0'+tt.hours/10))+string(rune('0'+tt.hours%10)), func(t *testing.T) {
			result := hoursToIndex(tt.hours)
			if result != tt.expected {
				t.Errorf("hoursToIndex(%d): got %d, want %d", tt.hours, result, tt.expected)
			}
		})
	}
}

func TestFindRateInTable(t *testing.T) {
	// Test with Entlastung table
	tests := []struct {
		income   float64
		hours    int
		expected float64
	}{
		// First bracket: 35000.01
		{35000.01, 30, 48.00},
		{35000.01, 45, 66.00},
		{38000.00, 30, 48.00}, // Still in first bracket

		// Second bracket: 40000.01
		{40000.01, 30, 80.00},
		{40000.01, 45, 110.00},

		// Third bracket: 45000.01
		{45000.01, 30, 120.00},
		{45000.01, 55, 195.00},

		// Fourth bracket: 50000.01
		{50000.01, 30, 168.00},
		{55000.00, 55, 273.00}, // Upper boundary still in this bracket

		// Below minimum
		{35000.00, 30, 0}, // Exactly at or below 35000 should return 0
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := findRateInTable(domain.FeeTableKrippeEntlastung, tt.income, tt.hours)
			if math.Abs(result-tt.expected) > 0.01 {
				t.Errorf("findRateInTable(Entlastung, %v, %d): got %v, want %v", tt.income, tt.hours, result, tt.expected)
			}
		})
	}
}

func TestGetSiblingDiscountFactor(t *testing.T) {
	tests := []struct {
		siblings int
		expected float64
	}{
		{1, 1.0},
		{2, 0.9},
		{3, 0.8},
		{4, 0.65},
		{5, 0.45},
		{6, 0.25},
		{7, 0.25},  // Capped at 6
		{10, 0.25}, // Capped at 6
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := getSiblingDiscountFactor(tt.siblings, 6)
			if math.Abs(result-tt.expected) > 0.001 {
				t.Errorf("getSiblingDiscountFactor(%d): got %v, want %v", tt.siblings, result, tt.expected)
			}
		})
	}
}
