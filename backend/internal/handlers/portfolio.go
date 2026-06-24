package handlers

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/NovaDrake76/grana-tracker/backend/db/sqlc"
	"github.com/NovaDrake76/grana-tracker/backend/internal/middleware"
	"github.com/NovaDrake76/grana-tracker/backend/internal/snapshots"
)

type PortfolioHandler struct {
	Queries   *sqlc.Queries
	Snapshots *snapshots.Service
}

// NewPortfolioHandler keeps the existing signature for tests that build the
// handler without a snapshot service. Use SetSnapshots to attach the cron
// component after construction (or pass it via NewPortfolioHandlerWithSnapshots
// from server wiring).
func NewPortfolioHandler(queries *sqlc.Queries) *PortfolioHandler {
	return &PortfolioHandler{Queries: queries}
}

// NewPortfolioHandlerWithSnapshots is what server.NewRouter calls so the
// GetHistory endpoint can lazily trigger a snapshot when the chart is empty.
func NewPortfolioHandlerWithSnapshots(queries *sqlc.Queries, snaps *snapshots.Service) *PortfolioHandler {
	return &PortfolioHandler{Queries: queries, Snapshots: snaps}
}

type portfolioResponse struct {
	ID          string  `json:"id"`
	UserID      string  `json:"user_id"`
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	Description *string `json:"description"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

// portfolioWithInvestmentsResponse is the nested 1:N payload returned by Get.
type portfolioWithInvestmentsResponse struct {
	portfolioResponse
	Investments []investmentResponse `json:"investments"`
}

type createPortfolioRequest struct {
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	Description *string `json:"description"`
}

type updatePortfolioRequest struct {
	Name        *string `json:"name"`
	Type        *string `json:"type"`
	Description *string `json:"description"`
}

func toPortfolioResponse(p sqlc.Portfolio) portfolioResponse {
	return portfolioResponse{
		ID:          uuidStr(p.ID),
		UserID:      uuidStr(p.UserID),
		Name:        p.Name,
		Type:        p.Type,
		Description: textPtr(p.Description),
		CreatedAt:   tsString(p.CreatedAt),
		UpdatedAt:   tsString(p.UpdatedAt),
	}
}

// returns every portfolio owned by the caller, newest first.
func (h *PortfolioHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	uid, err := parseUUID(userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid user id", "AUTH_ERROR")
		return
	}

	rows, err := h.Queries.ListPortfoliosByUser(r.Context(), pgUUID(uid))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list portfolios", "INTERNAL_ERROR")
		return
	}

	portfolios := make([]portfolioResponse, 0, len(rows))
	for _, p := range rows {
		portfolios = append(portfolios, toPortfolioResponse(p))
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": portfolios,
	})
}

// creates a portfolio for the caller; type must be "real" or "simulated".
func (h *PortfolioHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	uid, err := parseUUID(userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid user id", "AUTH_ERROR")
		return
	}

	var req createPortfolioRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "VALIDATION_ERROR")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required", "VALIDATION_ERROR")
		return
	}
	if req.Type != "real" && req.Type != "simulated" {
		writeError(w, http.StatusBadRequest, "type must be 'real' or 'simulated'", "VALIDATION_ERROR")
		return
	}

	created, err := h.Queries.CreatePortfolio(r.Context(), sqlc.CreatePortfolioParams{
		UserID:      pgUUID(uid),
		Name:        req.Name,
		Type:        req.Type,
		Description: pgTextPtr(req.Description),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create portfolio", "INTERNAL_ERROR")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"data":    toPortfolioResponse(created),
		"message": "portfolio created successfully",
	})
}

// fetches a single portfolio with its nested investments (the 1:N payload).
// returns 403 if it doesn't belong to the caller.
func (h *PortfolioHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	portfolioID := chi.URLParam(r, "id")

	pid, err := uuid.Parse(portfolioID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid portfolio id", "VALIDATION_ERROR")
		return
	}

	// gets portfolio
	p, err := h.Queries.GetPortfolioByID(r.Context(), pgUUID(pid))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "portfolio not found", "NOT_FOUND")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load portfolio", "INTERNAL_ERROR")
		return
	}

	//checks access
	if uuidStr(p.UserID) != userID {
		writeError(w, http.StatusForbidden, "access denied", "FORBIDDEN")
		return
	}

	//show investments from that portfolio
	invs, err := h.Queries.ListInvestmentsByPortfolio(r.Context(), p.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load investments", "INTERNAL_ERROR")
		return
	}

	nested := make([]investmentResponse, 0, len(invs))
	for _, i := range invs {
		nested = append(nested, toInvestmentResponse(i))
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": portfolioWithInvestmentsResponse{
			portfolioResponse: toPortfolioResponse(p),
			Investments:       nested,
		},
	})
}

// patches any subset of name/type/description after checking ownership.
func (h *PortfolioHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	portfolioID := chi.URLParam(r, "id")

	pid, err := uuid.Parse(portfolioID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid portfolio id", "VALIDATION_ERROR")
		return
	}

	current, err := h.Queries.GetPortfolioByID(r.Context(), pgUUID(pid))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "portfolio not found", "NOT_FOUND")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load portfolio", "INTERNAL_ERROR")
		return
	}
	if uuidStr(current.UserID) != userID {
		writeError(w, http.StatusForbidden, "access denied", "FORBIDDEN")
		return
	}

	var req updatePortfolioRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "VALIDATION_ERROR")
		return
	}

	name := current.Name
	pType := current.Type
	desc := current.Description
	if req.Name != nil {
		name = *req.Name
	}
	if req.Type != nil {
		if *req.Type != "real" && *req.Type != "simulated" {
			writeError(w, http.StatusBadRequest, "type must be 'real' or 'simulated'", "VALIDATION_ERROR")
			return
		}
		pType = *req.Type
	}
	if req.Description != nil {
		desc = pgTextPtr(req.Description)
	}

	updated, err := h.Queries.UpdatePortfolio(r.Context(), sqlc.UpdatePortfolioParams{
		ID:          pgUUID(pid),
		Name:        name,
		Type:        pType,
		Description: desc,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update portfolio", "INTERNAL_ERROR")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":    toPortfolioResponse(updated),
		"message": "portfolio updated successfully",
	})
}

// deletes a portfolio (and cascades to its investments) after checking ownership.
func (h *PortfolioHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	portfolioID := chi.URLParam(r, "id")

	pid, err := uuid.Parse(portfolioID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid portfolio id", "VALIDATION_ERROR")
		return
	}

	current, err := h.Queries.GetPortfolioByID(r.Context(), pgUUID(pid))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "portfolio not found", "NOT_FOUND")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load portfolio", "INTERNAL_ERROR")
		return
	}
	if uuidStr(current.UserID) != userID {
		writeError(w, http.StatusForbidden, "access denied", "FORBIDDEN")
		return
	}

	if err := h.Queries.DeletePortfolio(r.Context(), pgUUID(pid)); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete portfolio", "INTERNAL_ERROR")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "portfolio deleted successfully",
	})
}

// historyPoint is one (date, value) pair returned in the chart payload.
type historyPoint struct {
	Date  string `json:"date"`
	Value string `json:"value"`
}

// historyResponse is the envelope under {"data": ...} for GetHistory.
type historyResponse struct {
	PortfolioID string         `json:"portfolio_id"`
	Currency    string         `json:"currency"`
	Period      string         `json:"period"`
	Points      []historyPoint `json:"points"`
}

// historyPeriodDays maps the period query string to a window length in days.
// Keeping it as a small enum guards against accidentally letting a caller
// request "5y" and torching the database with a giant scan.
var historyPeriodDays = map[string]int{
	"7d":  7,
	"30d": 30,
	"90d": 90,
}

// returns the daily total_value series for a portfolio over the requested
// window (default 30d). When the snapshot table is empty AND the portfolio has
// at least one investment, we trigger an on-demand snapshot so the chart is
// never blank in the demo — the daily cron normally keeps it warm.
// route: GET /api/portfolios/{id}/history?period=30d
func (h *PortfolioHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	portfolioID := chi.URLParam(r, "id")

	pid, err := uuid.Parse(portfolioID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid portfolio id", "VALIDATION_ERROR")
		return
	}

	// ownership check — same pattern as Get.
	p, err := h.Queries.GetPortfolioByID(r.Context(), pgUUID(pid))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "portfolio not found", "NOT_FOUND")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load portfolio", "INTERNAL_ERROR")
		return
	}
	if uuidStr(p.UserID) != userID {
		writeError(w, http.StatusForbidden, "access denied", "FORBIDDEN")
		return
	}

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "30d"
	}
	days, ok := historyPeriodDays[period]
	if !ok {
		writeError(w, http.StatusBadRequest, "period must be one of 7d, 30d, 90d", "VALIDATION_ERROR")
		return
	}

	// UTC date math keeps the cutoff consistent with the snapshot writer,
	// which also truncates to UTC midnight.
	fromDate := time.Now().UTC().Truncate(24 * time.Hour).AddDate(0, 0, -days)

	rows, err := h.Queries.ListPortfolioSnapshotsInPeriod(r.Context(), sqlc.ListPortfolioSnapshotsInPeriodParams{
		PortfolioID:  pgUUID(pid),
		SnapshotDate: pgtype.Date{Time: fromDate, Valid: true},
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load history", "INTERNAL_ERROR")
		return
	}

	// Lazy snapshot: if the chart is empty but the portfolio has investments,
	// run a one-off snapshot so the demo (and freshly-onboarded users) see at
	// least one data point. Errors are logged, not returned — better to serve
	// an empty chart than a 500 in this edge case.
	if len(rows) == 0 && h.Snapshots != nil {
		invs, ierr := h.Queries.ListInvestmentsByPortfolio(r.Context(), pgUUID(pid))
		if ierr == nil && len(invs) > 0 {
			if serr := h.Snapshots.SnapshotPortfolio(r.Context(), pid); serr != nil {
				log.Printf("history: lazy snapshot for %s failed: %v", pid, serr)
			} else {
				rows, err = h.Queries.ListPortfolioSnapshotsInPeriod(r.Context(), sqlc.ListPortfolioSnapshotsInPeriodParams{
					PortfolioID:  pgUUID(pid),
					SnapshotDate: pgtype.Date{Time: fromDate, Valid: true},
				})
				if err != nil {
					writeError(w, http.StatusInternalServerError, "failed to load history", "INTERNAL_ERROR")
					return
				}
			}
		}
	}

	points := make([]historyPoint, 0, len(rows))
	currency := "BRL"
	for _, row := range rows {
		points = append(points, historyPoint{
			Date:  dateStr(row.SnapshotDate),
			Value: numericStr(row.TotalValue),
		})
		if row.Currency != "" {
			currency = row.Currency
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": historyResponse{
			PortfolioID: uuidStr(p.ID),
			Currency:    currency,
			Period:      period,
			Points:      points,
		},
	})
}
