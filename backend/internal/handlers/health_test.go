package handlers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/NovaDrake76/grana-tracker/backend/internal/server"
)

func TestHealthzAlwaysOK(t *testing.T) {
	// /healthz should not depend on DB — skip the requireDB gate and
	// construct a router against whatever pool we have (nil is fine,
	// because /healthz never touches it).
	r := server.NewRouter(testPool, testJWTSecret, "http://localhost:3000")

	rr, _ := doRequest(t, r, http.MethodGet, "/healthz", "", nil)
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
}

func TestReadyzWhenDBHealthy(t *testing.T) {
	requireDB(t)
	r := newTestRouter(t)

	rr, _ := doRequest(t, r, http.MethodGet, "/readyz", "", nil)
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
}

func TestReadyzWhenDBUnreachable(t *testing.T) {
	// Point at a DSN that cannot connect; /readyz must return 503.
	pool, err := pgxpool.NewWithConfig(context.Background(), mustParseConfig(t, "postgres://nobody:nobody@127.0.0.1:1/none?sslmode=disable&connect_timeout=1"))
	if err != nil {
		t.Fatalf("NewWithConfig: %v", err)
	}
	defer pool.Close()

	r := server.NewRouter(pool, testJWTSecret, "http://localhost:3000")

	rr, _ := doRequest(t, r, http.MethodGet, "/readyz", "", nil)
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rr.Code)
	}
}

func mustParseConfig(t *testing.T, dsn string) *pgxpool.Config {
	t.Helper()
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	return cfg
}
