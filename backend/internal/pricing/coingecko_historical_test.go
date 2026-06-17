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

// canned /coins/bitcoin/history response — only market_data.current_price.usd
// matters to us.
const cgHistoryPayload = `{
  "id": "bitcoin",
  "symbol": "btc",
  "market_data": {
    "current_price": {
      "usd": 65123.45,
      "eur": 60000.0
    }
  }
}`

func TestCoinGeckoFetchHistorical_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/coins/bitcoin/history" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		// CoinGecko expects DD-MM-YYYY, not YYYY-MM-DD.
		if got := r.URL.Query().Get("date"); got != "15-06-2025" {
			t.Errorf("date = %q, want 15-06-2025", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(cgHistoryPayload))
	}))
	defer srv.Close()

	src := NewCoinGeckoSource(srv.URL)
	src.pause = 0

	asset := sqlc.Asset{
		Ticker:     "BTC",
		AssetType:  "crypto",
		ExternalID: pgtype.Text{String: "bitcoin", Valid: true},
		Currency:   "USD",
	}
	date := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

	price, err := src.FetchHistorical(context.Background(), asset, date)
	if err != nil {
		t.Fatalf("FetchHistorical returned error: %v", err)
	}
	if price.Price != "65123.45" {
		t.Errorf("price = %q, want 65123.45", price.Price)
	}
	if price.Currency != "USD" {
		t.Errorf("currency = %q, want USD", price.Currency)
	}
	if price.Ticker != "BTC" {
		t.Errorf("ticker = %q, want BTC", price.Ticker)
	}
}

// TestCoinGeckoFetchHistorical_MissingMarketData mimics the response shape
// CoinGecko returns when the date is older than ~365 days on the free tier:
// the body parses but market_data is empty.
func TestCoinGeckoFetchHistorical_MissingMarketData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"bitcoin"}`))
	}))
	defer srv.Close()

	src := NewCoinGeckoSource(srv.URL)
	asset := sqlc.Asset{
		Ticker:     "BTC",
		AssetType:  "crypto",
		ExternalID: pgtype.Text{String: "bitcoin", Valid: true},
	}
	_, err := src.FetchHistorical(context.Background(), asset, time.Now().AddDate(-2, 0, 0))
	if err != ErrNotFound {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

// TestCoinGeckoFetchHistorical_NoExternalID short-circuits without firing any
// HTTP call when the asset row lacks a CoinGecko id.
func TestCoinGeckoFetchHistorical_NoExternalID(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer srv.Close()

	src := NewCoinGeckoSource(srv.URL)
	asset := sqlc.Asset{Ticker: "XYZ", AssetType: "crypto"}
	_, err := src.FetchHistorical(context.Background(), asset, time.Now())
	if err != ErrNotFound {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
	if called {
		t.Error("expected no HTTP call when external_id missing")
	}
}
