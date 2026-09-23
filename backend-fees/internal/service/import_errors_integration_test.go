package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

const importErrorsCSVHeader = "Bezeichnung Auftragskonto;IBAN Auftragskonto;BIC Auftragskonto;Bankname Auftragskonto;Buchungstag;Valutadatum;Name Zahlungsbeteiligter;IBAN Zahlungsbeteiligter;BIC (SWIFT-Code) Zahlungsbeteiligter;Buchungstext;Verwendungszweck;Betrag;Waehrung;Saldo nach Buchung\n"

// failingCreateTxRepo fails to save transactions from one payer IBAN.
type failingCreateTxRepo struct {
	repository.TransactionRepository
	failIBAN string
}

func (r failingCreateTxRepo) Create(ctx context.Context, tx *domain.BankTransaction) error {
	if tx.PayerIBAN != nil && *tx.PayerIBAN == r.failIBAN {
		return errors.New("disk full")
	}
	return r.TransactionRepository.Create(ctx, tx)
}

type failingBlacklistRepo struct {
	repository.KnownIBANRepository
}

func (failingBlacklistRepo) GetBlacklistedIBANs(context.Context) (map[string]bool, error) {
	return nil, errors.New("connection lost")
}

func TestImportService_ProcessCSV_ReportsAndStoresRowErrors(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	txRepo := failingCreateTxRepo{TransactionRepository: repository.NewPostgresTransactionRepository(testDB), failIBAN: "TESTERR000000002"}
	importService := service.NewImportService(
		txRepo,
		repository.NewPostgresFeeRepository(testDB),
		repository.NewPostgresChildRepository(testDB),
		repository.NewPostgresMatchRepository(testDB),
		repository.NewPostgresKnownIBANRepository(testDB),
		repository.NewPostgresWarningRepository(testDB),
		repository.NewTxManager(testDB),
	)

	csv := importErrorsCSVHeader +
		"Test;DE1234;BIC;Bank;02.01.2026;02.01.2026;Zahler Eins;TESTERR000000001;BIC;Transfer;Irgendwas;12,34;EUR;1000,00\n" +
		"Test;DE1234;BIC;Bank;03.01.2026;03.01.2026;Zahler Zwei;TESTERR000000002;BIC;Transfer;Irgendwas;56,78;EUR;1000,00\n"

	result, err := importService.ProcessCSV(context.Background(), strings.NewReader(csv), "errors.csv", uuid.New())
	if err != nil {
		t.Fatalf("ProcessCSV: %v", err)
	}
	if result.Imported != 1 || result.Skipped != 0 {
		t.Fatalf("imported=%d skipped=%d, want 1/0 (a failed row is an error, not a skip)", result.Imported, result.Skipped)
	}
	if len(result.Errors) != 1 || result.Errors[0].PayerName == nil || *result.Errors[0].PayerName != "Zahler Zwei" ||
		!strings.Contains(result.Errors[0].Message, "disk full") {
		t.Fatalf("errors = %+v, want one error for Zahler Zwei", result.Errors)
	}

	var stored struct {
		ErrorCount int                 `db:"error_count"`
		Errors     domain.ImportErrors `db:"errors"`
	}
	if err := testDB.Get(&stored, `SELECT error_count, errors FROM fees.import_batches WHERE id = $1`, result.BatchID); err != nil {
		t.Fatalf("load batch: %v", err)
	}
	if stored.ErrorCount != 1 || len(stored.Errors) != 1 || stored.Errors[0].Amount != 56.78 {
		t.Fatalf("stored batch errors = %+v", stored)
	}
}

func TestImportService_ProcessCSV_FailsWhenReferenceDataUnavailable(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	importService := service.NewImportService(
		repository.NewPostgresTransactionRepository(testDB),
		repository.NewPostgresFeeRepository(testDB),
		repository.NewPostgresChildRepository(testDB),
		repository.NewPostgresMatchRepository(testDB),
		failingBlacklistRepo{repository.NewPostgresKnownIBANRepository(testDB)},
		repository.NewPostgresWarningRepository(testDB),
		repository.NewTxManager(testDB),
	)

	var batchesBefore int
	if err := testDB.Get(&batchesBefore, `SELECT COUNT(*) FROM fees.import_batches`); err != nil {
		t.Fatal(err)
	}

	csv := importErrorsCSVHeader + "Test;DE1234;BIC;Bank;02.01.2026;02.01.2026;Zahler;TESTERR000000003;BIC;Transfer;x;10,00;EUR;1000,00\n"
	if _, err := importService.ProcessCSV(context.Background(), strings.NewReader(csv), "fail.csv", uuid.New()); err == nil {
		t.Fatal("expected error when the blacklist cannot be loaded")
	}

	var batchesAfter int
	if err := testDB.Get(&batchesAfter, `SELECT COUNT(*) FROM fees.import_batches`); err != nil {
		t.Fatal(err)
	}
	if batchesAfter != batchesBefore {
		t.Fatalf("batch created despite failed import (%d -> %d)", batchesBefore, batchesAfter)
	}
}

func TestTransactionRepository_CreateRejectsDuplicateBooking(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	txRepo := repository.NewPostgresTransactionRepository(testDB)
	bookingDate := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	if _, err := createTestTransaction(txRepo, "TESTDUP000000001", 45.40, bookingDate, "Essen"); err != nil {
		t.Fatal(err)
	}

	_, err := createTestTransaction(txRepo, "TESTDUP000000001", 45.40, bookingDate, "Essen")
	if !errors.Is(err, repository.ErrDuplicate) {
		t.Fatalf("second insert err = %v, want ErrDuplicate", err)
	}

	// Same booking with another description is a different transaction.
	if _, err := createTestTransaction(txRepo, "TESTDUP000000001", 45.40, bookingDate, "Essen Geschwister"); err != nil {
		t.Fatalf("different description: %v", err)
	}
}
