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

// failingResolveWarningRepo fails the last follow-up step after a match.
type failingResolveWarningRepo struct {
	repository.WarningRepository
}

func (failingResolveWarningRepo) ResolveByTransactionID(context.Context, uuid.UUID, domain.ResolutionType, string) error {
	return errors.New("resolve failed")
}

// A failing follow-up action must roll back the match and the trusted IBAN,
// instead of leaving a half-applied assignment behind.
func TestImportService_MatchIsAtomicWithFollowUpActions(t *testing.T) {
	ctx := context.Background()
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	txRepo := repository.NewPostgresTransactionRepository(testDB)
	matchRepo := repository.NewPostgresMatchRepository(testDB)
	knownIBANRepo := repository.NewPostgresKnownIBANRepository(testDB)
	warningRepo := failingResolveWarningRepo{repository.NewPostgresWarningRepository(testDB)}
	importService := service.NewImportService(txRepo, feeRepo, childRepo, matchRepo, knownIBANRepo, warningRepo, repository.NewTxManager(testDB))

	setup := func(t *testing.T) (*domain.FeeExpectation, *domain.BankTransaction) {
		t.Helper()
		cleanupTestData()
		t.Cleanup(cleanupTestData)
		child, err := createTestChild(childRepo, "AT")
		if err != nil {
			t.Fatal(err)
		}
		fee, err := createTestFee(feeRepo, child.ID, domain.FeeTypeFood, 45.40, 2026, 9)
		if err != nil {
			t.Fatal(err)
		}
		tx, err := createTestTransaction(txRepo, "TESTATOMIC0001", 45.40, time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC), "Essen")
		if err != nil {
			t.Fatal(err)
		}
		return fee, tx
	}

	assertNothingPersisted := func(t *testing.T, tx *domain.BankTransaction) {
		t.Helper()
		var matches int
		if err := testDB.Get(&matches, `SELECT COUNT(*) FROM fees.payment_matches WHERE transaction_id = $1`, tx.ID); err != nil {
			t.Fatal(err)
		}
		if matches != 0 {
			t.Fatalf("payment_matches = %d, want 0 after rollback", matches)
		}
		known, err := knownIBANRepo.GetByIBAN(ctx, *tx.PayerIBAN)
		if err != nil {
			t.Fatal(err)
		}
		if known != nil {
			t.Fatalf("IBAN was marked as %s, want rollback", known.Status)
		}
	}

	t.Run("manual match", func(t *testing.T) {
		fee, tx := setup(t)
		if _, err := importService.CreateManualMatch(ctx, tx.ID, fee.ID, uuid.New()); err == nil {
			t.Fatal("expected error from failing follow-up action")
		}
		assertNothingPersisted(t, tx)
	})

	t.Run("allocation", func(t *testing.T) {
		fee, tx := setup(t)
		_, err := importService.AllocateTransaction(ctx, tx.ID, uuid.New(), []service.AllocationInput{{ExpectationID: fee.ID, Amount: 45.40}})
		if err == nil {
			t.Fatal("expected error from failing follow-up action")
		}
		assertNothingPersisted(t, tx)
	})

	t.Run("confirm counts failure", func(t *testing.T) {
		fee, tx := setup(t)
		result, err := importService.ConfirmMatches(ctx, []service.MatchConfirmation{{TransactionID: tx.ID, ExpectationID: fee.ID}}, uuid.New())
		if err != nil {
			t.Fatal(err)
		}
		if result.Confirmed != 0 || result.Failed != 1 {
			t.Fatalf("confirmed=%d failed=%d, want 0/1", result.Confirmed, result.Failed)
		}
		assertNothingPersisted(t, tx)
	})
}
