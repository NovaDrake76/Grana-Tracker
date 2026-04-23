package handlers_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestRegisterHappyPath(t *testing.T) {
	requireDB(t)
	truncateAll(t)
	r := newTestRouter(t)

	rr, resp := doRequest(t, r, http.MethodPost, "/api/auth/register", "", map[string]string{
		"name":     "Alice",
		"email":    "alice@example.com",
		"password": "hunter2",
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body %s", rr.Code, rr.Body.String())
	}
	var tokens struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(resp.Data, &tokens); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Error("empty tokens in response")
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	requireDB(t)
	truncateAll(t)
	r := newTestRouter(t)

	registerUser(t, r, "dup@example.com", "hunter2")

	rr, resp := doRequest(t, r, http.MethodPost, "/api/auth/register", "", map[string]string{
		"name":     "Bob",
		"email":    "dup@example.com",
		"password": "hunter3",
	})
	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rr.Code)
	}
	if resp.Code != "DUPLICATE_ERROR" {
		t.Errorf("code = %q, want DUPLICATE_ERROR", resp.Code)
	}
}

func TestRegisterValidation(t *testing.T) {
	requireDB(t)
	truncateAll(t)
	r := newTestRouter(t)

	cases := []struct {
		name string
		body map[string]string
	}{
		{"missing email", map[string]string{"name": "A", "password": "hunter2"}},
		{"missing password", map[string]string{"name": "A", "email": "a@b.com"}},
		{"short password", map[string]string{"name": "A", "email": "a@b.com", "password": "x"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rr, resp := doRequest(t, r, http.MethodPost, "/api/auth/register", "", c.body)
			if rr.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400", rr.Code)
			}
			if resp.Code != "VALIDATION_ERROR" {
				t.Errorf("code = %q, want VALIDATION_ERROR", resp.Code)
			}
		})
	}
}

func TestLoginFlow(t *testing.T) {
	requireDB(t)
	truncateAll(t)
	r := newTestRouter(t)

	registerUser(t, r, "login@example.com", "hunter2")

	t.Run("valid creds", func(t *testing.T) {
		rr, _ := doRequest(t, r, http.MethodPost, "/api/auth/login", "", map[string]string{
			"email":    "login@example.com",
			"password": "hunter2",
		})
		if rr.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", rr.Code)
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		rr, _ := doRequest(t, r, http.MethodPost, "/api/auth/login", "", map[string]string{
			"email":    "login@example.com",
			"password": "nope",
		})
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", rr.Code)
		}
	})

	t.Run("unknown email", func(t *testing.T) {
		rr, _ := doRequest(t, r, http.MethodPost, "/api/auth/login", "", map[string]string{
			"email":    "ghost@example.com",
			"password": "hunter2",
		})
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", rr.Code)
		}
	})
}

func TestRefreshFlow(t *testing.T) {
	requireDB(t)
	truncateAll(t)
	r := newTestRouter(t)

	registerUser(t, r, "refresh@example.com", "hunter2")

	rr, resp := doRequest(t, r, http.MethodPost, "/api/auth/login", "", map[string]string{
		"email":    "refresh@example.com",
		"password": "hunter2",
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("login failed: %d", rr.Code)
	}
	var tokens struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(resp.Data, &tokens); err != nil {
		t.Fatalf("decode tokens: %v", err)
	}

	t.Run("valid refresh", func(t *testing.T) {
		rr, resp := doRequest(t, r, http.MethodPost, "/api/auth/refresh", "", map[string]string{
			"refresh_token": tokens.RefreshToken,
		})
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rr.Code)
		}
		var fresh struct {
			AccessToken string `json:"access_token"`
		}
		if err := json.Unmarshal(resp.Data, &fresh); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if fresh.AccessToken == "" {
			t.Error("empty access token in refresh response")
		}
	})

	t.Run("garbage refresh", func(t *testing.T) {
		rr, _ := doRequest(t, r, http.MethodPost, "/api/auth/refresh", "", map[string]string{
			"refresh_token": "not-a-jwt",
		})
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", rr.Code)
		}
	})

	t.Run("missing refresh", func(t *testing.T) {
		rr, _ := doRequest(t, r, http.MethodPost, "/api/auth/refresh", "", map[string]string{})
		if rr.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rr.Code)
		}
	})
}
