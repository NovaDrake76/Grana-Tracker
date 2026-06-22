# syntax=docker/dockerfile:1
# Build stage — go.mod declares go 1.23, so the toolchain must match.
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy module manifests first so `go mod download` is cached when only source
# files change. go.sum is required for reproducible builds.
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the backend source. .dockerignore (if present) keeps the
# context lean. CGO is disabled so the binary is fully static and can run on
# the scratch/alpine runtime image without libc.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# Runtime stage — alpine gives us ca-certificates (needed for HTTPS upstreams
# like CoinGecko / Alpha Vantage) and a real shell for debugging on Render.
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata \
    && addgroup -S app && adduser -S -G app app

WORKDIR /app

# Binary + runtime SQL migrations. The migration runner reads .sql files from
# db/schema at boot, so they MUST exist in the final image (not just the
# builder stage).
COPY --from=builder /out/server ./server
COPY --from=builder /app/db/schema ./db/schema

# Drop root. Render is happy with root but running as a non-privileged user
# is basic hygiene and matches OWASP container guidance.
USER app

# Documentation only — Render maps the public URL to whatever $PORT the
# process listens on (cmd/server/main.go honors PORT, defaulting to 8080).
EXPOSE 8080

CMD ["./server"]
