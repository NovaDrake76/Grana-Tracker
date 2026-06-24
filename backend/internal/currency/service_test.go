package currency

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// canned AwesomeAPI payload for USD->BRL — mirrors the real envelope shape
// (outer key is FROM+TO with no hyphen) so the parser exercises the same path
// it would in production.
const usdBrlPayload = `{"USDBRL":{"bid":"5.42","ask":"5.43","timestamp":"1700000000"}}`

// TestGetRate_HappyPath stands up an httptest server, points the Service at
// it, and asserts the bid is returned verbatim plus a recent fetchedAt.
func TestGetRate_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/USD-BRL"; got != want {
			t.Errorf("path = %q, want %q", got, want)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(usdBrlPayload))
	}))
	defer srv.Close()

	svc := NewService().WithBaseURL(srv.URL)
	rate, fetchedAt, err := svc.GetRate(context.Background(), "USD", "BRL")
	if err != nil {
		t.Fatalf("GetRate returned error: %v", err)
	}
	if rate != "5.42" {
		t.Errorf("rate = %q, want 5.42", rate)
	}
	if time.Since(fetchedAt) > time.Second {
		t.Errorf("fetchedAt = %v looks stale", fetchedAt)
	}
}

// TestGetRate_SameCurrency must never touch the network — identity conversion
// is a closed-form answer.
func TestGetRate_SameCurrency(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("GetRate must not call upstream for same-currency request")
	}))
	defer srv.Close()

	svc := NewService().WithBaseURL(srv.URL)
	rate, _, err := svc.GetRate(context.Background(), "BRL", "BRL")
	if err != nil {
		t.Fatalf("GetRate returned error: %v", err)
	}
	if rate != "1.00" {
		t.Errorf("rate = %q, want 1.00", rate)
	}
}

// TestGetRate_Cached asserts the second call inside the TTL window does not
// re-hit the server. The atomic counter is the contract here — a return value
// equality check could pass for the wrong reason (e.g. server returning same
// payload twice).
func TestGetRate_Cached(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(usdBrlPayload))
	}))
	defer srv.Close()

	svc := NewService().WithBaseURL(srv.URL)
	if _, _, err := svc.GetRate(context.Background(), "USD", "BRL"); err != nil {
		t.Fatalf("first GetRate: %v", err)
	}
	if _, _, err := svc.GetRate(context.Background(), "USD", "BRL"); err != nil {
		t.Fatalf("second GetRate: %v", err)
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("server hit %d times, want 1 (second call should use cache)", got)
	}
}

// TestGetRate_FallsBackToStaleOnError forces the upstream to fail after
// seeding the cache, then proves the stale value is served instead of a 5xx.
// This is the fallback contract handlers depend on so the dashboard never
// goes blank when AwesomeAPI hiccups.
func TestGetRate_FallsBackToStaleOnError(t *testing.T) {
	var fail atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fail.Load() {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(usdBrlPayload))
	}))
	defer srv.Close()

	svc := NewService().WithBaseURL(srv.URL)
	if _, _, err := svc.GetRate(context.Background(), "USD", "BRL"); err != nil {
		t.Fatalf("seed call: %v", err)
	}

	// Manually age the cache entry past TTL so the next call attempts a refetch.
	svc.mu.Lock()
	entry := svc.cache["USD-BRL"]
	entry.fetchedAt = time.Now().Add(-2 * time.Hour)
	svc.cache["USD-BRL"] = entry
	svc.mu.Unlock()

	fail.Store(true)
	rate, _, err := svc.GetRate(context.Background(), "USD", "BRL")
	if err != nil {
		t.Fatalf("expected fallback, got error: %v", err)
	}
	if rate != "5.42" {
		t.Errorf("stale rate = %q, want 5.42", rate)
	}
}

// TestConvert checks the decimal math: 100 USD at 5.42 = 542.00 BRL. We pre-
// seed the cache so the test stays hermetic without standing up a server.
func TestConvert(t *testing.T) {
	svc := NewService()
	svc.cache["USD-BRL"] = cachedRate{rate: "5.42", fetchedAt: time.Now()}

	got, err := svc.Convert(context.Background(), "100", "USD", "BRL")
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if got != "542.00" {
		t.Errorf("Convert = %q, want 542.00", got)
	}
}

// TestGetRate_UpstreamErrorNoCache surfaces the upstream error when no cached
// value exists — the handler then returns 502 so the UI can show "indisponível".
func TestGetRate_UpstreamErrorNoCache(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusBadGateway)
	}))
	defer srv.Close()

	svc := NewService().WithBaseURL(srv.URL)
	_, _, err := svc.GetRate(context.Background(), "USD", "BRL")
	if err == nil {
		t.Fatal("expected error when upstream fails and cache is empty")
	}
	if !strings.Contains(err.Error(), "awesomeapi") {
		t.Errorf("error %q should reference the upstream", err)
	}
}
