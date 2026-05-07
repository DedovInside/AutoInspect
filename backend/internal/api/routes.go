package api

import (
	"net/http"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/gin-gonic/gin"

	"github.com/DedovInside/AutoInspect/backend/internal/api/handlers"
	"github.com/DedovInside/AutoInspect/backend/internal/api/middleware"
	"github.com/DedovInside/AutoInspect/backend/internal/service"
)

func NewGinRouter(
	authHandler *handlers.AuthHandler,
	analysisHandler *handlers.AnalysisHandler,
	modelHandler *handlers.ModelHandler,
	modelTrainingRequestHandler *handlers.ModelTrainingRequestHandler,
	carServiceApplicationHandler *handlers.CarServiceApplicationHandler,
	tokenManager *service.TokenManager,
	cache service.SessionCache,
) *gin.Engine {
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

	analysisGroup := router.Group("/v1/analyses")
	analysisGroup.Use(middleware.Auth(tokenManager, cache))

	analysisGroup.POST("", analysisHandler.Submit)
	analysisGroup.GET("/:id", analysisHandler.GetByID)
	analysisGroup.GET("", analysisHandler.ListMine)
	analysisGroup.GET("/:id/images/:idx", analysisHandler.GetPresignedImageURL)

	modelTrainingRequestsGroup := router.Group("/v1/model-training-requests")
	modelTrainingRequestsGroup.Use(middleware.Auth(tokenManager, cache))
	modelTrainingRequestsGroup.POST("", modelTrainingRequestHandler.Create)
	modelTrainingRequestsGroup.GET("", modelTrainingRequestHandler.ListMine)

	carServiceApplicationsGroup := router.Group("/v1/car-service-applications")
	carServiceApplicationsGroup.Use(middleware.Auth(tokenManager, cache))
	carServiceApplicationsGroup.POST("", carServiceApplicationHandler.Create)
	carServiceApplicationsGroup.GET("/current", carServiceApplicationHandler.GetCurrent)
	carServiceApplicationsGroup.GET("", carServiceApplicationHandler.ListMine)

	adminGroup := router.Group("/v1/admin")
	adminGroup.Use(middleware.Auth(tokenManager, cache))
	adminGroup.Use(middleware.RequireRole(domain.RoleAdmin))
	adminGroup.POST("/models", modelHandler.Upload)
	adminGroup.GET("/models", modelHandler.List)
	adminGroup.PATCH("/models/:id/deactivate", modelHandler.Deactivate)
	adminGroup.GET("/model-training-requests", modelTrainingRequestHandler.AdminList)
	adminGroup.PATCH("/model-training-requests/:id/status", modelTrainingRequestHandler.AdminUpdateStatus)

	router.GET("/v1/analyses/ws", middleware.WSAuth(tokenManager, cache), analysisHandler.WSHandler)

	return router
}
