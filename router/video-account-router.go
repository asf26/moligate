package router

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/gin-gonic/gin"
)

// registerVideoAccountRoutes mounts the dedicated CTMOAI account API. These
// routes are intentionally separate from /channel: a video account owns its
// own upstream key-scoped model catalog and is not a generic relay Channel.
func registerVideoAccountRoutes(apiRouter *gin.RouterGroup) {
	adminRoute := apiRouter.Group("/video-accounts")
	adminRoute.Use(middleware.AdminAuth(), middleware.DisableCache())
	{
		adminRoute.GET("", controller.GetVideoAccounts)
		adminRoute.GET("/", controller.GetVideoAccounts)
		adminRoute.GET("/:id", controller.GetVideoAccount)
		adminRoute.POST("", controller.CreateVideoAccount)
		adminRoute.POST("/", controller.CreateVideoAccount)
		adminRoute.PUT("", controller.UpdateVideoAccount)
		adminRoute.PUT("/", controller.UpdateVideoAccount)
		adminRoute.PUT("/:id", controller.UpdateVideoAccount)
		adminRoute.POST("/:id/sync", controller.SyncVideoAccount)
		adminRoute.POST("/:id/refresh", controller.SyncVideoAccount)
		adminRoute.DELETE("/:id", controller.DeleteVideoAccount)
	}

	// The creation UI follows CTMOAI's API shape. It is user-authenticated, so
	// model/account visibility can be filtered by the caller's usable group.
	creationRoute := apiRouter.Group("/video-creation")
	creationRoute.Use(middleware.TokenOrUserAuth(), middleware.DisableCache())
	{
		creationRoute.GET("/catalog", controller.GetVideoCreationCatalog)
		creationRoute.GET("/private-groups/:key/models", controller.GetVideoCreationPrivateGroupModels)
	}
}
