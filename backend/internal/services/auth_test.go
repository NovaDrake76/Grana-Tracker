package services

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == "" {
		t.Fatal("hash is empty")
	}
	if hash == "correct horse battery staple" {
		t.Fatal("hash equals plaintext")
	}
	if !CheckPassword("correct horse battery staple", hash) {
		t.Fatal("correct password rejected")
	}
	if CheckPassword("wrong password", hash) {
		t.Fatal("wrong password accepted")
	}
}

func TestGenerateTokenPair(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret"

	pair, err := GenerateTokenPair(userID, secret)
	if err != nil {
		t.Fatalf("GenerateTokenPair: %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatal("empty token in pair")
	}
	if pair.AccessToken == pair.RefreshToken {
		t.Fatal("access and refresh token are identical")
	}

	for _, tok := range []string{pair.AccessToken, pair.RefreshToken} {
		claims, err := ValidateToken(tok, secret)
		if err != nil {
			t.Fatalf("ValidateToken: %v", err)
		}
		if claims.UserID != userID.String() {
			t.Fatalf("UserID mismatch: got %q want %q", claims.UserID, userID.String())
		}
	}
}

func TestValidateTokenRejectsWrongSecret(t *testing.T) {
	userID := uuid.New()
	pair, err := GenerateTokenPair(userID, "secret-a")
	if err != nil {
		t.Fatalf("GenerateTokenPair: %v", err)
	}
	if _, err := ValidateToken(pair.AccessToken, "secret-b"); err == nil {
		t.Fatal("token validated against wrong secret")
	}
}

func TestValidateTokenRejectsMalformed(t *testing.T) {
	if _, err := ValidateToken("not-a-jwt", "secret"); err == nil {
		t.Fatal("malformed token validated")
	}
	if _, err := ValidateToken("", "secret"); err == nil {
		t.Fatal("empty token validated")
	}
}

// TestValidateTokenRejectsAlgNone crafts an unsigned JWT with alg=none and
// confirms ValidateToken rejects it. This is the textbook JWT-confusion
// attack (CVE-2015-9235 class) and the signing-method pin in the keyfunc is
// the defence (OWASP A02 — Cryptographic Failures).
func TestValidateTokenRejectsAlgNone(t *testing.T) {
	claims := Claims{
		UserID: uuid.New().String(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	signed, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("sign with none: %v", err)
	}

	if _, err := ValidateToken(signed, "any-secret"); err == nil {
		t.Fatal("alg=none token was accepted; signing-method pin is missing")
	}
}

func TestValidateTokenRejectsExpired(t *testing.T) {
	secret := "test-secret"
	claims := Claims{
		UserID: uuid.New().String(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	_, err = ValidateToken(signed, secret)
	if err == nil {
		t.Fatal("expired token validated")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "expired") {
		t.Fatalf("expected expiry error, got: %v", err)
	}
}
