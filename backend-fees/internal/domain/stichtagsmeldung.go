package domain

import "time"

// StichtagsmeldungStats contains statistics for the Stichtagsmeldung report.
type StichtagsmeldungStats struct {
	NextStichtag        time.Time         `json:"nextStichtag"`
	DaysUntilStichtag   int               `json:"daysUntilStichtag"`
	U3IncomeBreakdown   U3IncomeBreakdown `json:"u3IncomeBreakdown"`
	TotalChildrenInKita int               `json:"totalChildrenInKita"`
}

// StichtagsmeldungReport contains the report data for a specific date.
type StichtagsmeldungReport struct {
	ReportDate          time.Time             `json:"reportDate"`
	U3IncomeBreakdown   U3IncomeBreakdown     `json:"u3IncomeBreakdown"`
	TotalChildrenInKita int                   `json:"totalChildrenInKita"`
	U3ChildrenCount     int                   `json:"u3ChildrenCount"`
	Ue3ChildrenCount    int                   `json:"ue3ChildrenCount"`
	CareHoursBreakdown  []CareHoursBreakdown  `json:"careHoursBreakdown"`
	LegalHoursBreakdown []LegalHoursBreakdown `json:"legalHoursBreakdown"`
}

// CareHoursBreakdown groups children by their contracted care hours at a given date.
type CareHoursBreakdown struct {
	CareHours *int `json:"careHours" binding:"optional"`
	Count     int  `json:"count"`
	U3Count   int  `json:"u3Count"`
	Ue3Count  int  `json:"ue3Count"`
}

// LegalHoursBreakdown groups children by their legal entitlement hours at a given date.
type LegalHoursBreakdown struct {
	LegalHours *int `json:"legalHours" binding:"optional"`
	Count      int  `json:"count"`
	U3Count    int  `json:"u3Count"`
	Ue3Count   int  `json:"ue3Count"`
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
	HouseholdIncome *int    `json:"householdIncome" binding:"optional"`
	IncomeStatus    *string `json:"incomeStatus" binding:"optional"`
	IsFosterFamily  bool    `json:"isFosterFamily"`
}

// KitaYearGroup classifies a child for the quarterly Stichtagsmeldung to the
// Landkreis (Jugendamt) by age and position relative to school entry.
type KitaYearGroup string

const (
	KitaYearGroupU3 KitaYearGroup = "U3"
	// KitaYearGroupOtherUe3 are children from 3 years who are neither in the last
	// nor in the second-to-last Kita year (form column "3. Lebensjahr").
	KitaYearGroupOtherUe3     KitaYearGroup = "OTHER_UE3"
	KitaYearGroupSecondToLast KitaYearGroup = "SECOND_TO_LAST_YEAR"
	KitaYearGroupLast         KitaYearGroup = "LAST_YEAR"
	KitaYearGroupDeferred     KitaYearGroup = "DEFERRED"
)

// SchoolEntryYear returns the year in whose August the child becomes schulpflichtig
// in Brandenburg (§ 37 BbgSchulG): the sixth year of life must be completed by
// 30 September. By § 188 Abs. 2 BGB it is completed on the day before the sixth
// birthday, so a child born on 1 October still starts school in that year.
func SchoolEntryYear(birthDate time.Time) int {
	sixth := birthDate.Year() + 6
	if birthDate.Month() < time.October || (birthDate.Month() == time.October && birthDate.Day() == 1) {
		return sixth
	}
	return sixth + 1
}

// KitaYearStart returns the year in which the Kita year (1 August – 31 July)
// containing the given date starts.
func KitaYearStart(date time.Time) int {
	if date.Month() >= time.August {
		return date.Year()
	}
	return date.Year() - 1
}

// ClassifyKitaYear places a child enrolled at the Stichtag into one group.
// Children who by age should already attend school but are still enrolled count
// as deferred (zurückgestellt); the report also counts them in the last Kita year.
func ClassifyKitaYear(birthDate, stichtag time.Time) KitaYearGroup {
	if stichtag.Before(birthDate.AddDate(3, 0, 0)) {
		return KitaYearGroupU3
	}
	switch entry, year := SchoolEntryYear(birthDate), KitaYearStart(stichtag); {
	case entry <= year:
		return KitaYearGroupDeferred
	case entry == year+1:
		return KitaYearGroupLast
	case entry == year+2:
		return KitaYearGroupSecondToLast
	default:
		return KitaYearGroupOtherUe3
	}
}

// JugendamtReport holds the counts for the quarterly Stichtagsmeldung to the
// Landkreis (Belegung and § 17a KitaG). It contains no fee or income data.
type JugendamtReport struct {
	Stichtag         time.Time        `json:"stichtag"`
	DaysUntil        int              `json:"daysUntil"` // negative once the Stichtag has passed
	Total            int              `json:"total"`
	U3               int              `json:"u3"`
	Ue3              int              `json:"ue3"`
	LastYear         int              `json:"lastYear"` // includes Deferred, as on the § 17a form
	Deferred         int              `json:"deferred"`
	SecondToLastYear int              `json:"secondToLastYear"`
	OtherUe3         int              `json:"otherUe3"`
	Children         []JugendamtChild `json:"children"`
}

// JugendamtChild lists one child enrolled at the Stichtag with its group.
type JugendamtChild struct {
	ID        string        `json:"id" db:"id"`
	FirstName string        `json:"firstName" db:"first_name"`
	LastName  string        `json:"lastName" db:"last_name"`
	BirthDate time.Time     `json:"birthDate" db:"birth_date"`
	Group     KitaYearGroup `json:"group" db:"-"`
}

// Add counts the child into the report.
func (r *JugendamtReport) Add(child JugendamtChild) {
	child.Group = ClassifyKitaYear(child.BirthDate, r.Stichtag)
	r.Children = append(r.Children, child)
	r.Total++
	switch child.Group {
	case KitaYearGroupU3:
		r.U3++
		return
	case KitaYearGroupDeferred:
		r.Deferred++
		r.LastYear++
	case KitaYearGroupLast:
		r.LastYear++
	case KitaYearGroupSecondToLast:
		r.SecondToLastYear++
	default:
		r.OtherUe3++
	}
	r.Ue3++
}

// StichtagForJugendamt returns the Stichtag the dashboard shows: the most recent
// one (1 March, June, September, December) while its report is still being
// prepared, i.e. up to ReportWindowDays after it, otherwise the next one.
func StichtagForJugendamt(today time.Time) time.Time {
	first := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, today.Location())
	recent := first.AddDate(0, -(int(first.Month()) % 3), 0)
	if today.Sub(recent) <= ReportWindowDays*24*time.Hour {
		return recent
	}
	return recent.AddDate(0, 3, 0)
}

// ReportWindowDays is how long after a Stichtag the dashboard keeps showing it.
const ReportWindowDays = 30
