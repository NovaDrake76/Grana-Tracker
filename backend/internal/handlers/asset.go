package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/NovaDrake76/grana-tracker/backend/db/sqlc"
	"github.com/NovaDrake76/grana-tracker/backend/internal/pricing"
)

// AssetHandler serves the asset catalog + price endpoints for US04 (autocomplete)
// and US08 (current quotes / refresh trigger).
type AssetHandler struct {
	Queries *sqlc.Queries
	Pricing *pricing.Service
}

func NewAssetHandler(queries *sqlc.Queries, svc *pricing.Service) *AssetHandler {
	return &AssetHandler{Queries: queries, Pricing: svc}
}

// assetResponse is the JSON payload returned by the autocomplete endpoint.
type assetResponse struct {
	ID        string  `json:"id"`
	Ticker    string  `json:"ticker"`
	Name      string  `json:"name"`
	AssetType string  `json:"asset_type"`
	Currency  string  `json:"currency"`
	Market    *string `json:"market"`
	Source    string  `json:"source"`
}

func toAssetResponse(a sqlc.Asset) assetResponse {
	return assetResponse{
		ID:        uuidStr(a.ID),
		Ticker:    a.Ticker,
		Name:      a.Name,
		AssetType: a.AssetType,
		Currency:  a.Currency,
		Market:    textPtr(a.Market),
		Source:    a.Source,
	}
}

// priceResponse is the JSON payload returned by GET /api/prices/{ticker}.
type priceResponse struct {
	Ticker    string `json:"ticker"`
	AssetType string `json:"asset_type"`
	Price     string `json:"price"`
	Currency  string `json:"currency"`
	FetchedAt string `json:"fetched_at"`
	Stale     bool   `json:"stale"`
}

// Search is GET /api/assets/search?q=val&type=stock&limit=10. Public —
// autocomplete needs to work before the user signs in.
func (h *AssetHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeError(w, http.StatusBadRequest, "q is required", "VALIDATION_ERROR")
		return
	}

	assetType := strings.TrimSpace(r.URL.Query().Get("type"))
	if assetType != "" {
		if _, ok := allowedAssetTypes[assetType]; !ok {
			writeError(w, http.StatusBadRequest, "type must be one of stock, crypto, etf, index", "VALIDATION_ERROR")
			return
		}
	}

	limit := int32(10)
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			writeError(w, http.StatusBadRequest, "limit must be a positive integer", "VALIDATION_ERROR")
			return
		}
		if parsed > 50 {
			parsed = 50
		}
		limit = int32(parsed)
	}

	rows, err := h.Queries.SearchAssets(r.Context(), sqlc.SearchAssetsParams{
		Ticker:  "%" + q + "%",
		Column2: assetType,
		Limit:   limit,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to search assets", "INTERNAL_ERROR")
		return
	}

	out := make([]assetResponse, 0, len(rows))
	for _, a := range rows {
		out = append(out, toAssetResponse(a))
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": out,
	})
}

// GetPrice is GET /api/prices/{ticker}?type=stock. Public so the front-end
// can render quotes on the landing dashboard before login.
func (h *AssetHandler) GetPrice(w http.ResponseWriter, r *http.Request) {
	ticker := strings.TrimSpace(chi.URLParam(r, "ticker"))
	if ticker == "" {
		writeError(w, http.StatusBadRequest, "ticker is required", "VALIDATION_ERROR")
		return
	}
	assetType := strings.TrimSpace(r.URL.Query().Get("type"))
	if _, ok := allowedAssetTypes[assetType]; !ok {
		writeError(w, http.StatusBadRequest, "type must be one of stock, crypto, etf, index", "VALIDATION_ERROR")
		return
	}

	price, err := h.Pricing.GetCurrent(r.Context(), ticker, assetType)
	if err != nil {
		if errors.Is(err, pricing.ErrNotFound) {
			writeError(w, http.StatusNotFound, "price not cached for this ticker", "NOT_FOUND")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load price", "INTERNAL_ERROR")
		return
	}

	resp := priceResponse{
		Ticker:    price.Ticker,
		AssetType: price.AssetType,
		Price:     price.Price,
		Currency:  price.Currency,
		FetchedAt: price.FetchedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		Stale:     price.Stale,
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": resp,
	})
}

// Refresh is POST /api/prices/refresh. Authenticated — calling it costs
// upstream API quota, so we don't let anonymous users trigger it.
func (h *AssetHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	summary, err := h.Pricing.RefreshAll(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "refresh failed", "INTERNAL_ERROR")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": map[string]interface{}{
			"refreshed":   summary.Refreshed,
			"errors":      summary.Errors,
			"duration_ms": summary.DurationMS,
		},
		"message": "refresh complete",
	})
}
