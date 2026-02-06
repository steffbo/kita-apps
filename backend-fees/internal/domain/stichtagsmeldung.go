package domain

import "time"

// StichtagsmeldungStats contains statistics for the Stichtagsmeldung report.
type StichtagsmeldungStats struct {
	NextStichtag        time.Time         `json:"nextStichtag"`
	DaysUntilStichtag   int               `json:"daysUntilStichtag"`
	U3IncomeBreakdown   U3IncomeBreakdown `json:"u3IncomeBreakdown"`
	TotalChildrenInKita int               `json:"totalChildrenInKita"`
}

// U3IncomeBreakdown groups U3 children by household income ranges.
// Foster families are excluded from these counts.
type U3IncomeBreakdown struct {
	UpTo20k     int `json:"upTo20k"`     // income ≤20,000
	From20To35k int `json:"from20To35k"` // >20,000 && ≤35,000
	From35To55k int `json:"from35To55k"` // >35,000 && ≤55,000
	Total       int `json:"total"`       // total U3 (excluding foster families)
}
