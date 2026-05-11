package middleware

import (
	"net/http"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/gin-gonic/gin"
)

func RequireRole(allowedRoles ...domain.Role) gin.HandlerFunc {
	allowed := make(map[domain.Role]struct{}, len(allowedRoles))
	for _, role := range allowedRoles {
		allowed[role] = struct{}{}
	}

	return func(c *gin.Context) {
		role, ok := UserRoleFromContext(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "unauthorized",
				"message": "missing user role",
			})
			return
		}

		if _, exists := allowed[role]; !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    "forbidden",
				"message": "insufficient role",
			})
			return
		}

		c.Next()
	}
}

func UserRoleFromContext(c *gin.Context) (domain.Role, bool) {
	value := c.Request.Context().Value(UserRoleContextKey)
	switch role := value.(type) {
	case domain.Role:
		return role, true
	case string:
		return domain.Role(role), role != ""
	default:
		return "", false
	}
}
