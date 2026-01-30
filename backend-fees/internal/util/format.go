package util

import "github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"

// MonthToGerman converts month number (1-12) to German name.
func MonthToGerman(month int) string {
	months := []string{
		"", "Januar", "Februar", "März", "April", "Mai", "Juni",
		"Juli", "August", "September", "Oktober", "November", "Dezember",
	}
	if month >= 1 && month <= 12 {
		return months[month]
	}
	return ""
}

// FeeTypeToGerman converts fee type to German label.
func FeeTypeToGerman(ft domain.FeeType) string {
	switch ft {
	case domain.FeeTypeFood:
		return "Essensgeld"
	case domain.FeeTypeMembership:
		return "Vereinsbeitrag"
	case domain.FeeTypeChildcare:
		return "Platzgeld"
	case domain.FeeTypeReminder:
		return "Mahngebühr"
	default:
		return string(ft)
	}
}
