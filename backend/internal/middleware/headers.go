package middleware

import "net/http"

// SecurityHeaders adds defence-in-depth response headers to every request
// (OWASP A05 — Security Misconfiguration). The headers neutralise common
// browser-side attack classes:
//   - X-Content-Type-Options: nosniff           => disables MIME sniffing
//   - X-Frame-Options: DENY                     => blocks click-jacking via iframes
//   - Referrer-Policy: no-referrer              => never leak the source URL
//   - Strict-Transport-Security                 => force HTTPS for a year
//   - Content-Security-Policy: default-src 'self' => restrict script/asset origins
//   - Permissions-Policy                        => deny geolocation/mic/camera
//   - Cross-Origin-Opener-Policy: same-origin   => isolate browsing context (Spectre defence)
//   - Cross-Origin-Resource-Policy: same-origin => block cross-origin embedding
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		h.Set("Content-Security-Policy", "default-src 'self'")
		h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Cross-Origin-Resource-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}
