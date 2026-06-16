package services

import (
	"regexp"
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

// TestHashRefreshToken pins the contract of HashRefreshToken: deterministic,
// collision-resistant (different inputs differ), 64 lowercase hex chars
// (SHA-256), and stable on the empty string. Storing only this hash means a
// database leak cannot be replayed against the refresh endpoint.
func TestHashRefreshToken(t *testing.T) {
	t.Run("deterministic", func(t *testing.T) {
		if HashRefreshToken("abc") != HashRefreshToken("abc") {
			t.Fatal("HashRefreshToken is not deterministic for the same input")
		}
	})

	t.Run("different_inputs_differ", func(t *testing.T) {
		if HashRefreshToken("abc") == HashRefreshToken("abd") {
			t.Fatal("HashRefreshToken collided on trivially different inputs")
		}
	})

	t.Run("length_and_format", func(t *testing.T) {
		got := HashRefreshToken("any-input")
		if len(got) != 64 {
			t.Fatalf("expected 64-char hex digest, got %d chars: %q", len(got), got)
		}
		hexRe := regexp.MustCompile(`^[0-9a-f]{64}$`)
		if !hexRe.MatchString(got) {
			t.Fatalf("digest is not 64 lowercase hex chars: %q", got)
		}
	})

	t.Run("empty_input", func(t *testing.T) {
		// SHA-256 of the empty string is a well-known constant.
		const emptySHA256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
		if got := HashRefreshToken(""); got != emptySHA256 {
			t.Fatalf("HashRefreshToken(\"\") = %q, want %q", got, emptySHA256)
		}
	})
}

// TestRefreshTokenExpiry pins both the helper and the constant. The constant
// must be exactly 7 days so the rotation policy on the refresh endpoint
// matches the documented OWASP-aligned threat model.
func TestRefreshTokenExpiry(t *testing.T) {
	if RefreshTokenTTL != 7*24*time.Hour {
		t.Fatalf("RefreshTokenTTL drifted: got %v, want %v", RefreshTokenTTL, 7*24*time.Hour)
	}

	want := time.Now().Add(RefreshTokenTTL)
	got := RefreshTokenExpiry()

	diff := got.Sub(want)
	if diff < -5*time.Second || diff > 5*time.Second {
		t.Fatalf("RefreshTokenExpiry off by %v (got %v, want ~%v)", diff, got, want)
	}
}
