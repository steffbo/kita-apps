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
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/email"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

// @title Kita Knirpsenstadt Fees API
// @version 1.0.0
// @description API für Beitragsverwaltung der Kita Knirpsenstadt
// @contact.name Kita Knirpsenstadt

// @host localhost:8081
// @BasePath /api/fees/v1
// @schemes http https

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description JWT Bearer token authentication

// @tag.name Auth
// @tag.description Authentifizierung und Autorisierung
// @tag.name Children
// @tag.description Kinderverwaltung
// @tag.name Parents
// @tag.description Elternverwaltung
// @tag.name Households
// @tag.description Haushaltsverwaltung
// @tag.name Members
// @tag.description Vereinsmitglieder
// @tag.name Fees
// @tag.description Beitragsverwaltung
// @tag.name Import
// @tag.description Bankdaten-Import
// @tag.name Calculator
// @tag.description Gebührenrechner

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
	warningRepo := repository.NewPostgresWarningRepository(db)
	settingsRepo := repository.NewPostgresSettingsRepository(db)
	emailLogRepo := repository.NewPostgresEmailLogRepository(db)

	// Initialize services
	jwtService := auth.NewJWTService(cfg.JWT.Secret, cfg.JWT.AccessExpiry, cfg.JWT.RefreshExpiry, cfg.JWT.Issuer)
	emailService := email.NewService(email.Config{
		Host:     cfg.SMTP.Host,
		Port:     cfg.SMTP.Port,
		From:     cfg.SMTP.From,
		Username: cfg.SMTP.Username,
		Password: cfg.SMTP.Password,
		UseTLS:   cfg.SMTP.UseTLS,
	})
	authService := service.NewAuthService(userRepo, refreshTokenRepo, emailService, emailLogRepo, cfg.SMTP.BaseURL, cfg.JWT.RefreshExpiry)

	childService := service.NewChildService(childRepo, parentRepo, householdRepo)
	parentService := service.NewParentService(parentRepo, childRepo, memberRepo)
	householdService := service.NewHouseholdService(householdRepo, parentRepo, childRepo)
	memberService := service.NewMemberService(memberRepo, householdRepo)
	feeService := service.NewFeeService(feeRepo, childRepo, householdRepo, matchRepo, transactionRepo)
	importService := service.NewImportService(transactionRepo, feeRepo, childRepo, matchRepo, knownIBANRepo, warningRepo)
	childImportService := service.NewChildImportService(childRepo, parentRepo)
	coverageService := service.NewCoverageService(feeRepo, childRepo, transactionRepo, matchRepo)
	reminderService := service.NewReminderService(feeRepo, childRepo, settingsRepo, emailLogRepo, emailService)

	// Initialize handlers
	handlers := &api.Handlers{
		Auth:        handler.NewAuthHandler(authService, jwtService),
		Child:       handler.NewChildHandler(childService, feeService, coverageService, feeRepo, matchRepo, transactionRepo),
		ChildImport: handler.NewChildImportHandler(childImportService),
		Parent:      handler.NewParentHandler(parentService),
		Household:   handler.NewHouseholdHandler(householdService),
		Member:      handler.NewMemberHandler(memberService),
		Fee:         handler.NewFeeHandler(feeService, importService, reminderService, emailLogRepo),
		Import:      handler.NewImportHandler(importService),
		BankingSync: handler.NewBankingSyncHandler(cfg.BankingSync.BaseURL, cfg.BankingSync.Token, cfg.BankingSync.Timeout),
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
