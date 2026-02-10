package domain

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"math"
)

// IncomeDetails represents the detailed income components for one parent
// as per the "Festsetzung des Elternbeitrages" calculation sheet (Sheet 2).
// All values are annual amounts in EUR, stored as positive magnitudes.
type IncomeDetails struct {
	// Employee Income (bei AN)
	GrossIncome         float64 `json:"grossIncome"`         // Bruttoeinkommen (Jahressummen)
	OtherIncome         float64 `json:"otherIncome"`         // + sonstige Einnahmen
	SocialSecurityShare float64 `json:"socialSecurityShare"` // - AN-Anteile SV
	PrivateInsurance    float64 `json:"privateInsurance"`    // - private KV/PV
	Tax                 float64 `json:"tax"`                 // - Lst bzw Est, KiSt, SolZu
	AdvertisingCosts    float64 `json:"advertisingCosts"`    // - WK-Pauschale

	// Self-Employed Income (bei Gewerbetreibenden / Selbständigen)
	Profit          float64 `json:"profit"`          // Gewinn aus Gewerbebetrieb oder selbst. Arbeit
	WelfareExpense  float64 `json:"welfareExpense"`  // - Abgabe für persönliche Daseinsfürsorge
	SelfEmployedTax float64 `json:"selfEmployedTax"` // - Steuern (Est, KiSt, SolZu)

	// Other Benefits (NOT included in fee-relevant household income)
	ParentalBenefit  float64 `json:"parentalBenefit"`  // Elterngeld
	MaternityBenefit float64 `json:"maternityBenefit"` // Mutterschaftsgeld
	Insurances       float64 `json:"insurances"`       // - Versicherungen

	// Maintenance (Unterhalt)
	MaintenanceToPay    float64 `json:"maintenanceToPay"`    // - Unterhalt (zu zahlen)
	MaintenanceReceived float64 `json:"maintenanceReceived"` // + Unterhalt (erhalten)
}

// CalculateEmployeeNet returns the net employee income portion.
func (i IncomeDetails) CalculateEmployeeNet() float64 {
	return i.GrossIncome + i.OtherIncome - i.SocialSecurityShare - i.PrivateInsurance - i.Tax - i.AdvertisingCosts
}

// CalculateSelfEmployedNet returns the net self-employed income portion.
func (i IncomeDetails) CalculateSelfEmployedNet() float64 {
	return i.Profit - i.WelfareExpense - i.SelfEmployedTax
}

// CalculateBenefitsNet returns the net benefits portion (Elterngeld + Mutterschaftsgeld - Versicherungen).
func (i IncomeDetails) CalculateBenefitsNet() float64 {
	return i.ParentalBenefit + i.MaternityBenefit - i.Insurances
}

// CalculateSubtotal calculates the Zwischensumme for one parent (before maintenance).
// This includes ALL income components (employee + self-employed + benefits).
func (i IncomeDetails) CalculateSubtotal() float64 {
	return i.CalculateEmployeeNet() + i.CalculateSelfEmployedNet() + i.CalculateBenefitsNet()
}

// CalculateNetIncome calculates the Jahresnettoeinkommen for one parent.
// This is the per-parent total shown in the Mutter/Vater columns,
// including Elterngeld and Mutterschaftsgeld.
func (i IncomeDetails) CalculateNetIncome() float64 {
	return i.CalculateSubtotal() - i.MaintenanceToPay + i.MaintenanceReceived
}

// CalculateFeeRelevantIncome calculates the income portion relevant for fee determination.
// Per Brandenburg Elternbeitragsgesetz, Elterngeld and Mutterschaftsgeld are excluded
// from the household income used for the fee bracket lookup.
// This corresponds to the "Gesamt" column in the Excel sheet.
func (i IncomeDetails) CalculateFeeRelevantIncome() float64 {
	return i.CalculateEmployeeNet() + i.CalculateSelfEmployedNet() - i.Insurances - i.MaintenanceToPay + i.MaintenanceReceived
}

// HouseholdIncomeCalculation represents the full income calculation sheet for a household.
// It reflects the structure of the "Festsetzung des Elternbeitrages" Excel sheet (Sheet 2).
type HouseholdIncomeCalculation struct {
	Parent1 IncomeDetails `json:"parent1"`
	Parent2 IncomeDetails `json:"parent2"`
}

// CalculateAnnualNetIncome computes the fee-relevant Jahresnettoeinkommen for the household.
// This is the "Gesamt" column total, which excludes Elterngeld and Mutterschaftsgeld.
// This value is used for the fee bracket lookup in the Satzung/Entlastung tables.
func (h HouseholdIncomeCalculation) CalculateAnnualNetIncome() float64 {
	return roundTo2(h.Parent1.CalculateFeeRelevantIncome() + h.Parent2.CalculateFeeRelevantIncome())
}

// CalculateFullNetIncome computes the sum of both parents' full net income (including benefits).
// This is for display purposes only - NOT used for fee bracket lookup.
func (h HouseholdIncomeCalculation) CalculateFullNetIncome() float64 {
	return roundTo2(h.Parent1.CalculateNetIncome() + h.Parent2.CalculateNetIncome())
}

func roundTo2(v float64) float64 {
	return math.Round(v*100) / 100
}

// Scan implements the sql.Scanner interface for reading JSONB from PostgreSQL.
func (h *HouseholdIncomeCalculation) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into HouseholdIncomeCalculation", src)
	}
	return json.Unmarshal(data, h)
}

// Value implements the driver.Valuer interface for writing JSONB to PostgreSQL.
func (h HouseholdIncomeCalculation) Value() (driver.Value, error) {
	return json.Marshal(h)
}
