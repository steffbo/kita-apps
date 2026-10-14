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

// TestCreateManualMatch_CapsAmount checks that a manual match never books more than
// the transaction carries or the fee still has open (a 30 EUR top-up on a 150 EUR
// fee with 120 EUR already paid must be matched with 30 EUR, not 150 EUR).
func TestCreateManualMatch_CapsAmount(t *testing.T) {
	requireTestDatabase(t)
	cleanupTestData()
	defer cleanupTestData()

	ctx := context.Background()
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	txRepo := repository.NewPostgresTransactionRepository(testDB)
	matchRepo := repository.NewPostgresMatchRepository(testDB)
	importService := service.NewImportService(txRepo, feeRepo, childRepo, matchRepo,
		repository.NewPostgresKnownIBANRepository(testDB), repository.NewPostgresWarningRepository(testDB),
		repository.NewTxManager(testDB), repository.NewPostgresFeeScheduleRepository(testDB))
	adminUserID := uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")

	child, err := createTestChild(childRepo, "CAP")
	if err != nil {
		t.Fatal(err)
	}
	due := time.Date(2026, 6, 5, 0, 0, 0, 0, time.UTC)
	fee, err := createTestFeeWithDueDate(feeRepo, child.ID, domain.FeeTypeChildcare, 150, 2026, 6, due)
	if err != nil {
		t.Fatal(err)
	}

	first, err := createTestTransaction(txRepo, "TESTDECAP0000000001", 120, due, "Beitrag Juni")
	if err != nil {
		t.Fatal(err)
	}
	topUp, err := createTestTransaction(txRepo, "TESTDECAP0000000001", 30, due.AddDate(0, 0, 3), "Nachzahlung Juni")
	if err != nil {
		t.Fatal(err)
	}

	m1, err := importService.CreateManualMatch(ctx, first.ID, fee.ID, adminUserID)
	if err != nil {
		t.Fatalf("first match: %v", err)
	}
	if m1.Amount != 120 {
		t.Errorf("first match amount = %.2f, want 120 (transaction amount)", m1.Amount)
	}

	m2, err := importService.CreateManualMatch(ctx, topUp.ID, fee.ID, adminUserID)
	if err != nil {
		t.Fatalf("top-up match: %v", err)
	}
	if m2.Amount != 30 {
		t.Errorf("top-up match amount = %.2f, want 30", m2.Amount)
	}

	total, err := matchRepo.GetTotalMatchedAmount(ctx, fee.ID)
	if err != nil {
		t.Fatal(err)
	}
	if total != 150 {
		t.Errorf("total matched = %.2f, want 150", total)
	}

	// The fee is paid now; a further manual match is rejected.
	extra, err := createTestTransaction(txRepo, "TESTDECAP0000000001", 150, due.AddDate(0, 0, 10), "doppelt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := importService.CreateManualMatch(ctx, extra.ID, fee.ID, adminUserID); !errors.Is(err, service.ErrInvalidInput) {
		t.Errorf("match on paid fee: err = %v, want ErrInvalidInput", err)
	}
}
