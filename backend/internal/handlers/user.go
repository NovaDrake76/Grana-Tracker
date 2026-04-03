package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/NovaDrake76/grana-tracker/backend/internal/middleware"
)

type UserHandler struct {
	DB *pgxpool.Pool
}

func NewUserHandler(db *pgxpool.Pool) *UserHandler {
	return &UserHandler{DB: db}
}

type userResponse struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	Email             string  `json:"email"`
	PreferredCurrency string  `json:"preferred_currency"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
}

type updateUserRequest struct {
	Name              *string `json:"name"`
	PreferredCurrency *string `json:"preferred_currency"`
}

func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	uid, err := parseUUID(userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid user id", "AUTH_ERROR")
		return
	}

	var id uuid.UUID
	var uName, email, currency string
	var createdAt, updatedAt time.Time
	err = h.DB.QueryRow(r.Context(),
		"SELECT id, name, email, preferred_currency, created_at, updated_at FROM users WHERE id = $1",
		uid,
	).Scan(&id, &uName, &email, &currency, &createdAt, &updatedAt)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found", "NOT_FOUND")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": userResponse{
			ID:                id.String(),
			Name:              uName,
			Email:             email,
			PreferredCurrency: currency,
			CreatedAt:         createdAt.Format(time.RFC3339),
			UpdatedAt:         updatedAt.Format(time.RFC3339),
		},
	})
}

func (h *UserHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	uid, err := parseUUID(userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid user id", "AUTH_ERROR")
		return
	}

	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "VALIDATION_ERROR")
		return
	}

	// get current values
	var currentName, currentCurrency string
	err = h.DB.QueryRow(r.Context(),
		"SELECT name, preferred_currency FROM users WHERE id = $1", uid,
	).Scan(&currentName, &currentCurrency)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found", "NOT_FOUND")
		return
	}

	name := currentName
	currency := currentCurrency
	if req.Name != nil {
		name = *req.Name
	}
	if req.PreferredCurrency != nil {
		currency = *req.PreferredCurrency
	}

	var retID uuid.UUID
	var retName, retEmail, retCurrency string
	var retCreatedAt, retUpdatedAt time.Time
	err = h.DB.QueryRow(r.Context(),
		"UPDATE users SET name = $2, preferred_currency = $3, updated_at = NOW() WHERE id = $1 RETURNING id, name, email, preferred_currency, created_at, updated_at",
		uid, name, currency,
	).Scan(&retID, &retName, &retEmail, &retCurrency, &retCreatedAt, &retUpdatedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update user", "INTERNAL_ERROR")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": userResponse{
			ID:                retID.String(),
			Name:              retName,
			Email:             retEmail,
			PreferredCurrency: retCurrency,
			CreatedAt:         retCreatedAt.Format(time.RFC3339),
			UpdatedAt:         retUpdatedAt.Format(time.RFC3339),
		},
		"message": "user updated successfully",
	})
}
