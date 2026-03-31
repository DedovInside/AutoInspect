package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

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

func Auth(tokenManager *service.TokenManager, cache denylistChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
		if !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": "unauthorized", "message": "missing bearer token"})
			return
		}

		tokenString := strings.TrimSpace(authHeader[len("Bearer "):])
		claims, err := tokenManager.ParseAccessToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": "unauthorized", "message": "invalid token"})
			return
		}

		if cache != nil {
			denylisted, cacheErr := cache.IsDenylistedJTI(c.Request.Context(), claims.RegisteredClaims.ID)
			if cacheErr != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": "unauthorized", "message": "token check failed"})
				return
			}
			if denylisted {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": "unauthorized", "message": "token revoked"})
				return
			}
		}

		uid, err := uuid.Parse(claims.UserID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": "unauthorized", "message": "invalid token subject"})
			return
		}

		c.Set(string(UserIDContextKey), uid)
		c.Set(string(UserRoleContextKey), claims.Role)
		c.Set(string(AccessJTIContextKey), claims.RegisteredClaims.ID)
		if claims.RegisteredClaims.ExpiresAt != nil {
			c.Set(string(AccessExpContextKey), claims.RegisteredClaims.ExpiresAt.Time)
		} else {
			c.Set(string(AccessExpContextKey), time.Time{})
		}

		ctx := context.WithValue(c.Request.Context(), UserIDContextKey, uid)
		ctx = context.WithValue(ctx, UserRoleContextKey, claims.Role)
		ctx = context.WithValue(ctx, AccessJTIContextKey, claims.RegisteredClaims.ID)
		if claims.RegisteredClaims.ExpiresAt != nil {
			ctx = context.WithValue(ctx, AccessExpContextKey, claims.RegisteredClaims.ExpiresAt.Time)
		} else {
			ctx = context.WithValue(ctx, AccessExpContextKey, time.Time{})
		}
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
