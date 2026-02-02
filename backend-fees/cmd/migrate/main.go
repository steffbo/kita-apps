package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "up":
		runMigration("up", 0)
	case "down":
		steps := 1
		if len(os.Args) > 2 {
			s, err := strconv.Atoi(os.Args[2])
			if err == nil {
				steps = s
			}
		}
		runMigration("down", steps)
	case "status":
		showStatus()
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: go run cmd/migrate/main.go <command>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  up              Run all pending migrations")
	fmt.Println("  down [n]        Rollback n migrations (default: 1)")
	fmt.Println("  status          Show current migration status")
	fmt.Println("  help            Show this help message")
	fmt.Println()
	fmt.Println("Environment:")
	fmt.Println("  DATABASE_URL    PostgreSQL connection URL")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  go run cmd/migrate/main.go up")
	fmt.Println("  go run cmd/migrate/main.go down 1")
	fmt.Println("  go run cmd/migrate/main.go status")
}

func getDatabaseURL() string {
	baseURL := os.Getenv("DATABASE_URL")
	if baseURL == "" {
		baseURL = "postgres://kita:kita_dev_password@localhost:5432/kita?sslmode=disable"
	}
	return baseURL
}

func createMigrateInstance() (*migrate.Migrate, string, error) {
	baseURL := getDatabaseURL()

	// Ensure fees schema exists
	db, err := sql.Open("postgres", baseURL)
	if err != nil {
		return nil, "", fmt.Errorf("failed to connect to database: %w", err)
	}
	_, err = db.Exec("CREATE SCHEMA IF NOT EXISTS fees")
	if err != nil {
		db.Close()
		return nil, "", fmt.Errorf("failed to create fees schema: %w", err)
	}
	db.Close()

	// Build URL with search_path
	databaseURL := baseURL
	if !strings.Contains(baseURL, "search_path=") {
		databaseURL = baseURL + "&search_path=fees"
	}

	m, err := migrate.New("file://migrations", databaseURL)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create migrate instance: %w", err)
	}

	return m, databaseURL, nil
}

func runMigration(direction string, steps int) {
	m, _, err := createMigrateInstance()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer m.Close()

	var migrateErr error
	if direction == "up" {
		if steps > 0 {
			migrateErr = m.Steps(steps)
		} else {
			migrateErr = m.Up()
		}
	} else {
		if steps > 0 {
			migrateErr = m.Steps(-steps)
		} else {
			migrateErr = m.Down()
		}
	}

	if migrateErr != nil && migrateErr != migrate.ErrNoChange {
		fmt.Fprintf(os.Stderr, "Migration failed: %v\n", migrateErr)
		os.Exit(1)
	}

	if migrateErr == migrate.ErrNoChange {
		fmt.Println("No migrations to apply")
	} else {
		fmt.Printf("Migrations applied successfully (%s)\n", direction)
	}
}

func showStatus() {
	m, _, err := createMigrateInstance()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer m.Close()

	// Get current version
	version, dirty, err := m.Version()
	if err != nil {
		if err == migrate.ErrNilVersion {
			fmt.Println("Current version: (none - no migrations applied yet)")
		} else {
			fmt.Fprintf(os.Stderr, "Error getting version: %v\n", err)
			os.Exit(1)
		}
	} else {
		status := "clean"
		if dirty {
			status = "DIRTY"
		}
		fmt.Printf("Current version: %d (%s)\n", version, status)
	}

	// Count available migrations
	migrationsPath := "migrations"
	files, err := filepath.Glob(filepath.Join(migrationsPath, "*.up.sql"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not list migrations: %v\n", err)
	} else {
		fmt.Printf("Available migrations: %d\n", len(files))
	}

	// Show next migration
	if version > 0 && len(files) > 0 {
		fmt.Printf("Next migration: %s\n", getMigrationName(files, int(version)))
	}
}

func getMigrationName(files []string, currentVersion int) string {
	for _, file := range files {
		base := filepath.Base(file)
		parts := strings.Split(base, "_")
		if len(parts) > 0 {
			v, err := strconv.Atoi(parts[0])
			if err == nil && v == currentVersion+1 {
				return strings.TrimSuffix(base, ".up.sql")
			}
		}
	}
	return "(none - up to date)"
}
