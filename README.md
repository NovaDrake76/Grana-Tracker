# Grana Tracker

Investment tracker where you register real and simulated positions (stocks, crypto, ETFs, indices) and see them together. Real and simulated portfolios are tracked the same way; the only difference is a flag at the portfolio level.

Project for DIM0547 (Desenvolvimento de Sistemas Web II com Go), 2026.1.

## Stack

- Backend: Go with Chi, pgx, JWT
- Database: PostgreSQL 16
- Frontend: Next.js 16 + Chakra UI

## Requirements

- Go 1.22+
- Node 20+
- Docker (for Postgres)

## Setup

Clone the repo and install the frontend deps:

```
cd frontend
npm install
cd ..
npm install
```

Create `backend/.env` from the example:

```
cp backend/.env.example backend/.env
```

Fill in `DATABASE_URL` and `JWT_SECRET` at minimum. A working local config:

```
DATABASE_URL=postgresql://granatracker:granatracker@localhost:5432/granatracker?sslmode=disable
JWT_SECRET=change-me
PORT=8080
FRONTEND_URL=http://localhost:3000
```

And `frontend/.env.local`:

```
NEXT_PUBLIC_API_URL=http://localhost:8080/api
```

## Running

From the repo root:

```
npm run dev
```

That starts Postgres in Docker, the Go API on :8080 and Next.js on :3000.

Individual parts if you need them:

```
npm run db         # postgres only
npm run db:stop    # stop postgres
npm run backend    # go api
npm run frontend   # next dev
```

The first time the API starts it runs the migration in `backend/db/migrations/001_init.up.sql`, so you don't need to do anything manually.

## Tests

The backend has two layers of tests:

- **Unit tests** (no DB): `internal/services`, `internal/middleware`, and handler helpers. Always run.
- **Integration tests** (real Postgres): `internal/handlers` — hit the real HTTP + SQL stack. Run only when `TEST_DATABASE_URL` is set; skip cleanly otherwise.

### Run only the unit tests

```
cd backend
go test ./...
```

Integration tests will print `SKIP: TEST_DATABASE_URL not set` — that's expected.

### Run everything (unit + integration)

First start a Postgres the tests can use. The simplest option is to reuse the dev one:

```
npm run db
```

Then point the tests at it and run:

```
cd backend
export TEST_DATABASE_URL="postgresql://granatracker:granatracker@localhost:5432/granatracker?sslmode=disable"
go test ./...
```

On Windows PowerShell:

```
$env:TEST_DATABASE_URL="postgresql://granatracker:granatracker@localhost:5432/granatracker?sslmode=disable"
go test ./...
```

The integration tests `TRUNCATE` every table between runs, so they will wipe your local dev data. If you want to keep it, spin up a second database (e.g. `createdb granatracker_test`) and point `TEST_DATABASE_URL` at that instead.

### Coverage

```
cd backend
go test ./... -cover
```

### CI

Every push and pull request runs `go build`, `go vet`, and `go test ./... -race -count=1` against a fresh Postgres 16 service container. Workflow file: `.github/workflows/ci.yml`.

## Project layout

```
backend/         Go API
  cmd/server/    entry point
  internal/      handlers, services, middleware
  db/            migrations and queries
frontend/        Next.js app
  src/app/       routes
  src/lib/       api client, theme
  src/context/   auth context
docker-compose.yml
```

## Current state

Working: registration, login, JWT refresh, user profile, portfolio CRUD, `/healthz` and `/readyz` probes, automated tests (unit + integration), GitHub Actions CI.

Still to do: investment CRUD, dashboard summary, live price fetching, charts, sqlc migration, Clean Architecture layering, OpenAPI docs.
