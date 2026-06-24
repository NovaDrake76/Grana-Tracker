# ADR-0001: Record architecture decisions

## Status

Accepted.

## Context

Grana Tracker is delivered for two distinct disciplines (DIM0547 and DIM0518) and is reviewed by different grading panels. Several architectural choices (sqlc, GraphQL library, deploy target, auth flow) were made early and are not obvious from the code alone. We need a lightweight place to capture *why* each choice was made so reviewers and future contributors do not have to reverse-engineer intent from diffs.

## Decision

We will record architecture decisions as MADR (Markdown ADR) files:

- One Markdown file per decision, stored under `docs/adr/`.
- Files are numbered sequentially with a 4-digit prefix: `NNNN-short-slug.md`.
- Each ADR has four sections: **Status**, **Context**, **Decision**, **Consequences**.
- ADRs are immutable once `Accepted`. To change a decision, write a new ADR that supersedes the old one and update the old ADR's status.

## Consequences

- Reviewers and graders can read the rationale directly on GitHub with zero tooling.
- ADRs render nicely in PR descriptions and link easily from `README.md` / `ARCHITECTURE.md`.
- Discipline of writing the ADR forces us to commit to a single approach instead of leaving competing patterns in the codebase.
- Cost: a small amount of writing overhead per non-trivial decision, and the team must remember to actually open the file.
