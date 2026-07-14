package api

import (
	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/DedovInside/AutoInspect/backend/internal/observability"
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
	carServiceProfileHandler *handlers.CarServiceProfileHandler,
	repairRequestHandler *handlers.RepairRequestHandler,
	carServiceReviewHandler *handlers.CarServiceReviewHandler,
	vehicleCatalogHandler *handlers.VehicleCatalogHandler,
	tokenManager *service.TokenManager,
	cache service.SessionCache,
	healthChecker *observability.HealthChecker,
	corsAllowedOrigins []string,
) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.RequestLogger())
	router.Use(observability.HTTPMetricsMiddleware())
	router.Use(middleware.CORS(corsAllowedOrigins))

	router.GET("/metrics", observability.MetricsHandler())
	router.GET("/health", healthChecker.SummaryGin)
	router.GET("/health/live", healthChecker.LiveGin)
	router.GET("/health/ready", healthChecker.ReadyGin)

	router.POST("/v1/auth/refresh", authHandler.Refresh)
	router.GET("/v1/auth/yandex/start", authHandler.YandexStart)
	router.GET("/v1/auth/yandex/callback", authHandler.YandexCallback)
	router.POST("/v1/auth/oauth/yandex", authHandler.YandexExchange)

	authGroup := router.Group("/v1/auth")
	authGroup.Use(middleware.Auth(tokenManager, cache))
	authGroup.GET("/me", authHandler.Me)
	authGroup.PATCH("/me", authHandler.UpdateMe)
	authGroup.POST("/logout", authHandler.Logout)

	analysisGroup := router.Group("/v1/analyses")
	analysisGroup.Use(middleware.Auth(tokenManager, cache))

	analysisGroup.POST("", analysisHandler.Submit)
	analysisGroup.GET("/:id", analysisHandler.GetByID)
	analysisGroup.GET("", analysisHandler.ListMine)
	analysisGroup.GET("/:id/images/:idx", analysisHandler.GetPresignedImageURL)
	analysisGroup.GET("/:id/car-services", analysisHandler.FindMatchingCarServices)

	modelTrainingRequestsGroup := router.Group("/v1/model-training-requests")
	modelTrainingRequestsGroup.Use(middleware.Auth(tokenManager, cache))
	modelTrainingRequestsGroup.POST("", modelTrainingRequestHandler.Create)
	modelTrainingRequestsGroup.GET("", modelTrainingRequestHandler.ListMine)

	modelsGroup := router.Group("/v1/models")
	modelsGroup.Use(middleware.Auth(tokenManager, cache))
	modelsGroup.GET("/specialized", modelHandler.ListAvailableSpecialized)

	vehicleCatalogGroup := router.Group("/v1/vehicle-catalog")
	vehicleCatalogGroup.Use(middleware.Auth(tokenManager, cache))
	vehicleCatalogGroup.GET("/makes", vehicleCatalogHandler.ListMakes)
	vehicleCatalogGroup.GET("/makes/:make_id/models", vehicleCatalogHandler.ListModels)
	vehicleCatalogGroup.GET("/models/:model_id/generations", vehicleCatalogHandler.ListGenerations)

	carServiceApplicationsGroup := router.Group("/v1/car-service-applications")
	carServiceApplicationsGroup.Use(middleware.Auth(tokenManager, cache))
	carServiceApplicationsGroup.POST("", carServiceApplicationHandler.Create)
	carServiceApplicationsGroup.GET("/current", carServiceApplicationHandler.GetCurrent)
	carServiceApplicationsGroup.GET("", carServiceApplicationHandler.ListMine)

	repairRequestsGroup := router.Group("/v1/repair-requests")
	repairRequestsGroup.Use(middleware.Auth(tokenManager, cache))
	repairRequestsGroup.POST("", repairRequestHandler.Create)
	repairRequestsGroup.GET("", repairRequestHandler.ListMine)
	repairRequestsGroup.GET("/:id", repairRequestHandler.GetMine)
	repairRequestsGroup.PATCH("/:id/cancel", repairRequestHandler.CancelMine)
	repairRequestsGroup.POST("/:id/review", carServiceReviewHandler.CreateForRepairRequest)
	repairRequestsGroup.GET("/:id/review", carServiceReviewHandler.GetByRepairRequest)
	repairRequestsGroup.PATCH("/:id/review", carServiceReviewHandler.UpdateForRepairRequest)
	repairRequestsGroup.DELETE("/:id/review", carServiceReviewHandler.DeleteForRepairRequest)

	reviewsGroup := router.Group("/v1/reviews")
	reviewsGroup.Use(middleware.Auth(tokenManager, cache))
	reviewsGroup.GET("/mine", carServiceReviewHandler.ListMine)

	carServicesGroup := router.Group("/v1/car-services")
	carServicesGroup.Use(middleware.Auth(tokenManager, cache))
	carServicesGroup.GET("/:id/reviews", carServiceReviewHandler.ListByCarService)

	carServiceProfileGroup := router.Group("/v1/car-service/profile")
	carServiceProfileGroup.Use(middleware.Auth(tokenManager, cache))
	carServiceProfileGroup.Use(middleware.RequireRole(domain.RoleCarService))
	carServiceProfileGroup.GET("", carServiceProfileHandler.GetMine)
	carServiceProfileGroup.PATCH("", carServiceProfileHandler.UpdateMine)
	carServiceProfileGroup.PATCH("/active", carServiceProfileHandler.SetActive)
	carServiceProfileGroup.POST("/images", carServiceProfileHandler.UploadImage)
	carServiceProfileGroup.GET("/images", carServiceProfileHandler.ListImages)
	carServiceProfileGroup.PATCH("/images/:id/primary", carServiceProfileHandler.SetPrimaryImage)
	carServiceProfileGroup.DELETE("/images/:id", carServiceProfileHandler.DeleteImage)
	carServiceProfileGroup.GET("/specialization-options", carServiceProfileHandler.ListSpecializationOptions)
	carServiceProfileGroup.GET("/specializations", carServiceProfileHandler.ListSpecializations)
	carServiceProfileGroup.PUT("/specializations", carServiceProfileHandler.ReplaceSpecializations)

	carServiceRepairRequestsGroup := router.Group("/v1/car-service/repair-requests")
	carServiceRepairRequestsGroup.Use(middleware.Auth(tokenManager, cache))
	carServiceRepairRequestsGroup.Use(middleware.RequireRole(domain.RoleCarService))
	carServiceRepairRequestsGroup.GET("", repairRequestHandler.ListIncoming)
	carServiceRepairRequestsGroup.GET("/:id", repairRequestHandler.GetIncomingDetails)
	carServiceRepairRequestsGroup.PATCH("/:id/accept", repairRequestHandler.AcceptIncoming)
	carServiceRepairRequestsGroup.PATCH("/:id/reject", repairRequestHandler.RejectIncoming)
	carServiceRepairRequestsGroup.PATCH("/:id/complete", repairRequestHandler.CompleteIncoming)

	adminGroup := router.Group("/v1/admin")
	adminGroup.Use(middleware.Auth(tokenManager, cache))
	adminGroup.Use(middleware.RequireRole(domain.RoleAdmin))
	adminGroup.POST("/models", modelHandler.Upload)
	adminGroup.GET("/models", modelHandler.List)
	adminGroup.PATCH("/models/:id/deactivate", modelHandler.Deactivate)
	adminGroup.GET("/model-training-requests", modelTrainingRequestHandler.AdminList)
	adminGroup.PATCH("/model-training-requests/:id/status", modelTrainingRequestHandler.AdminUpdateStatus)
	adminGroup.GET("/car-service-applications", carServiceApplicationHandler.AdminList)
	adminGroup.PATCH("/car-service-applications/:id/approve", carServiceApplicationHandler.AdminApprove)
	adminGroup.PATCH("/car-service-applications/:id/reject", carServiceApplicationHandler.AdminReject)
	adminGroup.GET("/vehicle-catalog/makes", vehicleCatalogHandler.AdminListMakes)
	adminGroup.POST("/vehicle-catalog/makes", vehicleCatalogHandler.AdminCreateMake)
	adminGroup.PATCH("/vehicle-catalog/makes/:id", vehicleCatalogHandler.AdminUpdateMake)
	adminGroup.PATCH("/vehicle-catalog/makes/:id/active", vehicleCatalogHandler.AdminSetMakeActive)
	adminGroup.GET("/vehicle-catalog/makes/:make_id/models", vehicleCatalogHandler.AdminListModels)
	adminGroup.POST("/vehicle-catalog/models", vehicleCatalogHandler.AdminCreateModel)
	adminGroup.PATCH("/vehicle-catalog/models/:id", vehicleCatalogHandler.AdminUpdateModel)
	adminGroup.PATCH("/vehicle-catalog/models/:id/active", vehicleCatalogHandler.AdminSetModelActive)
	adminGroup.GET("/vehicle-catalog/models/:model_id/generations", vehicleCatalogHandler.AdminListGenerations)
	adminGroup.POST("/vehicle-catalog/generations", vehicleCatalogHandler.AdminCreateGeneration)
	adminGroup.PATCH("/vehicle-catalog/generations/:id", vehicleCatalogHandler.AdminUpdateGeneration)
	adminGroup.PATCH("/vehicle-catalog/generations/:id/active", vehicleCatalogHandler.AdminSetGenerationActive)

	router.GET("/v1/analyses/ws", middleware.WSAuth(tokenManager, cache), analysisHandler.WSHandler)

	return router
}
