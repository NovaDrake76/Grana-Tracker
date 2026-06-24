# ADR-0002: Use sqlc instead of an ORM

## Status

Accepted.

## Context

The backend needs a data-access layer over PostgreSQL. The DIM0518 course syllabus mandates **sqlc** for the persistence layer, but we still evaluated the alternative most Go web developers reach for first: **GORM**.

- GORM: reflection-based, runtime query building, familiar `db.Where(...).Find(&x)` ergonomics, but no compile-time guarantees and a well-known tendency toward N+1 queries.
- sqlc: hand-written SQL in `db/queries/*.sql`, code-generated typed Go in `db/sqlc/`, queries are validated against the schema at generation time.

Course requirement aside, the data model is small (users, accounts, categories, transactions, budgets) and the queries are not dynamic — exactly the shape sqlc handles well.

## Decision

Use **sqlc exclusively** for all database access:

- SQL queries live in `backend/db/queries/*.sql` with `-- name: ... :one|:many|:exec` annotations.
- Generated code lives in `backend/db/sqlc/` and is checked in.
- HTTP and GraphQL handlers call `queries.MethodName(ctx, params)` directly; no repository wrapper unless behavior justifies one.

## Consequences

- Type safety: query signatures change at compile time when the schema changes, breaking the build instead of production.
- No hidden N+1 — every round trip is visible as a `.sql` file.
- Cost: every handler change that touches new columns requires an SQL edit plus `sqlc generate`. Devs unfamiliar with SQL pay a learning tax.
