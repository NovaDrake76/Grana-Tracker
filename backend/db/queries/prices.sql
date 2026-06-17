-- name: GetCurrentPrice :one
SELECT * FROM price_cache WHERE ticker = $1 AND asset_type = $2;

-- name: UpsertPrice :one
INSERT INTO price_cache (ticker, asset_type, price, currency, fetched_at)
VALUES ($1, $2, $3, $4, NOW())
ON CONFLICT (ticker, asset_type) DO UPDATE
  SET price = EXCLUDED.price,
      currency = EXCLUDED.currency,
      fetched_at = EXCLUDED.fetched_at
RETURNING *;

-- name: SnapshotPriceHistory :exec
INSERT INTO price_history (ticker, asset_type, price, currency, recorded_at)
VALUES ($1, $2, $3, $4, CURRENT_DATE)
ON CONFLICT (ticker, asset_type, recorded_at) DO UPDATE
  SET price = EXCLUDED.price,
      currency = EXCLUDED.currency;
