package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/NovaDrake76/grana-tracker/backend/db/sqlc"
	"github.com/NovaDrake76/grana-tracker/backend/internal/handlers"
	"github.com/NovaDrake76/grana-tracker/backend/internal/pricing"
)

// stubPricingSource returns a canned price for any asset it's asked about — used
// to drive the handler without hitting external APIs.
type stubPricingSource struct {
	priceByTicker map[string]string
}

func (s *stubPricingSource) Name() string { return "stub" }
func (s *stubPricingSource) Fetch(_ context.Context, assets []sqlc.Asset) (map[string]pricing.Price, error) {
	out := make(map[string]pricing.Price, len(assets))
	now := time.Now().UTC()
	for _, a := range assets {
		price, ok := s.priceByTicker[a.Ticker]
		if !ok {
			price = "1.00"
		}
		out[a.Ticker] = pricing.Price{
			Ticker:    a.Ticker,
			AssetType: a.AssetType,
			Price:     price,
			Currency:  a.Currency,
			FetchedAt: now,
		}
	}
	return out, nil
}

func newAssetTestRouter(t *testing.T) (chi.Router, *pricing.Service) {
	t.Helper()
	queries := sqlc.New(testPool)
	crypto := &stubPricingSource{priceByTicker: map[string]string{"BTC": "65000.00"}}
	stocks := &stubPricingSource{priceByTicker: map[string]string{"AAPL": "189.50"}}
	svc := pricing.NewService(queries, crypto, stocks)
	h := handlers.NewAssetHandler(queries, svc)

	r := chi.NewRouter()
	r.Route("/api", func(r chi.Router) {
		r.Get("/assets/search", h.Search)
		r.Get("/prices/{ticker}", h.GetPrice)
		r.Post("/prices/refresh", h.Refresh)
	})
	return r, svc
}

func TestSearchAssets_HappyPath(t *testing.T) {
	requireDB(t)

	r, _ := newAssetTestRouter(t)
	rr, resp := doRequest(t, r, http.MethodGet, "/api/assets/search?q=AAPL&type=stock&limit=5", "", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rr.Code, rr.Body.String())
	}
	var out []map[string]interface{}
	if err := json.Unmarshal(resp.Data, &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(out) == 0 {
		t.Fatalf("expected at least one asset, got 0")
	}
	if got := out[0]["ticker"]; got != "AAPL" {
		t.Errorf("ticker = %v, want AAPL", got)
	}
}

func TestSearchAssets_MissingQ(t *testing.T) {
	requireDB(t)
	r, _ := newAssetTestRouter(t)
	rr, _ := doRequest(t, r, http.MethodGet, "/api/assets/search", "", nil)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

func TestSearchAssets_BadType(t *testing.T) {
	requireDB(t)
	r, _ := newAssetTestRouter(t)
	rr, _ := doRequest(t, r, http.MethodGet, "/api/assets/search?q=AA&type=bogus", "", nil)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

func TestGetPrice_NotFound(t *testing.T) {
	requireDB(t)
	truncateAll(t)

	r, _ := newAssetTestRouter(t)
	rr, _ := doRequest(t, r, http.MethodGet, "/api/prices/BTC?type=crypto", "", nil)
	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 (no cache row yet)", rr.Code)
	}
}

func TestGetPrice_AfterRefresh(t *testing.T) {
	requireDB(t)
	truncateAll(t)

	r, svc := newAssetTestRouter(t)
	if _, err := svc.RefreshAll(context.Background()); err != nil {
		t.Fatalf("seed refresh: %v", err)
	}

	rr, resp := doRequest(t, r, http.MethodGet, "/api/prices/BTC?type=crypto", "", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rr.Code, rr.Body.String())
	}
	var price map[string]interface{}
	if err := json.Unmarshal(resp.Data, &price); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got := price["ticker"]; got != "BTC" {
		t.Errorf("ticker = %v, want BTC", got)
	}
	if got := price["stale"]; got != false {
		t.Errorf("stale = %v, want false right after refresh", got)
	}
}

func TestGetPrice_BadType(t *testing.T) {
	requireDB(t)
	r, _ := newAssetTestRouter(t)
	rr, _ := doRequest(t, r, http.MethodGet, "/api/prices/BTC?type=bogus", "", nil)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

func TestRefresh_PopulatesCache(t *testing.T) {
	requireDB(t)
	truncateAll(t)

	r, _ := newAssetTestRouter(t)
	rr, resp := doRequest(t, r, http.MethodPost, "/api/prices/refresh", "", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rr.Code, rr.Body.String())
	}
	var payload struct {
		Refreshed int `json:"refreshed"`
	}
	if err := json.Unmarshal(resp.Data, &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Refreshed == 0 {
		t.Errorf("refreshed = 0, want > 0")
	}
}

