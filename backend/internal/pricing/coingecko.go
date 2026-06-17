package pricing

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/NovaDrake76/grana-tracker/backend/db/sqlc"
)

// coinGeckoChunkSize keeps each /simple/price call under the free-tier 30/min
// rate limit. CoinGecko accepts CSV ids, so we batch up to 50 ids per call.
const coinGeckoChunkSize = 50

// CoinGeckoSource implements Source against the CoinGecko v3 free API.
// External ids on the `assets` table map directly to CoinGecko coin ids.
type CoinGeckoSource struct {
	BaseURL string        // overridable for httptest
	Client  *http.Client  // injected so tests can use httptest server
	Currency string       // "usd" — the API always quotes one currency per call
	pause   time.Duration // backoff between chunks
}

// NewCoinGeckoSource returns a ready-to-use adapter with sane defaults.
func NewCoinGeckoSource(baseURL string) *CoinGeckoSource {
	if baseURL == "" {
		baseURL = "https://api.coingecko.com/api/v3"
	}
	return &CoinGeckoSource{
		BaseURL:  baseURL,
		Client:   &http.Client{Timeout: 10 * time.Second},
		Currency: "usd",
		pause:    1500 * time.Millisecond,
	}
}

func (c *CoinGeckoSource) Name() string { return "coingecko" }

// Fetch makes one or more /simple/price calls and returns a map keyed by the
// asset ticker (NOT the CoinGecko id) so the Service can match it back to the
// row in our table.
func (c *CoinGeckoSource) Fetch(ctx context.Context, assets []sqlc.Asset) (map[string]Price, error) {
	out := map[string]Price{}
	if len(assets) == 0 {
		return out, nil
	}

	// Index CoinGecko id -> ticker so we can flip the response back into our
	// canonical keys. Assets without an external_id are skipped (free-text
	// tickers like "ABC" would otherwise crash the call).
	idToTicker := make(map[string]string, len(assets))
	var ids []string
	for _, a := range assets {
		if !a.ExternalID.Valid || a.ExternalID.String == "" {
			continue
		}
		idToTicker[a.ExternalID.String] = a.Ticker
		ids = append(ids, a.ExternalID.String)
	}
	if len(ids) == 0 {
		return out, nil
	}

	now := time.Now().UTC()

	for chunk := range chunkBy(ids, coinGeckoChunkSize) {
		quotes, err := c.fetchChunk(ctx, chunk)
		if err != nil {
			return out, err
		}
		for id, price := range quotes {
			ticker, ok := idToTicker[id]
			if !ok {
				continue
			}
			out[ticker] = Price{
				Ticker:    ticker,
				AssetType: "crypto",
				Price:     strconv.FormatFloat(price, 'f', -1, 64),
				Currency:  strings.ToUpper(c.Currency),
				FetchedAt: now,
			}
		}
		if c.pause > 0 && len(ids) > coinGeckoChunkSize {
			select {
			case <-ctx.Done():
				return out, ctx.Err()
			case <-time.After(c.pause):
			}
		}
	}

	return out, nil
}

// fetchChunk calls /simple/price for a single batch of ids.
func (c *CoinGeckoSource) fetchChunk(ctx context.Context, ids []string) (map[string]float64, error) {
	q := url.Values{}
	q.Set("ids", strings.Join(ids, ","))
	q.Set("vs_currencies", c.Currency)

	u := fmt.Sprintf("%s/simple/price?%s", strings.TrimRight(c.BaseURL, "/"), q.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("coingecko GET: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("coingecko status %d: %s", resp.StatusCode, string(body))
	}

	// {"bitcoin":{"usd":67234.12},"ethereum":{"usd":3210.4}}
	var raw map[string]map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode coingecko: %w", err)
	}

	out := make(map[string]float64, len(raw))
	for id, currencies := range raw {
		if v, ok := currencies[c.Currency]; ok {
			out[id] = v
		}
	}
	return out, nil
}

// chunkBy yields successive size-bounded slices of in. Implemented as a
// closure-returning function so we can use `range` over Go 1.23 iterators.
func chunkBy(in []string, size int) func(yield func([]string) bool) {
	return func(yield func([]string) bool) {
		for i := 0; i < len(in); i += size {
			j := i + size
			if j > len(in) {
				j = len(in)
			}
			if !yield(in[i:j]) {
				return
			}
		}
	}
}
