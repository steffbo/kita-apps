package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
)

func TestTxManager_WithTx(t *testing.T) {
	ctx := context.Background()
	txm := repository.NewTxManager(testDB)
	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	errBoom := errors.New("boom")

	countFees := func(t *testing.T, childID uuid.UUID) int {
		t.Helper()
		var n int
		if err := testDB.Get(&n, `SELECT COUNT(*) FROM fees.fee_expectations WHERE child_id = $1`, childID); err != nil {
			t.Fatal(err)
		}
		return n
	}

	t.Run("commits on success", func(t *testing.T) {
		cleanupTestData()
		defer cleanupTestData()
		child, err := createTestChild(childRepo, "TX")
		if err != nil {
			t.Fatal(err)
		}

		err = txm.WithTx(ctx, func(ctx context.Context) error {
			_, err := createTestFeeCtx(ctx, feeRepo, child.ID, 1)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		if got := countFees(t, child.ID); got != 1 {
			t.Fatalf("fees = %d, want 1", got)
		}
	})

	t.Run("rolls back all writes on error", func(t *testing.T) {
		cleanupTestData()
		defer cleanupTestData()
		child, err := createTestChild(childRepo, "TX")
		if err != nil {
			t.Fatal(err)
		}

		err = txm.WithTx(ctx, func(ctx context.Context) error {
			for month := 1; month <= 2; month++ {
				if _, err := createTestFeeCtx(ctx, feeRepo, child.ID, month); err != nil {
					return err
				}
			}
			return errBoom
		})
		if !errors.Is(err, errBoom) {
			t.Fatalf("err = %v, want boom", err)
		}
		if got := countFees(t, child.ID); got != 0 {
			t.Fatalf("fees = %d, want 0 after rollback", got)
		}
	})

	t.Run("repository-owned transaction joins the outer one", func(t *testing.T) {
		cleanupTestData()
		defer cleanupTestData()
		child, err := createTestChild(childRepo, "TX")
		if err != nil {
			t.Fatal(err)
		}
		originalName := child.LastName

		// ChildRepository.Update opens its own transaction; inside WithTx it
		// must neither commit early nor survive the outer rollback.
		err = txm.WithTx(ctx, func(ctx context.Context) error {
			child.LastName = "Geändert"
			if err := childRepo.Update(ctx, child); err != nil {
				return err
			}
			return errBoom
		})
		if !errors.Is(err, errBoom) {
			t.Fatalf("err = %v, want boom", err)
		}
		reloaded, err := childRepo.GetByID(ctx, child.ID)
		if err != nil {
			t.Fatal(err)
		}
		if reloaded.LastName != originalName {
			t.Fatalf("last name = %q, want %q (rolled back)", reloaded.LastName, originalName)
		}
	})

	t.Run("nil manager runs without transaction", func(t *testing.T) {
		var nilTxm *repository.TxManager
		called := false
		if err := nilTxm.WithTx(ctx, func(context.Context) error { called = true; return nil }); err != nil || !called {
			t.Fatalf("called=%v err=%v", called, err)
		}
	})
}

// createTestFeeCtx creates a monthly food fee using ctx, so it joins a transaction bound to ctx.
func createTestFeeCtx(ctx context.Context, feeRepo repository.FeeRepository, childID uuid.UUID, month int) (*domain.FeeExpectation, error) {
	fee := &domain.FeeExpectation{
		ID:        uuid.New(),
		ChildID:   childID,
		FeeType:   domain.FeeTypeFood,
		Year:      2026,
		Month:     &month,
		Amount:    45.40,
		DueDate:   time.Date(2026, time.Month(month), 5, 0, 0, 0, 0, time.UTC),
		CreatedAt: time.Now(),
	}
	return fee, feeRepo.Create(ctx, fee)
}
