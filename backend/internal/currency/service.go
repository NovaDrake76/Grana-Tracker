// Package currency exposes a small FX rate service that powers US09 (Moeda
// Preferida). It calls AwesomeAPI's economia.awesomeapi.com.br/json/last
// endpoint, caches every rate for one hour, and falls back to the last good
// value if the upstream is unreachable so the dashboard never goes blank.
//
// Only the dashboard / UI reads through this service — the snapshot pipeline
// keeps using its own static rate. Centralising conversion server-side would
// force every cached portfolio value to be recomputed on every render; instead
// the front-end fetches /api/currency/rate once per session and multiplies
// values locally.
package currency

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

// cacheTTL is how long we trust a fetched rate before going back to AwesomeAPI.
// One hour matches the dashboard refresh cadence; rates rarely move enough in
// that window to mislead a casual portfolio view.
const cacheTTL = 1 * time.Hour

// cachedRate is the in-memory record per "FROM-TO" pair. fetchedAt is the wall
// time we received the response so callers can render "atualizado às HH:MM".
type cachedRate struct {
	rate      string
	fetchedAt time.Time
}

// Service holds the HTTP client, the rate cache, and the mutex that protects
// it. It is safe to share across handlers — every public method takes the lock
// only for the brief read/write window.
type Service struct {
	httpClient *http.Client
	baseURL    string
	cache      map[string]cachedRate
	mu         sync.Mutex
}

// NewService returns a Service with a 5s HTTP timeout (AwesomeAPI is normally
// sub-second; a long timeout would block the dashboard render on an outage).
func NewService() *Service {
	return &Service{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		baseURL:    "https://economia.awesomeapi.com.br/json/last",
		cache:      make(map[string]cachedRate),
	}
}

// WithBaseURL overrides the upstream base for tests pointing at an httptest
// server. Returns the same Service so callers can chain it during construction.
func (s *Service) WithBaseURL(url string) *Service {
	s.baseURL = url
	return s
}

// awesomePayload mirrors the JSON envelope AwesomeAPI emits. The outer key is
// the FROM+TO pair concatenated without a hyphen (e.g. "USDBRL"); we decode
// into a generic map and pick the first value so the parser is agnostic to
// the requested pair.
type awesomeQuote struct {
	Bid       string `json:"bid"`
	Ask       string `json:"ask"`
	Timestamp string `json:"timestamp"`
}

// GetRate returns the latest FROM->TO rate as a decimal string plus the time
// it was fetched. Calling with from == to short-circuits to "1.00" so callers
// can ask for any pair without special-casing the identity conversion.
//
// On upstream failure GetRate returns the last cached value (no matter how
// stale) and logs a warning. Only if there is no cached entry at all do we
// surface the error to the handler.
func (s *Service) GetRate(ctx context.Context, from, to string) (string, time.Time, error) {
	from = strings.ToUpper(strings.TrimSpace(from))
	to = strings.ToUpper(strings.TrimSpace(to))
	if from == to {
		return "1.00", time.Now(), nil
	}

	key := from + "-" + to

	s.mu.Lock()
	cached, hasCached := s.cache[key]
	s.mu.Unlock()
	if hasCached && time.Since(cached.fetchedAt) < cacheTTL {
		return cached.rate, cached.fetchedAt, nil
	}

	rate, err := s.fetch(ctx, from, to)
	if err != nil {
		if hasCached {
			log.Printf("currency: upstream %s-%s failed (%v), serving stale rate from %s", from, to, err, cached.fetchedAt.Format(time.RFC3339))
			return cached.rate, cached.fetchedAt, nil
		}
		return "", time.Time{}, err
	}

	now := time.Now()
	s.mu.Lock()
	s.cache[key] = cachedRate{rate: rate, fetchedAt: now}
	s.mu.Unlock()
	return rate, now, nil
}

// fetch calls AwesomeAPI and pulls the "bid" out of the (single) inner object.
// The outer key varies per request ("USDBRL", "BRLUSD"…), so we iterate the
// map rather than hard-coding it.
func (s *Service) fetch(ctx context.Context, from, to string) (string, error) {
	url := fmt.Sprintf("%s/%s-%s", s.baseURL, from, to)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("call awesomeapi: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("awesomeapi status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload map[string]awesomeQuote
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if len(payload) == 0 {
		return "", fmt.Errorf("awesomeapi: empty payload")
	}
	for _, q := range payload {
		if q.Bid == "" {
			return "", fmt.Errorf("awesomeapi: missing bid")
		}
		return q.Bid, nil
	}
	return "", fmt.Errorf("awesomeapi: no quote in payload")
}

// Convert multiplies amount by the FROM->TO rate and returns a decimal string
// rounded to 2 places. Uses math/big.Float so callers do not lose precision on
// large portfolio totals; the snapshots package already uses big.Float for the
// same reason, so this stays consistent with the rest of the money math.
func (s *Service) Convert(ctx context.Context, amount, from, to string) (string, error) {
	rate, _, err := s.GetRate(ctx, from, to)
	if err != nil {
		return "", err
	}
	amt, _, err := big.ParseFloat(strings.TrimSpace(amount), 10, 128, big.ToNearestEven)
	if err != nil {
		return "", fmt.Errorf("parse amount %q: %w", amount, err)
	}
	r, _, err := big.ParseFloat(strings.TrimSpace(rate), 10, 128, big.ToNearestEven)
	if err != nil {
		return "", fmt.Errorf("parse rate %q: %w", rate, err)
	}
	product := new(big.Float).SetPrec(128).Mul(amt, r)
	return product.Text('f', 2), nil
}
