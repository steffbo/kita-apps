package letter

import (
	"bytes"
	"math"
	"os"
	"reflect"
	"testing"
	"time"
)

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func TestBuildFeePeriods(t *testing.T) {
	row := func(month time.Month, fee float64, previous bool) MonthRow {
		return MonthRow{date(2026, month, 1), "Krippe", 40, fee, 45, previous}
	}
	rows := []MonthRow{row(3, 100, false), row(1, 100, true), row(2, 100, true),
		row(1, 999, false), row(4, 100, false), row(6, 100, false)}
	copyRows := append([]MonthRow(nil), rows...)
	for _, test := range []struct {
		open   bool
		labels []string
	}{
		{false, []string{"Jan. 2026 – Feb. 2026", "März 2026 – Apr. 2026", "Juni 2026"}},
		{true, []string{"Jan. 2026 – Feb. 2026", "März 2026 – Apr. 2026", "ab Juni 2026"}},
	} {
		periods := BuildFeePeriods(rows, test.open)
		labels := make([]string, len(periods))
		for i, p := range periods {
			labels[i] = p.Label
		}
		if !reflect.DeepEqual(labels, test.labels) {
			t.Fatalf("labels: %v", labels)
		}
		if !periods[0].Previous || periods[0].ChildcareFee != 100 || periods[1].Previous {
			t.Fatalf("first duplicate or previous flag lost: %+v", periods)
		}
	}
	if !reflect.DeepEqual(rows, copyRows) {
		t.Fatal("input mutated")
	}
	if len(BuildFeePeriods(nil, false)) != 0 {
		t.Fatal("expected empty periods")
	}
	for _, field := range []string{"hours", "type", "childcare", "food", "previous"} {
		a, b := row(1, 100, false), row(2, 100, false)
		switch field {
		case "hours":
			b.CareHours++
		case "type":
			b.CareType = "Kindergarten"
		case "childcare":
			b.ChildcareFee++
		case "food":
			b.FoodFee++
		case "previous":
			b.Previous = true
		}
		if len(BuildFeePeriods([]MonthRow{a, b}, false)) != 2 {
			t.Errorf("did not split %s", field)
		}
	}
	if got := BuildFeePeriods([]MonthRow{row(1, 100, false)}, true)[0].Label; got != "ab Jan. 2026" {
		t.Fatal(got)
	}
	if got := BuildFeePeriods([]MonthRow{row(1, 100, false)}, false)[0].Label; got != "Jan. 2026" {
		t.Fatal(got)
	}
	acrossYear := []MonthRow{row(12, 100, false), row(1, 100, false)}
	acrossYear[1].Month = date(2027, 1, 1)
	got := BuildFeePeriods(acrossYear, false)
	if len(got) != 1 || got[0].Label != "Dez. 2026 – Jan. 2027" {
		t.Fatalf("year boundary: %+v", got)
	}
}

func TestFormats(t *testing.T) {
	for _, test := range []struct {
		amount float64
		want   string
	}{
		{1.005, "1,01 €"}, {-1.005, "-1,01 €"}, {0, "0,00 €"},
		{312.3, "312,30 €"}, {1234.5, "1.234,50 €"},
		{1234567.891, "1.234.567,89 €"}, {math.Copysign(0, -1), "0,00 €"},
		{-0.004, "0,00 €"}, {-1234.565, "-1.234,57 €"},
	} {
		if got := FormatEuro(test.amount); got != test.want {
			t.Errorf("%v: %s", test.amount, got)
		}
	}
	for i, month := range []string{
		"Jan.", "Feb.", "März", "Apr.", "Mai", "Juni", "Juli", "Aug.", "Sept.", "Okt.", "Nov.", "Dez.",
	} {
		if got := FormatMonthShort(date(2026, time.Month(i+1), 1)); got != month+" 2026" {
			t.Error(got)
		}
	}
	if got := FormatDateLong(date(2026, 3, 11)); got != "11. März 2026" {
		t.Error(got)
	}
	if got := FormatMonthYear(date(2026, 3, 11)); got != "März 2026" {
		t.Error(got)
	}
	if got := FormatDate(date(2024, 2, 5)); got != "05.02.2024" {
		t.Error(got)
	}
}

// Sample identity and contributions are synthetic; fixed wording, board and bank come from the Vue template.
func sampleLetter() EinstufungLetter {
	return EinstufungLetter{
		LetterDate: date(2026, 10, 11), Year: 2026, ChildName: "Vorname Nachname", MemberNumber: "123",
		BirthDate: date(2024, 2, 5), EntryDate: date(2026, 8, 1), ValidFrom: date(2026, 10, 1),
		BasisText: "Die Einstufung wurde aufgrund der eingereichten Einkommensnachweise und nach der " +
			"Elternbeitragssatzung vorgenommen. Bei Änderungen informiert uns bitte umgehend.",
		Periods: []FeePeriod{
			{"Aug. 2026 – Sept. 2026", "Krippe", 40, 312.30, 45, true},
			{"Okt. 2026 – Feb. 2027", "Krippe", 40, 280, 45, false},
			{"ab März 2027", "Kindergarten", 40, 240, 45, false},
		},
		MembershipNote: "Vereinsbeitrag: 30,00 € jährlich – für dieses Jahr bereits bezahlt, " +
			"in den oben genannten Monatsbeträgen nicht enthalten.",
		MembershipFee: 30,
		Board: []BoardMember{
			{"André Rüger", "1. Vorsitzender"}, {"Sarah Thränhardt", "2. Vorsitzende / Bauliches"},
			{"Marcus Rehaag", "Finanzen"}, {"Stefan Remer", "Elternarbeit"},
			{"Samantha Lahl", "Schriftführerin"}, {"Dennis Braak", "Personal"},
		},
		Bank: BankAccount{"Knirpsenstadt e.\u202fV.", "DE33 3702 0500 0003 3214 00", "BFSWDE33XXX",
			"Bank für Sozialwirtschaft AG"},
	}
}

func TestRenderEinstufung(t *testing.T) {
	l := sampleLetter()
	pdf := renderEinstufung(l)
	if err := pdf.Error(); err != nil {
		t.Fatal(err)
	}
	if got := pdf.PageNo(); got != 1 {
		t.Fatalf("standard letter: %d pages", got)
	}
	five := l
	five.Periods = append(append([]FeePeriod(nil), l.Periods...), l.Periods[1:]...)
	if got := renderEinstufung(five).PageNo(); got != 1 {
		t.Fatalf("five periods: %d pages", got)
	}
	out, err := RenderEinstufung(l)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(out, []byte("%PDF-")) {
		t.Fatal("missing PDF header")
	}
	l.Provisional = true
	l.ChildName = "Änne Müller „Knirps“"
	l.BasisText += " Ä Ö Ü ä ö ü ß € „“ –."
	l.Periods = nil
	for i := 1; i <= 12; i++ {
		l.Periods = append(l.Periods, FeePeriod{FormatMonthShort(date(2026, time.Month(i), 1)),
			"Kindergarten", 40, 312.3, 45, i == 1})
	}
	if _, err := RenderEinstufung(l); err != nil {
		t.Fatal(err)
	}
	l.Periods = append(l.Periods, l.Periods...)
	l.Periods = append(l.Periods, l.Periods...)
	pdf = renderEinstufung(l)
	if pdf.PageNo() < 2 {
		t.Fatal("long table did not paginate")
	}
	var outLong bytes.Buffer
	if err := pdf.Output(&outLong); err != nil {
		t.Fatal(err)
	}
}

func TestWriteSampleLetter(t *testing.T) {
	path := os.Getenv("LETTER_SAMPLE_OUT")
	if path == "" {
		t.Skip("LETTER_SAMPLE_OUT is not set")
	}
	out, err := RenderEinstufung(sampleLetter())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, out, 0644); err != nil {
		t.Fatal(err)
	}
}
