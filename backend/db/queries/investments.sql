-- name: CreateInvestment :one
INSERT INTO investments (portfolio_id, ticker, asset_type, amount_invested, quantity, purchase_date, notes)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, portfolio_id, ticker, asset_type, amount_invested, quantity, purchase_date, notes, created_at, updated_at;

-- name: ListInvestmentsByPortfolio :many
SELECT id, portfolio_id, ticker, asset_type, amount_invested, quantity, purchase_date, notes, created_at, updated_at
FROM investments
WHERE portfolio_id = $1
ORDER BY purchase_date DESC, created_at DESC;

-- name: GetInvestmentByID :one
SELECT id, portfolio_id, ticker, asset_type, amount_invested, quantity, purchase_date, notes, created_at, updated_at
FROM investments
WHERE id = $1;

-- name: GetInvestmentWithOwner :one
-- joins through portfolios so handlers can check ownership in a single round trip.
SELECT i.id, i.portfolio_id, i.ticker, i.asset_type, i.amount_invested, i.quantity, i.purchase_date, i.notes, i.created_at, i.updated_at, p.user_id
FROM investments i
JOIN portfolios p ON p.id = i.portfolio_id
WHERE i.id = $1;

-- name: UpdateInvestment :one
UPDATE investments
SET ticker = $2, asset_type = $3, amount_invested = $4, quantity = $5, purchase_date = $6, notes = $7, updated_at = NOW()
WHERE id = $1
RETURNING id, portfolio_id, ticker, asset_type, amount_invested, quantity, purchase_date, notes, created_at, updated_at;

-- name: DeleteInvestment :exec
DELETE FROM investments
WHERE id = $1;
