-- name: CreatePortfolio :one
INSERT INTO portfolios (user_id, name, type, description)
VALUES ($1, $2, $3, $4)
RETURNING id, user_id, name, type, description, created_at, updated_at;

-- name: ListPortfoliosByUser :many
SELECT id, user_id, name, type, description, created_at, updated_at
FROM portfolios
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: GetPortfolioByID :one
SELECT id, user_id, name, type, description, created_at, updated_at
FROM portfolios
WHERE id = $1;

-- name: UpdatePortfolio :one
UPDATE portfolios
SET name = $2, type = $3, description = $4, updated_at = NOW()
WHERE id = $1
RETURNING id, user_id, name, type, description, created_at, updated_at;

-- name: DeletePortfolio :exec
DELETE FROM portfolios
WHERE id = $1;
