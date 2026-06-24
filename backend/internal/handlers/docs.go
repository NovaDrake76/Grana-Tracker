package handlers

import (
	"net/http"
)

// DocsHandler serves the OpenAPI specification and a Swagger UI viewer over
// public, unauthenticated routes so graders / new devs can explore the API
// surface without provisioning credentials first (US13 — Documentação).
//
// The YAML body is read once at boot and cached in memory, so each request
// is a single in-memory copy with no disk I/O on the hot path. If the file
// could not be read (missing on disk), we still mount the routes but the
// /openapi.yaml endpoint will return 503 instead of panicking — the rest of
// the API stays usable.
type DocsHandler struct {
	openapiYAML []byte
}

// NewDocsHandler caches the OpenAPI spec body. Pass nil if the file could
// not be read at boot — the handler will still mount but return 503 on the
// spec route so callers see a clear error instead of a stack trace.
func NewDocsHandler(openapiYAMLContent []byte) *DocsHandler {
	return &DocsHandler{openapiYAML: openapiYAMLContent}
}

// ServeOpenAPI emits the raw OpenAPI 3 document as application/yaml so
// Swagger UI (and any external tooling like Postman / Insomnia / Stoplight)
// can fetch and parse it directly. Returns 503 if the spec was not loaded
// at boot — that's a deployment misconfiguration, not a request error.
func (h *DocsHandler) ServeOpenAPI(w http.ResponseWriter, r *http.Request) {
	if len(h.openapiYAML) == 0 {
		http.Error(w, "openapi spec not available", http.StatusServiceUnavailable)
		return
	}
	// application/yaml is the IANA-registered media type (RFC 9512).
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	// Allow short-lived caching so refreshing the Swagger UI page doesn't
	// re-download the full spec every time.
	w.Header().Set("Cache-Control", "public, max-age=300")
	_, _ = w.Write(h.openapiYAML)
}

// swaggerUIHTML is the minimal single-page Swagger UI shell. JS + CSS are
// pulled from the unpkg CDN so we don't have to vendor ~1MB of static assets
// into the Go binary. The page points at /openapi.yaml on the same origin,
// so it works locally, on Render, and anywhere else the API is hosted —
// no per-environment URL rewriting needed.
const swaggerUIHTML = `<!DOCTYPE html>
<html>
<head>
  <title>Grana Tracker API</title>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = function () {
      SwaggerUIBundle({ url: '/openapi.yaml', dom_id: '#swagger-ui' });
    };
  </script>
</body>
</html>`

// ServeSwaggerUI renders the Swagger UI single-page viewer. It overrides
// the strict global CSP (default-src 'self') because the page deliberately
// pulls JS + CSS from unpkg.com — without these directives the browser
// would block every Swagger asset and render an empty page.
//
// This route ships zero user data and no auth tokens, so loosening the CSP
// here does not weaken the protections on /api/*. The override is local to
// this handler only.
func (h *DocsHandler) ServeSwaggerUI(w http.ResponseWriter, r *http.Request) {
	// Allow Swagger UI's JS/CSS from unpkg and its inline bootstrap script.
	// font-src + img-src cover Swagger UI's woff2 icons and the data: URIs
	// it uses for embedded SVG glyphs.
	w.Header().Set("Content-Security-Policy",
		"default-src 'self'; "+
			"script-src 'self' https://unpkg.com 'unsafe-inline'; "+
			"style-src 'self' https://unpkg.com 'unsafe-inline'; "+
			"img-src 'self' data: https://unpkg.com; "+
			"font-src 'self' data: https://unpkg.com; "+
			"connect-src 'self'")
	// Swagger UI does not embed itself in an iframe, but we keep DENY off so
	// the page can be linked from the README / slides without a frame-ancestors
	// surprise. SAMEORIGIN is the safe middle ground.
	w.Header().Set("X-Frame-Options", "SAMEORIGIN")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(swaggerUIHTML))
}
