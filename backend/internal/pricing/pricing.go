// Package pricing encapsulates external market-price providers (CoinGecko +
// Alpha Vantage) and the bookkeeping that keeps the price_cache /
// price_history tables warm. Handlers read prices through Service; the
// daily refresh goroutine writes them.
package pricing

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/NovaDrake76/grana-tracker/backend/db/sqlc"
)

// ErrNotFound is returned by GetCurrent when the price_cache row does not exist
// for the requested ticker/asset_type pair.
var ErrNotFound = errors.New("pricing: price not cached")

// ErrSkipped is returned by a Source.Fetch when the source has been disabled at
// runtime (e.g. Alpha Vantage missing API key). The Service treats it as a
// non-fatal signal and falls back to the existing cache.
var ErrSkipped = errors.New("pricing: source skipped")

// Price is the in-memory representation of a quote returned to handlers.
// Price is encoded as a decimal string so callers can decide rounding.
type Price struct {
	Ticker    string    `json:"ticker"`
	AssetType string    `json:"asset_type"`
	Price     string    `json:"price"`
	Currency  string    `json:"currency"`
	FetchedAt time.Time `json:"fetched_at"`
	// Stale is true when the cache row is older than the configured TTL OR
	// when the latest refresh attempt failed and we are serving last-known
	// data. Front-end can render a "data may be outdated" indicator.
	Stale bool `json:"stale"`
}

// Source fetches a batch of quotes for assets it knows how to handle.
// Implementations should be tolerant to partial failures: return the map of
// tickers it could resolve and an error describing the rest. The Service
// guarantees the input slice will only contain assets routed to this source
// (e.g. only crypto assets to CoinGecko).
type Source interface {
	Name() string
	Fetch(ctx context.Context, assets []sqlc.Asset) (map[string]Price, error)
}

// RefreshSummary describes what RefreshAll did. The HTTP handler returns it.
type RefreshSummary struct {
	Refreshed  int           `json:"refreshed"`
	Errors     []string      `json:"errors"`
	DurationMS int64         `json:"duration_ms"`
	StartedAt  time.Time     `json:"started_at"`
	Took       time.Duration `json:"-"`
}

// StaleAfter controls when GetCurrent marks a cached price as outdated.
// 26h gives the daily refresh a 2h grace window before users see a warning.
const StaleAfter = 26 * time.Hour

// Service is the orchestrator: it knows which Source handles each asset_type,
// keeps the cache warm via RefreshAll, and reads it back via GetCurrent.
type Service struct {
	queries *sqlc.Queries
	sources map[string]Source // keyed by asset_type
	mu      sync.Mutex        // serialises RefreshAll calls so two cron ticks cannot interleave writes
}

// NewService wires the default routing: crypto -> CoinGecko, stock/etf/index ->
// Alpha Vantage. Callers can override individual sources before use (handy in
// tests).
func NewService(queries *sqlc.Queries, coinGecko, alphaVantage Source) *Service {
	s := &Service{
		queries: queries,
		sources: make(map[string]Source),
	}
	if coinGecko != nil {
		s.sources["crypto"] = coinGecko
	}
	if alphaVantage != nil {
		s.sources["stock"] = alphaVantage
		s.sources["etf"] = alphaVantage
		s.sources["index"] = alphaVantage
	}
	return s
}

// RegisterSource lets tests inject a stub for a specific asset_type.
func (s *Service) RegisterSource(assetType string, src Source) {
	s.sources[assetType] = src
}

// GetCurrent returns the latest cached price, or ErrNotFound if no row exists.
// It also flags Stale = true when fetched_at is older than StaleAfter so the
// UI can surface that.
func (s *Service) GetCurrent(ctx context.Context, ticker, assetType string) (*Price, error) {
	row, err := s.queries.GetCurrentPrice(ctx, sqlc.GetCurrentPriceParams{
		Ticker:    ticker,
		AssetType: assetType,
	})
	if err != nil {
		// pgx.ErrNoRows is the only "expected" error here; collapse everything
		// to ErrNotFound and let the handler decide.
		return nil, ErrNotFound
	}
	priceStr := numericString(row.Price)
	currency := ""
	if row.Currency.Valid {
		currency = row.Currency.String
	}
	fetched := time.Time{}
	if row.FetchedAt.Valid {
		fetched = row.FetchedAt.Time
	}
	stale := !fetched.IsZero() && time.Since(fetched) > StaleAfter
	return &Price{
		Ticker:    row.Ticker,
		AssetType: row.AssetType,
		Price:     priceStr,
		Currency:  currency,
		FetchedAt: fetched,
		Stale:     stale,
	}, nil
}

// RefreshAll walks the asset catalog, groups by asset_type, calls each Source,
// and upserts price_cache + price_history. Per-ticker errors are collected but
// never abort the whole refresh — we always want to write whatever we did get.
func (s *Service) RefreshAll(ctx context.Context) (RefreshSummary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	start := time.Now()
	summary := RefreshSummary{StartedAt: start}

	assets, err := s.queries.ListAllAssets(ctx)
	if err != nil {
		summary.DurationMS = time.Since(start).Milliseconds()
		summary.Errors = append(summary.Errors, fmt.Sprintf("list assets: %v", err))
		return summary, fmt.Errorf("list assets: %w", err)
	}

	// Group by asset_type so each source receives one batch call.
	byType := make(map[string][]sqlc.Asset)
	for _, a := range assets {
		byType[a.AssetType] = append(byType[a.AssetType], a)
	}

	for assetType, group := range byType {
		src, ok := s.sources[assetType]
		if !ok {
			summary.Errors = append(summary.Errors, fmt.Sprintf("no source for %s", assetType))
			continue
		}
		quotes, err := src.Fetch(ctx, group)
		if errors.Is(err, ErrSkipped) {
			log.Printf("pricing: source %s skipped for %s", src.Name(), assetType)
			summary.Errors = append(summary.Errors, fmt.Sprintf("%s skipped", src.Name()))
			continue
		}
		if err != nil {
			log.Printf("pricing: source %s for %s returned: %v", src.Name(), assetType, err)
			summary.Errors = append(summary.Errors, fmt.Sprintf("%s: %v", src.Name(), err))
			// fall through — partial success is allowed
		}
		for ticker, q := range quotes {
			if err := s.persist(ctx, ticker, assetType, q); err != nil {
				summary.Errors = append(summary.Errors, fmt.Sprintf("persist %s: %v", ticker, err))
				continue
			}
			summary.Refreshed++
		}
	}

	summary.Took = time.Since(start)
	summary.DurationMS = summary.Took.Milliseconds()
	return summary, nil
}

// persist writes one quote to price_cache + price_history.
func (s *Service) persist(ctx context.Context, ticker, assetType string, q Price) error {
	num, err := decimalFromString(q.Price)
	if err != nil {
		return fmt.Errorf("parse price %q: %w", q.Price, err)
	}
	currency := q.Currency
	if currency == "" {
		currency = "USD"
	}
	if _, err := s.queries.UpsertPrice(ctx, sqlc.UpsertPriceParams{
		Ticker:    ticker,
		AssetType: assetType,
		Price:     num,
		Currency:  pgtype.Text{String: currency, Valid: true},
	}); err != nil {
		return fmt.Errorf("upsert: %w", err)
	}
	if err := s.queries.SnapshotPriceHistory(ctx, sqlc.SnapshotPriceHistoryParams{
		Ticker:    ticker,
		AssetType: assetType,
		Price:     num,
		Currency:  pgtype.Text{String: currency, Valid: true},
	}); err != nil {
		return fmt.Errorf("history: %w", err)
	}
	return nil
}

// decimalFromString turns a plain decimal literal into pgtype.Numeric.
func decimalFromString(s string) (pgtype.Numeric, error) {
	var n pgtype.Numeric
	if s == "" {
		return n, errors.New("empty price")
	}
	if err := n.Scan(s); err != nil {
		return n, err
	}
	return n, nil
}

// numericString renders a pgtype.Numeric back to a flat decimal string.
func numericString(n pgtype.Numeric) string {
	if !n.Valid {
		return ""
	}
	v, err := n.Value()
	if err != nil || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	default:
		return fmt.Sprintf("%v", t)
	}
}
