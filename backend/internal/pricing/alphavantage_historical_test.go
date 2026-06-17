package pricing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/NovaDrake76/grana-tracker/backend/db/sqlc"
)

// canned TIME_SERIES_DAILY payload — three trading days, with 2025-06-14
// (Saturday) and 2025-06-15 (Sunday) deliberately absent so the rollback path
// exercises itself.
const avHistoryPayload = `{
  "Meta Data": {
    "1. Information": "Daily Prices (open, high, low, close) and Volumes",
    "2. Symbol": "AAPL"
  },
  "Time Series (Daily)": {
    "2025-06-16": {"1. open": "190.00", "2. high": "192.00", "3. low": "189.00", "4. close": "191.50", "5. volume": "12345"},
    "2025-06-13": {"1. open": "188.00", "2. high": "190.00", "3. low": "187.00", "4. close": "189.70", "5. volume": "22345"},
    "2025-06-12": {"1. open": "187.00", "2. high": "189.00", "3. low": "186.00", "4. close": "188.20", "5. volume": "32345"}
  }
}`

func TestAlphaVantageFetchHistorical_ExactDate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/query" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("function"); got != "TIME_SERIES_DAILY" {
			t.Errorf("function = %q, want TIME_SERIES_DAILY", got)
		}
		if got := r.URL.Query().Get("symbol"); got != "AAPL" {
			t.Errorf("symbol = %q, want AAPL", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(avHistoryPayload))
	}))
	defer srv.Close()

	src := NewAlphaVantageSource(srv.URL, "real-key")
	src.pause = 0

	asset := sqlc.Asset{Ticker: "AAPL", AssetType: "stock", Currency: "USD"}
	date := time.Date(2025, 6, 16, 0, 0, 0, 0, time.UTC)

	price, err := src.FetchHistorical(context.Background(), asset, date)
	if err != nil {
		t.Fatalf("FetchHistorical returned error: %v", err)
	}
	if price.Price != "191.50" {
		t.Errorf("price = %q, want 191.50", price.Price)
	}
	if price.Currency != "USD" {
		t.Errorf("currency = %q, want USD", price.Currency)
	}
}

// TestAlphaVantageFetchHistorical_WeekendRollback asks for Sunday 2025-06-15.
// The fixture has no entry for 06-15 or 06-14; the function must step back to
// Friday 06-13 and return its closing price.
func TestAlphaVantageFetchHistorical_WeekendRollback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(avHistoryPayload))
	}))
	defer srv.Close()

	src := NewAlphaVantageSource(srv.URL, "real-key")
	src.pause = 0

	asset := sqlc.Asset{Ticker: "AAPL", AssetType: "stock", Currency: "USD"}
	date := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC) // Sunday

	price, err := src.FetchHistorical(context.Background(), asset, date)
	if err != nil {
		t.Fatalf("FetchHistorical returned error: %v", err)
	}
	if price.Price != "189.70" {
		t.Errorf("price = %q, want 189.70 (rolled back to Friday)", price.Price)
	}
}

// TestAlphaVantageFetchHistorical_B3SkipsUpstream ensures BVMF assets never
// trigger an HTTP call (Alpha Vantage free tier does not cover Brazilian
// equities) — they short-circuit to ErrNotFound.
func TestAlphaVantageFetchHistorical_B3SkipsUpstream(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer srv.Close()

	src := NewAlphaVantageSource(srv.URL, "real-key")
	asset := sqlc.Asset{
		Ticker:    "PETR4",
		AssetType: "stock",
		Currency:  "BRL",
		Market:    pgtype.Text{String: "BVMF", Valid: true},
	}
	_, err := src.FetchHistorical(context.Background(), asset, time.Now())
	if err != ErrNotFound {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
	if called {
		t.Error("B3 lookup must not hit the network")
	}
}

// TestAlphaVantageFetchHistorical_PlaceholderKey mirrors the live-fetch
// short-circuit: missing/placeholder API key returns ErrSkipped without
// touching the network.
func TestAlphaVantageFetchHistorical_PlaceholderKey(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer srv.Close()

	src := NewAlphaVantageSource(srv.URL, "")
	_, err := src.FetchHistorical(context.Background(), sqlc.Asset{Ticker: "AAPL", AssetType: "stock"}, time.Now())
	if err != ErrSkipped {
		t.Errorf("err = %v, want ErrSkipped", err)
	}
	if called {
		t.Error("placeholder key must not trigger an HTTP call")
	}
}

// TestAlphaVantageFetchHistorical_RateLimited surfaces the "Note" envelope as
// ErrNotFound (so the HTTP handler returns a clean 404).
func TestAlphaVantageFetchHistorical_RateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Note":"Thank you for using Alpha Vantage..."}`))
	}))
	defer srv.Close()

	src := NewAlphaVantageSource(srv.URL, "real-key")
	src.pause = 0

	asset := sqlc.Asset{Ticker: "AAPL", AssetType: "stock"}
	_, err := src.FetchHistorical(context.Background(), asset, time.Now())
	if err != ErrNotFound {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}
