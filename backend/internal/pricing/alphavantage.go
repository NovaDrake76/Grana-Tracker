package pricing

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/NovaDrake76/grana-tracker/backend/db/sqlc"
)

// AlphaVantageSource implements Source against the Alpha Vantage GLOBAL_QUOTE
// endpoint. The free tier is brutal (25 calls/day, 5 calls/min) so the daily
// refresh will only ever ask for a few tickers; for student development we
// keep a 200ms delay between requests to stay polite.
type AlphaVantageSource struct {
	BaseURL string
	APIKey  string
	Client  *http.Client
	pause   time.Duration
}

// NewAlphaVantageSource reads the API key at construction time. An empty key
// (or the placeholder "your-key" / "changeme") puts the source into "skipped"
// mode: every Fetch returns ErrSkipped and the Service falls back to the
// existing cache rather than serving zeros.
func NewAlphaVantageSource(baseURL, apiKey string) *AlphaVantageSource {
	if baseURL == "" {
		baseURL = "https://www.alphavantage.co"
	}
	return &AlphaVantageSource{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Client:  &http.Client{Timeout: 10 * time.Second},
		pause:   200 * time.Millisecond,
	}
}

func (a *AlphaVantageSource) Name() string { return "alphavantage" }

// keyPlaceholders catches common "I forgot to set the env var" defaults.
var keyPlaceholders = map[string]struct{}{
	"":          {},
	"your-key":  {},
	"changeme":  {},
	"YOUR_KEY":  {},
	"demo":      {}, // demo key only returns IBM
}

// Fetch calls GLOBAL_QUOTE once per ticker. Free tier caps us at 25 calls/day
// so we let partial responses through: we log per-ticker failures and keep
// going.
func (a *AlphaVantageSource) Fetch(ctx context.Context, assets []sqlc.Asset) (map[string]Price, error) {
	out := map[string]Price{}
	if len(assets) == 0 {
		return out, nil
	}

	if _, isPlaceholder := keyPlaceholders[a.APIKey]; isPlaceholder {
		log.Printf("pricing: ALPHA_VANTAGE_API_KEY not configured — skipping %d stocks", len(assets))
		return out, ErrSkipped
	}

	now := time.Now().UTC()

	for _, asset := range assets {
		select {
		case <-ctx.Done():
			return out, ctx.Err()
		default:
		}

		price, currency, err := a.fetchOne(ctx, asset.Ticker)
		if err != nil {
			log.Printf("pricing: alphavantage %s: %v", asset.Ticker, err)
			continue
		}
		// Alpha Vantage always quotes USD; we honour the row currency for
		// presentation but the raw price stays in USD on disk.
		if currency == "" {
			currency = asset.Currency
		}
		out[asset.Ticker] = Price{
			Ticker:    asset.Ticker,
			AssetType: asset.AssetType,
			Price:     price,
			Currency:  currency,
			FetchedAt: now,
		}

		if a.pause > 0 {
			select {
			case <-ctx.Done():
				return out, ctx.Err()
			case <-time.After(a.pause):
			}
		}
	}

	return out, nil
}

// fetchOne is one HTTP round trip to GLOBAL_QUOTE.
func (a *AlphaVantageSource) fetchOne(ctx context.Context, ticker string) (price, currency string, err error) {
	q := url.Values{}
	q.Set("function", "GLOBAL_QUOTE")
	q.Set("symbol", ticker)
	q.Set("apikey", a.APIKey)

	u := fmt.Sprintf("%s/query?%s", strings.TrimRight(a.BaseURL, "/"), q.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := a.Client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("GET: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", "", fmt.Errorf("alphavantage status %d: %s", resp.StatusCode, string(body))
	}

	// {"Global Quote":{"01. symbol":"AAPL","05. price":"189.7000",...}}
	var payload struct {
		GlobalQuote map[string]string `json:"Global Quote"`
		Note        string            `json:"Note"`
		Info        string            `json:"Information"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", "", fmt.Errorf("decode: %w", err)
	}
	if payload.Note != "" || payload.Info != "" {
		// Hit rate limit — surface as an error so the Service logs it.
		msg := payload.Note
		if msg == "" {
			msg = payload.Info
		}
		return "", "", fmt.Errorf("alphavantage rate-limited: %s", msg)
	}

	if len(payload.GlobalQuote) == 0 {
		return "", "", fmt.Errorf("alphavantage: empty response for %s", ticker)
	}
	priceStr := payload.GlobalQuote["05. price"]
	if priceStr == "" {
		return "", "", fmt.Errorf("alphavantage: no price field for %s", ticker)
	}
	return priceStr, "USD", nil
}
