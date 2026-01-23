package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/handler"
	customMiddleware "github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/middleware"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/auth"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/config"
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

	// Health check (public)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// API v1 routes
	r.Route("/api/fees/v1", func(r chi.Router) {
		// Public routes
		r.Route("/auth", func(r chi.Router) {
			r.Post("/login", handlers.Auth.Login)
			r.Post("/refresh", handlers.Auth.Refresh)
		})

		// Public childcare fee calculator
		r.Get("/childcare-fee/calculate", handlers.Fee.CalculateChildcareFee)

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(customMiddleware.AuthMiddleware(handlers.JWTService))

			// Auth
			r.Post("/auth/logout", handlers.Auth.Logout)
			r.Get("/auth/me", handlers.Auth.Me)

			// Children
			r.Route("/children", func(r chi.Router) {
				r.Get("/", handlers.Child.List)
				r.Post("/", handlers.Child.Create)
				r.Get("/{id}", handlers.Child.Get)
				r.Put("/{id}", handlers.Child.Update)
				r.Delete("/{id}", handlers.Child.Delete)
				r.Post("/{id}/parents", handlers.Child.LinkParent)
				r.Delete("/{id}/parents/{parentId}", handlers.Child.UnlinkParent)

				// Child import routes
				r.Route("/import", func(r chi.Router) {
					r.Post("/parse", handlers.ChildImport.Parse)
					r.Post("/preview", handlers.ChildImport.Preview)
					r.Post("/execute", handlers.ChildImport.Execute)
				})
			})

			// Parents
			r.Route("/parents", func(r chi.Router) {
				r.Get("/", handlers.Parent.List)
				r.Post("/", handlers.Parent.Create)
				r.Get("/{id}", handlers.Parent.Get)
				r.Put("/{id}", handlers.Parent.Update)
				r.Delete("/{id}", handlers.Parent.Delete)
			})

			// Households
			r.Route("/households", func(r chi.Router) {
				r.Get("/", handlers.Household.List)
				r.Post("/", handlers.Household.Create)
				r.Get("/{id}", handlers.Household.Get)
				r.Put("/{id}", handlers.Household.Update)
				r.Delete("/{id}", handlers.Household.Delete)
				r.Post("/{id}/parents", handlers.Household.LinkParent)
				r.Post("/{id}/children", handlers.Household.LinkChild)
			})

			// Members (Vereinsmitglieder)
			r.Route("/members", func(r chi.Router) {
				r.Get("/", handlers.Member.List)
				r.Post("/", handlers.Member.Create)
				r.Get("/{id}", handlers.Member.Get)
				r.Put("/{id}", handlers.Member.Update)
				r.Delete("/{id}", handlers.Member.Delete)
			})

			// Fees
			r.Route("/fees", func(r chi.Router) {
				r.Get("/", handlers.Fee.List)
				r.Get("/overview", handlers.Fee.Overview)
				r.Post("/generate", handlers.Fee.Generate)
				r.Get("/{id}", handlers.Fee.Get)
				r.Put("/{id}", handlers.Fee.Update)
				r.Delete("/{id}", handlers.Fee.Delete)
				r.Post("/{id}/reminder", handlers.Fee.CreateReminder)
			})

			// Import
			r.Route("/import", func(r chi.Router) {
				r.Post("/upload", handlers.Import.Upload)
				r.Post("/confirm", handlers.Import.Confirm)
				r.Get("/history", handlers.Import.History)
				r.Get("/transactions", handlers.Import.UnmatchedTransactions)
				r.Post("/match", handlers.Import.ManualMatch)
				r.Post("/rescan", handlers.Import.Rescan)
				r.Post("/transactions/{id}/dismiss", handlers.Import.DismissTransaction)
				r.Get("/blacklist", handlers.Import.GetBlacklist)
				r.Delete("/blacklist/{iban}", handlers.Import.RemoveFromBlacklist)
				r.Get("/trusted", handlers.Import.GetTrustedIBANs)
				r.Post("/trusted/{iban}/link", handlers.Import.LinkIBANToChild)
				r.Delete("/trusted/{iban}/link", handlers.Import.UnlinkIBANFromChild)
				r.Get("/warnings", handlers.Import.GetWarnings)
				r.Post("/warnings/{id}/dismiss", handlers.Import.DismissWarning)
			})
		})
	})

	return r
}

// Handlers holds all HTTP handlers.
type Handlers struct {
	Auth        *handler.AuthHandler
	Child       *handler.ChildHandler
	ChildImport *handler.ChildImportHandler
	Parent      *handler.ParentHandler
	Household   *handler.HouseholdHandler
	Member      *handler.MemberHandler
	Fee         *handler.FeeHandler
	Import      *handler.ImportHandler
	JWTService  *auth.JWTService
}
