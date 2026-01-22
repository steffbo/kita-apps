package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/handler"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/auth"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/config"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

func main() {
	// Configure logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})

	// Load configuration
	cfg := config.Load()
	log.Info().Str("port", cfg.Server.Port).Msg("Starting fees service")

	// Connect to database
	db, err := connectDB(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	// Initialize repositories
	userRepo := repository.NewPostgresUserRepository(db)
	refreshTokenRepo := repository.NewPostgresRefreshTokenRepository(db)
	childRepo := repository.NewPostgresChildRepository(db)
	parentRepo := repository.NewPostgresParentRepository(db)
	householdRepo := repository.NewPostgresHouseholdRepository(db)
	memberRepo := repository.NewPostgresMemberRepository(db)
	feeRepo := repository.NewPostgresFeeRepository(db)
	transactionRepo := repository.NewPostgresTransactionRepository(db)
	matchRepo := repository.NewPostgresMatchRepository(db)
	knownIBANRepo := repository.NewPostgresKnownIBANRepository(db)

	// Initialize services
	jwtService := auth.NewJWTService(cfg.JWT.Secret, cfg.JWT.AccessExpiry, cfg.JWT.RefreshExpiry, cfg.JWT.Issuer)
	authService := service.NewAuthService(userRepo, refreshTokenRepo, cfg.JWT.RefreshExpiry)
	childService := service.NewChildService(childRepo, parentRepo, householdRepo)
	parentService := service.NewParentService(parentRepo, childRepo)
	householdService := service.NewHouseholdService(householdRepo, parentRepo, childRepo)
	memberService := service.NewMemberService(memberRepo, householdRepo)
	feeService := service.NewFeeService(feeRepo, childRepo, householdRepo, matchRepo, transactionRepo)
	importService := service.NewImportService(transactionRepo, feeRepo, childRepo, matchRepo, knownIBANRepo)
	childImportService := service.NewChildImportService(childRepo, parentRepo)

	// Initialize handlers
	handlers := &api.Handlers{
		Auth:        handler.NewAuthHandler(authService, jwtService),
		Child:       handler.NewChildHandler(childService),
		ChildImport: handler.NewChildImportHandler(childImportService),
		Parent:      handler.NewParentHandler(parentService),
		Household:   handler.NewHouseholdHandler(householdService),
		Member:      handler.NewMemberHandler(memberService),
		Fee:         handler.NewFeeHandler(feeService, importService),
		Import:      handler.NewImportHandler(importService),
		JWTService:  jwtService,
	}

	// Create router
	router := api.NewRouter(cfg, handlers)

	// Create server
	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server in goroutine
	go func() {
		log.Info().Str("addr", server.Addr).Msg("HTTP server listening")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("HTTP server error")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Server shutdown error")
	}

	log.Info().Msg("Server stopped")
}

func connectDB(cfg *config.Config) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", cfg.Database.URL)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

	// Ping to verify connection
	if err := db.Ping(); err != nil {
		return nil, err
	}

	log.Info().Msg("Connected to database")
	return db, nil
}
