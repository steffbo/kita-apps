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
type U3IncomeBreakdown struct {
	UpTo20k      int `json:"upTo20k"`      // income ≤20,000
	From20To35k  int `json:"from20To35k"`  // >20,000 && ≤35,000
	From35To55k  int `json:"from35To55k"`  // >35,000 && ≤55,000
	MaxAccepted  int `json:"maxAccepted"`  // income_status = MAX_ACCEPTED
	FosterFamily int `json:"fosterFamily"` // income_status = FOSTER_FAMILY
	Total        int `json:"total"`        // total U3 children
}

// U3ChildDetail contains details of a U3 child for the Stichtagsmeldung modal.
type U3ChildDetail struct {
	ID              string  `json:"id"`
	MemberNumber    string  `json:"memberNumber"`
	FirstName       string  `json:"firstName"`
	LastName        string  `json:"lastName"`
	BirthDate       string  `json:"birthDate"`
	HouseholdIncome *int    `json:"householdIncome"`
	IncomeStatus    *string `json:"incomeStatus"`
	IsFosterFamily  bool    `json:"isFosterFamily"`
}
