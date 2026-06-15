package handlers

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/NovaDrake76/grana-tracker/backend/db/sqlc"
	"github.com/NovaDrake76/grana-tracker/backend/internal/middleware"
)

type UserHandler struct {
	Queries *sqlc.Queries
}

func NewUserHandler(queries *sqlc.Queries) *UserHandler {
	return &UserHandler{Queries: queries}
}

type userResponse struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Email             string `json:"email"`
	PreferredCurrency string `json:"preferred_currency"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
}

type updateUserRequest struct {
	Name              *string `json:"name"`
	PreferredCurrency *string `json:"preferred_currency"`
}

// returns the profile of the user identified by the JWT.
func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	uid, err := parseUUID(userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid user id", "AUTH_ERROR")
		return
	}

	row, err := h.Queries.GetUserByID(r.Context(), pgUUID(uid))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "user not found", "NOT_FOUND")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load user", "INTERNAL_ERROR")
		return
	}

	currency := ""
	if row.PreferredCurrency.Valid {
		currency = row.PreferredCurrency.String
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": userResponse{
			ID:                uuidStr(row.ID),
			Name:              row.Name,
			Email:             row.Email,
			PreferredCurrency: currency,
			CreatedAt:         tsString(row.CreatedAt),
			UpdatedAt:         tsString(row.UpdatedAt),
		},
	})
}

// patches name and/or preferred_currency on the authenticated user.
func (h *UserHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	uid, err := parseUUID(userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid user id", "AUTH_ERROR")
		return
	}

	var req updateUserRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "VALIDATION_ERROR")
		return
	}

	current, err := h.Queries.GetUserByID(r.Context(), pgUUID(uid))
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found", "NOT_FOUND")
		return
	}

	name := current.Name
	if req.Name != nil {
		name = *req.Name
	}

	currency := current.PreferredCurrency
	if req.PreferredCurrency != nil {
		currency = pgText(*req.PreferredCurrency)
	}

	updated, err := h.Queries.UpdateUser(r.Context(), sqlc.UpdateUserParams{
		ID:                pgUUID(uid),
		Name:              name,
		PreferredCurrency: currency,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update user", "INTERNAL_ERROR")
		return
	}

	cur := ""
	if updated.PreferredCurrency.Valid {
		cur = updated.PreferredCurrency.String
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": userResponse{
			ID:                uuidStr(updated.ID),
			Name:              updated.Name,
			Email:             updated.Email,
			PreferredCurrency: cur,
			CreatedAt:         tsString(updated.CreatedAt),
			UpdatedAt:         tsString(updated.UpdatedAt),
		},
		"message": "user updated successfully",
	})
}
