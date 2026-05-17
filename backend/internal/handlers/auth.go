package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/NovaDrake76/grana-tracker/backend/db/sqlc"
	"github.com/NovaDrake76/grana-tracker/backend/internal/services"
)

type AuthHandler struct {
	Queries *sqlc.Queries
	Secret  string
}

func NewAuthHandler(queries *sqlc.Queries, secret string) *AuthHandler {
	return &AuthHandler{Queries: queries, Secret: secret}
}

type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// validates the request, hashes the password, inserts the user, and returns a token pair.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "VALIDATION_ERROR")
		return
	}

	if req.Name == "" || req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "name, email, and password are required", "VALIDATION_ERROR")
		return
	}

	if len(req.Password) < 6 {
		writeError(w, http.StatusBadRequest, "password must be at least 6 characters", "VALIDATION_ERROR")
		return
	}

	hash, err := services.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password", "INTERNAL_ERROR")
		return
	}

	created, err := h.Queries.CreateUser(r.Context(), sqlc.CreateUserParams{
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: hash,
	})
	if err != nil {
		if isDuplicateKeyError(err) {
			writeError(w, http.StatusConflict, "email already registered", "DUPLICATE_ERROR")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create user", "INTERNAL_ERROR")
		return
	}

	tokens, err := services.GenerateTokenPair(uuidFromBytes(created.ID.Bytes), h.Secret)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate tokens", "INTERNAL_ERROR")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"data":    tokens,
		"message": "user registered successfully",
	})
}

// checks email + password and returns a fresh token pair on success.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "VALIDATION_ERROR")
		return
	}

	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required", "VALIDATION_ERROR")
		return
	}

	user, err := h.Queries.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusUnauthorized, "invalid email or password", "AUTH_ERROR")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to look up user", "INTERNAL_ERROR")
		return
	}

	if !services.CheckPassword(req.Password, user.PasswordHash) {
		writeError(w, http.StatusUnauthorized, "invalid email or password", "AUTH_ERROR")
		return
	}

	tokens, err := services.GenerateTokenPair(uuidFromBytes(user.ID.Bytes), h.Secret)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate tokens", "INTERNAL_ERROR")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": tokens,
	})
}

// exchanges a valid refresh token for a new access/refresh pair.
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "VALIDATION_ERROR")
		return
	}

	if req.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "refresh_token is required", "VALIDATION_ERROR")
		return
	}

	claims, err := services.ValidateToken(req.RefreshToken, h.Secret)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid refresh token", "AUTH_ERROR")
		return
	}

	uid, err := parseUUID(claims.UserID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid token claims", "AUTH_ERROR")
		return
	}

	tokens, err := services.GenerateTokenPair(uid, h.Secret)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate tokens", "INTERNAL_ERROR")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": tokens,
	})
}
