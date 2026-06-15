package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// initMigrationName is the filename of the original schema migration. When the
// schema_migrations table is first introduced on a database that already has
// the core tables (because 001 was applied by the legacy runner), we record
// this name without re-executing the SQL.
const initMigrationName = "001_init.up.sql"

// RunMigrations applies every db/schema/*.up.sql file in lexical order that
// has not yet been recorded in the schema_migrations bookkeeping table. Each
// migration runs inside its own transaction; on any error the transaction is
// rolled back and a wrapped error is returned.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool, schemaDir string) error {
	if err := ensureMigrationsTable(ctx, pool); err != nil {
		return fmt.Errorf("ensure schema_migrations table: %w", err)
	}

	if err := markPreExistingSchema(ctx, pool); err != nil {
		return fmt.Errorf("mark pre-existing schema: %w", err)
	}

	files, err := listMigrationFiles(schemaDir)
	if err != nil {
		return fmt.Errorf("list migrations in %s: %w", schemaDir, err)
	}

	for _, file := range files {
		name := filepath.Base(file)

		applied, err := migrationApplied(ctx, pool, name)
		if err != nil {
			return fmt.Errorf("check migration %s: %w", name, err)
		}
		if applied {
			log.Printf("skipping %s (already applied)", name)
			continue
		}

		sqlBytes, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", file, err)
		}

		log.Printf("applying %s", name)
		if err := applyMigration(ctx, pool, name, string(sqlBytes)); err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
	}

	log.Println("migrations applied successfully")
	return nil
}

// ensureMigrationsTable creates the bookkeeping table if it does not exist.
func ensureMigrationsTable(ctx context.Context, pool *pgxpool.Pool) error {
	const stmt = `CREATE TABLE IF NOT EXISTS schema_migrations (
        name TEXT PRIMARY KEY,
        applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    )`
	_, err := pool.Exec(ctx, stmt)
	return err
}

// markPreExistingSchema records 001_init.up.sql as applied when the bookkeeping
// table is empty but the core tables already exist (i.e. an older runner
// applied 001 before schema_migrations was introduced). This avoids the
// CREATE TABLE-without-IF-NOT-EXISTS statements in 001_init from blowing up
// on existing databases.
func markPreExistingSchema(ctx context.Context, pool *pgxpool.Pool) error {
	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&count); err != nil {
		return fmt.Errorf("count schema_migrations: %w", err)
	}
	if count > 0 {
		return nil
	}

	var usersExists bool
	const existsQuery = `SELECT EXISTS (
        SELECT 1 FROM information_schema.tables
        WHERE table_schema = 'public' AND table_name = 'users'
    )`
	if err := pool.QueryRow(ctx, existsQuery).Scan(&usersExists); err != nil {
		return fmt.Errorf("check users table existence: %w", err)
	}
	if !usersExists {
		return nil
	}

	if _, err := pool.Exec(ctx,
		`INSERT INTO schema_migrations (name) VALUES ($1) ON CONFLICT DO NOTHING`,
		initMigrationName,
	); err != nil {
		return fmt.Errorf("insert pre-existing migration marker: %w", err)
	}
	log.Printf("marked pre-existing schema as applied (%s)", initMigrationName)
	return nil
}

// listMigrationFiles returns every *.up.sql file in schemaDir sorted by name.
func listMigrationFiles(schemaDir string) ([]string, error) {
	entries, err := os.ReadDir(schemaDir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".up.sql") {
			continue
		}
		files = append(files, filepath.Join(schemaDir, name))
	}
	sort.Strings(files)
	return files, nil
}

// migrationApplied reports whether a row already exists for name.
func migrationApplied(ctx context.Context, pool *pgxpool.Pool, name string) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE name = $1)`,
		name,
	).Scan(&exists)
	return exists, err
}

// applyMigration runs sql and records the migration inside a single
// transaction, rolling back on any error.
func applyMigration(ctx context.Context, pool *pgxpool.Pool, name, sql string) error {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, sql); err != nil {
		return fmt.Errorf("exec sql: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO schema_migrations (name) VALUES ($1)`,
		name,
	); err != nil {
		return fmt.Errorf("record migration: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}
