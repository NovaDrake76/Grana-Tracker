package pricing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/NovaDrake76/grana-tracker/backend/db/sqlc"
)

// canned CoinGecko payload for the two assets the test injects.
const cgPayload = `{"bitcoin":{"usd":67234.12},"ethereum":{"usd":3210.4}}`

// TestCoinGeckoFetch_HappyPath spins up an httptest server that returns the
// canned payload, points the adapter at it, and asserts both prices are
// resolved back through external_id to the correct ticker.
func TestCoinGeckoFetch_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/simple/price" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("ids"); got == "" {
			t.Error("ids query param missing")
		}
		if got := r.URL.Query().Get("vs_currencies"); got != "usd" {
			t.Errorf("vs_currencies = %q, want usd", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(cgPayload))
	}))
	defer srv.Close()

	src := NewCoinGeckoSource(srv.URL)
	src.pause = 0

	assets := []sqlc.Asset{
		{
			Ticker:     "BTC",
			AssetType:  "crypto",
			ExternalID: pgtype.Text{String: "bitcoin", Valid: true},
			Currency:   "USD",
		},
		{
			Ticker:     "ETH",
			AssetType:  "crypto",
			ExternalID: pgtype.Text{String: "ethereum", Valid: true},
			Currency:   "USD",
		},
	}

	quotes, err := src.Fetch(context.Background(), assets)
	if err != nil {
		t.Fatalf("Fetch returned error: %v", err)
	}
	if len(quotes) != 2 {
		t.Fatalf("got %d quotes, want 2", len(quotes))
	}
	if got := quotes["BTC"].Price; got != "67234.12" {
		t.Errorf("BTC price = %q, want 67234.12", got)
	}
	if got := quotes["ETH"].Price; got != "3210.4" {
		t.Errorf("ETH price = %q, want 3210.4", got)
	}
	if got := quotes["BTC"].Currency; got != "USD" {
		t.Errorf("BTC currency = %q, want USD", got)
	}
}

// TestCoinGeckoFetch_EmptyInput must return an empty map without firing any
// HTTP call.
func TestCoinGeckoFetch_EmptyInput(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Fetch should not call the API for empty input")
	}))
	defer srv.Close()

	src := NewCoinGeckoSource(srv.URL)
	quotes, err := src.Fetch(context.Background(), nil)
	if err != nil {
		t.Fatalf("Fetch returned error: %v", err)
	}
	if len(quotes) != 0 {
		t.Errorf("got %d quotes, want 0", len(quotes))
	}
}

// TestCoinGeckoFetch_SkipsAssetsWithoutExternalID protects against rows in
// the catalog that were inserted without a CoinGecko id — without this guard
// the request would crash with a 404.
func TestCoinGeckoFetch_SkipsAssetsWithoutExternalID(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	src := NewCoinGeckoSource(srv.URL)
	assets := []sqlc.Asset{{Ticker: "XYZ", AssetType: "crypto"}}
	quotes, err := src.Fetch(context.Background(), assets)
	if err != nil {
		t.Fatalf("Fetch returned error: %v", err)
	}
	if called {
		t.Error("expected no HTTP call when no external_id")
	}
	if len(quotes) != 0 {
		t.Errorf("got %d quotes, want 0", len(quotes))
	}
}
