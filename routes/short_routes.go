package routes

import (
	shortlink "github.com/adehusnim37/lihatin-go/controllers/shortlink"
	"github.com/adehusnim37/lihatin-go/middleware"
	"github.com/adehusnim37/lihatin-go/repositories"
	"github.com/gin-gonic/gin"
)

func RegisterShortRoutes(rg *gin.RouterGroup, shortController *shortlink.Controller, userRepo repositories.UserRepository) {
	shortGroup := rg.Group("/short")
	{
		shortGroup.Use(middleware.RateLimitMiddleware(20))
		shortGroup.Use(middleware.OptionalAuth(userRepo))
		shortGroup.POST("", shortController.Create)
		shortGroup.GET("/:code", shortController.Redirect)
	}

	protectedShort := rg.Group("users/me/shorts")
	{
		protectedShort.Use(middleware.AuthMiddleware(userRepo))
		protectedShort.Use(middleware.RateLimitMiddleware(100))
		protectedShort.GET("/:code", shortController.GetShortLink)
		protectedShort.PUT("/:code", shortController.UpdateShortLink)
		protectedShort.GET("", shortController.ListShortLinks) // ✅ UNIVERSAL: Auto-detects role and filters accordingly
		protectedShort.GET("/:code/stats", shortController.GetShortLinkStats)
		protectedShort.GET("/:code/views", shortController.GetShortLinkViewsPaginated) // New route for paginated views
		protectedShort.DELETE("/:code", shortController.DeleteShortLink)
	}

	// ✅ ADMIN ROUTES: Only accessible by admin users
	protectedAdminShort := rg.Group("admin/shorts")
	{
		protectedAdminShort.Use(middleware.AuthMiddleware(userRepo))
		protectedAdminShort.Use(middleware.RequireRole("admin")) // Ensures only admin access

		// ✅ UNIVERSAL ENDPOINT: Same endpoint, but admin gets all data
		protectedAdminShort.GET("", shortController.ListShortLinks) // Will return all short links for admin
		protectedAdminShort.GET("/:code/views", shortController.GetShortLinkViewsPaginated)
		protectedAdminShort.DELETE("/:code", shortController.DeleteShortLink)
		protectedAdminShort.DELETE("/bulk-delete", shortController.AdminBulkDeleteShortLinks)
		protectedAdminShort.PUT("/:code/banned", shortController.AdminBannedShortLink)
		protectedAdminShort.PUT("/:code/unban", shortController.AdminUnbanShortLink)
		protectedAdminShort.POST("/:code/edotensei", shortController.ReviveShortLink) // Revive deleted short link
		// protectedAdminShort.GET("stats/:code", shortController.AdminGetShortLinkStats)
		// protectedAdminShort.GET("/stats", shortController.AdminGetAllShortLinksStats)
		// protectedAdminShort.GET("/export", shortController.AdminExportShortLinks)
	}
}
