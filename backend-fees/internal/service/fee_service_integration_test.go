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

func TestFeeService_GetByID_PartialMatch(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	householdRepo := repository.NewPostgresHouseholdRepository(testDB)
	matchRepo := repository.NewPostgresMatchRepository(testDB)
	txRepo := repository.NewPostgresTransactionRepository(testDB)

	child, err := createTestChild(childRepo, "PM")
	if err != nil {
		t.Fatal(err)
	}

	fee, err := createTestFee(feeRepo, child.ID, domain.FeeTypeFood, 45.40, time.Now().Year(), int(time.Now().Month()))
	if err != nil {
		t.Fatal(err)
	}

	tx, err := createTestTransaction(txRepo, "TESTDE444555666777", 20.00, time.Now(),
		child.FirstName+" "+child.LastName+" Essensgeld "+child.MemberNumber)
	if err != nil {
		t.Fatal(err)
	}

	match := &domain.PaymentMatch{
		ID:            uuid.New(),
		TransactionID: tx.ID,
		ExpectationID: fee.ID,
		Amount:        20.00,
		MatchType:     domain.MatchTypeManual,
		MatchedAt:     time.Now(),
	}
	if err := matchRepo.Create(context.Background(), match); err != nil {
		t.Fatalf("matchRepo.Create failed: %v", err)
	}

	feeService := service.NewFeeService(feeRepo, childRepo, householdRepo, matchRepo, txRepo)
	loaded, err := feeService.GetByID(context.Background(), fee.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if loaded.MatchedAmount != 20.00 {
		t.Fatalf("Expected matchedAmount 20.00, got %.2f", loaded.MatchedAmount)
	}
	if loaded.Remaining < 25.39 || loaded.Remaining > 25.41 {
		t.Fatalf("Expected remaining ~25.40, got %.2f", loaded.Remaining)
	}
	if loaded.IsPaid {
		t.Fatalf("Expected fee to be unpaid for partial match")
	}
	if len(loaded.PartialMatches) != 1 {
		t.Fatalf("Expected 1 partial match, got %d", len(loaded.PartialMatches))
	}
}

func TestFeeService_GetByID_FullMatch(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	childRepo := repository.NewPostgresChildRepository(testDB)
	feeRepo := repository.NewPostgresFeeRepository(testDB)
	householdRepo := repository.NewPostgresHouseholdRepository(testDB)
	matchRepo := repository.NewPostgresMatchRepository(testDB)
	txRepo := repository.NewPostgresTransactionRepository(testDB)

	child, err := createTestChild(childRepo, "FM")
	if err != nil {
		t.Fatal(err)
	}

	fee, err := createTestFee(feeRepo, child.ID, domain.FeeTypeFood, 45.40, time.Now().Year(), int(time.Now().Month()))
	if err != nil {
		t.Fatal(err)
	}

	tx, err := createTestTransaction(txRepo, "TESTDE888999000111", 45.40, time.Now(),
		child.FirstName+" "+child.LastName+" Essensgeld "+child.MemberNumber)
	if err != nil {
		t.Fatal(err)
	}

	match := &domain.PaymentMatch{
		ID:            uuid.New(),
		TransactionID: tx.ID,
		ExpectationID: fee.ID,
		Amount:        45.40,
		MatchType:     domain.MatchTypeAuto,
		MatchedAt:     time.Now(),
	}
	if err := matchRepo.Create(context.Background(), match); err != nil {
		t.Fatalf("matchRepo.Create failed: %v", err)
	}

	feeService := service.NewFeeService(feeRepo, childRepo, householdRepo, matchRepo, txRepo)
	loaded, err := feeService.GetByID(context.Background(), fee.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if loaded.MatchedAmount < 45.39 || loaded.MatchedAmount > 45.41 {
		t.Fatalf("Expected matchedAmount 45.40, got %.2f", loaded.MatchedAmount)
	}
	if loaded.Remaining != 0 {
		t.Fatalf("Expected remaining 0.00, got %.2f", loaded.Remaining)
	}
	if !loaded.IsPaid {
		t.Fatalf("Expected fee to be paid")
	}
}
