-- name: UpsertPortfolioSnapshot :exec
INSERT INTO portfolio_snapshots (portfolio_id, snapshot_date, total_value, currency)
VALUES ($1, $2, $3, $4)
ON CONFLICT (portfolio_id, snapshot_date) DO UPDATE
  SET total_value = EXCLUDED.total_value,
      currency = EXCLUDED.currency;

-- name: ListPortfolioSnapshotsInPeriod :many
SELECT id, portfolio_id, snapshot_date, total_value, currency, created_at
FROM portfolio_snapshots
WHERE portfolio_id = $1 AND snapshot_date >= $2
ORDER BY snapshot_date ASC;
