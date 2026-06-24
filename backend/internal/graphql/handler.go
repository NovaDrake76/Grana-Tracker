package graphql

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/graphql-go/graphql"
)

// Handler serves POST /api/graphql. It expects the standard GraphQL-over-HTTP
// JSON shape ({query, variables, operationName}) and returns the canonical
// {data, errors} envelope. Auth is enforced upstream by the Authenticate
// middleware — by the time we get here, middleware.GetUserID(ctx) is non-empty.
type Handler struct {
	Schema graphql.Schema
}

// NewHandler returns an http.Handler that executes queries against the supplied
// schema. The schema is built once at boot via NewSchema, so per-request
// allocation is just JSON decode + graphql.Do.
func NewHandler(schema graphql.Schema) *Handler {
	return &Handler{Schema: schema}
}

// graphqlRequest is the standard payload shape from graphql.org's spec for the
// GraphQL-over-HTTP transport.
type graphqlRequest struct {
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables"`
	OperationName string                 `json:"operationName"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req graphqlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Query == "" {
		writeError(w, http.StatusBadRequest, "query is required")
		return
	}

	result := graphql.Do(graphql.Params{
		Schema:         h.Schema,
		RequestString:  req.Query,
		VariableValues: req.Variables,
		OperationName:  req.OperationName,
		Context:        r.Context(),
	})

	// GraphQL keeps a 200 OK even on field-level resolver errors — errors land
	// in the `errors` array, not the HTTP status. Only protocol-level failures
	// (parse/validate) get a non-200 above.
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Printf("graphql encode: %v", err)
	}
}

// writeError emits a GraphQL-shaped error envelope so the front-end has a
// single error branch regardless of whether the failure was at the HTTP layer
// or the resolver layer.
func writeError(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"data":   nil,
		"errors": []map[string]interface{}{{"message": message}},
	})
}
