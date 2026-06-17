-- name: SearchAssets :many
-- ILIKE on ticker OR name, optional asset_type filter, LIMIT $3.
-- args: query string ("%val%"), asset_type or empty string, limit
SELECT * FROM assets
 WHERE (ticker ILIKE $1 OR name ILIKE $1)
   AND ($2 = '' OR asset_type = $2)
 ORDER BY (ticker ILIKE $1) DESC, ticker ASC
 LIMIT $3;

-- name: GetAssetByTicker :one
SELECT * FROM assets WHERE LOWER(ticker) = LOWER($1) AND asset_type = $2;

-- name: ListAssetsByType :many
SELECT * FROM assets WHERE asset_type = $1 ORDER BY ticker;

-- name: ListAllAssets :many
SELECT * FROM assets ORDER BY asset_type, ticker;
