package service_test

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
)

var (
	testDB      *sqlx.DB
	pgContainer *postgres.PostgresContainer
)

// setupTestContainer starts a PostgreSQL container and runs migrations.
// This is called once per test suite from TestMain.
func setupTestContainer() error {
	ctx := context.Background()

	// Start PostgreSQL container
	container, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("kita_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to start postgres container: %w", err)
	}
	pgContainer = container

	// Get connection string
	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return fmt.Errorf("failed to get connection string: %w", err)
	}

	// Connect to database with retry logic
	// The container may report ready before accepting connections
	var connectErr error
	for i := 0; i < 10; i++ {
		testDB, connectErr = sqlx.Connect("postgres", connStr)
		if connectErr == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if connectErr != nil {
		return fmt.Errorf("failed to connect to test database after retries: %w", connectErr)
	}

	// Run migrations
	if err := runMigrations(connStr); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// runMigrations applies all database migrations to the test database.
func runMigrations(connStr string) error {
	// Get path to migrations folder relative to this file
	_, filename, _, _ := runtime.Caller(0)
	migrationsPath := filepath.Join(filepath.Dir(filename), "..", "..", "migrations")

	m, err := migrate.New(
		"file://"+migrationsPath,
		connStr,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// teardownTestContainer stops and removes the PostgreSQL container.
// This is called from TestMain after all tests complete.
func teardownTestContainer() {
	if testDB != nil {
		testDB.Close()
	}
	if pgContainer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		pgContainer.Terminate(ctx)
	}
}

// cleanupTestData removes test data created during tests.
// Test data is identified by having "T" prefix in member_number (children).
func cleanupTestData() {
	if testDB == nil {
		return
	}

	// Clean up in reverse order of dependencies
	testDB.Exec("DELETE FROM fees.transaction_warnings WHERE transaction_id IN (SELECT id FROM fees.bank_transactions WHERE payer_iban LIKE 'TEST%')")
	testDB.Exec("DELETE FROM fees.payment_matches WHERE transaction_id IN (SELECT id FROM fees.bank_transactions WHERE payer_iban LIKE 'TEST%')")
	testDB.Exec("DELETE FROM fees.bank_transactions WHERE payer_iban LIKE 'TEST%'")
	testDB.Exec("DELETE FROM fees.known_ibans WHERE iban LIKE 'TEST%'")
	testDB.Exec("DELETE FROM fees.fee_expectations WHERE child_id IN (SELECT id FROM fees.children WHERE member_number LIKE 'T%')")
	testDB.Exec("DELETE FROM fees.child_parents WHERE child_id IN (SELECT id FROM fees.children WHERE member_number LIKE 'T%')")
	testDB.Exec("DELETE FROM fees.children WHERE member_number LIKE 'T%'")
	testDB.Exec("DELETE FROM fees.households WHERE name LIKE 'TEST%'")
}

// =============================================================================
// Test Helper Functions
// =============================================================================

// createTestChild creates a child for testing with a unique member number.
func createTestChild(childRepo repository.ChildRepository, suffix string) (*domain.Child, error) {
	// Member number must be max 10 chars, use last 2 chars of suffix + 4 digit random
	shortSuffix := suffix
	if len(shortSuffix) > 2 {
		shortSuffix = shortSuffix[:2]
	}
	memberNum := fmt.Sprintf("T%s%05d", shortSuffix, time.Now().UnixNano()%100000)
	if len(memberNum) > 10 {
		memberNum = memberNum[:10]
	}

	child := &domain.Child{
		ID:           uuid.New(),
		MemberNumber: memberNum,
		FirstName:    "Test",
		LastName:     "Kind",
		BirthDate:    time.Now().AddDate(-2, 0, 0), // 2 years old
		EntryDate:    time.Now().AddDate(-1, 0, 0),
		IsActive:     true,
	}

	if err := childRepo.Create(context.Background(), child); err != nil {
		return nil, fmt.Errorf("failed to create test child: %w", err)
	}

	return child, nil
}

// createTestFee creates a fee expectation for testing.
func createTestFee(feeRepo repository.FeeRepository, childID uuid.UUID, feeType domain.FeeType, amount float64, year int, month int) (*domain.FeeExpectation, error) {
	fee := &domain.FeeExpectation{
		ID:        uuid.New(),
		ChildID:   childID,
		FeeType:   feeType,
		Year:      year,
		Month:     &month,
		Amount:    amount,
		DueDate:   time.Date(year, time.Month(month), 5, 0, 0, 0, 0, time.UTC),
		CreatedAt: time.Now(),
	}

	if err := feeRepo.Create(context.Background(), fee); err != nil {
		return nil, fmt.Errorf("failed to create test fee: %w", err)
	}

	return fee, nil
}

// createTestFeeWithDueDate creates a fee expectation with a specific due date.
func createTestFeeWithDueDate(feeRepo repository.FeeRepository, childID uuid.UUID, feeType domain.FeeType, amount float64, year int, month int, dueDate time.Time) (*domain.FeeExpectation, error) {
	fee := &domain.FeeExpectation{
		ID:        uuid.New(),
		ChildID:   childID,
		FeeType:   feeType,
		Year:      year,
		Month:     &month,
		Amount:    amount,
		DueDate:   dueDate,
		CreatedAt: time.Now(),
	}

	if err := feeRepo.Create(context.Background(), fee); err != nil {
		return nil, fmt.Errorf("failed to create test fee: %w", err)
	}

	return fee, nil
}

// createTestTransaction creates a bank transaction for testing.
func createTestTransaction(txRepo repository.TransactionRepository, payerIBAN string, amount float64, bookingDate time.Time, description string) (*domain.BankTransaction, error) {
	tx := &domain.BankTransaction{
		ID:          uuid.New(),
		BookingDate: bookingDate,
		ValueDate:   bookingDate,
		PayerName:   stringPtr("Test Payer"),
		PayerIBAN:   stringPtr(payerIBAN),
		Description: stringPtr(description),
		Amount:      amount,
		Currency:    "EUR",
		ImportedAt:  time.Now(),
	}

	if err := txRepo.Create(context.Background(), tx); err != nil {
		return nil, fmt.Errorf("failed to create test transaction: %w", err)
	}

	return tx, nil
}

// createTrustedIBAN creates a trusted IBAN entry, optionally linked to a child.
func createTrustedIBAN(knownIBANRepo repository.KnownIBANRepository, iban string, childID *uuid.UUID) error {
	knownIBAN := &domain.KnownIBAN{
		IBAN:      iban,
		PayerName: stringPtr("Test Payer"),
		Status:    domain.KnownIBANStatusTrusted,
		ChildID:   childID,
		Reason:    stringPtr("Test trusted IBAN"),
	}

	return knownIBANRepo.Create(context.Background(), knownIBAN)
}

// stringPtr returns a pointer to a string.
func stringPtr(s string) *string {
	return &s
}

// intPtr returns a pointer to an int.
func intPtr(i int) *int {
	return &i
}
