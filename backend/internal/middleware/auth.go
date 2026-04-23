package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/NovaDrake76/grana-tracker/backend/internal/services"
)

type contextKey string

const UserIDKey contextKey = "user_id"

type AuthMiddleware struct {
	Secret string
}

func NewAuthMiddleware(secret string) *AuthMiddleware {
	return &AuthMiddleware{Secret: secret}
}

// validates the Bearer JWT and stores user_id in the request context; 401 otherwise.
func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenStr := extractBearerToken(r.Header.Get("Authorization"))
		if tokenStr == "" {
			http.Error(w, `{"error":"unauthorized","code":"UNAUTHORIZED"}`, http.StatusUnauthorized)
			return
		}

		claims, err := services.ValidateToken(tokenStr, m.Secret)
		if err != nil {
			http.Error(w, `{"error":"invalid token","code":"UNAUTHORIZED"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func extractBearerToken(header string) string {
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return parts[1]
}

func GetUserID(ctx context.Context) string {
	if id, ok := ctx.Value(UserIDKey).(string); ok {
		return id
	}
	return ""
}
