package api

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jmoiron/sqlx"

	"github.com/knirpsenstadt/kita-apps/backend-portal/internal/api/handler"
	portalMiddleware "github.com/knirpsenstadt/kita-apps/backend-portal/internal/api/middleware"
	"github.com/knirpsenstadt/kita-apps/backend-portal/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-portal/internal/auth"
	"github.com/knirpsenstadt/kita-apps/backend-portal/internal/config"
	"github.com/knirpsenstadt/kita-apps/backend-portal/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-portal/internal/service"
)

func NewRouter(cfg *config.Config, db *sqlx.DB) http.Handler {
	r := chi.NewRouter()

	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.Server.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	health := healthHandler(db)
	jwtService := auth.NewJWTService(cfg.JWT.Secret, cfg.JWT.AccessExpiry, cfg.JWT.RefreshExpiry, cfg.JWT.Issuer)
	userRepo := repository.NewUserRepository(db)
	refreshTokenRepo := repository.NewRefreshTokenRepository(db)
	authHandler := handler.NewAuthHandler(service.NewAuthService(userRepo, refreshTokenRepo, jwtService))

	r.Get("/health", health)

	r.Route("/api/portal/v1", func(r chi.Router) {
		r.Get("/health", health)

		r.Route("/auth", func(r chi.Router) {
			r.Post("/login", authHandler.Login)
			r.Post("/refresh", authHandler.Refresh)

			r.Group(func(r chi.Router) {
				r.Use(portalMiddleware.AuthMiddleware(jwtService))
				r.Get("/me", authHandler.Me)
				r.Post("/logout", authHandler.Logout)
			})
		})
	})

	return r
}

func healthHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			response.Error(w, http.StatusServiceUnavailable, "database unavailable")
			return
		}

		response.Success(w, map[string]string{
			"service": "portal",
			"status":  "ok",
		})
	}
}
