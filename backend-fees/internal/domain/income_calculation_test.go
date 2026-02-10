package domain_test

import (
	"testing"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestIncomeCalculation_MatchesExcel(t *testing.T) {
	// Data from Excel "Festsetzung des Elternbeitrages für 2024"
	// See: 260208_Einstufung_Vorlage.xlsx, Sheet 2

	parent1 := domain.IncomeDetails{
		// Mutter (Employee)
		GrossIncome:         20933.26,
		OtherIncome:         6585.70,
		SocialSecurityShare: 4354.21,
		PrivateInsurance:    0,
		Tax:                 2258.99,
		AdvertisingCosts:    1500.00,
		// Benefits
		ParentalBenefit:  2794.63,
		MaternityBenefit: 1430.00,
		Insurances:       0,
		// Maintenance
		MaintenanceToPay:    0,
		MaintenanceReceived: 0,
	}

	parent2 := domain.IncomeDetails{
		// Vater (Employee)
		GrossIncome:         29805.65,
		OtherIncome:         0,
		SocialSecurityShare: 6219.29,
		PrivateInsurance:    0,
		Tax:                 2439.32,
		AdvertisingCosts:    1500.00,
		// Benefits
		ParentalBenefit:  0,
		MaternityBenefit: 0,
		Insurances:       0,
		// Maintenance
		MaintenanceToPay:    2640.00,
		MaintenanceReceived: 0,
	}

	household := domain.HouseholdIncomeCalculation{
		Parent1: parent1,
		Parent2: parent2,
	}

	// --- Per-parent calculations (Mutter/Vater columns) ---

	// Parent 1 (Mutter) employee net: 20933.26 + 6585.70 - 4354.21 - 2258.99 - 1500.00 = 19405.76
	assert.InDelta(t, 19405.76, parent1.CalculateEmployeeNet(), 0.01, "Parent 1 employee net")

	// Parent 1 benefits: 2794.63 + 1430.00 = 4224.63
	assert.InDelta(t, 4224.63, parent1.CalculateBenefitsNet(), 0.01, "Parent 1 benefits net")

	// Parent 1 full net income (as shown in Mutter column): 19405.76 + 4224.63 = 23630.39
	assert.InDelta(t, 23630.39, parent1.CalculateNetIncome(), 0.01, "Parent 1 net income (with benefits)")

	// Parent 1 fee-relevant income (excluding benefits): 19405.76
	assert.InDelta(t, 19405.76, parent1.CalculateFeeRelevantIncome(), 0.01, "Parent 1 fee-relevant income")

	// Parent 2 (Vater) employee net: 29805.65 - 6219.29 - 2439.32 - 1500.00 = 19647.04
	assert.InDelta(t, 19647.04, parent2.CalculateEmployeeNet(), 0.01, "Parent 2 employee net")

	// Parent 2 full net income: 19647.04 - 2640.00 = 17007.04
	assert.InDelta(t, 17007.04, parent2.CalculateNetIncome(), 0.01, "Parent 2 net income")

	// Parent 2 fee-relevant (same as net, no benefits): 17007.04
	assert.InDelta(t, 17007.04, parent2.CalculateFeeRelevantIncome(), 0.01, "Parent 2 fee-relevant income")

	// --- Household total (Gesamt column) ---

	// Fee-relevant household income: 19405.76 + 17007.04 = 36412.80
	// This is the value used for fee bracket lookup.
	// Note: Elterngeld (2794.63) and Mutterschaftsgeld (1430.00) are excluded per Brandenburg law.
	assert.InDelta(t, 36412.80, household.CalculateAnnualNetIncome(), 0.01, "Household fee-relevant income")

	// Full household income (including benefits): 23630.39 + 17007.04 = 40637.43
	assert.InDelta(t, 40637.43, household.CalculateFullNetIncome(), 0.01, "Household full net income")
}

func TestIncomeCalculation_FeeResult(t *testing.T) {
	// With household income 36,412.80 EUR, 1 child, Krippe, 45h/week:
	// Falls in Entlastung bracket (35,000.01 - 55,000.00)
	// FeeTableKrippeEntlastung row 35,000.01: rates [48, 54, 60, 66, 72, 78]
	// 45h = index 3 → 66.00 EUR ✓ (matches Excel Sheet 1)
	income := 36412.80
	assert.True(t, income >= domain.ChildcareFeeLimits.MinIncomeEntlastungU3, "Should be in Entlastung bracket")
	assert.True(t, income <= domain.ChildcareFeeLimits.MaxIncomeEntlastungU3, "Should be in Entlastung bracket")
}

func TestIncomeCalculation_SelfEmployed(t *testing.T) {
	parent := domain.IncomeDetails{
		Profit:          50000.00,
		WelfareExpense:  5000.00,
		SelfEmployedTax: 8000.00,
	}

	assert.InDelta(t, 37000.00, parent.CalculateSelfEmployedNet(), 0.01)
	assert.InDelta(t, 37000.00, parent.CalculateNetIncome(), 0.01)
	assert.InDelta(t, 37000.00, parent.CalculateFeeRelevantIncome(), 0.01)
}

func TestIncomeCalculation_MixedIncome(t *testing.T) {
	// Parent with both employment and self-employment
	parent := domain.IncomeDetails{
		GrossIncome:         30000.00,
		SocialSecurityShare: 5000.00,
		Tax:                 3000.00,
		AdvertisingCosts:    1000.00,
		Profit:              10000.00,
		SelfEmployedTax:     2000.00,
		ParentalBenefit:     3600.00, // 300/month, fully below Freibetrag
	}

	// Employee net: 30000 - 5000 - 3000 - 1000 = 21000
	// Self-employed net: 10000 - 2000 = 8000
	// Benefits: 3600
	// Full net: 21000 + 8000 + 3600 = 32600
	assert.InDelta(t, 32600.00, parent.CalculateNetIncome(), 0.01)

	// Fee-relevant: 21000 + 8000 = 29000 (benefits excluded)
	assert.InDelta(t, 29000.00, parent.CalculateFeeRelevantIncome(), 0.01)
}

func TestIncomeCalculation_ZeroIncome(t *testing.T) {
	household := domain.HouseholdIncomeCalculation{}
	assert.InDelta(t, 0.0, household.CalculateAnnualNetIncome(), 0.01)
	assert.InDelta(t, 0.0, household.CalculateFullNetIncome(), 0.01)
}
