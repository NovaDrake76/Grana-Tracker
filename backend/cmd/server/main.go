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

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/NovaDrake76/grana-tracker/backend/internal/handlers"
	"github.com/NovaDrake76/grana-tracker/backend/internal/middleware"
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

	runMigrations(ctx, pool)

	authMiddleware := middleware.NewAuthMiddleware(jwtSecret)
	authHandler := handlers.NewAuthHandler(pool, jwtSecret)
	userHandler := handlers.NewUserHandler(pool)
	portfolioHandler := handlers.NewPortfolioHandler(pool)

	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{frontendURL},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	r.Route("/api", func(r chi.Router) {
		// public routes
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)
			r.Post("/refresh", authHandler.Refresh)
		})

		// protected routes
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
			})
		})
	})

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: r,
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

// reads the init SQL and executes it; treats "already applied" as a no-op.
func runMigrations(ctx context.Context, pool *pgxpool.Pool) {
	migrationSQL, err := os.ReadFile("db/migrations/001_init.up.sql")
	if err != nil {
		log.Printf("no migration file found, skipping: %v", err)
		return
	}

	_, err = pool.Exec(ctx, string(migrationSQL))
	if err != nil {
		// tables likely already exist
		log.Printf("migration note: %v", err)
		return
	}
	log.Println("migrations applied successfully")
}
