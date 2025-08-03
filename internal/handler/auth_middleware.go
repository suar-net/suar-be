package handler

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/suar-net/suar-be/internal/model"
	"github.com/suar-net/suar-be/internal/service"
)

type contextKey string

const userContextKey = contextKey("user")

type AuthMiddleware struct {
	authService service.IAuthService
	logger      *log.Logger
}

func NewAuthMiddleware(s service.IAuthService, l *log.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		authService: s,
		logger:      l,
	}
}

// middleware untuk memeriksa token JWT
func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			respondWithError(w, http.StatusUnauthorized, "Authorization header is required")
			return
		}

		headerParts := strings.Split(authHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			respondWithError(w, http.StatusUnauthorized, "Authorization header format must be Bearer {token}")
			return
		}

		tokenString := headerParts[1]

		claims, err := m.authService.ValidateToken(r.Context(), tokenString)
		if err != nil {
			if errors.Is(err, service.ErrTokenExpired) {
				respondWithError(w, http.StatusUnauthorized, "Token has expired")
			} else {
				respondWithError(w, http.StatusUnauthorized, "Invalid token")
			}
			return
		}

		// Simpan claims di dalam context untuk digunakan oleh handler selanjutnya.
		ctx := context.WithValue(r.Context(), userContextKey, claims)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// helper untuk mendapatkan claims pengguna dari context.
func GetUserFromContext(ctx context.Context) (*model.Claims, bool) {
	claims, ok := ctx.Value(userContextKey).(*model.Claims)
	return claims, ok
}
