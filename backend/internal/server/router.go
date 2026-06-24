package server

import (
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/NovaDrake76/grana-tracker/backend/db/sqlc"
	"github.com/NovaDrake76/grana-tracker/backend/internal/currency"
	gqlapi "github.com/NovaDrake76/grana-tracker/backend/internal/graphql"
	"github.com/NovaDrake76/grana-tracker/backend/internal/handlers"
	"github.com/NovaDrake76/grana-tracker/backend/internal/middleware"
	"github.com/NovaDrake76/grana-tracker/backend/internal/pricing"
	"github.com/NovaDrake76/grana-tracker/backend/internal/snapshots"
)

// parseAllowedOrigins splits FRONTEND_URL on commas and trims whitespace so the
// same env var can carry both the Vercel production URL and the local dev
// origin (e.g. "https://grana-tracker.vercel.app,http://localhost:3000").
// Empty entries are dropped so trailing commas don't widen CORS to "".
func parseAllowedOrigins(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// NewRouter wires every route and middleware in one place so main.go and
// integration tests build the exact same HTTP surface. It also returns the
// configured pricing.Service and snapshots.Service so main.go can hold a
// reference for the daily refresh / snapshot goroutine without recreating the
// source adapters or the FX rate.
func NewRouter(pool *pgxpool.Pool, jwtSecret, frontendURL string) (chi.Router, *pricing.Service, *snapshots.Service) {
	queries := sqlc.New(pool)

	// Build pricing service with real upstream adapters. Env vars override
	// defaults so tests can point at httptest servers without recompiling.
	coingeckoBase := os.Getenv("COINGECKO_BASE_URL")
	alphaKey := os.Getenv("ALPHA_VANTAGE_API_KEY")
	alphaBase := os.Getenv("ALPHA_VANTAGE_BASE_URL")
	pricingSvc := pricing.NewService(
		queries,
		pricing.NewCoinGeckoSource(coingeckoBase),
		pricing.NewAlphaVantageSource(alphaBase, alphaKey),
	)

	// Snapshot service runs alongside the pricing cron so the US06 history
	// chart can read a daily total per portfolio. USD positions are converted
	// to BRL with a placeholder rate until US09 ships a live FX feed.
	snapshotsSvc := snapshots.NewService(queries, "5.00")

	// Currency service backs US09 (Moeda Preferida). It is HTTP-only with an
	// in-memory cache, so we instantiate it once and share it across requests.
	currencySvc := currency.NewService()

	authMiddleware := middleware.NewAuthMiddleware(jwtSecret)
	authHandler := handlers.NewAuthHandler(queries, pool, jwtSecret)
	userHandler := handlers.NewUserHandler(queries)
	portfolioHandler := handlers.NewPortfolioHandlerWithSnapshots(queries, snapshotsSvc)
	investmentHandler := handlers.NewInvestmentHandler(queries)
	healthHandler := handlers.NewHealthHandler(pool)
	assetHandler := handlers.NewAssetHandler(queries, pricingSvc)
	currencyHandler := handlers.NewCurrencyHandler(currencySvc)

	// Build the GraphQL schema once at boot — it's stateless after construction,
	// so every request just runs graphql.Do against the cached schema. A schema
	// build error is fatal because the rest of the surface is useless without it.
	gqlSchema, err := gqlapi.NewSchema(queries)
	if err != nil {
		panic("failed to build graphql schema: " + err.Error())
	}
	graphqlHandler := gqlapi.NewHandler(gqlSchema)

	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(middleware.SecurityHeaders)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   parseAllowedOrigins(frontendURL),
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

		// Public asset / price endpoints — autocomplete and quote display do
		// not require a session.
		r.Route("/assets", func(r chi.Router) {
			r.Get("/search", assetHandler.Search)
		})
		// Historical lookup is routed BEFORE /prices/{ticker} so chi does not
		// match "historical" as a ticker value.
		r.Get("/prices/historical", assetHandler.GetHistoricalPrice)
		r.Get("/prices/{ticker}", assetHandler.GetPrice)

		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)

			// Logout lives behind Authenticate (NOT in the public /auth rate-
			// limited group above) so the caller must prove they hold a valid
			// access token before they can revoke a refresh token.
			r.Post("/auth/logout", authHandler.Logout)

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
				r.Get("/{id}/history", portfolioHandler.GetHistory)
				r.Post("/{id}/investments", investmentHandler.Create)
			})

			r.Route("/investments", func(r chi.Router) {
				r.Get("/{id}", investmentHandler.Get)
				r.Put("/{id}", investmentHandler.Update)
				r.Delete("/{id}", investmentHandler.Delete)
			})

			// Mutating the cache costs upstream API quota — keep it behind auth.
			r.Post("/prices/refresh", assetHandler.Refresh)

			// Currency conversion sits behind auth so abusers can't drain the
			// free AwesomeAPI tier on our IP; rates are user-facing anyway.
			r.Get("/currency/rate", currencyHandler.GetRate)

			// Single GraphQL endpoint — same Authenticate middleware as REST so
			// resolvers can trust middleware.GetUserID(ctx) is set.
			r.Post("/graphql", graphqlHandler.ServeHTTP)
		})
	})

	return r, pricingSvc, snapshotsSvc
}
