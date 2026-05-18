package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/NovaDrake76/grana-tracker/backend/db/sqlc"
	"github.com/NovaDrake76/grana-tracker/backend/internal/middleware"
)

type PortfolioHandler struct {
	Queries *sqlc.Queries
}

func NewPortfolioHandler(queries *sqlc.Queries) *PortfolioHandler {
	return &PortfolioHandler{Queries: queries}
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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
