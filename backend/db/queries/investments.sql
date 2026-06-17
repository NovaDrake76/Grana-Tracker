-- name: CreateInvestment :one
INSERT INTO investments (portfolio_id, ticker, asset_type, amount_invested, quantity, purchase_price, currency, purchase_date, notes)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: ListInvestmentsByPortfolio :many
SELECT *
FROM investments
WHERE portfolio_id = $1
ORDER BY purchase_date DESC, created_at DESC;

-- name: GetInvestmentByID :one
SELECT *
FROM investments
WHERE id = $1;

-- name: GetInvestmentWithOwner :one
-- joins through portfolios so handlers can check ownership in a single round trip.
SELECT i.*, p.user_id
FROM investments i
JOIN portfolios p ON p.id = i.portfolio_id
WHERE i.id = $1;

-- name: UpdateInvestment :one
UPDATE investments
SET ticker = $2, asset_type = $3, amount_invested = $4, quantity = $5, purchase_price = $6, currency = $7, purchase_date = $8, notes = $9, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteInvestment :exec
DELETE FROM investments
WHERE id = $1;
