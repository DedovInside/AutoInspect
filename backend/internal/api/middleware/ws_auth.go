package middleware

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/DedovInside/AutoInspect/backend/internal/service"
)

func WSAuth(tokenManager *service.TokenManager, cache denylistChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, _ := ExtractToken(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "unauthorized",
				"message": "JWT token required (?token=... or Authorization: Bearer ...)",
			})
			return
		}

		claims, err := tokenManager.ParseAccessToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "invalid_token",
				"message": "Failed to parse or verify token",
			})
			return
		}

		if cache != nil {
			denylisted, cacheErr := cache.IsDenylistedJTI(c.Request.Context(), claims.ID)
			if cacheErr != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"code":    "unauthorized",
					"message": "token check failed",
				})
				return
			}
			if denylisted {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"code":    "unauthorized",
					"message": "token revoked",
				})
				return
			}
		}

		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "invalid_token",
				"message": "Invalid user ID in token",
			})
			return
		}

		ctx := c.Request.Context()
		ctx = contextWithUserID(ctx, userID)
		ctx = contextWithUserRole(ctx, string(claims.Role))
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

func contextWithUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, UserIDContextKey, userID)
}

func contextWithUserRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, UserRoleContextKey, role)
}

func UserIDFromContext(c *gin.Context) (uuid.UUID, bool) {
	ctx := c.Request.Context()
	uid, ok := ctx.Value(UserIDContextKey).(uuid.UUID)
	return uid, ok
}
