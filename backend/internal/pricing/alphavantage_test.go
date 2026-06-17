package pricing

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/NovaDrake76/grana-tracker/backend/db/sqlc"
)

// canned GLOBAL_QUOTE payload — only the "05. price" field matters to us.
const avPayload = `{
  "Global Quote": {
    "01. symbol": "AAPL",
    "05. price": "189.7000",
    "10. change percent": "0.45%"
  }
}`

func TestAlphaVantageFetch_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/query" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("function"); got != "GLOBAL_QUOTE" {
			t.Errorf("function = %q, want GLOBAL_QUOTE", got)
		}
		if got := r.URL.Query().Get("symbol"); got != "AAPL" {
			t.Errorf("symbol = %q, want AAPL", got)
		}
		if got := r.URL.Query().Get("apikey"); got != "real-key" {
			t.Errorf("apikey = %q, want real-key", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(avPayload))
	}))
	defer srv.Close()

	src := NewAlphaVantageSource(srv.URL, "real-key")
	src.pause = 0

	assets := []sqlc.Asset{{Ticker: "AAPL", AssetType: "stock", Currency: "USD"}}
	quotes, err := src.Fetch(context.Background(), assets)
	if err != nil {
		t.Fatalf("Fetch returned error: %v", err)
	}
	if got := quotes["AAPL"].Price; got != "189.7000" {
		t.Errorf("AAPL price = %q, want 189.7000", got)
	}
	if got := quotes["AAPL"].Currency; got != "USD" {
		t.Errorf("currency = %q, want USD", got)
	}
}

// TestAlphaVantageFetch_PlaceholderKey verifies the source short-circuits with
// ErrSkipped when no real key is configured, so the Service can keep serving
// last-known cache instead of corrupting it with zeroes.
func TestAlphaVantageFetch_PlaceholderKey(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer srv.Close()

	for _, key := range []string{"", "your-key", "changeme", "YOUR_KEY", "demo"} {
		src := NewAlphaVantageSource(srv.URL, key)
		_, err := src.Fetch(context.Background(), []sqlc.Asset{{Ticker: "AAPL", AssetType: "stock"}})
		if !errors.Is(err, ErrSkipped) {
			t.Errorf("key=%q got err=%v, want ErrSkipped", key, err)
		}
	}
	if called {
		t.Error("placeholder key should not trigger HTTP call")
	}
}

// TestAlphaVantageFetch_RateLimitNote translates the upstream "Note" envelope
// into a logged failure (no quote in the map) without aborting subsequent
// tickers.
func TestAlphaVantageFetch_RateLimitNote(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"Note":"Thank you for using Alpha Vantage! Our standard API call frequency is..."}`))
	}))
	defer srv.Close()

	src := NewAlphaVantageSource(srv.URL, "real-key")
	src.pause = 0

	quotes, err := src.Fetch(context.Background(), []sqlc.Asset{{Ticker: "AAPL", AssetType: "stock"}})
	if err != nil {
		t.Fatalf("Fetch returned error: %v", err)
	}
	if _, ok := quotes["AAPL"]; ok {
		t.Error("AAPL should not be in the result when rate-limited")
	}
}

// TestAlphaVantageFetch_EmptyAssets must not hit the network.
func TestAlphaVantageFetch_EmptyAssets(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not call API for empty input")
	}))
	defer srv.Close()

	src := NewAlphaVantageSource(srv.URL, "real-key")
	quotes, err := src.Fetch(context.Background(), nil)
	if err != nil {
		t.Fatalf("Fetch returned error: %v", err)
	}
	if len(quotes) != 0 {
		t.Errorf("got %d quotes, want 0", len(quotes))
	}
}
