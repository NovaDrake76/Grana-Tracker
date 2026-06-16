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

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%s", port),
		Handler:           server.NewRouter(pool, jwtSecret, frontendURL),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

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
