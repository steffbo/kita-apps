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
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/frontend"
)

// NewRouter creates and configures the HTTP router.
func NewRouter(cfg *config.Config, handlers *Handlers) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(customMiddleware.Logging)
	r.Use(middleware.Recoverer)
	r.Use(customMiddleware.MaxBodySize(customMiddleware.MaxUploadBytes))

	// CORS is only needed for cross-origin browser clients. The Beiträge frontend
	// is served same-origin (embedded in prod, Vite proxy in dev).
	if len(cfg.Server.CORSOrigins) > 0 {
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   cfg.Server.CORSOrigins,
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
			ExposedHeaders:   []string{"Link"},
			AllowCredentials: cfg.Server.CORSAllowCredentials,
			MaxAge:           300,
		}))
	}

	// Health check (public)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Embedded frontend route (served when built with -tags embed_frontend)
	r.Mount("/beitraege", frontend.BeitraegeHandler())

	// API v1 routes
	r.Route("/api/fees/v1", func(r chi.Router) {
		// Public routes
		r.Route("/auth", func(r chi.Router) {
			r.Post("/login", handlers.Auth.Login)
			// Refresh and logout authenticate via the httpOnly refresh cookie.
			r.Post("/refresh", handlers.Auth.Refresh)
			r.Post("/logout", handlers.Auth.Logout)
			r.Post("/invitation-password", handlers.Invitation.SetPassword)
		})

		// Public childcare fee calculator
		r.Get("/childcare-fee/calculate", handlers.Fee.CalculateChildcareFee)

		// CSV import upload (JWT or import token)
		r.With(customMiddleware.ImportAuthMiddleware(handlers.JWTService, cfg.Import.Token)).
			Post("/import/upload", handlers.Import.Upload)

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(customMiddleware.AuthMiddleware(handlers.JWTService))

			// Auth
			r.Get("/auth/me", handlers.Auth.Me)
			r.Post("/auth/change-password", handlers.Auth.ChangePassword)

			r.Route("/me", func(r chi.Router) {
				r.Use(customMiddleware.RequireRole(string(domain.UserRolePARENT)))
				r.Get("/", handlers.ParentAccount.Me)
				r.Get("/fees", handlers.ParentAccount.Fees)
				r.Get("/parent-work", handlers.ParentAccount.ParentWork)
				r.Post("/parent-work/entries", handlers.ParentAccount.SubmitEntry)
				r.Post("/parent-work/entries/{id}/withdraw", handlers.ParentAccount.WithdrawEntry)
				r.Put("/contact", handlers.ParentAccount.Contact)
				r.Put("/parents/{id}/contact", handlers.ParentAccount.ParentContact)
				r.Get("/reports", handlers.ParentAccount.OwnReports)
				r.Post("/reports", handlers.ParentAccount.CreateReport)
			})
			r.With(customMiddleware.RequireRole(string(domain.UserRoleAdmin))).
				Get("/activity", handlers.ParentAccount.Activity)
			r.Route("/parent-reports", func(r chi.Router) {
				r.Use(customMiddleware.RequireRole(string(domain.UserRoleAdmin)))
				r.Get("/", handlers.ParentAccount.StaffReports)
				r.Post("/{id}/resolve", handlers.ParentAccount.ResolveReport)
			})

			// User management (admin only)
			r.Route("/users", func(r chi.Router) {
				r.Use(customMiddleware.RequireRole(string(domain.UserRoleAdmin)))
				r.Get("/", handlers.User.List)
				r.Post("/", handlers.User.Create)
				r.Get("/invitation-candidates", handlers.Invitation.Candidates)
				r.Post("/invitations", handlers.Invitation.Invite)
				r.Post("/{id}/invitation", handlers.Invitation.Resend)
				r.Put("/{id}", handlers.User.Update)
				r.Post("/{id}/password", handlers.User.SetPassword)
				r.Post("/{id}/impersonate", handlers.User.Impersonate)
			})

			// Parent-work account routes.
			r.Route("/parent-work", func(r chi.Router) {
				r.Use(customMiddleware.RequireRole(
					string(domain.UserRoleAdmin), string(domain.UserRoleParentWork)))
				r.Get("/overview", handlers.ParentWork.Overview)
				r.Get("/households", handlers.ParentWork.Households)
				r.Get("/households/{id}", handlers.ParentWork.Household)
				r.Put("/households/{id}/override", handlers.ParentWork.SaveOverride)
				r.Delete("/households/{id}/override", handlers.ParentWork.DeleteOverride)
				r.Post("/entries", handlers.ParentWork.CreateEntry)
				r.Put("/entries/{id}", handlers.ParentWork.UpdateEntry)
				r.Post("/entries/{id}/void", handlers.ParentWork.VoidEntry)
				r.Post("/entries/{id}/approve", handlers.ParentWork.ApproveEntry)
				r.Post("/entries/{id}/reject", handlers.ParentWork.RejectEntry)
				r.Group(func(r chi.Router) {
					r.Use(customMiddleware.RequireRole(string(domain.UserRoleAdmin)))
					r.Get("/board-terms", handlers.ParentWork.Terms)
					r.Post("/board-terms", handlers.ParentWork.CreateTerm)
					r.Put("/board-terms/{id}", handlers.ParentWork.UpdateTerm)
					r.Delete("/board-terms/{id}", handlers.ParentWork.DeleteTerm)
				})
				r.Get("/rules", handlers.ParentWork.Rules)
				r.Post("/import/parse", handlers.ParentWork.ParseImport)
				r.Post("/import/preview", handlers.ParentWork.PreviewImport)
				r.Post("/import/execute", handlers.ParentWork.ExecuteImport)
				r.With(customMiddleware.RequireRole(string(domain.UserRoleAdmin))).
					Post("/rules", handlers.ParentWork.CreateRule)
				r.With(customMiddleware.RequireRole(string(domain.UserRoleAdmin))).
					Put("/rules/{id}", handlers.ParentWork.UpdateRule)
			})

			r.Group(func(r chi.Router) {
				r.Use(customMiddleware.RequireRole(
					string(domain.UserRoleAdmin), string(domain.UserRoleUser)))

				// Children
				r.Route("/children", func(r chi.Router) {
					r.Get("/", handlers.Child.List)
					r.Get("/next-member-number", handlers.Child.NextMemberNumber)
					r.Post("/", handlers.Child.Create)
					r.Get("/{id}", handlers.Child.Get)
					r.With(customMiddleware.RequireRole(string(domain.UserRoleAdmin))).
						Get("/{id}/changes", handlers.ParentAccount.ChildChanges)
					r.Put("/{id}", handlers.Child.Update)
					r.Delete("/{id}", handlers.Child.Delete)
					r.Get("/{id}/care-hours-history", handlers.Child.GetCareHoursHistory)
					r.Post("/{id}/care-hours-history", handlers.Child.AddCareHoursHistory)
					r.Get("/{id}/legal-hours-history", handlers.Child.GetLegalHoursHistory)
					r.Post("/{id}/legal-hours-history", handlers.Child.AddLegalHoursHistory)
					r.Get("/{id}/ledger", handlers.Child.GetLedger)
					r.Get("/{id}/timeline", handlers.Child.GetTimeline)
					r.Post("/{id}/parents", handlers.Child.LinkParent)
					r.Delete("/{id}/parents/{parentId}", handlers.Child.UnlinkParent)

					// Child notes
					r.Get("/{id}/notes", handlers.ChildNote.ListByChild)
					r.Post("/{id}/notes", handlers.ChildNote.Create)
					r.Put("/{id}/notes/{noteId}", handlers.ChildNote.Update)
					r.Delete("/{id}/notes/{noteId}", handlers.ChildNote.Delete)

					// Child import routes
					r.Route("/import", func(r chi.Router) {
						r.Post("/parse", handlers.ChildImport.Parse)
						r.Post("/preview", handlers.ChildImport.Preview)
						r.Post("/execute", handlers.ChildImport.Execute)
					})
				})

				// Notes (global list across children)
				r.Get("/notes", handlers.ChildNote.ListAll)

				// Parents
				r.Route("/parents", func(r chi.Router) {
					r.Get("/", handlers.Parent.List)
					r.Post("/", handlers.Parent.Create)
					r.Get("/{id}", handlers.Parent.Get)
					r.With(customMiddleware.RequireRole(string(domain.UserRoleAdmin))).
						Get("/{id}/changes", handlers.ParentAccount.ParentChanges)
					r.Put("/{id}", handlers.Parent.Update)
					r.Delete("/{id}", handlers.Parent.Delete)
					r.Post("/{id}/member", handlers.Parent.CreateMember)
					r.Delete("/{id}/member", handlers.Parent.UnlinkMember)
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
					r.Get("/count", handlers.Member.CountAsOf)
					r.Get("/{id}", handlers.Member.Get)
					r.Put("/{id}", handlers.Member.Update)
					r.Delete("/{id}", handlers.Member.Delete)
				})

				// Banking sync (admin only)
				r.Route("/banking-sync", func(r chi.Router) {
					r.With(customMiddleware.RequireRole(string(domain.UserRoleAdmin))).Post("/run", handlers.BankingSync.Run)
					r.With(customMiddleware.RequireRole(string(domain.UserRoleAdmin))).Get("/status", handlers.BankingSync.Status)
					r.With(customMiddleware.RequireRole(string(domain.UserRoleAdmin))).Post("/cancel", handlers.BankingSync.Cancel)
				})

				// Fees
				r.Route("/fees", func(r chi.Router) {
					r.Get("/", handlers.Fee.List)
					r.Post("/", handlers.Fee.Create)
					r.Get("/overview", handlers.Fee.Overview)
					r.Post("/generate", handlers.Fee.Generate)
					r.With(customMiddleware.RequireRole(string(domain.UserRoleAdmin))).Get("/reminders/settings", handlers.Fee.GetReminderSettings)
					r.With(customMiddleware.RequireRole(string(domain.UserRoleAdmin))).Put("/reminders/settings", handlers.Fee.UpdateReminderSettings)
					r.With(customMiddleware.RequireRole(string(domain.UserRoleAdmin))).Get("/email-logs", handlers.Fee.GetEmailLogs)
					r.With(customMiddleware.RequireRole(string(domain.UserRoleAdmin))).Get("/reminder-cases", handlers.Fee.GetReminderCases)
					r.With(customMiddleware.RequireRole(string(domain.UserRoleAdmin))).Post("/reminder-cases/{householdId}/preview", handlers.Fee.PreviewReminderCase)
					r.With(customMiddleware.RequireRole(string(domain.UserRoleAdmin))).Post("/reminder-cases/{householdId}/send", handlers.Fee.SendReminderCase)
					r.Get("/{id}", handlers.Fee.Get)
					r.Put("/{id}", handlers.Fee.Update)
					r.Delete("/{id}", handlers.Fee.Delete)
					r.Post("/{id}/reminder", handlers.Fee.CreateReminder)
				})

				// Einstufungen (fee classifications)
				r.Route("/einstufungen", func(r chi.Router) {
					r.Get("/", handlers.Einstufung.List)
					r.Post("/", handlers.Einstufung.Create)
					r.Post("/calculate-income", handlers.Einstufung.CalculateIncome)
					r.Get("/child/{childId}", handlers.Einstufung.GetForChild)
					r.Get("/household/{householdId}", handlers.Einstufung.ListForHousehold)
					r.Get("/{id}", handlers.Einstufung.Get)
					r.Put("/{id}", handlers.Einstufung.Update)
					r.Post("/{id}/follow-ups", handlers.Einstufung.CreateFollowUp)
					r.Delete("/{id}", handlers.Einstufung.Delete)
				})

				// Fee regulation versions (Beitragsordnung); only planned versions are writable
				r.Route("/fee-schedules", func(r chi.Router) {
					r.Get("/", handlers.FeeSchedule.List)
					r.With(customMiddleware.RequireRole(string(domain.UserRoleAdmin))).Post("/", handlers.FeeSchedule.Create)
					r.With(customMiddleware.RequireRole(string(domain.UserRoleAdmin))).Put("/{id}", handlers.FeeSchedule.Update)
					r.With(customMiddleware.RequireRole(string(domain.UserRoleAdmin))).Delete("/{id}", handlers.FeeSchedule.Delete)
				})

				// Stichtagsmeldung
				r.Route("/stichtagsmeldung", func(r chi.Router) {
					r.Get("/stats", handlers.Stichtagsmeldung.GetStats)
					r.Get("/report", handlers.Stichtagsmeldung.GetReport)
					r.Get("/children", handlers.Stichtagsmeldung.GetU3Children)
				})

				// Import
				r.Route("/import", func(r chi.Router) {
					r.Post("/confirm", handlers.Import.Confirm)
					r.Get("/history", handlers.Import.History)
					r.Get("/transactions", handlers.Import.UnmatchedTransactions)
					r.Get("/transactions/matched", handlers.Import.MatchedTransactions)
					r.Get("/transactions/{id}/suggestions", handlers.Import.TransactionSuggestions)
					r.Get("/transactions/unmatched/child/{id}", handlers.Import.ChildUnmatchedSuggestions)
					r.Post("/transactions/{id}/dismiss", handlers.Import.DismissTransaction)
					r.Post("/transactions/{id}/hide", handlers.Import.HideTransaction)
					r.Post("/transactions/{id}/unmatch", handlers.Import.UnmatchTransaction)
					r.Post("/transactions/{id}/allocate", handlers.Import.AllocateTransaction)
					r.Post("/match", handlers.Import.ManualMatch)
					r.Post("/rescan", handlers.Import.Rescan)
					r.Get("/blacklist", handlers.Import.GetBlacklist)
					r.Delete("/blacklist/{iban}", handlers.Import.RemoveFromBlacklist)
					r.Get("/trusted", handlers.Import.GetTrustedIBANs)
					r.Get("/trusted/child/{id}", handlers.Import.ChildTrustedIBANs)
					r.Post("/trusted/{iban}/link", handlers.Import.LinkIBANToChild)
					r.Delete("/trusted/{iban}/link", handlers.Import.UnlinkIBANFromChild)
					r.Get("/warnings", handlers.Import.GetWarnings)
					r.Post("/warnings/{id}/dismiss", handlers.Import.DismissWarning)
					r.Post("/warnings/{id}/resolve-late-fee", handlers.Import.ResolveLateFee)
				})

			})

		})
	})

	return r
}

// Handlers holds all HTTP handlers.
type Handlers struct {
	Auth             *handler.AuthHandler
	User             *handler.UserHandler
	Invitation       *handler.AccountInvitationHandler
	Child            *handler.ChildHandler
	ChildImport      *handler.ChildImportHandler
	ChildNote        *handler.ChildNoteHandler
	Parent           *handler.ParentHandler
	Household        *handler.HouseholdHandler
	Member           *handler.MemberHandler
	Fee              *handler.FeeHandler
	Einstufung       *handler.EinstufungHandler
	FeeSchedule      *handler.FeeScheduleHandler
	ParentWork       *handler.ParentWorkHandler
	ParentAccount    *handler.ParentAccountHandler
	Import           *handler.ImportHandler
	BankingSync      *handler.BankingSyncHandler
	Stichtagsmeldung *handler.StichtagsmeldungHandler
	JWTService       *auth.JWTService
}
