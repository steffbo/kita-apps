package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/knirpsenstadt/kita-apps/backend-portal/internal/api"
	"github.com/knirpsenstadt/kita-apps/backend-portal/internal/auth"
	"github.com/knirpsenstadt/kita-apps/backend-portal/internal/config"
	"github.com/knirpsenstadt/kita-apps/backend-portal/internal/repository"
)

func main() {
	_ = godotenv.Load()

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})

	cfg := config.Load()
	log.Info().Str("port", cfg.Server.Port).Msg("Starting portal service")

	db, err := connectDB(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	if err := bootstrapAdmin(context.Background(), cfg, db); err != nil {
		log.Fatal().Err(err).Msg("Failed to bootstrap portal admin")
	}

	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      api.NewRouter(cfg, db),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		log.Info().Str("addr", server.Addr).Msg("HTTP server listening")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("HTTP server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Server shutdown error")
	}
}

func bootstrapAdmin(ctx context.Context, cfg *config.Config, db *sqlx.DB) error {
	if cfg.Bootstrap.AdminEmail == "" && cfg.Bootstrap.AdminPassword == "" {
		return nil
	}
	if cfg.Bootstrap.AdminEmail == "" || cfg.Bootstrap.AdminPassword == "" {
		log.Warn().Msg("Skipping portal admin bootstrap because email or password is missing")
		return nil
	}

	hash, err := auth.HashPassword(cfg.Bootstrap.AdminPassword)
	if err != nil {
		return err
	}

	user, err := repository.NewUserRepository(db).EnsureBootstrapAdmin(ctx, cfg.Bootstrap.AdminEmail, hash)
	if err != nil {
		return err
	}
	if user != nil {
		log.Info().Str("email", user.Email).Msg("Portal bootstrap admin ensured")
	}
	return nil
}

func connectDB(cfg *config.Config) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", cfg.Database.URL)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	log.Info().Msg("Connected to database")
	return db, nil
}
