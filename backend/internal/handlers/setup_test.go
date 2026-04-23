package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/NovaDrake76/grana-tracker/backend/internal/db"
	"github.com/NovaDrake76/grana-tracker/backend/internal/server"
)

const testJWTSecret = "test-secret-do-not-use-in-prod"

var testPool *pgxpool.Pool

// TestMain brings up a shared pgx pool for the whole package when
// TEST_DATABASE_URL is set. Without it we leave testPool nil and each
// integration test skips via requireDB — that way `go test ./...` passes
// locally on a dev machine without Postgres running.
func TestMain(m *testing.M) {
	log.SetOutput(io.Discard) // silence handler noise during tests

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		os.Exit(m.Run())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.SetOutput(os.Stderr)
		log.Fatalf("connect to test DB: %v", err)
	}

	migrationsDir := os.Getenv("TEST_MIGRATIONS_DIR")
	if migrationsDir == "" {
		migrationsDir = "../../db/migrations"
	}
	if err := db.RunMigrations(ctx, pool, migrationsDir); err != nil {
		log.SetOutput(os.Stderr)
		log.Fatalf("run migrations: %v", err)
	}

	testPool = pool
	code := m.Run()
	pool.Close()
	os.Exit(code)
}

// requireDB skips the calling test when TEST_DATABASE_URL is not configured.
func requireDB(t *testing.T) {
	t.Helper()
	if testPool == nil {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}
}

// truncateAll wipes every table; call at the start of a test to isolate state.
func truncateAll(t *testing.T) {
	t.Helper()
	_, err := testPool.Exec(context.Background(),
		"TRUNCATE users, portfolios, investments, price_cache, price_history RESTART IDENTITY CASCADE")
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

// newTestRouter builds the exact same router main.go does, against the test pool.
func newTestRouter(t *testing.T) chi.Router {
	t.Helper()
	return server.NewRouter(testPool, testJWTSecret, "http://localhost:3000")
}

type apiResponse struct {
	Data    json.RawMessage `json:"data"`
	Message string          `json:"message"`
	Error   string          `json:"error"`
	Code    string          `json:"code"`
}

// doRequest fires an HTTP request through a chi router and returns the response body.
func doRequest(t *testing.T, r chi.Router, method, path, token string, body interface{}) (*httptest.ResponseRecorder, apiResponse) {
	t.Helper()

	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req := httptest.NewRequest(method, path, reqBody)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	var out apiResponse
	if rr.Body.Len() > 0 {
		// some error paths go through http.Error which wraps the JSON in a trailing newline; ignore decode errors on empty.
		_ = json.Unmarshal(rr.Body.Bytes(), &out)
	}
	return rr, out
}

// registerUser hits POST /auth/register and returns the access token.
// It fails the test if the register call does not return 201.
func registerUser(t *testing.T, r chi.Router, email, password string) string {
	t.Helper()
	rr, resp := doRequest(t, r, http.MethodPost, "/api/auth/register", "", map[string]string{
		"name":     "Test " + strings.Split(email, "@")[0],
		"email":    email,
		"password": password,
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("register %s: status %d, body %s", email, rr.Code, rr.Body.String())
	}
	var tokens struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(resp.Data, &tokens); err != nil {
		t.Fatalf("decode tokens: %v", err)
	}
	if tokens.AccessToken == "" {
		t.Fatal("empty access token")
	}
	return tokens.AccessToken
}
