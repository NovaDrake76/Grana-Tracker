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
	godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	jwtSecret := os.Getenv("JWT_SECRET")
	port := os.Getenv("PORT")
	frontendURL := os.Getenv("FRONTEND_URL")

	if dbURL == "" || jwtSecret == "" {
		log.Fatal("DATABASE_URL and JWT_SECRET are required")
	}
	if port == "" {
		port = "8080"
	}
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
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

	if err := db.RunMigrations(ctx, pool, "db/migrations"); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: server.NewRouter(pool, jwtSecret, frontendURL),
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
