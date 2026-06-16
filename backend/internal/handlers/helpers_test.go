package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestWriteJSON(t *testing.T) {
	rr := httptest.NewRecorder()
	writeJSON(rr, 201, map[string]string{"hello": "world"})

	if rr.Code != 201 {
		t.Errorf("status = %d, want 201", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if body["hello"] != "world" {
		t.Errorf("body[hello] = %q, want world", body["hello"])
	}
}

func TestWriteError(t *testing.T) {
	rr := httptest.NewRecorder()
	writeError(rr, 400, "bad request", "VALIDATION_ERROR")

	if rr.Code != 400 {
		t.Errorf("status = %d, want 400", rr.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if body["error"] != "bad request" {
		t.Errorf("body[error] = %q, want 'bad request'", body["error"])
	}
	if body["code"] != "VALIDATION_ERROR" {
		t.Errorf("body[code] = %q, want VALIDATION_ERROR", body["code"])
	}
}

func TestIsDuplicateKeyError(t *testing.T) {
	uniqueViolation := &pgconn.PgError{Code: "23505", Message: "duplicate key value violates unique constraint"}
	otherPg := &pgconn.PgError{Code: "23503", Message: "foreign key violation"}

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"real pg 23505", uniqueViolation, true},
		{"real pg 23505 wrapped", fmt.Errorf("create user: %w", uniqueViolation), true},
		{"other pg code", otherPg, false},
		{"generic error", errors.New("timeout connecting to DB"), false},
		{"nil error", nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isDuplicateKeyError(c.err); got != c.want {
				t.Errorf("isDuplicateKeyError(%v) = %v, want %v", c.err, got, c.want)
			}
		})
	}
}

func TestParseUUID(t *testing.T) {
	if _, err := parseUUID("not-a-uuid"); err == nil {
		t.Error("parseUUID accepted garbage")
	}
	if _, err := parseUUID("550e8400-e29b-41d4-a716-446655440000"); err != nil {
		t.Errorf("parseUUID rejected valid uuid: %v", err)
	}
}
