package handlers_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

type investmentDTO struct {
	ID             string  `json:"id"`
	PortfolioID    string  `json:"portfolio_id"`
	Ticker         string  `json:"ticker"`
	AssetType      string  `json:"asset_type"`
	AmountInvested string  `json:"amount_invested"`
	Quantity       *string `json:"quantity"`
	PurchaseDate   string  `json:"purchase_date"`
	Notes          *string `json:"notes"`
}

type portfolioWithInvestmentsDTO struct {
	portfolioDTO
	Investments []investmentDTO `json:"investments"`
}

// helper: register + create one portfolio, return token and portfolio id.
func setupOnePortfolio(t *testing.T, email string) (token string, portfolioID string) {
	t.Helper()
	r := newTestRouter(t)
	token = registerUser(t, r, email, "hunter22")
	_, resp := doRequest(t, r, http.MethodPost, "/api/portfolios", token, map[string]string{
		"name": "Test PF",
		"type": "real",
	})
	var pf portfolioDTO
	if err := json.Unmarshal(resp.Data, &pf); err != nil {
		t.Fatalf("decode portfolio: %v", err)
	}
	return token, pf.ID
}

func TestInvestmentCRUD(t *testing.T) {
	requireDB(t)
	truncateAll(t)

	r := newTestRouter(t)
	token := registerUser(t, r, "inv-crud@example.com", "hunter22")

	// create a portfolio to host the investment
	_, resp := doRequest(t, r, http.MethodPost, "/api/portfolios", token, map[string]string{
		"name": "Tech",
		"type": "real",
	})
	var pf portfolioDTO
	if err := json.Unmarshal(resp.Data, &pf); err != nil {
		t.Fatalf("decode portfolio: %v", err)
	}

	// create investment
	rr, resp := doRequest(t, r, http.MethodPost, "/api/portfolios/"+pf.ID+"/investments", token, map[string]interface{}{
		"ticker":          "aapl",
		"asset_type":      "stock",
		"amount_invested": "1000.50",
		"quantity":        "5.25",
		"purchase_date":   "2025-01-15",
		"notes":           "first buy",
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201, body %s", rr.Code, rr.Body.String())
	}
	var inv investmentDTO
	if err := json.Unmarshal(resp.Data, &inv); err != nil {
		t.Fatalf("decode investment: %v", err)
	}
	if inv.ID == "" {
		t.Fatal("empty investment id")
	}
	if inv.Ticker != "aapl" {
		t.Errorf("ticker = %q, want aapl", inv.Ticker)
	}
	if inv.AssetType != "stock" {
		t.Errorf("asset_type = %q, want stock", inv.AssetType)
	}
	if inv.AmountInvested != "1000.50" && inv.AmountInvested != "1000.5" {
		t.Errorf("amount_invested = %q, want 1000.50", inv.AmountInvested)
	}
	if inv.PurchaseDate != "2025-01-15" {
		t.Errorf("purchase_date = %q, want 2025-01-15", inv.PurchaseDate)
	}

	// get one
	rr, resp = doRequest(t, r, http.MethodGet, "/api/investments/"+inv.ID, token, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("get status = %d, want 200", rr.Code)
	}
	var got investmentDTO
	if err := json.Unmarshal(resp.Data, &got); err != nil {
		t.Fatalf("decode get: %v", err)
	}
	if got.ID != inv.ID {
		t.Errorf("get id = %q, want %q", got.ID, inv.ID)
	}

	// update
	newTicker := "AAPL"
	newAmount := "2000.00"
	rr, resp = doRequest(t, r, http.MethodPut, "/api/investments/"+inv.ID, token, map[string]interface{}{
		"ticker":          newTicker,
		"amount_invested": newAmount,
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("update status = %d, want 200, body %s", rr.Code, rr.Body.String())
	}
	var updated investmentDTO
	if err := json.Unmarshal(resp.Data, &updated); err != nil {
		t.Fatalf("decode update: %v", err)
	}
	if updated.Ticker != "AAPL" {
		t.Errorf("ticker after update = %q, want AAPL", updated.Ticker)
	}
	if updated.AmountInvested != "2000.00" && updated.AmountInvested != "2000" {
		t.Errorf("amount after update = %q, want 2000.00", updated.AmountInvested)
	}

	// delete
	rr, _ = doRequest(t, r, http.MethodDelete, "/api/investments/"+inv.ID, token, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("delete status = %d, want 200", rr.Code)
	}

	// confirm gone
	rr, _ = doRequest(t, r, http.MethodGet, "/api/investments/"+inv.ID, token, nil)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("get-after-delete status = %d, want 404", rr.Code)
	}
}

func TestPortfolioGetEmbedsInvestments(t *testing.T) {
	requireDB(t)
	truncateAll(t)

	r := newTestRouter(t)
	token := registerUser(t, r, "inv-embed@example.com", "hunter22")

	_, resp := doRequest(t, r, http.MethodPost, "/api/portfolios", token, map[string]string{
		"name": "Crypto",
		"type": "simulated",
	})
	var pf portfolioDTO
	if err := json.Unmarshal(resp.Data, &pf); err != nil {
		t.Fatalf("decode portfolio: %v", err)
	}

	// add two investments
	for _, body := range []map[string]interface{}{
		{
			"ticker":          "BTC",
			"asset_type":      "crypto",
			"amount_invested": "500.00",
			"quantity":        "0.01",
			"purchase_date":   "2025-02-01",
		},
		{
			"ticker":          "ETH",
			"asset_type":      "crypto",
			"amount_invested": "300.00",
			"quantity":        "0.15",
			"purchase_date":   "2025-03-10",
		},
	} {
		rr, _ := doRequest(t, r, http.MethodPost, "/api/portfolios/"+pf.ID+"/investments", token, body)
		if rr.Code != http.StatusCreated {
			t.Fatalf("seed status = %d, body %v", rr.Code, body)
		}
	}

	// the nested-payload endpoint
	rr, resp := doRequest(t, r, http.MethodGet, "/api/portfolios/"+pf.ID, token, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("get portfolio status = %d", rr.Code)
	}
	var nested portfolioWithInvestmentsDTO
	if err := json.Unmarshal(resp.Data, &nested); err != nil {
		t.Fatalf("decode nested: %v", err)
	}
	if nested.ID != pf.ID {
		t.Errorf("nested id = %q, want %q", nested.ID, pf.ID)
	}
	if len(nested.Investments) != 2 {
		t.Fatalf("nested investments count = %d, want 2", len(nested.Investments))
	}
	// ordered by purchase_date DESC: ETH (2025-03-10) before BTC (2025-02-01)
	if nested.Investments[0].Ticker != "ETH" || nested.Investments[1].Ticker != "BTC" {
		t.Errorf("unexpected order: %q, %q", nested.Investments[0].Ticker, nested.Investments[1].Ticker)
	}
}

func TestInvestmentOwnershipIsolation(t *testing.T) {
	requireDB(t)
	truncateAll(t)

	r := newTestRouter(t)
	aliceToken, alicePF := setupOnePortfolio(t, "inv-iso-alice@example.com")
	bobToken := registerUser(t, r, "inv-iso-bob@example.com", "hunter22")

	// alice creates investment
	_, resp := doRequest(t, r, http.MethodPost, "/api/portfolios/"+alicePF+"/investments", aliceToken, map[string]interface{}{
		"ticker":          "MSFT",
		"asset_type":      "stock",
		"amount_invested": "1500.00",
		"purchase_date":   "2025-04-01",
	})
	var inv investmentDTO
	if err := json.Unmarshal(resp.Data, &inv); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// bob can't read it
	rr, _ := doRequest(t, r, http.MethodGet, "/api/investments/"+inv.ID, bobToken, nil)
	if rr.Code != http.StatusForbidden {
		t.Errorf("bob get status = %d, want 403", rr.Code)
	}

	// bob can't update it
	rr, _ = doRequest(t, r, http.MethodPut, "/api/investments/"+inv.ID, bobToken, map[string]interface{}{
		"ticker": "PWND",
	})
	if rr.Code != http.StatusForbidden {
		t.Errorf("bob update status = %d, want 403", rr.Code)
	}

	// bob can't delete it
	rr, _ = doRequest(t, r, http.MethodDelete, "/api/investments/"+inv.ID, bobToken, nil)
	if rr.Code != http.StatusForbidden {
		t.Errorf("bob delete status = %d, want 403", rr.Code)
	}

	// bob can't even add to alice's portfolio
	rr, _ = doRequest(t, r, http.MethodPost, "/api/portfolios/"+alicePF+"/investments", bobToken, map[string]interface{}{
		"ticker":          "BAD",
		"asset_type":      "stock",
		"amount_invested": "1.00",
		"purchase_date":   "2025-04-01",
	})
	if rr.Code != http.StatusForbidden {
		t.Errorf("bob create-in-alice-pf status = %d, want 403", rr.Code)
	}
}

func TestInvestmentValidation(t *testing.T) {
	requireDB(t)
	truncateAll(t)

	r := newTestRouter(t)
	token, pf := setupOnePortfolio(t, "inv-val@example.com")

	t.Run("missing ticker", func(t *testing.T) {
		rr, _ := doRequest(t, r, http.MethodPost, "/api/portfolios/"+pf+"/investments", token, map[string]interface{}{
			"asset_type":      "stock",
			"amount_invested": "100",
			"purchase_date":   "2025-04-01",
		})
		if rr.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rr.Code)
		}
	})

	t.Run("bad asset_type", func(t *testing.T) {
		rr, _ := doRequest(t, r, http.MethodPost, "/api/portfolios/"+pf+"/investments", token, map[string]interface{}{
			"ticker":          "X",
			"asset_type":      "moon",
			"amount_invested": "100",
			"purchase_date":   "2025-04-01",
		})
		if rr.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rr.Code)
		}
	})

	t.Run("bad date", func(t *testing.T) {
		rr, _ := doRequest(t, r, http.MethodPost, "/api/portfolios/"+pf+"/investments", token, map[string]interface{}{
			"ticker":          "X",
			"asset_type":      "stock",
			"amount_invested": "100",
			"purchase_date":   "yesterday",
		})
		if rr.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rr.Code)
		}
	})

	t.Run("missing token", func(t *testing.T) {
		rr, _ := doRequest(t, r, http.MethodGet, "/api/investments/00000000-0000-0000-0000-000000000000", "", nil)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", rr.Code)
		}
	})
}

func TestPortfolioDeleteCascadesInvestments(t *testing.T) {
	requireDB(t)
	truncateAll(t)

	r := newTestRouter(t)
	token, pf := setupOnePortfolio(t, "inv-cascade@example.com")

	// add an investment
	_, resp := doRequest(t, r, http.MethodPost, "/api/portfolios/"+pf+"/investments", token, map[string]interface{}{
		"ticker":          "TSLA",
		"asset_type":      "stock",
		"amount_invested": "750.00",
		"purchase_date":   "2025-04-01",
	})
	var inv investmentDTO
	if err := json.Unmarshal(resp.Data, &inv); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// delete the portfolio
	rr, _ := doRequest(t, r, http.MethodDelete, "/api/portfolios/"+pf, token, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("delete portfolio status = %d", rr.Code)
	}

	// the investment is gone too (ON DELETE CASCADE)
	rr, _ = doRequest(t, r, http.MethodGet, "/api/investments/"+inv.ID, token, nil)
	if rr.Code != http.StatusNotFound {
		t.Errorf("investment after pf delete status = %d, want 404", rr.Code)
	}
}
