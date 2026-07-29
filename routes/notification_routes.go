package routes

import (
	"github.com/adehusnim37/lihatin-go/controllers"
	"github.com/adehusnim37/lihatin-go/controllers/notification"
	"github.com/adehusnim37/lihatin-go/middleware"
	"github.com/adehusnim37/lihatin-go/repositories/authrepo"
	"github.com/adehusnim37/lihatin-go/repositories/userrepo"
	"github.com/gin-gonic/gin"
)

func RegisterNotificationRoutes(
	rg *gin.RouterGroup,
	baseController *controllers.BaseController,
	userRepo userrepo.UserRepository,
	userAuthRepo *authrepo.UserAuthRepository,
) {
	controller := notification.NewController(baseController)

	rg.GET("/notifications/unsubscribe", controller.Unsubscribe)
	rg.POST("/notifications/unsubscribe", controller.Unsubscribe)

	protected := rg.Group("/notifications")
	protected.Use(middleware.AuthMiddleware(userRepo, userAuthRepo), middleware.RequireEmailVerification())
	{
		protected.GET("/preferences", controller.GetPreferences)
		protected.PATCH("/preferences", controller.UpdatePreferences)
	}

	admin := rg.Group("/auth/admin/promotional-campaigns")
	admin.Use(middleware.AuthMiddleware(userRepo, userAuthRepo), middleware.AdminAuth(), middleware.RequireEmailVerification())
	{
		admin.GET("", controller.ListCampaigns)
		admin.POST("", controller.CreateCampaign)
		admin.POST("/image", controller.UploadCampaignImage)
		admin.GET("/:id", controller.GetCampaign)
		admin.GET("/:id/deliveries", controller.ListCampaignDeliveries)
		admin.PUT("/:id", controller.UpdateCampaign)
		admin.POST("/:id/schedule", controller.ScheduleCampaign)
		admin.POST("/:id/cancel", controller.CancelCampaign)
		admin.DELETE("/:id", controller.DeleteCampaign)
	}
}
