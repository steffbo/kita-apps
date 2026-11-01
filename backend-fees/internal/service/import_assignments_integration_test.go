package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

func TestImportService_DonationAssignment(t *testing.T) {
	requireTestDatabase(t)
	cleanupTestData()
	t.Cleanup(cleanupTestData)
	ctx := context.Background()
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	txRepo := repository.NewPostgresTransactionRepository(testDB)
	matchRepo := repository.NewPostgresMatchRepository(testDB)
	svc := service.NewImportService(txRepo, feeRepo, childRepo, matchRepo, repository.NewPostgresKnownIBANRepository(testDB),
		repository.NewPostgresWarningRepository(testDB), repository.NewTxManager(testDB), repository.NewPostgresFeeScheduleRepository(testDB))
	user := uuid.New()
	day := func(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }
	mk := func(iban string, amount float64, date time.Time) *domain.BankTransaction {
		t.Helper()
		tx, err := createTestTransaction(txRepo, iban, amount, date, "Spende")
		if err != nil {
			t.Fatal(err)
		}
		return tx
	}
	a := mk("TESTDON00001", 100.10, day(2026, 3, 1))
	b := mk("TESTDON00002", 200.20, day(2026, 10, 5))
	c := mk("TESTDON00003", 50, day(2025, 12, 24))
	out := mk("TESTDON00004", -20, day(2026, 1, 2))

	for _, tx := range []*domain.BankTransaction{a, b, c} {
		if err := svc.AssignTransaction(ctx, tx.ID, domain.AssignmentInput{Kind: domain.AssignmentDonation, Note: "  Sommerfest "}, user); err != nil {
			t.Fatal(err)
		}
	}
	// Not assignable: second assignment, outgoing, unknown kind, overlong note, unknown ID.
	if err := svc.AssignTransaction(ctx, a.ID, domain.AssignmentInput{Kind: domain.AssignmentDonation, Note: ""}, user); !errors.Is(err, service.ErrInvalidInput) {
		t.Fatalf("double assignment: %v", err)
	}
	if err := svc.AssignTransaction(ctx, out.ID, domain.AssignmentInput{Kind: domain.AssignmentDonation, Note: ""}, user); !errors.Is(err, service.ErrInvalidInput) {
		t.Fatalf("outgoing: %v", err)
	}
	if err := svc.AssignTransaction(ctx, out.ID, domain.AssignmentInput{Kind: domain.AssignmentKind("SUBSIDY"), Note: ""}, user); !errors.Is(err, service.ErrInvalidInput) {
		t.Fatalf("kind: %v", err)
	}
	if err := svc.AssignTransaction(ctx, uuid.New(), domain.AssignmentInput{Kind: domain.AssignmentDonation}, user); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("unknown: %v", err)
	}

	// Grouped by booking year, newest first, sums in cents.
	report, err := svc.GetDonations(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if report.Count != 3 || report.Total != 350.30 || len(report.Years) != 2 {
		t.Fatalf("report %+v", report)
	}
	y := report.Years[0]
	if y.Year != 2026 || y.Count != 2 || y.Total != 300.30 || y.Transactions[0].ID != b.ID || y.Transactions[0].Assignment.Note != "Sommerfest" {
		t.Fatalf("2026 %+v", y)
	}
	if report.Years[1].Year != 2025 || report.Years[1].Total != 50 {
		t.Fatalf("2025 %+v", report.Years[1])
	}

	// Assigned transactions count as matched, not open, and survive IBAN dismissal.
	open, _, err := txRepo.ListTransactions(ctx, "offen", "TESTDON00001", "date", "desc", 0, 10)
	if err != nil || len(open) != 0 {
		t.Fatalf("open %v %v", open, err)
	}
	done, _, err := txRepo.ListTransactions(ctx, "zugeordnet", "TESTDON00001", "date", "desc", 0, 10)
	if err != nil || len(done) != 1 || done[0].Assignment == nil || done[0].MatchedAmount == nil || *done[0].MatchedAmount != 100.10 {
		t.Fatalf("matched %+v %v", done, err)
	}
	if n, err := txRepo.DeleteUnmatchedByIBAN(ctx, "TESTDON00001"); err != nil || n != 0 {
		t.Fatalf("dismiss deleted %d %v", n, err)
	}

	// Fee matches and donations exclude each other.
	child, err := createTestChild(childRepo, "DN")
	if err != nil {
		t.Fatal(err)
	}
	fee, err := createTestFee(feeRepo, child.ID, domain.FeeTypeFood, 100.10, 2026, 3)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateManualMatch(ctx, a.ID, fee.ID, user); !errors.Is(err, service.ErrInvalidInput) {
		t.Fatalf("fee match on donation: %v", err)
	}
	if err := matchRepo.Create(ctx, &domain.PaymentMatch{ID: uuid.New(), TransactionID: a.ID, ExpectationID: fee.ID, Amount: 1, MatchType: domain.MatchTypeManual, MatchedAt: time.Now()}); !errors.Is(err, repository.ErrDuplicate) {
		t.Fatalf("repository fee match on donation: %v", err)
	}
	d := mk("TESTDON00005", 100.10, day(2026, 3, 2))
	if _, err := svc.CreateManualMatch(ctx, d.ID, fee.ID, user); err != nil {
		t.Fatal(err)
	}
	if err := svc.AssignTransaction(ctx, d.ID, domain.AssignmentInput{Kind: domain.AssignmentDonation, Note: ""}, user); !errors.Is(err, service.ErrInvalidInput) {
		t.Fatalf("donation on fee match: %v", err)
	}

	// Unmatch undoes the assignment and the transaction is open again.
	res, err := svc.UnmatchTransaction(ctx, a.ID, false)
	if err != nil || res.MatchesRemoved != 1 {
		t.Fatalf("unmatch %+v %v", res, err)
	}
	open, _, err = txRepo.ListTransactions(ctx, "offen", "TESTDON00001", "date", "desc", 0, 10)
	if err != nil || len(open) != 1 || open[0].Assignment != nil {
		t.Fatalf("open after unmatch %+v %v", open, err)
	}
	if report, _ = svc.GetDonations(ctx); report.Count != 2 {
		t.Fatalf("report after unmatch %+v", report)
	}

	// Household: stored with the donation, shown in the report, unknown ids refused, editable.
	hh := uuid.New()
	other := uuid.New()
	for id, name := range map[uuid.UUID]string{hh: "TESTFamilie Spende", other: "TESTFamilie Andere"} {
		if _, err := testDB.Exec(`INSERT INTO fees.households (id, name) VALUES ($1, $2)`, id, name); err != nil {
			t.Fatal(err)
		}
	}
	e := mk("TESTDON00006", 40, day(2026, 5, 5))
	if err := svc.AssignTransaction(ctx, e.ID, domain.AssignmentInput{Kind: domain.AssignmentDonation, HouseholdID: ptr(uuid.New())}, user); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("unknown household: %v", err)
	}
	if err := svc.AssignTransaction(ctx, e.ID, domain.AssignmentInput{Kind: domain.AssignmentDonation, HouseholdID: &hh}, user); err != nil {
		t.Fatal(err)
	}
	report, _ = svc.GetDonations(ctx)
	var got *domain.TransactionAssignment
	for _, y := range report.Years {
		for _, tx := range y.Transactions {
			if tx.ID == e.ID {
				got = tx.Assignment
			}
		}
	}
	if got == nil || got.HouseholdID == nil || *got.HouseholdID != hh || got.HouseholdName != "TESTFamilie Spende" {
		t.Fatalf("household not reported: %+v", got)
	}
	if err := svc.UpdateAssignment(ctx, e.ID, domain.AssignmentInput{Kind: domain.AssignmentDonation, Note: "neu", HouseholdID: &other}); err != nil {
		t.Fatal(err)
	}
	if err := svc.UpdateAssignment(ctx, uuid.New(), domain.AssignmentInput{Kind: domain.AssignmentDonation}); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("update unknown: %v", err)
	}

	// Suggestion for the next open payment of the same IBAN (earlier donation), and from a trusted child IBAN.
	next := mk("TESTDON00006", 40, day(2026, 6, 5))
	open, _, err = txRepo.ListTransactions(ctx, "offen", "TESTDON00006", "date", "desc", 0, 10)
	if err != nil || len(open) != 1 || open[0].ID != next.ID || open[0].HouseholdSuggestion == nil || open[0].HouseholdSuggestion.HouseholdID != other {
		t.Fatalf("iban suggestion %+v %v", open, err)
	}
	if _, err := testDB.Exec(`UPDATE fees.children SET household_id=$1 WHERE id=$2`, hh, child.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.Exec(`INSERT INTO fees.known_ibans (iban, status, child_id) VALUES ('TESTDON00007','trusted',$1)`, child.ID); err != nil {
		t.Fatal(err)
	}
	f := mk("TESTDON00007", 15, day(2026, 6, 6))
	open, _, _ = txRepo.ListTransactions(ctx, "offen", "TESTDON00007", "date", "desc", 0, 10)
	if len(open) != 1 || open[0].ID != f.ID || open[0].HouseholdSuggestion == nil || open[0].HouseholdSuggestion.HouseholdID != hh {
		t.Fatalf("child suggestion %+v", open)
	}
	if err := svc.AssignTransaction(ctx, f.ID, domain.AssignmentInput{Kind: domain.AssignmentDonation}, user); err != nil {
		t.Fatal(err)
	}
	done, _, _ = txRepo.ListTransactions(ctx, "zugeordnet", "TESTDON00007", "date", "desc", 0, 10)
	if len(done) != 1 || done[0].HouseholdSuggestion != nil {
		t.Fatalf("assigned row keeps suggestion %+v", done)
	}
}

func ptr[T any](v T) *T { return &v }
