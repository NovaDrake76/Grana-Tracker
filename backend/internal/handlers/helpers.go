package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message, code string) {
	writeJSON(w, status, map[string]string{
		"error": message,
		"code":  code,
	})
}

// detects postgres unique-violation errors so we can return 409 instead of 500.
func isDuplicateKeyError(err error) bool {
	return strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "23505")
}

func uuidFromBytes(b [16]byte) uuid.UUID {
	return uuid.UUID(b)
}

func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

// pgUUID wraps a google/uuid value in the pgtype shape sqlc expects.
func pgUUID(u uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: u, Valid: true}
}

// uuidStr unwraps a pgtype.UUID back into the canonical hex string.
func uuidStr(p pgtype.UUID) string {
	return uuid.UUID(p.Bytes).String()
}

// pgText converts a Go string into pgtype.Text (always Valid for non-nil callers).
func pgText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: true}
}

// pgTextPtr maps a *string to pgtype.Text — nil pointer becomes a NULL value.
func pgTextPtr(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

// textPtr converts pgtype.Text back into *string for JSON responses.
func textPtr(p pgtype.Text) *string {
	if !p.Valid {
		return nil
	}
	v := p.String
	return &v
}

// tsString formats a pgtype.Timestamp as RFC3339 for JSON output.
func tsString(ts pgtype.Timestamp) string {
	if !ts.Valid {
		return ""
	}
	return ts.Time.Format(time.RFC3339)
}

// parseNumeric turns a decimal string (e.g. "123.45") into pgtype.Numeric.
// An empty string maps to a NULL value rather than an error.
func parseNumeric(s string) (pgtype.Numeric, error) {
	var n pgtype.Numeric
	if s == "" {
		return n, nil
	}
	if err := n.Scan(s); err != nil {
		return n, err
	}
	return n, nil
}

// numericStr emits a pgtype.Numeric as a plain decimal string for JSON.
func numericStr(n pgtype.Numeric) string {
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

// numericStrPtr returns nil when the numeric is NULL so JSON encodes `null`.
func numericStrPtr(n pgtype.Numeric) *string {
	if !n.Valid {
		return nil
	}
	s := numericStr(n)
	return &s
}

// parseDate accepts a YYYY-MM-DD string and returns a pgtype.Date.
func parseDate(s string) (pgtype.Date, error) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return pgtype.Date{}, err
	}
	return pgtype.Date{Time: t, Valid: true}, nil
}

// dateStr formats a pgtype.Date as YYYY-MM-DD; empty when NULL.
func dateStr(d pgtype.Date) string {
	if !d.Valid {
		return ""
	}
	return d.Time.Format("2006-01-02")
}
