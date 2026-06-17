package handlers

import (
	"errors"
	"math/big"
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
	PurchasePrice  string  `json:"purchase_price"`
	Currency       string  `json:"currency"`
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
	PurchasePrice  string  `json:"purchase_price"`
	Currency       string  `json:"currency"`
	PurchaseDate   string  `json:"purchase_date"`
	Notes          *string `json:"notes"`
}

type updateInvestmentRequest struct {
	Ticker         *string `json:"ticker"`
	AssetType      *string `json:"asset_type"`
	AmountInvested *string `json:"amount_invested"`
	Quantity       *string `json:"quantity"`
	PurchasePrice  *string `json:"purchase_price"`
	Currency       *string `json:"currency"`
	PurchaseDate   *string `json:"purchase_date"`
	Notes          *string `json:"notes"`
}

var allowedAssetTypes = map[string]struct{}{
	"stock":  {},
	"crypto": {},
	"etf":    {},
	"index":  {},
}

// allowedCurrencies matches the CHECK constraint on investments.currency.
var allowedCurrencies = map[string]struct{}{
	"BRL": {},
	"USD": {},
}

// purchasePriceString renders the purchase_price column. NULL becomes the
// em-dash placeholder so the UI can show a clear "missing" cell without
// special-casing nil checks in JSON consumers.
func purchasePriceString(n pgtype.Numeric) string {
	if !n.Valid {
		return "—"
	}
	return numericStr(n)
}

func toInvestmentResponse(i sqlc.Investment) investmentResponse {
	return investmentResponse{
		ID:             uuidStr(i.ID),
		PortfolioID:    uuidStr(i.PortfolioID),
		Ticker:         i.Ticker,
		AssetType:      i.AssetType,
		AmountInvested: numericStr(i.AmountInvested),
		Quantity:       numericStrPtr(i.Quantity),
		PurchasePrice:  purchasePriceString(i.PurchasePrice),
		Currency:       i.Currency,
		PurchaseDate:   dateStr(i.PurchaseDate),
		Notes:          textPtr(i.Notes),
		CreatedAt:      tsString(i.CreatedAt),
		UpdatedAt:      tsString(i.UpdatedAt),
	}
}

// derivePurchasePrice computes amount_invested / quantity as a big.Float and
// returns the decimal string. Used when the caller omitted purchase_price but
// supplied quantity > 0 — we never want a NULL purchase_price after this
// migration if we can avoid it.
func derivePurchasePrice(amount, qty pgtype.Numeric) (pgtype.Numeric, bool) {
	if !amount.Valid || !qty.Valid {
		return pgtype.Numeric{}, false
	}
	amountF, err := numericToBigFloat(amount)
	if err != nil {
		return pgtype.Numeric{}, false
	}
	qtyF, err := numericToBigFloat(qty)
	if err != nil {
		return pgtype.Numeric{}, false
	}
	if qtyF.Sign() <= 0 {
		return pgtype.Numeric{}, false
	}
	res := new(big.Float).SetPrec(128).Quo(amountF, qtyF)
	// 8 decimal places matches the DECIMAL(18,8) column.
	text := res.Text('f', 8)
	var out pgtype.Numeric
	if err := out.Scan(text); err != nil {
		return pgtype.Numeric{}, false
	}
	return out, true
}

// numericToBigFloat converts pgtype.Numeric to *big.Float by going through the
// string representation, avoiding precision loss from float64.
func numericToBigFloat(n pgtype.Numeric) (*big.Float, error) {
	s := numericStr(n)
	if s == "" {
		return nil, errors.New("empty numeric")
	}
	f, _, err := big.ParseFloat(s, 10, 128, big.ToNearestEven)
	return f, err
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
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "VALIDATION_ERROR")
		return
	}

	if msg := validateInvestmentFields(req.Ticker, req.AssetType, req.AmountInvested, req.PurchaseDate); msg != "" {
		writeError(w, http.StatusBadRequest, msg, "VALIDATION_ERROR")
		return
	}

	currency := req.Currency
	if currency == "" {
		currency = "BRL"
	}
	if _, ok := allowedCurrencies[currency]; !ok {
		writeError(w, http.StatusBadRequest, "currency must be one of BRL, USD", "VALIDATION_ERROR")
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

	// purchase_price authoritativeness: if the caller supplied it, use it as is.
	// If it is missing but we know quantity > 0, derive amount/quantity so the
	// row is not stranded with NULL purchase_price.
	var purchasePrice pgtype.Numeric
	if req.PurchasePrice != "" {
		purchasePrice, err = parseNumeric(req.PurchasePrice)
		if err != nil || !purchasePrice.Valid {
			writeError(w, http.StatusBadRequest, "purchase_price must be a valid decimal", "VALIDATION_ERROR")
			return
		}
	} else if qty.Valid {
		if derived, ok := derivePurchasePrice(amount, qty); ok {
			purchasePrice = derived
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
		PurchasePrice:  purchasePrice,
		Currency:       currency,
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
			PurchasePrice:  purchasePriceString(row.PurchasePrice),
			Currency:       row.Currency,
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
	if err := decodeJSON(r, &req); err != nil {
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

	currency := current.Currency
	if req.Currency != nil {
		if *req.Currency != "" {
			if _, ok := allowedCurrencies[*req.Currency]; !ok {
				writeError(w, http.StatusBadRequest, "currency must be one of BRL, USD", "VALIDATION_ERROR")
				return
			}
			currency = *req.Currency
		}
	}

	// purchase_price update rules:
	//   - explicit value wins ("" clears it and we attempt a backfill)
	//   - omitted field keeps the existing value
	//   - if cleared/missing and quantity > 0, derive amount/quantity
	purchasePrice := current.PurchasePrice
	purchasePriceProvided := false
	if req.PurchasePrice != nil {
		purchasePriceProvided = true
		if *req.PurchasePrice == "" {
			purchasePrice = pgtype.Numeric{}
		} else {
			parsed, err := parseNumeric(*req.PurchasePrice)
			if err != nil || !parsed.Valid {
				writeError(w, http.StatusBadRequest, "purchase_price must be a valid decimal", "VALIDATION_ERROR")
				return
			}
			purchasePrice = parsed
		}
	}
	// If amount or qty changed and the caller did not pin purchase_price
	// explicitly, refresh it from amount/qty when both are available.
	if !purchasePriceProvided && !purchasePrice.Valid && qty.Valid {
		if derived, ok := derivePurchasePrice(amount, qty); ok {
			purchasePrice = derived
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
		PurchasePrice:  purchasePrice,
		Currency:       currency,
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
