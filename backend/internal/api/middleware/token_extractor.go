package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

type TokenSource int

const (
	TokenSourceNone TokenSource = iota
	TokenSourceHeader
	TokenSourceQuery
	TokenSourceCookie
)

func ExtractToken(c *gin.Context) (string, TokenSource) {
	if auth := c.GetHeader("Authorization"); auth != "" {
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
			token := strings.TrimSpace(parts[1])
			if token != "" {
				return token, TokenSourceHeader
			}
		}
	}

	if token := strings.TrimSpace(c.Query("token")); token != "" {
		return token, TokenSourceQuery
	}

	if cookie, err := c.Cookie("auth_token"); err == nil && cookie != "" {
		return strings.TrimSpace(cookie), TokenSourceCookie
	}

	return "", TokenSourceNone
}
