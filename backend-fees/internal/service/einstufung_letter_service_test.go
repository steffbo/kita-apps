package service

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/util"
)

func letterDay(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func TestBuildEinstufungLetter_FollowUpShowsPreviousPeriods(t *testing.T) {
	sourceID := uuid.New()
	until := letterDay(2026, time.March, 31)
	previous := &domain.Einstufung{ValidUntil: &until, AnnualMembershipFee: 30, MonthlyTable: []domain.EinstufungMonthRow{
		{Year: 2026, Month: 1, CareType: "Krippe", CareHoursPerWeek: 45, ChildcareFee: 300, FoodFee: 45.4},
		{Year: 2026, Month: 2, CareType: "Krippe", CareHoursPerWeek: 45, ChildcareFee: 300, FoodFee: 45.4},
		{Year: 2026, Month: 3, CareType: "Krippe", CareHoursPerWeek: 45, ChildcareFee: 300, FoodFee: 45.4},
	}}
	e := &domain.Einstufung{
		Year: 2026, SourceEinstufungID: &sourceID, EffectiveFromMonth: letterDay(2026, time.April, 1),
		IncomeBasis: domain.IncomeBasisIncome, FeeRule: "Satzung", Provisional: true,
		Child: &domain.Child{FirstName: "Ada", LastName: "Beispiel", MemberNumber: "11001"},
		MonthlyTable: []domain.EinstufungMonthRow{
			{Year: 2026, Month: 4, CareType: "Krippe", CareHoursPerWeek: 45, ChildcareFee: 200, FoodFee: 45.4},
			{Year: 2026, Month: 5, CareType: "Krippe", CareHoursPerWeek: 45, ChildcareFee: 200, FoodFee: 45.4},
		},
	}

	l := buildEinstufungLetter(e, previous, letterDay(2026, time.April, 2))

	labels := []string{}
	for _, p := range l.Periods {
		labels = append(labels, p.Label)
	}
	if want := []string{"Jan. 2026 – März 2026", "ab Apr. 2026"}; !reflect.DeepEqual(labels, want) {
		t.Fatalf("labels = %v, want %v", labels, want)
	}
	if !l.Periods[0].Previous || l.Periods[1].Previous || l.Periods[1].ChildcareFee != 200 {
		t.Fatalf("periods = %+v", l.Periods)
	}
	if l.ChildName != "Ada Beispiel" || !l.ValidFrom.Equal(letterDay(2026, time.April, 1)) {
		t.Fatalf("child %q validFrom %v", l.ChildName, l.ValidFrom)
	}
	if want := "Vereinsbeitrag: 30,00 € jährlich – für dieses Jahr bereits bezahlt, " +
		"in den oben genannten Monatsbeträgen nicht enthalten."; l.MembershipNote != want {
		t.Fatalf("membership note = %q", l.MembershipNote)
	}
	if want := "Die Einstufung wurde aufgrund der eingereichten Einkommensnachweise und nach der " +
		"Elternbeitragssatzung vorgenommen. Die Einstufung ist vorläufig, bis die fehlenden Nachweise " +
		"vorliegen. Bei Änderungen informiert uns bitte umgehend."; l.BasisText != want {
		t.Fatalf("basis = %q", l.BasisText)
	}
	previous.AnnualMembershipFee = 0
	if l := buildEinstufungLetter(e, previous, letterDay(2026, time.April, 2)); l.MembershipNote != "" {
		t.Fatalf("membership note without any fee = %q", l.MembershipNote)
	}
}

func TestLetterBasisText(t *testing.T) {
	for _, test := range []struct {
		e    domain.Einstufung
		want string
	}{
		{domain.Einstufung{IncomeBasis: domain.IncomeBasisHighestRate},
			"Die Einstufung erfolgte aufgrund der freiwilligen Anerkennung des Höchstsatzes."},
		{domain.Einstufung{IncomeBasis: domain.IncomeBasisNotRequired},
			"Die Betreuung im Kindergarten ist beitragsfrei; Einkommensnachweise sind dafür nicht erforderlich."},
		{domain.Einstufung{IncomeBasis: domain.IncomeBasisIncome, FeeRule: "Entlastung (Stufe 2)"},
			"Die Einstufung wurde aufgrund der eingereichten Einkommensnachweise und nach dem " +
				"Elternentlastungsgesetz 2023/2024 vorgenommen. Bei Änderungen informiert uns bitte umgehend."},
		{domain.Einstufung{IncomeBasis: domain.IncomeBasisIncome, FeeRule: "Beitragsfrei (Entlastung)"},
			"Die Einstufung wurde aufgrund der eingereichten Einkommensnachweise vorgenommen. Gemäß " +
				"Elternentlastungsgesetz ist der Beitrag beitragsfrei. Bei Änderungen informiert uns bitte umgehend."},
	} {
		if got := letterBasisText(&test.e); got != test.want {
			t.Errorf("basis %s/%s = %q", test.e.IncomeBasis, test.e.FeeRule, got)
		}
	}
}

func TestLetterIntroduction(t *testing.T) {
	for basis, want := range map[domain.IncomeBasis]string{
		domain.IncomeBasisIncome:      "auf Grundlage der eingereichten Nachweise und der Elternbeitragsordnung des Trägers",
		domain.IncomeBasisHighestRate: "auf Grundlage eurer Anerkennung des Höchstsatzes und der Elternbeitragsordnung des Trägers",
		domain.IncomeBasisNotRequired: "auf Grundlage der Elternbeitragsordnung des Trägers",
	} {
		want += " haben wir die Elternbeiträge für euer Kind wie folgt eingestuft."
		if got := letterIntroduction(&domain.Einstufung{IncomeBasis: basis}); got != want {
			t.Errorf("%s: %q", basis, got)
		}
	}
}

func TestBuildEinstufungLetter_NewMembershipDueAndFallbackRow(t *testing.T) {
	e := &domain.Einstufung{Year: 2026, ValidFrom: letterDay(2026, time.August, 1), AnnualMembershipFee: 30,
		CareType: domain.ChildAgeTypeKindergarten, CareHoursPerWeek: 35, MonthlyFoodFee: 45.4}
	l := buildEinstufungLetter(e, nil, letterDay(2026, time.July, 1))
	if len(l.Periods) != 1 || l.Periods[0].Label != "ab Aug. 2026" || l.Periods[0].CareType != "Kindergarten" {
		t.Fatalf("periods = %+v", l.Periods)
	}
	if want := "Vereinsbeitrag: 30,00 € jährlich – einmalig fällig mit dem ersten Beitrag im Aug. 2026, " +
		"in den Monatsbeträgen nicht enthalten."; l.MembershipNote != want {
		t.Fatalf("membership note = %q", l.MembershipNote)
	}
	e.AnnualMembershipFee = 0
	if l := buildEinstufungLetter(e, nil, letterDay(2026, time.July, 1)); l.MembershipNote != "" {
		t.Fatalf("membership note without fee = %q", l.MembershipNote)
	}
}

func TestLetterDate(t *testing.T) {
	restore := util.SetClock(func() time.Time { return time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC) })
	defer restore()

	// 23:30 UTC on 11 Oct is already 12 Oct in Berlin.
	sent := time.Date(2026, 10, 11, 23, 30, 0, 0, time.UTC)
	e := &domain.Einstufung{UpdatedAt: sent.Add(-time.Hour), LetterSentAt: &sent}
	if got := letterDate(e); !got.Equal(letterDay(2026, time.October, 12)) {
		t.Fatalf("unchanged since sending: %v", got)
	}
	e.UpdatedAt = sent.Add(time.Minute)
	if got := letterDate(e); !got.Equal(letterDay(2026, time.October, 20)) {
		t.Fatalf("changed after sending: %v", got)
	}
	if got := letterDate(&domain.Einstufung{}); !got.Equal(letterDay(2026, time.October, 20)) {
		t.Fatalf("never sent: %v", got)
	}
}

type staticBoard []domain.BoardTerm

func (b staticBoard) ListTerms(context.Context) ([]domain.BoardTerm, error) { return b, nil }

func TestBoardAt(t *testing.T) {
	ended := letterDay(2025, time.October, 15)
	s := &EinstufungLetterService{board: staticBoard{
		{MemberName: "Pia Personal", Office: "Personal", StartDate: letterDay(2025, time.April, 30), EndDate: &ended},
		{MemberName: "Paul Personal", Office: "Personal", StartDate: letterDay(2025, time.October, 16)},
		{MemberName: "Kai Kasse", Office: "Kassenprüfung", StartDate: letterDay(2024, time.January, 1)},
		{MemberName: "Eva Eltern", Office: "Elternarbeit", StartDate: letterDay(2024, time.January, 1)},
		{MemberName: "Vera Vorsitz", Office: "Vorsitz", StartDate: letterDay(2025, time.April, 30)},
		{MemberName: "Zoe Zukunft", Office: "Finanzen", StartDate: letterDay(2027, time.January, 1)},
	}}
	members, err := s.boardAt(context.Background(), letterDay(2026, time.October, 12))
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, m := range members {
		names = append(names, m.Name+" ("+m.Role+")")
	}
	want := []string{"Vera Vorsitz (Vorsitz)", "Eva Eltern (Elternarbeit)", "Paul Personal (Personal)",
		"Kai Kasse (Kassenprüfung)"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("board = %v, want %v", names, want)
	}
}

func TestLetterFormatting(t *testing.T) {
	if got := formatIBAN("de33370205000003321400"); got != "DE33 3702 0500 0003 3214 00" {
		t.Fatalf("iban = %q", got)
	}
	e := &domain.Einstufung{Year: 2026, Child: &domain.Child{FirstName: "Anna Lena", LastName: "Görs/Meier"}}
	if got := einstufungFilename(e); got != "Einstufung_2026_Görs-Meier_Anna_Lena.pdf" {
		t.Fatalf("filename = %q", got)
	}
	email := func(s string) *string { return &s }
	household := &domain.Household{Parents: []domain.Parent{
		{Email: email(" a@example.test ")}, {Email: email("A@example.test")}, {Email: nil}, {Email: email("b@example.test")},
	}}
	if got := parentEmails(household); !reflect.DeepEqual(got, []string{"a@example.test", "b@example.test"}) {
		t.Fatalf("recipients = %v", got)
	}
}
