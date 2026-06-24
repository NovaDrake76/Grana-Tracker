package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/NovaDrake76/grana-tracker/backend/internal/currency"
)

// We deliberately reuse handlers.allowedCurrencies (declared in investment.go)
// as the whitelist: both endpoints accept the same enum, and US09's
// users.preferred_currency CHECK constraint mirrors investments.currency.
// Keeping a single source of truth means adding a third currency later is a
// one-line change.

type CurrencyHandler struct {
	Service *currency.Service
}

func NewCurrencyHandler(svc *currency.Service) *CurrencyHandler {
	return &CurrencyHandler{Service: svc}
}

// rateResponse is the JSON shape the dashboard consumes. fetched_at is RFC3339
// so the front-end can render a localised "atualizado às" label without
// re-parsing a numeric timestamp.
type rateResponse struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Rate      string `json:"rate"`
	FetchedAt string `json:"fetched_at"`
}

// GetRate returns the FROM->TO exchange rate. Defaults are USD->BRL because
// that is the dominant pair for our crypto / S&P 500 holdings; the front-end
// flips them when the user picks USD as their preferred currency.
func (h *CurrencyHandler) GetRate(w http.ResponseWriter, r *http.Request) {
	from := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("from")))
	to := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("to")))
	if from == "" {
		from = "USD"
	}
	if to == "" {
		to = "BRL"
	}
	if _, ok := allowedCurrencies[from]; !ok {
		writeError(w, http.StatusBadRequest, "unsupported 'from' currency", "VALIDATION_ERROR")
		return
	}
	if _, ok := allowedCurrencies[to]; !ok {
		writeError(w, http.StatusBadRequest, "unsupported 'to' currency", "VALIDATION_ERROR")
		return
	}

	rate, fetchedAt, err := h.Service.GetRate(r.Context(), from, to)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to fetch exchange rate", "UPSTREAM_ERROR")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": rateResponse{
			From:      from,
			To:        to,
			Rate:      rate,
			FetchedAt: fetchedAt.Format(time.RFC3339),
		},
	})
}
