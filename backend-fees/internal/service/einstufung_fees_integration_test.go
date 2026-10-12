package service_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/util"
)

func childcareFee(t *testing.T, childID uuid.UUID, month int) *domain.FeeExpectation {
	t.Helper()
	fee, err := repository.NewPostgresFeeRepository(testDB).GetByChildFeePeriod(t.Context(), childID, domain.FeeTypeChildcare, 2026, month)
	if err != nil {
		t.Fatal(err)
	}
	return fee
}

func monthRow(t *testing.T, e *domain.Einstufung, month int) domain.EinstufungMonthRow {
	t.Helper()
	for _, row := range e.MonthlyTable {
		if row.Year == 2026 && row.Month == month {
			return row
		}
	}
	t.Fatalf("no monthly row for 2026-%02d", month)
	return domain.EinstufungMonthRow{}
}

func TestEinstufungFees_GenerateUsesValidEinstufung(t *testing.T) {
	f := newEinstufungFixture(t)
	child, household := f.child(t, "TEFG", domain.IncomeStatusUnknown, date(2025, time.March, 1), date(2026, time.January, 1))
	created, _, err := f.service.Create(f.ctx, service.CreateEinstufungInput{
		ChildID: child.ID, Year: 2026, IncomeBasis: domain.IncomeBasisHighestRate, ChildrenCount: 3,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Household data diverging from the Einstufung must not decide the fee.
	household = f.household(t, household.ID)
	one := 1
	household.ChildrenCountForFees = &one
	if err := f.householdRepo.Update(f.ctx, household); err != nil {
		t.Fatal(err)
	}

	march := 3
	if _, err := f.fees.Generate(f.ctx, 2026, &march); err != nil {
		t.Fatal(err)
	}
	fee := childcareFee(t, child.ID, 3)
	want := monthRow(t, created, 3).ChildcareFee
	if fee == nil || fee.Amount != want {
		t.Fatalf("generated %+v, want %.2f from the Einstufung", fee, want)
	}

	// Without an Einstufung for the month, household data still decide.
	if err := f.service.Delete(f.ctx, created.ID); err != nil {
		t.Fatal(err)
	}
	april := 4
	if _, err := f.fees.Generate(f.ctx, 2026, &april); err != nil {
		t.Fatal(err)
	}
	if fallback := childcareFee(t, child.ID, 4); fallback == nil || fallback.Amount <= want {
		t.Fatalf("household fallback %+v, want more than the sibling rate %.2f", fallback, want)
	}
}

func TestEinstufungFees_SaveAlignsGeneratedMonths(t *testing.T) {
	f := newEinstufungFixture(t)
	// Turns three on 10 May 2026: January to April are Krippe months, May is fee-free.
	child, _ := f.child(t, "TEFS", domain.IncomeStatusMaxAccepted, date(2023, time.May, 10), date(2026, time.January, 1))
	for _, month := range []int{1, 2, 3, 5} {
		m := month
		if _, err := f.fees.Generate(f.ctx, 2026, &m); err != nil {
			t.Fatal(err)
		}
	}
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	stale, err := createTestFee(feeRepo, child.ID, domain.FeeTypeChildcare, 100, 2026, 5)
	if err != nil {
		t.Fatal(err)
	}

	// Pay March in full at the household rate.
	paid := childcareFee(t, child.ID, 3)
	tx, err := createTestTransaction(repository.NewPostgresTransactionRepository(testDB), "TESTDE000EINSTUFUNG", paid.Amount,
		date(2026, time.March, 5), "Platzgeld")
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.NewPostgresMatchRepository(testDB).Create(f.ctx, &domain.PaymentMatch{ID: uuid.New(),
		TransactionID: tx.ID, ExpectationID: paid.ID, Amount: paid.Amount, MatchType: domain.MatchTypeManual,
		MatchedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}

	created, changes, err := f.service.Create(f.ctx, service.CreateEinstufungInput{
		ChildID: child.ID, Year: 2026, IncomeBasis: domain.IncomeBasisHighestRate, ChildrenCount: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	reduced := monthRow(t, created, 1).ChildcareFee
	if reduced >= paid.Amount {
		t.Fatalf("sibling rate %.2f should be below the single-child rate %.2f", reduced, paid.Amount)
	}
	// January and February follow the Einstufung, paid March is only reported,
	// April is not generated yet, the stale May fee is removed.
	if changes.Updated != 2 || changes.Removed != 1 || changes.Created != 0 || len(changes.CreditReviewRequired) != 1 ||
		changes.CreditReviewRequired[0].Month != 3 {
		t.Fatalf("changes = %+v", changes)
	}
	for _, month := range []int{1, 2} {
		if fee := childcareFee(t, child.ID, month); fee.Amount != reduced {
			t.Fatalf("2026-%02d = %.2f, want %.2f", month, fee.Amount, reduced)
		}
	}
	if fee := childcareFee(t, child.ID, 3); fee.Amount != paid.Amount {
		t.Fatalf("paid March changed to %.2f", fee.Amount)
	}
	if childcareFee(t, child.ID, 4) != nil {
		t.Fatal("April was created before generation")
	}
	if _, err := feeRepo.GetByID(f.ctx, stale.ID); err == nil {
		t.Fatal("stale May CHILDCARE fee was kept")
	}

	// Back to one child: unpaid months rise again, paid March is settled.
	one := 1
	_, changes, err = f.service.Update(f.ctx, created.ID, service.UpdateEinstufungInput{ChildrenCount: &one})
	if err != nil {
		t.Fatal(err)
	}
	if changes.Updated != 2 || changes.Skipped != 1 || len(changes.CreditReviewRequired) != 0 {
		t.Fatalf("changes after update = %+v", changes)
	}
	if fee := childcareFee(t, child.ID, 1); fee.Amount != paid.Amount {
		t.Fatalf("January = %.2f, want %.2f", fee.Amount, paid.Amount)
	}
}

func TestEinstufungWorklist_FiltersAndMissing(t *testing.T) {
	f := newEinstufungFixture(t)
	restore := util.SetClock(func() time.Time { return time.Date(2026, 10, 12, 10, 0, 0, 0, time.UTC) })
	defer restore()

	covered, _ := f.child(t, "TEWA", domain.IncomeStatusUnknown, date(2025, time.March, 1), date(2026, time.January, 1))
	uncovered, _ := f.child(t, "TEWB", domain.IncomeStatusUnknown, date(2025, time.April, 1), date(2026, time.February, 1))
	f.child(t, "TEWC", domain.IncomeStatusUnknown, date(2025, time.May, 1), date(2026, time.December, 15))
	f.child(t, "TEWD", domain.IncomeStatusUnknown, date(2025, time.June, 1), date(2027, time.June, 1))       // starts too late
	f.child(t, "TEWE", domain.IncomeStatusUnknown, date(2023, time.January, 1), date(2025, time.January, 1)) // Kindergarten

	if _, _, err := f.service.Create(f.ctx, service.CreateEinstufungInput{
		ChildID: covered.ID, Year: 2026, IncomeBasis: domain.IncomeBasisHighestRate, ChildrenCount: 1, Provisional: true,
	}); err != nil {
		t.Fatal(err)
	}
	// Ended in June: the child needs a new Einstufung for October.
	ended, _, err := f.service.Create(f.ctx, service.CreateEinstufungInput{
		ChildID: uncovered.ID, Year: 2026, IncomeBasis: domain.IncomeBasisHighestRate, ChildrenCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.Exec(`UPDATE fees.einstufungen SET valid_until = '2026-06-30' WHERE id = $1`, ended.ID); err != nil {
		t.Fatal(err)
	}

	missing, err := f.service.ListMissing(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, m := range missing {
		if strings.HasPrefix(m.MemberNumber, "TEW") {
			got[m.MemberNumber] = m.Month.Format("2006-01")
		}
	}
	if len(got) != 2 || got["TEWB"] != "2026-10" || got["TEWC"] != "2026-12" {
		t.Fatalf("missing = %v", got)
	}

	list := func(search, status string) int64 {
		t.Helper()
		_, total, err := f.service.ListByYear(f.ctx, repository.EinstufungListFilter{Year: 2026, Search: search, Status: status}, 0, 50)
		if err != nil {
			t.Fatal(err)
		}
		return total
	}
	if n := list("", ""); n != 2 {
		t.Fatalf("all = %d", n)
	}
	if n := list("tewa test", ""); n != 1 {
		t.Fatalf("search by name = %d", n)
	}
	if n := list("TEWB", "provisional"); n != 0 {
		t.Fatalf("search and status = %d", n)
	}
	if n := list("%", ""); n != 0 {
		t.Fatalf("wildcard search = %d", n)
	}
	if n := list("", "provisional"); n != 1 {
		t.Fatalf("provisional = %d", n)
	}
	if n := list("", "current"); n != 1 {
		t.Fatalf("current = %d", n)
	}
	if n := list("", "unsent"); n != 2 {
		t.Fatalf("unsent = %d", n)
	}
	if _, _, err := f.service.ListByYear(f.ctx, repository.EinstufungListFilter{Year: 2026, Status: "bogus"}, 0, 50); err == nil {
		t.Fatal("unknown status accepted")
	}
}
