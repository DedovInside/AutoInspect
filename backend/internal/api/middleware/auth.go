package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/service"
	"github.com/google/uuid"
)

type contextKey string

const (
	UserIDContextKey    contextKey = "auth_user_id"
	UserRoleContextKey  contextKey = "auth_user_role"
	AccessJTIContextKey contextKey = "auth_access_jti"
	AccessExpContextKey contextKey = "auth_access_exp"
)

type denylistChecker interface {
	IsDenylistedJTI(ctx context.Context, jti string) (bool, error)
}

func Auth(tokenManager *service.TokenManager, cache denylistChecker) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
			if !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
				http.Error(w, "missing bearer token", http.StatusUnauthorized)
				return
			}

			tokenString := strings.TrimSpace(authHeader[len("Bearer "):])
			claims, err := tokenManager.ParseAccessToken(tokenString)
			if err != nil {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			if cache != nil {
				denylisted, cacheErr := cache.IsDenylistedJTI(r.Context(), claims.RegisteredClaims.ID)
				if cacheErr != nil {
					http.Error(w, "token check failed", http.StatusUnauthorized)
					return
				}
				if denylisted {
					http.Error(w, "token revoked", http.StatusUnauthorized)
					return
				}
			}

			uid, err := uuid.Parse(claims.UserID)
			if err != nil {
				http.Error(w, "invalid token subject", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDContextKey, uid)
			ctx = context.WithValue(ctx, UserRoleContextKey, claims.Role)
			ctx = context.WithValue(ctx, AccessJTIContextKey, claims.RegisteredClaims.ID)
			if claims.RegisteredClaims.ExpiresAt != nil {
				ctx = context.WithValue(ctx, AccessExpContextKey, claims.RegisteredClaims.ExpiresAt.Time)
			} else {
				ctx = context.WithValue(ctx, AccessExpContextKey, time.Time{})
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
