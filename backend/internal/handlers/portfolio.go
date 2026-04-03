package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/NovaDrake76/grana-tracker/backend/internal/middleware"
)

type PortfolioHandler struct {
	DB *pgxpool.Pool
}

func NewPortfolioHandler(db *pgxpool.Pool) *PortfolioHandler {
	return &PortfolioHandler{DB: db}
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

func (h *PortfolioHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	uid, err := parseUUID(userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid user id", "AUTH_ERROR")
		return
	}

	rows, err := h.DB.Query(r.Context(),
		"SELECT id, user_id, name, type, description, created_at, updated_at FROM portfolios WHERE user_id = $1 ORDER BY created_at DESC",
		uid,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list portfolios", "INTERNAL_ERROR")
		return
	}
	defer rows.Close()

	portfolios := []portfolioResponse{}
	for rows.Next() {
		var p portfolioResponse
		var desc *string
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.Type, &desc, &createdAt, &updatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to scan portfolio", "INTERNAL_ERROR")
			return
		}
		p.Description = desc
		p.CreatedAt = createdAt.Format(time.RFC3339)
		p.UpdatedAt = updatedAt.Format(time.RFC3339)
		portfolios = append(portfolios, p)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": portfolios,
	})
}

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

	var p portfolioResponse
	var desc *string
	var createdAt, updatedAt time.Time
	err = h.DB.QueryRow(r.Context(),
		"INSERT INTO portfolios (user_id, name, type, description) VALUES ($1, $2, $3, $4) RETURNING id, user_id, name, type, description, created_at, updated_at",
		uid, req.Name, req.Type, req.Description,
	).Scan(&p.ID, &p.UserID, &p.Name, &p.Type, &desc, &createdAt, &updatedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create portfolio", "INTERNAL_ERROR")
		return
	}
	p.Description = desc
	p.CreatedAt = createdAt.Format(time.RFC3339)
	p.UpdatedAt = updatedAt.Format(time.RFC3339)

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"data":    p,
		"message": "portfolio created successfully",
	})
}

func (h *PortfolioHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	portfolioID := chi.URLParam(r, "id")

	pid, err := uuid.Parse(portfolioID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid portfolio id", "VALIDATION_ERROR")
		return
	}

	var p portfolioResponse
	var desc *string
	var createdAt, updatedAt time.Time
	err = h.DB.QueryRow(r.Context(),
		"SELECT id, user_id, name, type, description, created_at, updated_at FROM portfolios WHERE id = $1",
		pid,
	).Scan(&p.ID, &p.UserID, &p.Name, &p.Type, &desc, &createdAt, &updatedAt)
	if err != nil {
		writeError(w, http.StatusNotFound, "portfolio not found", "NOT_FOUND")
		return
	}

	if p.UserID != userID {
		writeError(w, http.StatusForbidden, "access denied", "FORBIDDEN")
		return
	}

	p.Description = desc
	p.CreatedAt = createdAt.Format(time.RFC3339)
	p.UpdatedAt = updatedAt.Format(time.RFC3339)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": p,
	})
}

func (h *PortfolioHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	portfolioID := chi.URLParam(r, "id")

	pid, err := uuid.Parse(portfolioID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid portfolio id", "VALIDATION_ERROR")
		return
	}

	// verify ownership
	var ownerID string
	err = h.DB.QueryRow(r.Context(), "SELECT user_id FROM portfolios WHERE id = $1", pid).Scan(&ownerID)
	if err != nil {
		writeError(w, http.StatusNotFound, "portfolio not found", "NOT_FOUND")
		return
	}
	if ownerID != userID {
		writeError(w, http.StatusForbidden, "access denied", "FORBIDDEN")
		return
	}

	var req updatePortfolioRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "VALIDATION_ERROR")
		return
	}

	// get current values
	var currentName, currentType string
	var currentDesc *string
	h.DB.QueryRow(r.Context(),
		"SELECT name, type, description FROM portfolios WHERE id = $1", pid,
	).Scan(&currentName, &currentType, &currentDesc)

	name := currentName
	pType := currentType
	desc := currentDesc
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
		desc = req.Description
	}

	var p portfolioResponse
	var retDesc *string
	var createdAt, updatedAt time.Time
	err = h.DB.QueryRow(r.Context(),
		"UPDATE portfolios SET name = $2, type = $3, description = $4, updated_at = NOW() WHERE id = $1 RETURNING id, user_id, name, type, description, created_at, updated_at",
		pid, name, pType, desc,
	).Scan(&p.ID, &p.UserID, &p.Name, &p.Type, &retDesc, &createdAt, &updatedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update portfolio", "INTERNAL_ERROR")
		return
	}
	p.Description = retDesc
	p.CreatedAt = createdAt.Format(time.RFC3339)
	p.UpdatedAt = updatedAt.Format(time.RFC3339)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":    p,
		"message": "portfolio updated successfully",
	})
}

func (h *PortfolioHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	portfolioID := chi.URLParam(r, "id")

	pid, err := uuid.Parse(portfolioID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid portfolio id", "VALIDATION_ERROR")
		return
	}

	// verify ownership
	var ownerID string
	err = h.DB.QueryRow(r.Context(), "SELECT user_id FROM portfolios WHERE id = $1", pid).Scan(&ownerID)
	if err != nil {
		writeError(w, http.StatusNotFound, "portfolio not found", "NOT_FOUND")
		return
	}
	if ownerID != userID {
		writeError(w, http.StatusForbidden, "access denied", "FORBIDDEN")
		return
	}

	_, err = h.DB.Exec(r.Context(), "DELETE FROM portfolios WHERE id = $1", pid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete portfolio", "INTERNAL_ERROR")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "portfolio deleted successfully",
	})
}
