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

Hoje o backend tem ~33 testes verdes divididos em duas camadas:

- **Unit tests** (no DB): `internal/services`, `internal/middleware`, and handler helpers. Always run.
- **Integration tests** (real Postgres): `internal/handlers` — hit the real HTTP + SQL stack. Run only when `TEST_DATABASE_URL` is set; skip cleanly otherwise.

O CI (`.github/workflows/ci.yml`) roda `go build`, `go vet` e `go test ./... -race -count=1` contra um Postgres 16 fresco a cada push/PR.

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

Sprint 2 + Sprint 3 finalizados. Entregue nesta versão:

- Auth completa: JWT HS256 (signing method pinned), bcrypt (DefaultCost), refresh token com rotação e *theft response* (família invalidada em reuso), rate-limit por IP (10 req/min) em `/api/auth`, security response headers.
- CRUD completo de users, portfolios e investments, todo migrado para **sqlc** (queries tipadas e parametrizadas).
- Endpoint `GET /api/portfolios/{id}` devolve o portfólio com os investimentos aninhados (1:N).
- 33+ testes verdes rodando no CI (unit + integração com Postgres real).
- Cobertura OWASP Top 10: A02 (criptografia), A03 (injeção), A05 (misconfiguration / headers / CORS) e A07 (auth & identity).
- `/healthz` e `/readyz` probes.
- OpenAPI 3.0.3 publicada — ver [API Reference](#api-reference).

Ainda em aberto (fora do escopo desta entrega): dashboard summary, live price fetching e charts.

## API Reference

Toda a documentação da API vive em `docs/`:

- [`docs/openapi.yaml`](docs/openapi.yaml) — spec OpenAPI 3.0.3
- [`docs/GranaTracker.postman_collection.json`](docs/GranaTracker.postman_collection.json) — coleção Postman (importável)
- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) — diagrama do sistema e decisões técnicas
- [`docs/ERD.md`](docs/ERD.md) — modelo de dados

## Segurança

Controles aplicados no backend (cobrindo OWASP A02/A03/A05/A07):

- Senhas com **bcrypt** (`DefaultCost`).
- **JWT HS256** com signing method *pinned* na validação (impede `alg: none` e troca de algoritmo).
- **Refresh token com rotação**: cada refresh invalida o anterior; reuso detectado invalida toda a família (*theft response*).
- **Rate-limit por IP**: 10 req/min em `/api/auth/*` para mitigar brute-force.
- **Security response headers** aplicados globalmente (`X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`, etc.).
- **SQL 100% parametrizado** via sqlc — nenhuma string concatenada.
- **CORS escopado por env** (`FRONTEND_URL`), sem `*` em produção.
- **Ownership / IDOR check** em todas as rotas `/{id}`: o `user_id` do JWT precisa bater com o dono do recurso.
- **401 genérico** em login para evitar enumeração de usuário ("invalid credentials" sempre, sem distinguir email vs senha).
- **`DisallowUnknownFields`** em todos os decoders JSON, recusando payloads com campos extras.

## Time

DIM0547 — Desenvolvimento de Sistemas Web II:

- Breno Jalmir
- Nathan Araújo
- Heitor Vinícius
