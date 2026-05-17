package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/NovaDrake76/grana-tracker/backend/db/sqlc"
	"github.com/NovaDrake76/grana-tracker/backend/internal/middleware"
)

type InvestmentHandler struct {
	Queries *sqlc.Queries
}

func NewInvestmentHandler(queries *sqlc.Queries) *InvestmentHandler {
	return &InvestmentHandler{Queries: queries}
}

type investmentResponse struct {
	ID             string  `json:"id"`
	PortfolioID    string  `json:"portfolio_id"`
	Ticker         string  `json:"ticker"`
	AssetType      string  `json:"asset_type"`
	AmountInvested string  `json:"amount_invested"`
	Quantity       *string `json:"quantity"`
	PurchaseDate   string  `json:"purchase_date"`
	Notes          *string `json:"notes"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

type createInvestmentRequest struct {
	Ticker         string  `json:"ticker"`
	AssetType      string  `json:"asset_type"`
	AmountInvested string  `json:"amount_invested"`
	Quantity       *string `json:"quantity"`
	PurchaseDate   string  `json:"purchase_date"`
	Notes          *string `json:"notes"`
}

type updateInvestmentRequest struct {
	Ticker         *string `json:"ticker"`
	AssetType      *string `json:"asset_type"`
	AmountInvested *string `json:"amount_invested"`
	Quantity       *string `json:"quantity"`
	PurchaseDate   *string `json:"purchase_date"`
	Notes          *string `json:"notes"`
}

var allowedAssetTypes = map[string]struct{}{
	"stock":  {},
	"crypto": {},
	"etf":    {},
	"index":  {},
}

func toInvestmentResponse(i sqlc.Investment) investmentResponse {
	return investmentResponse{
		ID:             uuidStr(i.ID),
		PortfolioID:    uuidStr(i.PortfolioID),
		Ticker:         i.Ticker,
		AssetType:      i.AssetType,
		AmountInvested: numericStr(i.AmountInvested),
		Quantity:       numericStrPtr(i.Quantity),
		PurchaseDate:   dateStr(i.PurchaseDate),
		Notes:          textPtr(i.Notes),
		CreatedAt:      tsString(i.CreatedAt),
		UpdatedAt:      tsString(i.UpdatedAt),
	}
}

// adds an investment to a portfolio the caller owns.
// route: POST /api/portfolios/{id}/investments
func (h *InvestmentHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	portfolioID := chi.URLParam(r, "id")

	pid, err := uuid.Parse(portfolioID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid portfolio id", "VALIDATION_ERROR")
		return
	}

	portfolio, err := h.Queries.GetPortfolioByID(r.Context(), pgUUID(pid))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "portfolio not found", "NOT_FOUND")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load portfolio", "INTERNAL_ERROR")
		return
	}
	if uuidStr(portfolio.UserID) != userID {
		writeError(w, http.StatusForbidden, "access denied", "FORBIDDEN")
		return
	}

	var req createInvestmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "VALIDATION_ERROR")
		return
	}

	if msg := validateInvestmentFields(req.Ticker, req.AssetType, req.AmountInvested, req.PurchaseDate); msg != "" {
		writeError(w, http.StatusBadRequest, msg, "VALIDATION_ERROR")
		return
	}

	amount, err := parseNumeric(req.AmountInvested)
	if err != nil || !amount.Valid {
		writeError(w, http.StatusBadRequest, "amount_invested must be a valid decimal", "VALIDATION_ERROR")
		return
	}

	var qty pgtype.Numeric
	if req.Quantity != nil && *req.Quantity != "" {
		qty, err = parseNumeric(*req.Quantity)
		if err != nil {
			writeError(w, http.StatusBadRequest, "quantity must be a valid decimal", "VALIDATION_ERROR")
			return
		}
	}

	date, err := parseDate(req.PurchaseDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "purchase_date must be YYYY-MM-DD", "VALIDATION_ERROR")
		return
	}

	created, err := h.Queries.CreateInvestment(r.Context(), sqlc.CreateInvestmentParams{
		PortfolioID:    pgUUID(pid),
		Ticker:         req.Ticker,
		AssetType:      req.AssetType,
		AmountInvested: amount,
		Quantity:       qty,
		PurchaseDate:   date,
		Notes:          pgTextPtr(req.Notes),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create investment", "INTERNAL_ERROR")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"data":    toInvestmentResponse(created),
		"message": "investment created successfully",
	})
}

// fetches one investment after checking the caller owns its portfolio.
// route: GET /api/investments/{id}
func (h *InvestmentHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	investmentID := chi.URLParam(r, "id")

	iid, err := uuid.Parse(investmentID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid investment id", "VALIDATION_ERROR")
		return
	}

	row, err := h.Queries.GetInvestmentWithOwner(r.Context(), pgUUID(iid))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "investment not found", "NOT_FOUND")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load investment", "INTERNAL_ERROR")
		return
	}
	if uuidStr(row.UserID) != userID {
		writeError(w, http.StatusForbidden, "access denied", "FORBIDDEN")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": investmentResponse{
			ID:             uuidStr(row.ID),
			PortfolioID:    uuidStr(row.PortfolioID),
			Ticker:         row.Ticker,
			AssetType:      row.AssetType,
			AmountInvested: numericStr(row.AmountInvested),
			Quantity:       numericStrPtr(row.Quantity),
			PurchaseDate:   dateStr(row.PurchaseDate),
			Notes:          textPtr(row.Notes),
			CreatedAt:      tsString(row.CreatedAt),
			UpdatedAt:      tsString(row.UpdatedAt),
		},
	})
}

// patches any subset of fields on an investment the caller owns.
// route: PUT /api/investments/{id}
func (h *InvestmentHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	investmentID := chi.URLParam(r, "id")

	iid, err := uuid.Parse(investmentID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid investment id", "VALIDATION_ERROR")
		return
	}

	current, err := h.Queries.GetInvestmentWithOwner(r.Context(), pgUUID(iid))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "investment not found", "NOT_FOUND")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load investment", "INTERNAL_ERROR")
		return
	}
	if uuidStr(current.UserID) != userID {
		writeError(w, http.StatusForbidden, "access denied", "FORBIDDEN")
		return
	}

	var req updateInvestmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "VALIDATION_ERROR")
		return
	}

	ticker := current.Ticker
	if req.Ticker != nil {
		ticker = *req.Ticker
	}

	assetType := current.AssetType
	if req.AssetType != nil {
		if _, ok := allowedAssetTypes[*req.AssetType]; !ok {
			writeError(w, http.StatusBadRequest, "asset_type must be one of stock, crypto, etf, index", "VALIDATION_ERROR")
			return
		}
		assetType = *req.AssetType
	}

	amount := current.AmountInvested
	if req.AmountInvested != nil {
		parsed, err := parseNumeric(*req.AmountInvested)
		if err != nil || !parsed.Valid {
			writeError(w, http.StatusBadRequest, "amount_invested must be a valid decimal", "VALIDATION_ERROR")
			return
		}
		amount = parsed
	}

	qty := current.Quantity
	if req.Quantity != nil {
		if *req.Quantity == "" {
			qty = pgtype.Numeric{}
		} else {
			parsed, err := parseNumeric(*req.Quantity)
			if err != nil {
				writeError(w, http.StatusBadRequest, "quantity must be a valid decimal", "VALIDATION_ERROR")
				return
			}
			qty = parsed
		}
	}

	date := current.PurchaseDate
	if req.PurchaseDate != nil {
		parsed, err := parseDate(*req.PurchaseDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, "purchase_date must be YYYY-MM-DD", "VALIDATION_ERROR")
			return
		}
		date = parsed
	}

	notes := current.Notes
	if req.Notes != nil {
		notes = pgTextPtr(req.Notes)
	}

	updated, err := h.Queries.UpdateInvestment(r.Context(), sqlc.UpdateInvestmentParams{
		ID:             pgUUID(iid),
		Ticker:         ticker,
		AssetType:      assetType,
		AmountInvested: amount,
		Quantity:       qty,
		PurchaseDate:   date,
		Notes:          notes,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update investment", "INTERNAL_ERROR")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":    toInvestmentResponse(updated),
		"message": "investment updated successfully",
	})
}

// deletes an investment after checking the caller owns its portfolio.
// route: DELETE /api/investments/{id}
func (h *InvestmentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	investmentID := chi.URLParam(r, "id")

	iid, err := uuid.Parse(investmentID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid investment id", "VALIDATION_ERROR")
		return
	}

	current, err := h.Queries.GetInvestmentWithOwner(r.Context(), pgUUID(iid))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "investment not found", "NOT_FOUND")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load investment", "INTERNAL_ERROR")
		return
	}
	if uuidStr(current.UserID) != userID {
		writeError(w, http.StatusForbidden, "access denied", "FORBIDDEN")
		return
	}

	if err := h.Queries.DeleteInvestment(r.Context(), pgUUID(iid)); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete investment", "INTERNAL_ERROR")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "investment deleted successfully",
	})
}

func validateInvestmentFields(ticker, assetType, amount, purchaseDate string) string {
	if ticker == "" {
		return "ticker is required"
	}
	if amount == "" {
		return "amount_invested is required"
	}
	if purchaseDate == "" {
		return "purchase_date is required"
	}
	if _, ok := allowedAssetTypes[assetType]; !ok {
		return "asset_type must be one of stock, crypto, etf, index"
	}
	return ""
}
