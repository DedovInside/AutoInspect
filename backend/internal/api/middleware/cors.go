package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	corsAllowMethods = "GET,POST,PUT,PATCH,DELETE,OPTIONS"
	corsAllowHeaders = "Authorization,Content-Type,Idempotency-Key"
)

func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	allowAll := false
	for _, origin := range allowedOrigins {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		if origin == "*" {
			allowAll = true
			continue
		}
		allowed[origin] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && isAllowedOrigin(origin, allowed, allowAll) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Methods", corsAllowMethods)
			c.Header("Access-Control-Allow-Headers", corsAllowHeaders)
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func isAllowedOrigin(origin string, allowed map[string]struct{}, allowAll bool) bool {
	if allowAll {
		return true
	}
	_, ok := allowed[origin]
	return ok
}
