package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// TestSecurityHeadersSetsAllHeaders verifies every hardening header is set on
// a sample GET response. We mount SecurityHeaders on a minimal chi router so
// the assertions hold against the exact middleware chain it will run in.
func TestSecurityHeadersSetsAllHeaders(t *testing.T) {
	r := chi.NewRouter()
	r.Use(SecurityHeaders)
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	want := map[string]string{
		"X-Content-Type-Options":       "nosniff",
		"X-Frame-Options":              "DENY",
		"Referrer-Policy":              "no-referrer",
		"Strict-Transport-Security":    "max-age=31536000; includeSubDomains",
		"Content-Security-Policy":      "default-src 'self'",
		"Permissions-Policy":           "geolocation=(), microphone=(), camera=()",
		"Cross-Origin-Opener-Policy":   "same-origin",
		"Cross-Origin-Resource-Policy": "same-origin",
	}
	for header, expected := range want {
		got := rr.Header().Get(header)
		if got != expected {
			t.Errorf("%s = %q, want %q", header, got, expected)
		}
	}
}
