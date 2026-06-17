package pricing

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/NovaDrake76/grana-tracker/backend/db/sqlc"
	"github.com/NovaDrake76/grana-tracker/backend/internal/db"
)

// priceEquals compares two decimal price strings numerically (with a tiny
// epsilon) so test assertions are insensitive to trailing-zero formatting:
// Postgres NUMERIC(18,8) round-trips "65000.00" as "65000.00000000", which is
// still the correct value.
func priceEquals(got string, want float64) bool {
	gotF, err := strconv.ParseFloat(got, 64)
	if err != nil {
		return false
	}
	return math.Abs(gotF-want) < 1e-6
}

// stubSource always returns the same canned price; lets us assert that
// RefreshAll plumbed assets through to a source and persisted the result.
type stubSource struct {
	name     string
	calls    int
	override map[string]string // ticker -> price string
}

func (s *stubSource) Name() string { return s.name }

func (s *stubSource) Fetch(_ context.Context, assets []sqlc.Asset) (map[string]Price, error) {
	s.calls++
	out := make(map[string]Price, len(assets))
	now := time.Now().UTC()
	for _, a := range assets {
		price := "100.00"
		if v, ok := s.override[a.Ticker]; ok {
			price = v
		}
		out[a.Ticker] = Price{
			Ticker:    a.Ticker,
			AssetType: a.AssetType,
			Price:     price,
			Currency:  a.Currency,
			FetchedAt: now,
		}
	}
	return out, nil
}

// requirePool skips when TEST_DATABASE_URL is unset, then provisions an
// ephemeral per-test database so the pricing suite cannot race the handler
// suite (which targets the shared grana_test DB and would observe our
// TRUNCATE calls otherwise).
func requirePool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	adminDSN := os.Getenv("TEST_DATABASE_URL")
	if adminDSN == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	adminPool, err := pgxpool.New(ctx, adminDSN)
	if err != nil {
		t.Fatalf("connect admin DB: %v", err)
	}
	defer adminPool.Close()

	dbName := fmt.Sprintf("grana_pricing_test_%d", time.Now().UnixNano())
	if _, err := adminPool.Exec(ctx, fmt.Sprintf(`DROP DATABASE IF EXISTS %s`, dbName)); err != nil {
		t.Fatalf("drop ephemeral DB: %v", err)
	}
	if _, err := adminPool.Exec(ctx, fmt.Sprintf(`CREATE DATABASE %s`, dbName)); err != nil {
		t.Fatalf("create ephemeral DB: %v", err)
	}

	ephemeralDSN, err := swapDatabaseInDSN(adminDSN, dbName)
	if err != nil {
		t.Fatalf("rewrite DSN: %v", err)
	}

	pool, err := pgxpool.New(ctx, ephemeralDSN)
	if err != nil {
		t.Fatalf("connect ephemeral DB: %v", err)
	}

	schema := os.Getenv("TEST_SCHEMA_DIR")
	if schema == "" {
		schema = "../../db/schema"
	}
	if err := db.RunMigrations(ctx, pool, schema); err != nil {
		t.Fatalf("run migrations: %v", err)
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

// swapDatabaseInDSN replaces the path (database name) component of a
// postgres://... URL while keeping credentials, host, port, and options.
func swapDatabaseInDSN(dsn, dbName string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}
	u.Path = "/" + dbName
	return u.String(), nil
}

func TestRefreshAll_PopulatesCacheAndHistory(t *testing.T) {
	pool := requirePool(t)
	ctx := context.Background()

	// isolate state
	if _, err := pool.Exec(ctx, `TRUNCATE price_cache, price_history RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	queries := sqlc.New(pool)
	crypto := &stubSource{name: "stub-crypto", override: map[string]string{"BTC": "65000.00"}}
	stocks := &stubSource{name: "stub-stocks", override: map[string]string{"AAPL": "189.50"}}

	svc := NewService(queries, crypto, stocks)

	summary, err := svc.RefreshAll(ctx)
	if err != nil {
		t.Fatalf("RefreshAll: %v", err)
	}
	if summary.Refreshed == 0 {
		t.Fatalf("expected at least one refreshed asset, got 0 (errors=%v)", summary.Errors)
	}

	// spot-check two well-known seed entries
	btc, err := svc.GetCurrent(ctx, "BTC", "crypto")
	if err != nil {
		t.Fatalf("GetCurrent BTC: %v", err)
	}
	if !priceEquals(btc.Price, 65000.00) {
		t.Errorf("BTC price = %q, want 65000.00", btc.Price)
	}
	if btc.Stale {
		t.Errorf("BTC should not be stale right after refresh")
	}

	aapl, err := svc.GetCurrent(ctx, "AAPL", "stock")
	if err != nil {
		t.Fatalf("GetCurrent AAPL: %v", err)
	}
	if !priceEquals(aapl.Price, 189.50) {
		t.Errorf("AAPL price = %q, want 189.50", aapl.Price)
	}

	// price_history should have a row for today per ticker
	var n int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM price_history WHERE recorded_at = CURRENT_DATE`).Scan(&n); err != nil {
		t.Fatalf("count history: %v", err)
	}
	if n == 0 {
		t.Errorf("expected price_history rows for today, got 0")
	}
}

// TestRefreshAll_SkippedSourceLeavesCacheIntact mirrors the "Alpha Vantage key
// not set" production scenario: the stub returns ErrSkipped, and the previous
// cached price is preserved.
func TestRefreshAll_SkippedSourceLeavesCacheIntact(t *testing.T) {
	pool := requirePool(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `TRUNCATE price_cache, price_history RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	queries := sqlc.New(pool)
	crypto := &stubSource{name: "stub-crypto", override: map[string]string{"BTC": "65000.00"}}
	stocks := &skippingSource{}

	svc := NewService(queries, crypto, stocks)
	summary, err := svc.RefreshAll(ctx)
	if err != nil {
		t.Fatalf("RefreshAll: %v", err)
	}
	if summary.Refreshed == 0 {
		t.Fatalf("crypto branch should have written some rows")
	}

	// AAPL was never quoted, so GetCurrent must return ErrNotFound.
	if _, err := svc.GetCurrent(ctx, "AAPL", "stock"); err == nil {
		t.Errorf("expected ErrNotFound for AAPL, got nil")
	}
}

type skippingSource struct{}

func (skippingSource) Name() string { return "skip" }
func (skippingSource) Fetch(_ context.Context, _ []sqlc.Asset) (map[string]Price, error) {
	return nil, ErrSkipped
}
