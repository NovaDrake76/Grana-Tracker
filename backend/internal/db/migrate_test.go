package db

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// requireDB skips the calling test when TEST_DATABASE_URL is not configured.
// Mirrors the helper used by the handlers package so `go test ./...` still
// passes on a dev machine without Postgres running.
func requireDB(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping migrate integration test")
	}
	return dsn
}

// newEphemeralDB connects to the admin DB defined by TEST_DATABASE_URL, drops
// + creates a uniquely-named database for THIS test, returns a pool against
// the fresh DB, and registers cleanup that closes the pool and drops the DB.
//
// Using a dedicated database per test means the regular grana_test schema is
// never touched, and the runner is exercised against a guaranteed-clean state
// — which is exactly what RunMigrations is designed for.
func newEphemeralDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	adminDSN := requireDB(t)

	// Silence the migration runner's log.Printf calls; otherwise tests are noisy.
	log.SetOutput(io.Discard)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	adminPool, err := pgxpool.New(ctx, adminDSN)
	if err != nil {
		t.Fatalf("connect admin DB: %v", err)
	}
	defer adminPool.Close()

	// Build a unique DB name. Postgres identifiers can't start with a digit
	// and must be <= 63 chars; nanosecond timestamp is plenty unique here.
	dbName := fmt.Sprintf("grana_mig_test_%d", time.Now().UnixNano())

	// Terminate any stale connections (defence in depth) and recreate the DB.
	if _, err := adminPool.Exec(ctx, fmt.Sprintf(`DROP DATABASE IF EXISTS %s`, dbName)); err != nil {
		t.Fatalf("drop ephemeral DB: %v", err)
	}
	if _, err := adminPool.Exec(ctx, fmt.Sprintf(`CREATE DATABASE %s`, dbName)); err != nil {
		t.Fatalf("create ephemeral DB: %v", err)
	}

	// Rewrite the DSN to point at the new database while preserving the
	// credentials/host/options from TEST_DATABASE_URL.
	ephemeralDSN, err := swapDatabaseInDSN(adminDSN, dbName)
	if err != nil {
		t.Fatalf("rewrite DSN: %v", err)
	}

	pool, err := pgxpool.New(ctx, ephemeralDSN)
	if err != nil {
		t.Fatalf("connect ephemeral DB: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()

		dropCtx, dropCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer dropCancel()

		cleanupPool, err := pgxpool.New(dropCtx, adminDSN)
		if err != nil {
			return
		}
		defer cleanupPool.Close()
		_, _ = cleanupPool.Exec(dropCtx, fmt.Sprintf(`DROP DATABASE IF EXISTS %s`, dbName))
	})

	return pool
}

// swapDatabaseInDSN replaces the path component (i.e. the database name) in a
// postgres URL-style DSN.
func swapDatabaseInDSN(dsn, dbName string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}
	u.Path = "/" + dbName
	return u.String(), nil
}

// writeSchemaDir creates a temp directory and writes each file in files there,
// returning the directory path. Decouples the runner test from the real
// product schema so changes to product migrations cannot break these tests.
func writeSchemaDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return dir
}

func TestRunMigrations(t *testing.T) {
	t.Run("fresh_db_applies_all_in_lexical_order", func(t *testing.T) {
		pool := newEphemeralDB(t)
		ctx := context.Background()

		dir := writeSchemaDir(t, map[string]string{
			"001_a.up.sql": `CREATE TABLE t1 (id INT PRIMARY KEY);`,
			"002_b.up.sql": `CREATE TABLE t2 (id INT PRIMARY KEY);`,
			// non-.up.sql files MUST be ignored by listMigrationFiles
			"README.md":     `not a migration`,
			"003_a.down.sql": `DROP TABLE t1;`,
		})

		if err := RunMigrations(ctx, pool, dir); err != nil {
			t.Fatalf("RunMigrations: %v", err)
		}

		// Both migrations recorded, in lexical order.
		rows, err := pool.Query(ctx, `SELECT name FROM schema_migrations ORDER BY applied_at ASC`)
		if err != nil {
			t.Fatalf("query schema_migrations: %v", err)
		}
		var got []string
		for rows.Next() {
			var n string
			if err := rows.Scan(&n); err != nil {
				t.Fatalf("scan: %v", err)
			}
			got = append(got, n)
		}
		rows.Close()

		want := []string{"001_a.up.sql", "002_b.up.sql"}
		if len(got) != len(want) {
			t.Fatalf("rows: got %v, want %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("order mismatch at %d: got %q want %q (full got=%v)", i, got[i], want[i], got)
			}
		}

		// Both tables exist.
		for _, table := range []string{"t1", "t2"} {
			var exists bool
			if err := pool.QueryRow(ctx,
				`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name=$1)`,
				table,
			).Scan(&exists); err != nil {
				t.Fatalf("check %s exists: %v", table, err)
			}
			if !exists {
				t.Fatalf("expected table %s to exist", table)
			}
		}
	})

	t.Run("idempotent_rerun_skips_applied", func(t *testing.T) {
		pool := newEphemeralDB(t)
		ctx := context.Background()

		dir := writeSchemaDir(t, map[string]string{
			"001_a.up.sql": `CREATE TABLE t1 (id INT PRIMARY KEY);`,
			"002_b.up.sql": `CREATE TABLE t2 (id INT PRIMARY KEY);`,
		})

		if err := RunMigrations(ctx, pool, dir); err != nil {
			t.Fatalf("first run: %v", err)
		}
		if err := RunMigrations(ctx, pool, dir); err != nil {
			t.Fatalf("second run: %v", err)
		}

		var count int
		if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&count); err != nil {
			t.Fatalf("count schema_migrations: %v", err)
		}
		if count != 2 {
			t.Fatalf("expected 2 rows after idempotent rerun, got %d", count)
		}
	})

	t.Run("errors_propagate", func(t *testing.T) {
		pool := newEphemeralDB(t)
		ctx := context.Background()

		dir := writeSchemaDir(t, map[string]string{
			"001_a.up.sql":      `CREATE TABLE t1 (id INT PRIMARY KEY);`,
			"002_b.up.sql":      `CREATE TABLE t2 (id INT PRIMARY KEY);`,
			"003_broken.up.sql": `SELECT FROM;`, // intentionally invalid SQL
		})

		err := RunMigrations(ctx, pool, dir)
		if err == nil {
			t.Fatal("expected RunMigrations to return error for broken migration")
		}

		// Broken migration MUST NOT be marked applied; the transaction rolls back.
		var brokenApplied bool
		if err := pool.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE name=$1)`,
			"003_broken.up.sql",
		).Scan(&brokenApplied); err != nil {
			t.Fatalf("check 003 applied: %v", err)
		}
		if brokenApplied {
			t.Fatal("003_broken.up.sql should NOT be marked applied after rollback")
		}

		// Sanity: the two earlier valid migrations ARE applied (lexical order).
		var firstTwo int
		if err := pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM schema_migrations WHERE name IN ($1,$2)`,
			"001_a.up.sql", "002_b.up.sql",
		).Scan(&firstTwo); err != nil {
			t.Fatalf("count first two: %v", err)
		}
		if firstTwo != 2 {
			t.Fatalf("expected the two valid migrations to remain applied, got %d", firstTwo)
		}
	})

	t.Run("legacy_db_marked_pre_existing", func(t *testing.T) {
		pool := newEphemeralDB(t)
		ctx := context.Background()

		// Simulate a legacy database where 001_init was applied by the OLD
		// runner before schema_migrations existed: public.users is already
		// there with real data, but schema_migrations is absent.
		if _, err := pool.Exec(ctx,
			`CREATE TABLE users (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), tag TEXT)`,
		); err != nil {
			t.Fatalf("create legacy users: %v", err)
		}
		// Sentinel row — if RunMigrations re-ran 001_init this row would be gone.
		if _, err := pool.Exec(ctx,
			`INSERT INTO users (tag) VALUES ('sentinel')`,
		); err != nil {
			t.Fatalf("insert sentinel: %v", err)
		}

		// Stub a 001_init.up.sql that WOULD blow up if it ran against the
		// legacy schema (no IF NOT EXISTS, conflicting users definition).
		dir := writeSchemaDir(t, map[string]string{
			"001_init.up.sql": `CREATE TABLE users (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), email TEXT);`,
		})

		if err := RunMigrations(ctx, pool, dir); err != nil {
			t.Fatalf("RunMigrations on legacy DB: %v", err)
		}

		// markPreExistingSchema must have recorded 001_init without running it.
		var marked bool
		if err := pool.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE name=$1)`,
			"001_init.up.sql",
		).Scan(&marked); err != nil {
			t.Fatalf("check 001 marker: %v", err)
		}
		if !marked {
			t.Fatal("expected 001_init.up.sql to be marked as applied via markPreExistingSchema")
		}

		// Sentinel still present => 001_init SQL was NOT re-executed.
		var tag string
		err := pool.QueryRow(ctx, `SELECT tag FROM users WHERE tag='sentinel'`).Scan(&tag)
		if err != nil {
			t.Fatalf("legacy data wiped! 001_init was re-executed: %v", err)
		}
		if tag != "sentinel" {
			t.Fatalf("sentinel mismatch: %q", tag)
		}

		// And the legacy schema (tag column) is intact — proof that the stubbed
		// 001 with (id, email) shape was NOT applied on top of it.
		var hasTag bool
		if err := pool.QueryRow(ctx,
			`SELECT EXISTS (
                SELECT 1 FROM information_schema.columns
                WHERE table_schema='public' AND table_name='users' AND column_name='tag'
            )`,
		).Scan(&hasTag); err != nil {
			t.Fatalf("check tag column: %v", err)
		}
		if !hasTag {
			t.Fatal("legacy users.tag column missing — migration ran on top of pre-existing schema")
		}
	})
}

// guard against an unused-import warning if strings ends up unused in some
// future edit; cheap helper kept for symmetry with handlers/setup_test.go.
var _ = strings.HasSuffix
