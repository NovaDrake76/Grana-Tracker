package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/NovaDrake76/grana-tracker/backend/internal/services"
)

func TestExtractBearerToken(t *testing.T) {
	cases := []struct {
		name, header, want string
	}{
		{"empty", "", ""},
		{"no scheme", "abc", ""},
		{"bearer mixed case", "Bearer xyz", "xyz"},
		{"bearer lower case", "bearer xyz", "xyz"},
		{"bearer upper case", "BEARER xyz", "xyz"},
		{"wrong scheme", "Basic xyz", ""},
		{"missing token", "Bearer ", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := extractBearerToken(c.header)
			if got != c.want {
				t.Errorf("extractBearerToken(%q) = %q, want %q", c.header, got, c.want)
			}
		})
	}
}

func TestGetUserID(t *testing.T) {
	if id := GetUserID(context.Background()); id != "" {
		t.Errorf("empty context returned %q, want empty", id)
	}
	ctx := context.WithValue(context.Background(), UserIDKey, "abc-123")
	if id := GetUserID(ctx); id != "abc-123" {
		t.Errorf("populated context returned %q, want abc-123", id)
	}
}

func TestAuthenticateMissingHeader(t *testing.T) {
	mw := NewAuthMiddleware("secret")
	h := mw.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler called despite missing auth header")
	}))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/whatever", nil))

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}

func TestAuthenticateInvalidToken(t *testing.T) {
	mw := NewAuthMiddleware("secret")
	h := mw.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler called despite invalid token")
	}))

	req := httptest.NewRequest(http.MethodGet, "/whatever", nil)
	req.Header.Set("Authorization", "Bearer garbage")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}

func TestAuthenticateValidTokenInjectsUserID(t *testing.T) {
	secret := "secret"
	userID := uuid.New()
	pair, err := services.GenerateTokenPair(userID, secret)
	if err != nil {
		t.Fatalf("GenerateTokenPair: %v", err)
	}

	called := false
	mw := NewAuthMiddleware(secret)
	h := mw.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if got := GetUserID(r.Context()); got != userID.String() {
			t.Errorf("context UserID = %q, want %q", got, userID.String())
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/whatever", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if !called {
		t.Fatal("next handler was not called")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
}
