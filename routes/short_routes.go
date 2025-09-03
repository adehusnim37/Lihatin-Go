package routes

import (
	shortlink "github.com/adehusnim37/lihatin-go/controllers/short-link"
	"github.com/adehusnim37/lihatin-go/middleware"
	"github.com/adehusnim37/lihatin-go/repositories"
	"github.com/gin-gonic/gin"
)

func RegisterShortRoutes(rg *gin.RouterGroup, shortController *shortlink.Controller, userRepo repositories.UserRepository) {
	shortGroup := rg.Group("/short")
	{
		shortGroup.Use(middleware.RateLimitMiddleware(20))
		shortGroup.POST("", middleware.OptionalAuth(userRepo), shortController.Create)
		shortGroup.GET("/:code", shortController.Redirect)
	}

	protectedShort := rg.Group("users/me/shorts")
	{
		protectedShort.Use(middleware.RateLimitMiddleware(100))
		protectedShort.Use(middleware.AuthMiddleware(userRepo))
		protectedShort.GET("/:code", shortController.GetShortLink)
		protectedShort.PUT("/:code", shortController.UpdateShortLink)
		protectedShort.GET("", shortController.ListUserShortLinks)
		protectedShort.GET("stats/:code", shortController.GetShortLinkStats)
		protectedShort.DELETE("/:code", shortController.DeleteShortLink)
	}

	protectedAdminShort := rg.Group("admin/shorts")
	{
		protectedAdminShort.Use(middleware.AuthMiddleware(userRepo))

	}
}
