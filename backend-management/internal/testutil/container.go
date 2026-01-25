package testutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestContainer holds a running PostgreSQL test container.
type TestContainer struct {
	Container *postgres.PostgresContainer
	DB        *sqlx.DB
	ConnStr   string
}

// SetupPostgres starts a PostgreSQL container and runs migrations.
// Call Cleanup() when done.
func SetupPostgres(ctx context.Context) (*TestContainer, error) {
	// Get migrations path relative to this file
	_, filename, _, _ := runtime.Caller(0)
	migrationsPath := filepath.Join(filepath.Dir(filename), "..", "..", "migrations")

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
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get connection string: %w", err)
	}

	// Retry connection a few times as the container may still be starting
	var db *sqlx.DB
	for i := 0; i < 10; i++ {
		db, err = sqlx.Connect("postgres", connStr)
		if err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Run migrations
	if err := runMigrations(db, migrationsPath); err != nil {
		db.Close()
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return &TestContainer{
		Container: container,
		DB:        db,
		ConnStr:   connStr,
	}, nil
}

// Cleanup terminates the container and closes connections.
func (tc *TestContainer) Cleanup(ctx context.Context) error {
	if tc.DB != nil {
		tc.DB.Close()
	}
	if tc.Container != nil {
		return tc.Container.Terminate(ctx)
	}
	return nil
}

// runMigrations executes all up migrations.
func runMigrations(db *sqlx.DB, migrationsPath string) error {
	// Read and execute migration files in order
	migrations := []string{
		"000001_initial_schema.up.sql",
		"000003_add_special_day_end_date.up.sql",
		"000004_add_springer_group.up.sql",
		// Note: 000002_seed_data.up.sql is skipped for tests - we create our own test data
	}

	for _, migration := range migrations {
		path := filepath.Join(migrationsPath, migration)
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", migration, err)
		}

		if _, err := db.Exec(string(content)); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", migration, err)
		}
	}

	return nil
}
