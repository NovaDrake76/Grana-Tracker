package handlers_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/NovaDrake76/grana-tracker/backend/internal/handlers"
)

// TestServeSwaggerUIReturnsHTML confirms the /docs handler emits a valid
// HTML shell with the Swagger UI bootstrap script and a CSP that whitelists
// unpkg.com — otherwise the browser would block every Swagger asset and the
// page would render blank.
func TestServeSwaggerUIReturnsHTML(t *testing.T) {
	h := handlers.NewDocsHandler([]byte("openapi: 3.0.0"))

	rr := httptest.NewRecorder()
	h.ServeSwaggerUI(rr, httptest.NewRequest(http.MethodGet, "/docs", nil))
	resp := rr.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html...", ct)
	}
	csp := resp.Header.Get("Content-Security-Policy")
	if !strings.Contains(csp, "unpkg.com") {
		t.Errorf("CSP does not allow unpkg.com: %q", csp)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "SwaggerUIBundle") {
		t.Errorf("body missing SwaggerUIBundle bootstrap")
	}
	if !strings.Contains(string(body), "/openapi.yaml") {
		t.Errorf("body must point at /openapi.yaml")
	}
}

// TestServeOpenAPIWithSpec checks the YAML route returns the cached bytes
// verbatim and tags the response as application/yaml so external tools
// (Postman, Stoplight) parse it correctly.
func TestServeOpenAPIWithSpec(t *testing.T) {
	spec := []byte("openapi: 3.0.0\ninfo:\n  title: Test\n")
	h := handlers.NewDocsHandler(spec)

	rr := httptest.NewRecorder()
	h.ServeOpenAPI(rr, httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil))
	resp := rr.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/yaml") {
		t.Errorf("Content-Type = %q, want application/yaml...", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != string(spec) {
		t.Errorf("body mismatch: got %q, want %q", body, spec)
	}
}

// TestServeOpenAPIWithoutSpec proves the graceful-degradation path: a missing
// spec at boot must NOT panic and the route must return 503 so callers see a
// clear deployment-misconfiguration signal.
func TestServeOpenAPIWithoutSpec(t *testing.T) {
	h := handlers.NewDocsHandler(nil)

	rr := httptest.NewRecorder()
	h.ServeOpenAPI(rr, httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil))
	if rr.Result().StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rr.Result().StatusCode)
	}
}
