package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/NovaDrake76/grana-tracker/backend/internal/db"
	"github.com/NovaDrake76/grana-tracker/backend/internal/pricing"
	"github.com/NovaDrake76/grana-tracker/backend/internal/server"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("godotenv: %v (continuing — .env is optional)", err)
	}

	dbURL := os.Getenv("DATABASE_URL")
	jwtSecret := os.Getenv("JWT_SECRET")
	port := os.Getenv("PORT")
	frontendURL := os.Getenv("FRONTEND_URL")

	if dbURL == "" || jwtSecret == "" {
		log.Fatal("DATABASE_URL and JWT_SECRET are required")
	}
	if len(jwtSecret) < 32 {
		log.Fatal("JWT_SECRET must be at least 32 bytes (use a high-entropy random string)")
	}
	if port == "" {
		port = "8080"
	}
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
		log.Printf("WARN: FRONTEND_URL not set, defaulting to http://localhost:3000 with AllowCredentials=true; do NOT use this default in production")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}
	log.Println("connected to database")

	if err := db.RunMigrations(ctx, pool, "db/schema"); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	handler, pricingSvc := server.NewRouter(pool, jwtSecret, frontendURL)

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%s", port),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Best-effort warm-up + daily refresh of price_cache / price_history.
	// We run this in a goroutine so a slow upstream cannot delay server boot,
	// and we log (rather than fatal) on errors — stale prices are degraded but
	// the rest of the app should keep serving.
	go runPricingRefresh(pricingSvc)

	go func() {
		log.Printf("server starting on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}
	log.Println("server stopped")
}

// runPricingRefresh kicks off one immediate refresh on boot and then a ticker
// every 24h. Errors are logged, never fatal.
func runPricingRefresh(svc *pricing.Service) {
	summary, err := svc.RefreshAll(context.Background())
	log.Printf("pricing initial refresh: %d updated, errors=%d, err=%v", summary.Refreshed, len(summary.Errors), err)

	t := time.NewTicker(24 * time.Hour)
	defer t.Stop()
	for range t.C {
		s, err := svc.RefreshAll(context.Background())
		log.Printf("pricing daily refresh: %d updated, errors=%d, err=%v", s.Refreshed, len(s.Errors), err)
	}
}
