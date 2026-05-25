package main

import (
	"database/sql"
	"flag"
	"fmt"
	"net/url"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

func main() {
	var (
		direction = flag.String("direction", "up", "Migration direction: up or down")
		steps     = flag.Int("steps", 0, "Number of migrations to run (0 = all)")
		dbURL     = flag.String("database-url", "", "Database URL (overrides DATABASE_URL env)")
	)
	flag.Parse()

	baseURL := *dbURL
	if baseURL == "" {
		baseURL = os.Getenv("DATABASE_URL")
	}
	if baseURL == "" {
		baseURL = "postgres://kita:kita_dev_password@localhost:5432/kita?sslmode=disable"
	}

	db, err := sql.Open("postgres", baseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	if _, err := db.Exec("CREATE SCHEMA IF NOT EXISTS portal"); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create portal schema: %v\n", err)
		os.Exit(1)
	}
	_ = db.Close()

	m, err := migrate.New("file://migrations", migrationDatabaseURL(baseURL))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create migrate instance: %v\n", err)
		os.Exit(1)
	}
	defer m.Close()

	switch *direction {
	case "up":
		if *steps > 0 {
			err = m.Steps(*steps)
		} else {
			err = m.Up()
		}
	case "down":
		if *steps > 0 {
			err = m.Steps(-*steps)
		} else {
			err = m.Down()
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown direction: %s (use 'up' or 'down')\n", *direction)
		os.Exit(1)
	}

	if err != nil && err != migrate.ErrNoChange {
		fmt.Fprintf(os.Stderr, "Migration failed: %v\n", err)
		os.Exit(1)
	}

	if err == migrate.ErrNoChange {
		fmt.Println("No migrations to apply")
		return
	}
	fmt.Printf("Migrations applied successfully (%s)\n", *direction)
}

func migrationDatabaseURL(databaseURL string) string {
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		return databaseURL
	}

	// Keep migrate bookkeeping outside the portal schema so down migrations can drop it.
	query := parsed.Query()
	query.Set("search_path", "public")
	query.Set("x-migrations-table", "portal_schema_migrations")
	parsed.RawQuery = query.Encode()
	return parsed.String()
}
