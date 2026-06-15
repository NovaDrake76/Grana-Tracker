package services

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// RefreshTokenTTL controls how long a refresh token is considered valid before
// the database row is rejected as expired.
const RefreshTokenTTL = 7 * 24 * time.Hour

// AccessTokenTTL controls how long a short-lived access JWT is valid.
const AccessTokenTTL = 15 * time.Minute

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// HashRefreshToken returns a hex-encoded SHA-256 digest of the refresh token.
// We only store the hash in the database so a database leak does not let an
// attacker spend tokens directly.
func HashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// RefreshTokenExpiry returns the absolute timestamp at which a refresh token
// minted right now should expire. Handlers persist this in the refresh_tokens
// row alongside the hash.
func RefreshTokenExpiry() time.Time {
	return time.Now().Add(RefreshTokenTTL)
}

// builds a short-lived access token (15 min) and a long-lived refresh token (7 days), both HS256.
func GenerateTokenPair(userID uuid.UUID, secret string) (*TokenPair, error) {
	now := time.Now()

	accessClaims := Claims{
		UserID: userID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessStr, err := accessToken.SignedString([]byte(secret))
	if err != nil {
		return nil, err
	}

	// Add a jti so every refresh token is unique even when minted in the same
	// nanosecond — without this, rotating twice in a row could collide on the
	// JWT bytes (and therefore the SHA-256 hash, which has a UNIQUE constraint).
	refreshClaims := Claims{
		UserID: userID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			ExpiresAt: jwt.NewNumericDate(now.Add(RefreshTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshStr, err := refreshToken.SignedString([]byte(secret))
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessStr,
		RefreshToken: refreshStr,
	}, nil
}

// parses the token, verifies the HS256 signature against our secret, and returns the claims.
func ValidateToken(tokenStr, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Defence-in-depth: reject `alg: none` and any non-HMAC signing method
		// even before checking the secret.
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}

	return claims, nil
}
