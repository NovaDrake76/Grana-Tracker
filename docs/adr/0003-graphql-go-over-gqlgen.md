# ADR-0003: Use graphql-go instead of gqlgen for the bonus GraphQL endpoint

## Status

Accepted.

## Context

DIM0518 offers a bonus for exposing an additional API surface beyond REST: either **gRPC** or **GraphQL**. We chose GraphQL because the frontend already speaks JSON over HTTP and GraphQL gives reviewers a tangible second-protocol demo with minimal client work.

Two Go libraries dominate the ecosystem:

- **gqlgen** (`99designs/gqlgen`): schema-first. You write `.graphql` files, run `go generate`, and implement the produced resolver interfaces. Idiomatic for large schemas but adds a second codegen toolchain (`gqlgen.yml`, generated `models_gen.go`, `generated.go`).
- **graphql-go** (`graphql-go/graphql`): programmatic. You declare types, fields and resolvers as Go values. More verbose per field, but zero configuration and zero generated files.

The Grana Tracker GraphQL surface is small — a handful of read queries over transactions, budgets and dashboard summaries — so schema size is not the deciding factor.

## Decision

Use **graphql-go** for the GraphQL endpoint, mounted under `/graphql` and wired in `backend/internal/graphql/`.

## Consequences

- Only one codegen toolchain in the build (`sqlc`); no `gqlgen generate` step and no `gqlgen.yml` to keep in sync.
- The schema lives in Go, so refactors flow through normal compiler errors instead of regeneration.
- Cost: schema definitions are noisier than `.graphql` files, and we lose gqlgen's free schema-to-Go binding. Acceptable at this scale.
