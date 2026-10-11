package letter

import (
	"fmt"
	"math"
	"math/big"
	"sort"
	"strconv"
	"time"
)

var monthsLong = [...]string{
	"Januar", "Februar", "März", "April", "Mai", "Juni",
	"Juli", "August", "September", "Oktober", "November", "Dezember",
}
var monthsShort = [...]string{
	"Jan.", "Feb.", "März", "Apr.", "Mai", "Juni",
	"Juli", "Aug.", "Sept.", "Okt.", "Nov.", "Dez.",
}

// FormatEuro formats an amount as "1.234,56 €".
func FormatEuro(amount float64) string {
	if math.IsNaN(amount) || math.IsInf(amount, 0) {
		return strconv.FormatFloat(amount, 'f', 2, 64) + " €"
	}
	// Round the shortest decimal representation, avoiding binary half-cent artifacts.
	decimal, _ := new(big.Rat).SetString(strconv.FormatFloat(math.Abs(amount), 'f', -1, 64))
	decimal.Mul(decimal, big.NewRat(100, 1))
	decimal.Add(decimal, big.NewRat(1, 2))
	cents := new(big.Int).Quo(decimal.Num(), decimal.Denom())
	digits := cents.String()
	for len(digits) < 3 {
		digits = "0" + digits
	}
	parts := []string{digits[:len(digits)-2], digits[len(digits)-2:]}

	whole := parts[0]
	for i := len(whole) - 3; i > 0; i -= 3 {
		whole = whole[:i] + "." + whole[i:]
	}
	if amount < 0 && cents.Sign() != 0 {
		whole = "-" + whole
	}
	return whole + "," + parts[1] + " €"
}

// FormatDateLong formats as "11. Oktober 2026".
func FormatDateLong(t time.Time) string {
	return fmt.Sprintf("%d. %s", t.Day(), FormatMonthYear(t))
}

// FormatMonthYear formats as "Oktober 2026".
func FormatMonthYear(t time.Time) string {
	return fmt.Sprintf("%s %d", monthsLong[t.Month()-1], t.Year())
}

// FormatMonthShort formats as "Okt. 2026".
func FormatMonthShort(t time.Time) string {
	return fmt.Sprintf("%s %d", monthsShort[t.Month()-1], t.Year())
}

// FormatDate formats as "05.02.2024".
func FormatDate(t time.Time) string { return t.Format("02.01.2006") }

// BuildFeePeriods groups consecutive months with equal values into table periods.
func BuildFeePeriods(rows []MonthRow, openEnded bool) []FeePeriod {
	ordered := append([]MonthRow(nil), rows...)
	monthID := func(t time.Time) int { return t.Year()*12 + int(t.Month()) }
	sort.SliceStable(ordered, func(i, j int) bool {
		return monthID(ordered[i].Month) < monthID(ordered[j].Month)
	})
	unique := make([]MonthRow, 0, len(ordered))
	for _, row := range ordered {
		if len(unique) == 0 || monthID(unique[len(unique)-1].Month) != monthID(row.Month) {
			unique = append(unique, row)
		}
	}
	periods := make([]FeePeriod, 0)
	for start, end := 0, 0; start < len(unique); start = end {
		first := unique[start]
		end = start + 1
		for end < len(unique) {
			next := unique[end]
			if monthID(next.Month) != monthID(unique[end-1].Month)+1 ||
				first.CareType != next.CareType || first.CareHours != next.CareHours ||
				first.ChildcareFee != next.ChildcareFee || first.FoodFee != next.FoodFee ||
				first.Previous != next.Previous {
				break
			}
			end++
		}
		label := FormatMonthShort(first.Month)
		if openEnded && end == len(unique) {
			label = "ab " + label
		} else if end-start > 1 {
			label += " – " + FormatMonthShort(unique[end-1].Month)
		}
		periods = append(periods, FeePeriod{
			Label: label, CareType: first.CareType, CareHours: first.CareHours,
			ChildcareFee: first.ChildcareFee, FoodFee: first.FoodFee, Previous: first.Previous,
		})
	}
	return periods
}
