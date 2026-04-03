-- name: CreateUser :one
INSERT INTO users (name, email, password_hash)
VALUES ($1, $2, $3)
RETURNING id, name, email, preferred_currency, created_at, updated_at;

-- name: GetUserByEmail :one
SELECT id, name, email, password_hash, preferred_currency, created_at, updated_at
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, name, email, preferred_currency, created_at, updated_at
FROM users
WHERE id = $1;

-- name: UpdateUser :one
UPDATE users
SET name = $2, preferred_currency = $3, updated_at = NOW()
WHERE id = $1
RETURNING id, name, email, preferred_currency, created_at, updated_at;
