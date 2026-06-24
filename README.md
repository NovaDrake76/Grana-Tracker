# Grana Tracker

Tracker de investimentos onde o usuário cadastra posições reais e simuladas (ações, cripto, ETFs, índices) e visualiza todas juntas. Carteiras reais e simuladas são tratadas da mesma forma; a diferença é apenas uma flag no nível do portfólio.

Projeto da disciplina DIM0547 (Desenvolvimento de Sistemas Web II com Go), 2026.1.

## Demo

- Frontend: https://grana-tracker.vercel.app
- Backend (health): https://grana-tracker-api.onrender.com/healthz

> O backend roda no plano free do Render — o primeiro request após inatividade pode levar ~30s para acordar o container.

## Stack

| Camada     | Tecnologias                                                       |
| ---------- | ----------------------------------------------------------------- |
| Backend    | Go 1.23, Chi, sqlc, pgx, JWT HS256                                |
| Banco      | PostgreSQL 16                                                     |
| Frontend   | Next.js 16, React 19, Chakra UI v3, Recharts                      |
| Deploy     | Render (API + Postgres) + Vercel (frontend)                       |

## Requisitos

- Go 1.23+
- Node 20+
- Docker (para o Postgres local)

## Setup

Clone o repo e instale as deps do frontend:

```
cd frontend
npm install
cd ..
npm install
```

Crie `backend/.env` a partir do exemplo:

```
cp backend/.env.example backend/.env
```

Preencha `DATABASE_URL` e `JWT_SECRET` no mínimo. Config local que funciona:

```
DATABASE_URL=postgresql://granatracker:granatracker@localhost:5432/granatracker?sslmode=disable
JWT_SECRET=change-me
PORT=8080
FRONTEND_URL=http://localhost:3000
```

E `frontend/.env.local`:

```
NEXT_PUBLIC_API_URL=http://localhost:8080/api
```

## Rodando local

Na raiz do repo:

```
npm run dev
```

Isso sobe o Postgres no Docker, a API Go em :8080 e o Next.js em :3000.

Partes isoladas, se precisar:

```
npm run db         # postgres only
npm run db:stop    # stop postgres
npm run backend    # go api
npm run frontend   # next dev
```

Na primeira execução, a API roda a migration em `backend/db/migrations/001_init.up.sql` automaticamente.

## Testes

O backend tem 65 testes verdes divididos em duas camadas:

- **Unit tests** (sem DB): `internal/services`, `internal/middleware` e helpers de handler. Sempre rodam.
- **Integration tests** (Postgres real): `internal/handlers` — atingem a stack HTTP + SQL completa. Rodam apenas quando `TEST_DATABASE_URL` está definida; pulam limpo caso contrário.

O CI (`.github/workflows/ci.yml`) executa `go build`, `go vet` e `go test ./... -race -count=1` contra um Postgres 16 fresco a cada push/PR.

### Rodar só os unit tests

```
cd backend
go test ./...
```

Os integration tests imprimem `SKIP: TEST_DATABASE_URL not set` — comportamento esperado.

### Rodar tudo (unit + integração)

```
npm run db
cd backend
export TEST_DATABASE_URL="postgresql://granatracker:granatracker@localhost:5432/granatracker?sslmode=disable"
go test ./...
```

No PowerShell:

```
$env:TEST_DATABASE_URL="postgresql://granatracker:granatracker@localhost:5432/granatracker?sslmode=disable"
go test ./...
```

Os integration tests dão `TRUNCATE` nas tabelas entre runs, então vão limpar seu dev DB. Se quiser preservar, suba um segundo banco (`createdb granatracker_test`) e aponte `TEST_DATABASE_URL` para ele.

### Cobertura

```
cd backend
go test ./... -cover
```

## Estrutura do projeto

```
backend/         Go API
  cmd/server/    entry point
  internal/      handlers, services, middleware, graphql
  db/            migrations e queries (sqlc)
frontend/        Next.js app
  src/app/       rotas
  src/lib/       api client, theme
  src/context/   auth context
docker-compose.yml
render.yaml      blueprint de deploy
```

## Estado atual

Sprints 1 a 4 finalizados. Nesta versão:

- **Auth completa**: JWT HS256 (signing method *pinned*), bcrypt (`DefaultCost`), refresh token com rotação e *theft response* (família invalidada em reuso), rate-limit por IP (10 req/min) em `/api/auth`, endpoint de logout que revoga a família atual, security response headers.
- **CRUD completo** de users, portfolios e investments, todo migrado para **sqlc** (queries tipadas e parametrizadas).
- **Endpoint aninhado 1:N**: `GET /api/portfolios/{id}` devolve o portfólio com os investimentos aninhados.
- **Catálogo de ativos com autocomplete** no frontend e refresh diário de preços no backend.
- **Dashboard** com sumário (valor total, custo, P&L) e charts Recharts (donut por classe + bars por ativo).
- **OpenAPI 3.0.3** publicada e endpoint **GraphQL** (`/api/graphql`) com queries para users, portfolios e investments.
- **65 testes verdes** rodando no CI (unit + integração com Postgres real).
- **Cobertura OWASP Top 10**: A02 (criptografia), A03 (injeção), A05 (misconfiguration / headers / CORS) e A07 (auth & identity).
- **Probes** `/healthz` e `/readyz`.
- **Deploy** em produção: API + Postgres no Render, frontend no Vercel.

## API Reference

Documentação da API em `docs/`:

- [`docs/openapi.yaml`](docs/openapi.yaml) — spec OpenAPI 3.0.3
- [`docs/GranaTracker.postman_collection.json`](docs/GranaTracker.postman_collection.json) — coleção Postman (importável)
- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) — diagrama do sistema e decisões técnicas
- [`docs/ERD.md`](docs/ERD.md) — modelo de dados
- [`docs/DEPLOY.md`](docs/DEPLOY.md) — passo-a-passo de deploy (Render + Vercel)

Endpoint GraphQL alternativo: `POST /api/graphql` (mesmas regras de auth via Bearer token).

## Segurança

Controles aplicados no backend (cobrindo OWASP A02/A03/A05/A07):

- Senhas com **bcrypt** (`DefaultCost`).
- **JWT HS256** com signing method *pinned* na validação (impede `alg: none` e troca de algoritmo).
- **Refresh token com rotação**: cada refresh invalida o anterior; reuso detectado invalida toda a família (*theft response*). Logout revoga a família ativa.
- **Rate-limit por IP**: 10 req/min em `/api/auth/*` para mitigar brute-force.
- **Security response headers** aplicados globalmente (`X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`, etc.).
- **SQL 100% parametrizado** via sqlc — nenhuma string concatenada.
- **CORS multi-origem por env** (`FRONTEND_URL` aceita lista separada por vírgula), sem `*` em produção.
- **Ownership / IDOR check** em todas as rotas `/{id}`: o `user_id` do JWT precisa bater com o dono do recurso.
- **401 genérico** em login para evitar enumeração de usuário ("invalid credentials" sempre, sem distinguir email vs senha).
- **`DisallowUnknownFields`** em todos os decoders JSON, recusando payloads com campos extras.

## Equipe

DIM0547 — Desenvolvimento de Sistemas Web II:

- Breno Jalmir
- Heittor Vinicius
- Nathan Araujo
