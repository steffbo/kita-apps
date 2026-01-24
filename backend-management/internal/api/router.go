package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/knirpsenstadt/kita-apps/backend-management/internal/api/handler"
	customMiddleware "github.com/knirpsenstadt/kita-apps/backend-management/internal/api/middleware"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/auth"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/config"
)

// NewRouter creates and configures the HTTP router.
func NewRouter(cfg *config.Config, handlers *Handlers) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(customMiddleware.Logging)
	r.Use(middleware.Recoverer)

	// CORS configuration
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.Server.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health checks
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})
	// Match Spring Boot actuator path used by existing tooling.
	r.Get("/api/actuator/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"UP"}`))
	})

	r.Route("/api", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			// Public auth routes
			r.Post("/login", handlers.Auth.Login)
			r.Post("/refresh", handlers.Auth.Refresh)
			r.Post("/password-reset/request", handlers.Auth.RequestPasswordReset)
			r.Post("/password-reset/confirm", handlers.Auth.ConfirmPasswordReset)

			// Protected auth routes
			r.Group(func(r chi.Router) {
				r.Use(customMiddleware.AuthMiddleware(handlers.JWTService))
				r.Get("/me", handlers.Auth.Me)
				r.Post("/change-password", handlers.Auth.ChangePassword)
			})
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(customMiddleware.AuthMiddleware(handlers.JWTService))

			r.Route("/employees", func(r chi.Router) {
				r.Get("/", handlers.Employee.List)
				r.Get("/{id}", handlers.Employee.Get)
				r.Get("/{id}/assignments", handlers.Employee.Assignments)

				r.Group(func(r chi.Router) {
					r.Use(customMiddleware.RequireRole("ADMIN"))
					r.Post("/", handlers.Employee.Create)
					r.Put("/{id}", handlers.Employee.Update)
					r.Delete("/{id}", handlers.Employee.Delete)
					r.Post("/{id}/reset-password", handlers.Employee.ResetPassword)
				})
			})

			r.Route("/groups", func(r chi.Router) {
				r.Use(customMiddleware.RequireRole("ADMIN"))
				r.Get("/", handlers.Group.List)
				r.Post("/", handlers.Group.Create)
				r.Get("/{id}", handlers.Group.Get)
				r.Put("/{id}", handlers.Group.Update)
				r.Delete("/{id}", handlers.Group.Delete)
				r.Get("/{id}/assignments", handlers.Group.Assignments)
				r.Put("/{id}/assignments", handlers.Group.UpdateAssignments)
			})

			r.Route("/schedule", func(r chi.Router) {
				r.Get("/", handlers.Schedule.List)
				r.Get("/week", handlers.Schedule.Week)

				r.Group(func(r chi.Router) {
					r.Use(customMiddleware.RequireRole("ADMIN"))
					r.Post("/", handlers.Schedule.Create)
					r.Post("/bulk", handlers.Schedule.BulkCreate)
					r.Put("/{id}", handlers.Schedule.Update)
					r.Delete("/{id}", handlers.Schedule.Delete)
				})
			})

			r.Route("/time-tracking", func(r chi.Router) {
				r.Post("/clock-in", handlers.TimeTracking.ClockIn)
				r.Post("/clock-out", handlers.TimeTracking.ClockOut)
				r.Get("/current", handlers.TimeTracking.Current)
				r.Get("/comparison", handlers.TimeTracking.Comparison)

				r.Route("/entries", func(r chi.Router) {
					r.Get("/", handlers.TimeTracking.List)

					r.Group(func(r chi.Router) {
						r.Use(customMiddleware.RequireRole("ADMIN"))
						r.Post("/", handlers.TimeTracking.Create)
						r.Put("/{id}", handlers.TimeTracking.Update)
						r.Delete("/{id}", handlers.TimeTracking.Delete)
					})
				})
			})

			r.Route("/special-days", func(r chi.Router) {
				r.Use(customMiddleware.RequireRole("ADMIN"))
				r.Get("/", handlers.SpecialDay.List)
				r.Post("/", handlers.SpecialDay.Create)
				r.Get("/holidays/{year}", handlers.SpecialDay.Holidays)
				r.Put("/{id}", handlers.SpecialDay.Update)
				r.Delete("/{id}", handlers.SpecialDay.Delete)
			})

			r.Route("/statistics", func(r chi.Router) {
				r.Get("/employee/{id}", handlers.Statistics.Employee)
				r.Get("/weekly", handlers.Statistics.Weekly)

				r.Group(func(r chi.Router) {
					r.Use(customMiddleware.RequireRole("ADMIN"))
					r.Get("/overview", handlers.Statistics.Overview)
				})
			})

			r.Route("/export", func(r chi.Router) {
				r.Use(customMiddleware.RequireRole("ADMIN"))
				r.Get("/timesheet", handlers.Statistics.ExportTimesheet)
				r.Get("/schedule", handlers.Statistics.ExportSchedule)
			})
		})
	})

	return r
}

// Handlers holds all HTTP handlers.
type Handlers struct {
	Auth         *handler.AuthHandler
	Employee     *handler.EmployeeHandler
	Group        *handler.GroupHandler
	Schedule     *handler.ScheduleHandler
	TimeTracking *handler.TimeTrackingHandler
	SpecialDay   *handler.SpecialDayHandler
	Statistics   *handler.StatisticsHandler
	JWTService   *auth.JWTService
}
