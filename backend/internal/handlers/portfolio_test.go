package handlers_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

type portfolioDTO struct {
	ID     string `json:"id"`
	UserID string `json:"user_id"`
	Name   string `json:"name"`
	Type   string `json:"type"`
}

func TestPortfolioCRUD(t *testing.T) {
	requireDB(t)
	truncateAll(t)
	r := newTestRouter(t)

	token := registerUser(t, r, "pf-crud@example.com", "hunter2")

	// create
	rr, resp := doRequest(t, r, http.MethodPost, "/api/portfolios", token, map[string]string{
		"name": "Growth",
		"type": "real",
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201, body %s", rr.Code, rr.Body.String())
	}
	var created portfolioDTO
	if err := json.Unmarshal(resp.Data, &created); err != nil {
		t.Fatalf("decode created: %v", err)
	}
	if created.ID == "" || created.Name != "Growth" || created.Type != "real" {
		t.Fatalf("unexpected created body: %+v", created)
	}

	// list
	rr, resp = doRequest(t, r, http.MethodGet, "/api/portfolios", token, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("list status = %d, want 200", rr.Code)
	}
	var list []portfolioDTO
	if err := json.Unmarshal(resp.Data, &list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("list mismatch: %+v", list)
	}

	// get one
	rr, resp = doRequest(t, r, http.MethodGet, "/api/portfolios/"+created.ID, token, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("get status = %d, want 200", rr.Code)
	}
	var got portfolioDTO
	if err := json.Unmarshal(resp.Data, &got); err != nil {
		t.Fatalf("decode get: %v", err)
	}
	if got.Name != "Growth" {
		t.Errorf("name = %q, want Growth", got.Name)
	}

	// update
	rr, resp = doRequest(t, r, http.MethodPut, "/api/portfolios/"+created.ID, token, map[string]string{
		"name": "Renamed",
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("update status = %d, want 200", rr.Code)
	}
	var updated portfolioDTO
	if err := json.Unmarshal(resp.Data, &updated); err != nil {
		t.Fatalf("decode update: %v", err)
	}
	if updated.Name != "Renamed" {
		t.Errorf("name after update = %q, want Renamed", updated.Name)
	}

	// delete
	rr, _ = doRequest(t, r, http.MethodDelete, "/api/portfolios/"+created.ID, token, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("delete status = %d, want 200", rr.Code)
	}

	// confirm gone
	rr, _ = doRequest(t, r, http.MethodGet, "/api/portfolios/"+created.ID, token, nil)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("get-after-delete status = %d, want 404", rr.Code)
	}
}

func TestPortfolioOwnershipIsolation(t *testing.T) {
	requireDB(t)
	truncateAll(t)
	r := newTestRouter(t)

	aliceToken := registerUser(t, r, "alice@example.com", "hunter2")
	bobToken := registerUser(t, r, "bob@example.com", "hunter2")

	// alice creates a portfolio
	_, resp := doRequest(t, r, http.MethodPost, "/api/portfolios", aliceToken, map[string]string{
		"name": "Alice's Pot",
		"type": "real",
	})
	var alicePF portfolioDTO
	if err := json.Unmarshal(resp.Data, &alicePF); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// bob cannot see alice's portfolio in HIS list
	rr, resp := doRequest(t, r, http.MethodGet, "/api/portfolios", bobToken, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("bob list status = %d", rr.Code)
	}
	var bobList []portfolioDTO
	if err := json.Unmarshal(resp.Data, &bobList); err != nil {
		t.Fatalf("decode bob list: %v", err)
	}
	if len(bobList) != 0 {
		t.Errorf("bob's list should be empty, got %+v", bobList)
	}

	// bob gets 403 when fetching alice's portfolio by ID
	rr, _ = doRequest(t, r, http.MethodGet, "/api/portfolios/"+alicePF.ID, bobToken, nil)
	if rr.Code != http.StatusForbidden {
		t.Errorf("bob get alice pf status = %d, want 403", rr.Code)
	}

	// bob cannot update it
	rr, _ = doRequest(t, r, http.MethodPut, "/api/portfolios/"+alicePF.ID, bobToken, map[string]string{
		"name": "Pwned",
	})
	if rr.Code != http.StatusForbidden {
		t.Errorf("bob update status = %d, want 403", rr.Code)
	}

	// bob cannot delete it
	rr, _ = doRequest(t, r, http.MethodDelete, "/api/portfolios/"+alicePF.ID, bobToken, nil)
	if rr.Code != http.StatusForbidden {
		t.Errorf("bob delete status = %d, want 403", rr.Code)
	}
}

func TestPortfolioValidation(t *testing.T) {
	requireDB(t)
	truncateAll(t)
	r := newTestRouter(t)

	token := registerUser(t, r, "pf-val@example.com", "hunter2")

	t.Run("missing name", func(t *testing.T) {
		rr, _ := doRequest(t, r, http.MethodPost, "/api/portfolios", token, map[string]string{
			"type": "real",
		})
		if rr.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rr.Code)
		}
	})

	t.Run("bad type", func(t *testing.T) {
		rr, _ := doRequest(t, r, http.MethodPost, "/api/portfolios", token, map[string]string{
			"name": "x",
			"type": "fictional",
		})
		if rr.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rr.Code)
		}
	})

	t.Run("invalid uuid path", func(t *testing.T) {
		rr, _ := doRequest(t, r, http.MethodGet, "/api/portfolios/not-a-uuid", token, nil)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rr.Code)
		}
	})

	t.Run("missing token", func(t *testing.T) {
		rr, _ := doRequest(t, r, http.MethodGet, "/api/portfolios", "", nil)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", rr.Code)
		}
	})
}
