package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/NovaDrake76/grana-tracker/backend/db/sqlc"
	"github.com/NovaDrake76/grana-tracker/backend/internal/middleware"
	"github.com/NovaDrake76/grana-tracker/backend/internal/services"
)

// dummyHash is a precomputed bcrypt hash burned at startup so the Login path
// can spend equivalent CPU on unknown emails (defeats user-enumeration via
// response-time side channel).
var dummyHash string

// emailRegex is a permissive but real-shaped email check (local@domain.tld
// with no whitespace or extra @). Cheap to evaluate, catches obvious garbage.
var emailRegex = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

func init() {
	h, err := services.HashPassword("dummy-password-for-timing-equalisation")
	if err != nil {
		log.Printf("init dummyHash: %v", err)
		return
	}
	dummyHash = h
}

type AuthHandler struct {
	Queries *sqlc.Queries
	Pool    *pgxpool.Pool
	Secret  string
}

func NewAuthHandler(queries *sqlc.Queries, pool *pgxpool.Pool, secret string) *AuthHandler {
	return &AuthHandler{Queries: queries, Pool: pool, Secret: secret}
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
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "VALIDATION_ERROR")
		return
	}

	if req.Name == "" || req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "name, email, and password are required", "VALIDATION_ERROR")
		return
	}

	if !emailRegex.MatchString(req.Email) {
		writeError(w, http.StatusBadRequest, "invalid email format", "VALIDATION_ERROR")
		return
	}

	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters", "VALIDATION_ERROR")
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

	userID := uuidFromBytes(created.ID.Bytes)
	tokens, err := h.issueAndPersistTokenPair(r.Context(), userID)
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
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "VALIDATION_ERROR")
		return
	}

	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required", "VALIDATION_ERROR")
		return
	}

	if !emailRegex.MatchString(req.Email) {
		writeError(w, http.StatusBadRequest, "invalid email format", "VALIDATION_ERROR")
		return
	}

	user, err := h.Queries.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Burn equivalent bcrypt time so a missing-user response is not
			// noticeably faster than a wrong-password response.
			_ = services.CheckPassword(req.Password, dummyHash)
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

	userID := uuidFromBytes(user.ID.Bytes)
	tokens, err := h.issueAndPersistTokenPair(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate tokens", "INTERNAL_ERROR")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": tokens,
	})
}

// exchanges a valid refresh token for a new access/refresh pair, rotating
// the stored row atomically. Reuse of a revoked token triggers a family-wide
// revocation as a theft-detection response.
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := decodeJSON(r, &req); err != nil {
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

	if _, err := parseUUID(claims.UserID); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid token claims", "AUTH_ERROR")
		return
	}

	hash := services.HashRefreshToken(req.RefreshToken)

	tx, err := h.Pool.BeginTx(r.Context(), pgx.TxOptions{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to begin transaction", "INTERNAL_ERROR")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	qtx := h.Queries.WithTx(tx)

	row, err := qtx.GetRefreshTokenByHash(r.Context(), hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusUnauthorized, "invalid refresh token", "AUTH_ERROR")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to look up refresh token", "INTERNAL_ERROR")
		return
	}

	now := time.Now()

	// Expired (DB authoritative — JWT expiry alone is not enough since the
	// refresh row could have been forcibly revoked or aged out separately).
	// Fail-closed: a NULL/invalid expires_at is treated as expired rather
	// than passing through.
	if !row.ExpiresAt.Valid || !row.ExpiresAt.Time.After(now) {
		writeError(w, http.StatusUnauthorized, "invalid refresh token", "AUTH_ERROR")
		return
	}

	// Reuse of a revoked token => somebody (legitimate user or attacker) has
	// the same token after rotation. Revoke the entire family for that user
	// and commit so the punitive revocation actually persists.
	if row.RevokedAt.Valid {
		if err := qtx.RevokeAllUserRefreshTokens(r.Context(), row.UserID); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to revoke tokens", "INTERNAL_ERROR")
			return
		}
		if err := tx.Commit(r.Context()); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to commit", "INTERNAL_ERROR")
			return
		}
		writeError(w, http.StatusUnauthorized, "invalid refresh token", "AUTH_ERROR")
		return
	}

	ownerID := uuidFromBytes(row.UserID.Bytes)
	newPair, err := services.GenerateTokenPair(ownerID, h.Secret)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate tokens", "INTERNAL_ERROR")
		return
	}

	newHash := services.HashRefreshToken(newPair.RefreshToken)
	newRow, err := qtx.CreateRefreshToken(r.Context(), sqlc.CreateRefreshTokenParams{
		UserID:    row.UserID,
		TokenHash: newHash,
		ExpiresAt: pgtype.Timestamptz{Time: services.RefreshTokenExpiry(), Valid: true},
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to persist refresh token", "INTERNAL_ERROR")
		return
	}

	if err := qtx.RevokeRefreshToken(r.Context(), sqlc.RevokeRefreshTokenParams{
		ID:         row.ID,
		ReplacedBy: pgtype.UUID{Bytes: newRow.ID.Bytes, Valid: true},
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to rotate refresh token", "INTERNAL_ERROR")
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to commit", "INTERNAL_ERROR")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": newPair,
	})
}

// Logout revokes the supplied refresh token for the authenticated user. It is
// idempotent: an unknown or already-revoked token returns 204 No Content so a
// client retrying logout does not see a spurious error. Cross-user logout
// attempts (authenticated user A submitting user B's refresh token) are
// rejected with 403 so a stolen access token cannot be used to invalidate
// somebody else's session.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "VALIDATION_ERROR")
		return
	}

	if req.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "refresh_token is required", "VALIDATION_ERROR")
		return
	}

	if _, err := services.ValidateToken(req.RefreshToken, h.Secret); err != nil {
		writeError(w, http.StatusBadRequest, "invalid refresh token", "VALIDATION_ERROR")
		return
	}

	hash := services.HashRefreshToken(req.RefreshToken)

	row, err := h.Queries.GetRefreshTokenByHash(r.Context(), hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Idempotent: token row already gone (e.g. rotated out or pruned).
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to look up refresh token", "INTERNAL_ERROR")
		return
	}

	// Cross-user check: the authenticated principal must own the row they are
	// trying to revoke. Use the middleware-derived user_id rather than the
	// refresh token's claims so a forged-but-valid-sig token can't bypass.
	authUserID := middleware.GetUserID(r.Context())
	if authUserID == "" || uuidStr(row.UserID) != authUserID {
		writeError(w, http.StatusForbidden, "forbidden", "FORBIDDEN")
		return
	}

	// Already revoked — idempotent no-op.
	if row.RevokedAt.Valid {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// ReplacedBy stays NULL (Valid: false) — this is a logout, not a rotation,
	// so there is no successor token to point at.
	if err := h.Queries.RevokeRefreshToken(r.Context(), sqlc.RevokeRefreshTokenParams{
		ID:         row.ID,
		ReplacedBy: pgtype.UUID{Valid: false},
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to revoke refresh token", "INTERNAL_ERROR")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// issueAndPersistTokenPair mints a fresh access/refresh pair for userID and
// records the refresh token's SHA-256 hash so future /refresh calls can find
// and rotate it.
func (h *AuthHandler) issueAndPersistTokenPair(ctx context.Context, userID uuid.UUID) (*services.TokenPair, error) {
	tokens, err := services.GenerateTokenPair(userID, h.Secret)
	if err != nil {
		return nil, err
	}

	_, err = h.Queries.CreateRefreshToken(ctx, sqlc.CreateRefreshTokenParams{
		UserID:    pgUUID(userID),
		TokenHash: services.HashRefreshToken(tokens.RefreshToken),
		ExpiresAt: pgtype.Timestamptz{Time: services.RefreshTokenExpiry(), Valid: true},
	})
	if err != nil {
		return nil, err
	}

	return tokens, nil
}
