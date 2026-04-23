package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type HealthHandler struct {
	Pool *pgxpool.Pool
}

func NewHealthHandler(pool *pgxpool.Pool) *HealthHandler {
	return &HealthHandler{Pool: pool}
}

// liveness probe — process is up and can serve HTTP.
func (h *HealthHandler) Live(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// readiness probe — we can serve traffic iff the DB is reachable.
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.Pool.Ping(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database unreachable", "NOT_READY")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
