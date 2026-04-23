package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
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
