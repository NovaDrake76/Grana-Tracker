package server

import (
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/NovaDrake76/grana-tracker/backend/db/sqlc"
	"github.com/NovaDrake76/grana-tracker/backend/internal/handlers"
	"github.com/NovaDrake76/grana-tracker/backend/internal/middleware"
)

// NewRouter wires every route and middleware in one place so main.go and
// integration tests build the exact same HTTP surface.
func NewRouter(pool *pgxpool.Pool, jwtSecret, frontendURL string) chi.Router {
	queries := sqlc.New(pool)

	authMiddleware := middleware.NewAuthMiddleware(jwtSecret)
	authHandler := handlers.NewAuthHandler(queries, pool, jwtSecret)
	userHandler := handlers.NewUserHandler(queries)
	portfolioHandler := handlers.NewPortfolioHandler(queries)
	investmentHandler := handlers.NewInvestmentHandler(queries)
	healthHandler := handlers.NewHealthHandler(pool)

	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(middleware.SecurityHeaders)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{frontendURL},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	// health probes sit outside /api and outside auth so orchestrators can reach them.
	r.Get("/healthz", healthHandler.Live)
	r.Get("/readyz", healthHandler.Ready)

	r.Route("/api", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			// RealIP rewrites RemoteAddr from X-Real-IP / X-Forwarded-For so the
			// rate limiter buckets clients by their true source IP and not by
			// the reverse proxy's IP. Assumes the reverse proxy strips inbound
			// X-Forwarded-For from clients and sets its own — without that
			// hardening at the proxy, clients could spoof the header and dodge
			// the per-IP cap.
			r.Use(chimw.RealIP)
			// OWASP A07 (Identification & Authentication Failures): cap each
			// IP to 10 auth requests per minute to slow brute-force / credential
			// stuffing attacks against /register, /login, and /refresh.
			r.Use(httprate.LimitByIP(10, time.Minute))
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)
			r.Post("/refresh", authHandler.Refresh)
		})

		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)

			r.Route("/user", func(r chi.Router) {
				r.Get("/me", userHandler.GetMe)
				r.Put("/me", userHandler.UpdateMe)
			})

			r.Route("/portfolios", func(r chi.Router) {
				r.Get("/", portfolioHandler.List)
				r.Post("/", portfolioHandler.Create)
				r.Get("/{id}", portfolioHandler.Get)
				r.Put("/{id}", portfolioHandler.Update)
				r.Delete("/{id}", portfolioHandler.Delete)
				r.Post("/{id}/investments", investmentHandler.Create)
			})

			r.Route("/investments", func(r chi.Router) {
				r.Get("/{id}", investmentHandler.Get)
				r.Put("/{id}", investmentHandler.Update)
				r.Delete("/{id}", investmentHandler.Delete)
			})
		})
	})

	return r
}
