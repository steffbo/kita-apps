package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

func TestFeeService_GenerateYearly_CreatesOneMembershipFeePerMember(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	ctx := context.Background()
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	householdRepo := repository.NewPostgresHouseholdRepository(testDB)
	memberRepo := repository.NewPostgresMemberRepository(testDB)
	matchRepo := repository.NewPostgresMatchRepository(testDB)
	txRepo := repository.NewPostgresTransactionRepository(testDB)
	feeService := service.NewFeeService(feeRepo, childRepo, householdRepo, matchRepo, txRepo, repository.NewPostgresFeeScheduleRepository(testDB))

	const year = 2027
	createHousehold := func(name string) *domain.Household {
		h := &domain.Household{Name: "TEST " + name, MembershipStatus: domain.MembershipAssignmentStatusAssumed}
		if err := householdRepo.Create(ctx, h); err != nil {
			t.Fatalf("create household %s: %v", name, err)
		}
		return h
	}
	createChild := func(number string, householdID uuid.UUID, entry time.Time) *domain.Child {
		now := time.Now()
		c := &domain.Child{
			ID: uuid.New(), HouseholdID: &householdID, MemberNumber: number, FirstName: "Kind", LastName: number,
			BirthDate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), EntryDate: entry, IsActive: true, CreatedAt: now, UpdatedAt: now,
		}
		if err := childRepo.Create(ctx, c); err != nil {
			t.Fatalf("create child %s: %v", number, err)
		}
		return c
	}
	createMember := func(number string, householdID uuid.UUID, start time.Time, end *time.Time) *domain.Member {
		m := &domain.Member{MemberNumber: number, FirstName: "Mitglied", LastName: number, HouseholdID: &householdID, MembershipStart: start, MembershipEnd: end, IsActive: true}
		if err := memberRepo.Create(ctx, m); err != nil {
			t.Fatalf("create member %s: %v", number, err)
		}
		return m
	}
	memberFees := func(householdID uuid.UUID) []domain.FeeExpectation {
		fees, err := feeRepo.ListMembershipForHousehold(ctx, householdID, year)
		if err != nil {
			t.Fatalf("list membership fees: %v", err)
		}
		return fees
	}

	aug2025 := time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC)
	aug2026 := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)

	// Two members, two children, one fee already paid without a member link.
	both := createHousehold("Beide")
	older := createChild("TMG01", both.ID, aug2025)
	younger := createChild("TMG02", both.ID, aug2026)
	first := createMember("TM0001", both.ID, aug2025, nil)
	second := createMember("TM0002", both.ID, aug2025, nil)
	legacy := &domain.FeeExpectation{
		ID: uuid.New(), ChildID: older.ID, HouseholdID: &both.ID, FeeType: domain.FeeTypeMembership,
		Year: year, Amount: 30, DueDate: time.Date(year, 3, 31, 0, 0, 0, 0, time.UTC), CreatedAt: time.Now(),
	}
	if err := feeRepo.Create(ctx, legacy); err != nil {
		t.Fatal(err)
	}

	// One member; a second member whose membership ended before the year.
	single := createHousehold("Einzeln")
	createChild("TMG03", single.ID, aug2025)
	onlyMember := createMember("TM0003", single.ID, aug2025, nil)
	ended := time.Date(year-1, 12, 31, 0, 0, 0, 0, time.UTC)
	createMember("TM0004", single.ID, aug2025, &ended)

	// No known member: one household-level fee as before.
	none := createHousehold("Ohne")
	createChild("TMG04", none.ID, aug2025)

	if _, err := feeService.Generate(ctx, year, nil); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	bothFees := memberFees(both.ID)
	if len(bothFees) != 2 {
		t.Fatalf("expected 2 fees for two-member household, got %d", len(bothFees))
	}
	byMember := map[uuid.UUID]domain.FeeExpectation{}
	for _, fee := range bothFees {
		if fee.MemberID == nil {
			t.Fatalf("fee %s has no member", fee.ID)
		}
		byMember[*fee.MemberID] = fee
	}
	if fee := byMember[first.ID]; fee.ID != legacy.ID {
		t.Errorf("expected legacy fee to be adopted by first member, got %+v", fee)
	}
	if fee := byMember[second.ID]; fee.ChildID != younger.ID || fee.Amount != 30 {
		t.Errorf("expected new 30 EUR fee for second member on younger child, got %+v", fee)
	}

	singleFees := memberFees(single.ID)
	if len(singleFees) != 1 || singleFees[0].MemberID == nil || *singleFees[0].MemberID != onlyMember.ID {
		t.Errorf("expected exactly one fee for the active member, got %+v", singleFees)
	}

	noneFees := memberFees(none.ID)
	if len(noneFees) != 1 || noneFees[0].MemberID != nil {
		t.Errorf("expected one household-level fee without member, got %+v", noneFees)
	}

	// Re-running is idempotent.
	if _, err := feeService.Generate(ctx, year, nil); err != nil {
		t.Fatalf("second Generate failed: %v", err)
	}
	if got := len(memberFees(both.ID)) + len(memberFees(single.ID)) + len(memberFees(none.ID)); got != 4 {
		t.Errorf("expected 4 fees after re-run, got %d", got)
	}
}

func TestImportService_MembershipPaymentSettlesFeeOfNamedChild(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	ctx := context.Background()
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	householdRepo := repository.NewPostgresHouseholdRepository(testDB)
	memberRepo := repository.NewPostgresMemberRepository(testDB)
	matchRepo := repository.NewPostgresMatchRepository(testDB)
	txRepo := repository.NewPostgresTransactionRepository(testDB)
	scheduleRepo := repository.NewPostgresFeeScheduleRepository(testDB)
	feeService := service.NewFeeService(feeRepo, childRepo, householdRepo, matchRepo, txRepo, scheduleRepo)
	importService := service.NewImportService(txRepo, feeRepo, childRepo, matchRepo, repository.NewPostgresKnownIBANRepository(testDB),
		repository.NewPostgresWarningRepository(testDB), repository.NewTxManager(testDB), scheduleRepo)

	const year = 2027
	household := &domain.Household{Name: "TEST Zwei Mitglieder", MembershipStatus: domain.MembershipAssignmentStatusAssumed}
	if err := householdRepo.Create(ctx, household); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	var children []*domain.Child
	for i, name := range []string{"Leonidas", "Tinette"} {
		c := &domain.Child{
			ID: uuid.New(), HouseholdID: &household.ID, MemberNumber: "TMI0" + string(rune('1'+i)), FirstName: name, LastName: "Zweimitglied",
			BirthDate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), EntryDate: time.Date(2025, 8, 1+i, 0, 0, 0, 0, time.UTC),
			IsActive: true, CreatedAt: now, UpdatedAt: now,
		}
		if err := childRepo.Create(ctx, c); err != nil {
			t.Fatal(err)
		}
		children = append(children, c)
	}
	for _, number := range []string{"TM0011", "TM0012"} {
		m := &domain.Member{MemberNumber: number, FirstName: "Mitglied", LastName: number, HouseholdID: &household.ID,
			MembershipStart: time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC), IsActive: true}
		if err := memberRepo.Create(ctx, m); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := feeService.Generate(ctx, year, nil); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	var youngerFee *domain.FeeExpectation
	fees, err := feeRepo.ListMembershipForHousehold(ctx, household.ID, year)
	if err != nil {
		t.Fatal(err)
	}
	for i := range fees {
		if fees[i].ChildID == children[1].ID {
			youngerFee = &fees[i]
		}
	}
	if len(fees) != 2 || youngerFee == nil {
		t.Fatalf("expected one fee per member with one on the younger child, got %+v", fees)
	}

	tx, err := createTestTransaction(txRepo, "TESTDE000011112222", 30, time.Date(year, 2, 1, 0, 0, 0, 0, time.UTC),
		"Vereinsbeitrag 2027 Tinette Zweimitglied")
	if err != nil {
		t.Fatal(err)
	}

	result, err := importService.Rescan(ctx)
	if err != nil {
		t.Fatalf("Rescan failed: %v", err)
	}
	for _, suggestion := range result.Suggestions {
		if suggestion.Transaction.ID != tx.ID {
			continue
		}
		if suggestion.Expectation == nil || suggestion.Expectation.ID != youngerFee.ID {
			t.Fatalf("expected younger child's membership fee %s, got %+v", youngerFee.ID, suggestion.Expectation)
		}
		return
	}
	t.Fatalf("no suggestion for transaction %s", tx.ID)
}
