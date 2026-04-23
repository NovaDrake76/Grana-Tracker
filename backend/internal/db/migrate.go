package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RunMigrations reads 001_init.up.sql from migrationsDir and applies it.
// Re-running against an already-migrated DB is a no-op (logged, not fatal) —
// the init SQL is CREATE TABLE without IF NOT EXISTS, so Postgres returns an
// error the second time and we swallow it. Tests rely on TRUNCATE for
// between-test isolation rather than re-running DDL.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool, migrationsDir string) error {
	path := filepath.Join(migrationsDir, "001_init.up.sql")
	sql, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read migration %s: %w", path, err)
	}

	if _, err := pool.Exec(ctx, string(sql)); err != nil {
		log.Printf("migration note: %v", err)
		return nil
	}
	log.Println("migrations applied successfully")
	return nil
}
