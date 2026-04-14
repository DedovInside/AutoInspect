package api

import (
	"net/http"

	"github.com/DedovInside/AutoInspect/backend/internal/api/handlers"
	"github.com/DedovInside/AutoInspect/backend/internal/api/middleware"
	"github.com/DedovInside/AutoInspect/backend/internal/service"
	"github.com/gin-gonic/gin"
)

func NewGinRouter(authHandler *handlers.AuthHandler, tokenManager *service.TokenManager, cache service.SessionCache) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())

	router.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	router.POST("/v1/auth/refresh", authHandler.Refresh)
	router.GET("/v1/auth/yandex/start", authHandler.YandexStart)
	router.GET("/v1/auth/yandex/callback", authHandler.YandexCallback)
	router.POST("/v1/auth/oauth/yandex", authHandler.YandexExchange)

	authGroup := router.Group("/v1/auth")
	authGroup.Use(middleware.Auth(tokenManager, cache))
	authGroup.GET("/me", authHandler.Me)
	authGroup.POST("/logout", authHandler.Logout)

	return router
}
