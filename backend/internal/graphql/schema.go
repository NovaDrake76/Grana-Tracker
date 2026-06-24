// Package graphql wires a minimal GraphQL endpoint on top of the existing sqlc
// repository layer so the front-end can fetch the dashboard in a single round
// trip (DIM0547 final delivery bonus). It defines exactly one query — `me` —
// that returns the authenticated user with nested portfolios and investments,
// reusing the same DB queries already exercised by the REST handlers.
package graphql

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/graphql-go/graphql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/NovaDrake76/grana-tracker/backend/db/sqlc"
	"github.com/NovaDrake76/grana-tracker/backend/internal/middleware"
)

// NewSchema builds the GraphQL schema bound to the sqlc Queries instance.
// Every resolver reads user_id from the request context (set by the Authenticate
// middleware) — never from query args — to prevent IDOR.
func NewSchema(queries *sqlc.Queries) (graphql.Schema, error) {
	investmentType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Investment",
		Fields: graphql.Fields{
			"id":              &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"ticker":          &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"asset_type":      &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"amount_invested": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"quantity":        &graphql.Field{Type: graphql.String},
			"current_price":  &graphql.Field{Type: graphql.String, Resolve: resolveCurrentPrice(queries)},
			"purchase_price": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"currency":       &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		},
	})

	portfolioType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Portfolio",
		Fields: graphql.Fields{
			"id":   &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"name": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"type": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"investments": &graphql.Field{
				Type:    graphql.NewList(graphql.NewNonNull(investmentType)),
				Resolve: resolveInvestments(queries),
			},
		},
	})

	userType := graphql.NewObject(graphql.ObjectConfig{
		Name: "User",
		Fields: graphql.Fields{
			"id":    &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"email": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"name":  &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"portfolios": &graphql.Field{
				Type:    graphql.NewList(graphql.NewNonNull(portfolioType)),
				Resolve: resolvePortfolios(queries),
			},
		},
	})

	rootQuery := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"me": &graphql.Field{
				Type:    userType,
				Resolve: resolveMe(queries),
			},
		},
	})

	return graphql.NewSchema(graphql.SchemaConfig{
		Query: rootQuery,
	})
}

// resolveMe loads the authenticated user. It reads user_id from the request
// context (never from args) to prevent IDOR — the JWT-derived id is the
// single source of truth for the caller's identity.
func resolveMe(queries *sqlc.Queries) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		userID := middleware.GetUserID(p.Context)
		if userID == "" {
			return nil, errors.New("unauthorized")
		}
		uid, err := uuid.Parse(userID)
		if err != nil {
			return nil, errors.New("invalid user id")
		}
		row, err := queries.GetUserByID(p.Context, pgtype.UUID{Bytes: uid, Valid: true})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, errors.New("user not found")
			}
			return nil, fmt.Errorf("failed to load user: %w", err)
		}
		// Forward the parsed UUID via Source so resolvePortfolios doesn't
		// have to re-parse the string. The leading underscore marks it as a
		// resolver-only field — the GraphQL schema does not expose it.
		return map[string]interface{}{
			"id":      uuidString(row.ID),
			"email":   row.Email,
			"name":    row.Name,
			"_userID": uid,
		}, nil
	}
}

// resolvePortfolios returns every portfolio owned by the authenticated user.
// Reuses ListPortfoliosByUser — no new SQL.
func resolvePortfolios(queries *sqlc.Queries) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		uid, err := userUUIDFromSource(p)
		if err != nil {
			return nil, err
		}
		rows, err := queries.ListPortfoliosByUser(p.Context, pgtype.UUID{Bytes: uid, Valid: true})
		if err != nil {
			return nil, fmt.Errorf("failed to list portfolios: %w", err)
		}
		out := make([]map[string]interface{}, 0, len(rows))
		for _, r := range rows {
			out = append(out, map[string]interface{}{
				"id":     uuidString(r.ID),
				"name":   r.Name,
				"type":   r.Type,
				"_pgID":  r.ID, // pgtype.UUID forwarded to investments resolver
			})
		}
		return out, nil
	}
}

// resolveInvestments returns every investment for a given portfolio. The
// portfolio_id is read from the parent map (set by resolvePortfolios) — never
// from args — so the caller cannot point this at someone else's portfolio.
func resolveInvestments(queries *sqlc.Queries) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		parent, ok := p.Source.(map[string]interface{})
		if !ok {
			return nil, errors.New("invalid portfolio source")
		}
		pgID, ok := parent["_pgID"].(pgtype.UUID)
		if !ok {
			return nil, errors.New("invalid portfolio id")
		}
		rows, err := queries.ListInvestmentsByPortfolio(p.Context, pgID)
		if err != nil {
			return nil, fmt.Errorf("failed to list investments: %w", err)
		}
		out := make([]map[string]interface{}, 0, len(rows))
		for _, i := range rows {
			out = append(out, map[string]interface{}{
				"id":              uuidString(i.ID),
				"ticker":          i.Ticker,
				"asset_type":      i.AssetType,
				"amount_invested": numericString(i.AmountInvested),
				"quantity":        numericStringPtr(i.Quantity),
				"purchase_price":  purchasePriceField(i.PurchasePrice),
				"currency":        i.Currency,
			})
		}
		return out, nil
	}
}

// resolveCurrentPrice looks up the cached spot price for the parent investment.
// Reuses GetCurrentPrice — no new SQL. Returns nil (GraphQL null) when no
// cached row exists rather than an error, so a missing quote does not poison
// the whole response.
func resolveCurrentPrice(queries *sqlc.Queries) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		parent, ok := p.Source.(map[string]interface{})
		if !ok {
			return nil, nil
		}
		ticker, _ := parent["ticker"].(string)
		assetType, _ := parent["asset_type"].(string)
		if ticker == "" || assetType == "" {
			return nil, nil
		}
		row, err := queries.GetCurrentPrice(p.Context, sqlc.GetCurrentPriceParams{
			Ticker:    ticker,
			AssetType: assetType,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, nil
			}
			return nil, nil
		}
		return numericString(row.Price), nil
	}
}

// userUUIDFromSource pulls the parent user's UUID out of the Source map set
// by resolveMe. Falls back to the middleware-supplied string in the request
// context for safety, but in practice Source is always populated.
func userUUIDFromSource(p graphql.ResolveParams) (uuid.UUID, error) {
	if parent, ok := p.Source.(map[string]interface{}); ok {
		if uid, ok := parent["_userID"].(uuid.UUID); ok {
			return uid, nil
		}
	}
	idStr := middleware.GetUserID(p.Context)
	if idStr == "" {
		return uuid.Nil, errors.New("unauthorized")
	}
	return uuid.Parse(idStr)
}

// uuidString unwraps a pgtype.UUID into its canonical hex string for GraphQL.
func uuidString(p pgtype.UUID) string {
	return uuid.UUID(p.Bytes).String()
}

// numericString renders a pgtype.Numeric as a plain decimal string. Mirrors
// handlers.numericStr so the GraphQL payload matches the REST one byte-for-byte
// — same precision, same formatting rules.
func numericString(n pgtype.Numeric) string {
	if !n.Valid {
		return ""
	}
	v, err := n.Value()
	if err != nil || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	default:
		return fmt.Sprintf("%v", t)
	}
}

// numericStringPtr returns a *string so GraphQL can serialise NULL quantities
// as JSON null rather than empty string.
func numericStringPtr(n pgtype.Numeric) *string {
	if !n.Valid {
		return nil
	}
	s := numericString(n)
	return &s
}

// purchasePriceField mirrors handlers.purchasePriceString — NULL becomes the
// em-dash placeholder so consumers don't have to special-case nil.
func purchasePriceField(n pgtype.Numeric) string {
	if !n.Valid {
		return "—"
	}
	return numericString(n)
}
