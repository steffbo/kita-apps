// Package letter renders prepared fee classifications as German business letters
// and the children lists for the Stichtagsmeldung (child_lists.go).
// The caller selects periods, supplies the basis and membership wording, and provides
// current board and bank data. Rendering does not calculate fees or access application data.
// Fixed wording and layout follow the frontend EinstufungPDF template. Liberation Serif
// is embedded; horizontal text scaling approximates CSS letter spacing. Automatic
// German hyphenation is not reproduced.
package letter

import "time"

// EinstufungLetter holds everything printed on the Einstufung letter.
type EinstufungLetter struct {
	LetterDate     time.Time
	Year           int
	ChildName      string
	MemberNumber   string
	BirthDate      time.Time
	EntryDate      time.Time
	ValidFrom      time.Time
	Provisional    bool
	Introduction   string
	BasisText      string
	Periods        []FeePeriod
	MembershipNote string
	MembershipFee  float64
	Board          []BoardMember
	Bank           BankAccount
}

// FeePeriod is one contribution period printed in the table.
type FeePeriod struct {
	Label        string
	CareType     string
	CareHours    int
	ChildcareFee float64
	FoodFee      float64
	Previous     bool
}

// BoardMember is a name and role in the letter footer.
type BoardMember struct{ Name, Role string }

// BankAccount is the bank information in the letter footer.
type BankAccount struct{ Holder, IBAN, BIC, Bank string }

// MonthRow is one billed month of the contribution timeline.
type MonthRow struct {
	Month        time.Time // First day of the month, UTC.
	CareType     string
	CareHours    int
	ChildcareFee float64
	FoodFee      float64
	Previous     bool
}
