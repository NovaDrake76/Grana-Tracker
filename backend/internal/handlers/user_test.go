package handlers_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestGetMeRequiresAuth(t *testing.T) {
	requireDB(t)
	truncateAll(t)
	r := newTestRouter(t)

	rr, _ := doRequest(t, r, http.MethodGet, "/api/user/me", "", nil)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}

func TestGetMeReturnsProfile(t *testing.T) {
	requireDB(t)
	truncateAll(t)
	r := newTestRouter(t)

	token := registerUser(t, r, "profile@example.com", "hunter2")

	rr, resp := doRequest(t, r, http.MethodGet, "/api/user/me", token, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var profile struct {
		Email             string `json:"email"`
		PreferredCurrency string `json:"preferred_currency"`
	}
	if err := json.Unmarshal(resp.Data, &profile); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if profile.Email != "profile@example.com" {
		t.Errorf("email = %q, want profile@example.com", profile.Email)
	}
	if profile.PreferredCurrency != "BRL" {
		t.Errorf("preferred_currency = %q, want BRL (default)", profile.PreferredCurrency)
	}
}

func TestUpdateMePersistsChanges(t *testing.T) {
	requireDB(t)
	truncateAll(t)
	r := newTestRouter(t)

	token := registerUser(t, r, "update@example.com", "hunter2")

	newName := "Updated Name"
	newCurrency := "USD"
	rr, _ := doRequest(t, r, http.MethodPut, "/api/user/me", token, map[string]interface{}{
		"name":               newName,
		"preferred_currency": newCurrency,
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("update status = %d, want 200", rr.Code)
	}

	rr, resp := doRequest(t, r, http.MethodGet, "/api/user/me", token, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("re-fetch status = %d, want 200", rr.Code)
	}
	var profile struct {
		Name              string `json:"name"`
		PreferredCurrency string `json:"preferred_currency"`
	}
	if err := json.Unmarshal(resp.Data, &profile); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if profile.Name != newName {
		t.Errorf("name = %q, want %q", profile.Name, newName)
	}
	if profile.PreferredCurrency != newCurrency {
		t.Errorf("preferred_currency = %q, want %q", profile.PreferredCurrency, newCurrency)
	}
}
