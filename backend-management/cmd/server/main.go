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

	"github.com/knirpsenstadt/kita-apps/backend-management/internal/api"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/api/handler"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/auth"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/config"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/service"
)

func main() {
	// Configure logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})

	// Load configuration
	cfg := config.Load()
	log.Info().Str("port", cfg.Server.Port).Msg("Starting management service")

	// Connect to database
	db, err := connectDB(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	// Initialize repositories
	employeeRepo := repository.NewPostgresEmployeeRepository(db)
	groupRepo := repository.NewPostgresGroupRepository(db)
	assignmentRepo := repository.NewPostgresGroupAssignmentRepository(db)
	scheduleRepo := repository.NewPostgresScheduleRepository(db)
	timeEntryRepo := repository.NewPostgresTimeEntryRepository(db)
	specialDayRepo := repository.NewPostgresSpecialDayRepository(db)

	// Initialize services
	jwtService := auth.NewJWTService(cfg.JWT.Secret, cfg.JWT.AccessExpiry, cfg.JWT.RefreshExpiry, cfg.JWT.Issuer)
	authService := service.NewAuthService(employeeRepo, jwtService, time.Hour)
	employeeService := service.NewEmployeeService(employeeRepo, assignmentRepo, groupRepo)
	groupService := service.NewGroupService(groupRepo, assignmentRepo, employeeRepo)
	scheduleService := service.NewScheduleService(scheduleRepo, employeeRepo, groupRepo, specialDayRepo)
	timeTrackingService := service.NewTimeTrackingService(timeEntryRepo, employeeRepo, scheduleRepo)
	specialDayService := service.NewSpecialDayService(specialDayRepo)
	statisticsService := service.NewStatisticsService(employeeRepo, scheduleRepo, timeEntryRepo, groupRepo)

	// Initialize handlers
	handlers := &api.Handlers{
		Auth:         handler.NewAuthHandler(authService),
		Employee:     handler.NewEmployeeHandler(employeeService),
		Group:        handler.NewGroupHandler(groupService),
		Schedule:     handler.NewScheduleHandler(scheduleService),
		TimeTracking: handler.NewTimeTrackingHandler(timeTrackingService),
		SpecialDay:   handler.NewSpecialDayHandler(specialDayService),
		Statistics:   handler.NewStatisticsHandler(statisticsService),
		JWTService:   jwtService,
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
