// Package snapshots maintains a daily ledger of each portfolio's total value
// (portfolio_snapshots) so the front-end can render the "Histórico Patrimonial"
// chart for US06 without recomputing prices on every render. The Service is
// invoked by the same daily cron that refreshes price_cache (see cmd/server)
// and is also called on-demand by the HTTP handler when a freshly-created
// portfolio has no points yet.
package snapshots

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/NovaDrake76/grana-tracker/backend/db/sqlc"
)

// Service computes and persists daily portfolio_snapshots rows. UsdToBrlRate is
// the current USD->BRL conversion rate used to normalise USD-denominated
// investments into BRL before summing — US09 will replace the placeholder with
// a live FX feed; for now we ship a constant so the chart has plausible data.
type Service struct {
	Queries      *sqlc.Queries
	UsdToBrlRate string
}

// NewService wires the queries handle and the static FX rate (until US09).
func NewService(queries *sqlc.Queries, usdToBrlRate string) *Service {
	return &Service{Queries: queries, UsdToBrlRate: usdToBrlRate}
}

// SnapshotPortfolio sums (current_price * quantity) across every investment in
// the portfolio, converts USD positions into BRL using UsdToBrlRate, and
// upserts the result with today's date. Per-investment failures (missing
// price, missing quantity) are logged and skipped — we still want a snapshot
// for the partial total rather than no snapshot at all.
func (s *Service) SnapshotPortfolio(ctx context.Context, portfolioID uuid.UUID) error {
	investments, err := s.Queries.ListInvestmentsByPortfolio(ctx, pgtype.UUID{Bytes: portfolioID, Valid: true})
	if err != nil {
		return fmt.Errorf("list investments: %w", err)
	}

	rate, err := bigFloatFromString(s.UsdToBrlRate)
	if err != nil {
		return fmt.Errorf("invalid usd_to_brl_rate %q: %w", s.UsdToBrlRate, err)
	}

	total := new(big.Float).SetPrec(128)
	for _, inv := range investments {
		if !inv.Quantity.Valid {
			// no quantity → cannot compute market value from a unit price.
			continue
		}
		price, err := s.Queries.GetCurrentPrice(ctx, sqlc.GetCurrentPriceParams{
			Ticker:    inv.Ticker,
			AssetType: inv.AssetType,
		})
		if err != nil {
			// no cached price yet — skip, the daily refresh should populate it.
			log.Printf("snapshots: no price for %s (%s): %v", inv.Ticker, inv.AssetType, err)
			continue
		}
		priceF, err := numericToBigFloat(price.Price)
		if err != nil {
			continue
		}
		qtyF, err := numericToBigFloat(inv.Quantity)
		if err != nil {
			continue
		}
		value := new(big.Float).SetPrec(128).Mul(priceF, qtyF)
		if inv.Currency == "USD" {
			value = value.Mul(value, rate)
		}
		total = total.Add(total, value)
	}

	// 8 decimal places matches the DECIMAL(18,8) column on portfolio_snapshots.
	totalStr := total.Text('f', 8)
	totalNum, err := numericFromString(totalStr)
	if err != nil {
		return fmt.Errorf("encode total: %w", err)
	}

	today := pgtype.Date{Time: time.Now().UTC().Truncate(24 * time.Hour), Valid: true}
	if err := s.Queries.UpsertPortfolioSnapshot(ctx, sqlc.UpsertPortfolioSnapshotParams{
		PortfolioID:  pgtype.UUID{Bytes: portfolioID, Valid: true},
		SnapshotDate: today,
		TotalValue:   totalNum,
		Currency:     "BRL",
	}); err != nil {
		return fmt.Errorf("upsert snapshot: %w", err)
	}
	return nil
}

// SnapshotAll iterates every portfolio in the catalog and snapshots each one.
// A failure on one portfolio is logged and added to the returned slice but
// never aborts the whole run — the daily cron must keep going.
func (s *Service) SnapshotAll(ctx context.Context) []error {
	portfolios, err := s.Queries.ListAllPortfolios(ctx)
	if err != nil {
		return []error{fmt.Errorf("list portfolios: %w", err)}
	}

	var errs []error
	for _, p := range portfolios {
		pid := uuid.UUID(p.ID.Bytes)
		if err := s.SnapshotPortfolio(ctx, pid); err != nil {
			log.Printf("snapshots: portfolio %s failed: %v", pid, err)
			errs = append(errs, fmt.Errorf("portfolio %s: %w", pid, err))
		}
	}
	return errs
}

// bigFloatFromString parses a decimal literal into *big.Float at 128-bit
// precision; mirrors the helper used in investment.go to keep math consistent.
func bigFloatFromString(s string) (*big.Float, error) {
	if s == "" {
		return nil, errors.New("empty string")
	}
	f, _, err := big.ParseFloat(s, 10, 128, big.ToNearestEven)
	return f, err
}

// numericToBigFloat unwraps pgtype.Numeric into *big.Float via its string form
// so we never lose precision through float64.
func numericToBigFloat(n pgtype.Numeric) (*big.Float, error) {
	if !n.Valid {
		return nil, errors.New("null numeric")
	}
	v, err := n.Value()
	if err != nil || v == nil {
		return nil, errors.New("empty numeric value")
	}
	var s string
	switch t := v.(type) {
	case string:
		s = t
	case []byte:
		s = string(t)
	default:
		s = fmt.Sprintf("%v", t)
	}
	return bigFloatFromString(s)
}

// numericFromString turns a plain decimal literal into pgtype.Numeric.
func numericFromString(s string) (pgtype.Numeric, error) {
	var n pgtype.Numeric
	if err := n.Scan(s); err != nil {
		return n, err
	}
	return n, nil
}
