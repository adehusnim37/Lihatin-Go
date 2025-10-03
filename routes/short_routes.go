package routes

import (
	shortlink "github.com/adehusnim37/lihatin-go/controllers/shortlink"
	"github.com/adehusnim37/lihatin-go/middleware"
	"github.com/adehusnim37/lihatin-go/repositories"
	"github.com/gin-gonic/gin"
)

func RegisterShortRoutes(rg *gin.RouterGroup, shortController *shortlink.Controller, userRepo repositories.UserRepository, authRepo *repositories.AuthRepository) {
	shortGroup := rg.Group("/short")
	{
		shortGroup.Use(middleware.RateLimitMiddleware(25))
		shortGroup.Use(middleware.OptionalAuth(userRepo))
		shortGroup.POST("", shortController.Create)
		shortGroup.GET("/:code", shortController.Redirect)
		shortGroup.GET("check/:code", shortController.CheckShortLink)
	}

	// ✅ API ROUTES: Accessible by API key authentication (service-to-service)
	apiShort := rg.Group("api/short")
	{
		apiShort.Use(middleware.AuthRepositoryAPIKeyMiddleware(authRepo))
		apiShort.Use(middleware.RateLimitMiddleware(1000))
		// Higher rate limit for API access
		apiShort.POST("", middleware.CheckPermissionAPIKey(authRepo, []string{"create"}, false), shortController.Create)
		apiShort.GET("/:code", middleware.CheckPermissionAPIKey(authRepo, []string{"read"}, false), shortController.GetShortLink)
		apiShort.PUT("/:code", middleware.CheckPermissionAPIKey(authRepo, []string{"update"}, false), shortController.UpdateShortLink)
		apiShort.GET("", middleware.CheckPermissionAPIKey(authRepo, []string{"read"}, false), shortController.ListShortLinks)
		apiShort.GET("/:code/stats", middleware.CheckPermissionAPIKey(authRepo, []string{"read"}, false), shortController.GetShortLinkStats)
		apiShort.GET("/:code/views", middleware.CheckPermissionAPIKey(authRepo, []string{"read"}, false), shortController.GetShortLinkViewsPaginated)
		apiShort.DELETE("/:code", middleware.CheckPermissionAPIKey(authRepo, []string{"delete"}, false), shortController.DeleteShortLink)
		apiShort.GET("/stats", middleware.CheckPermissionAPIKey(authRepo, []string{"read"}, false), shortController.GetAllStatsShorts)
	}

	// ✅ PROTECTED ROUTES: Accessible by authenticated users (user or admin)

	protectedShort := rg.Group("users/me/shorts")
	{
		protectedShort.Use(middleware.AuthMiddleware(userRepo))
		protectedShort.Use(middleware.RateLimitMiddleware(100))
		protectedShort.POST("", shortController.Create)
		protectedShort.GET("/:code", shortController.GetShortLink)
		protectedShort.PUT("/:code", shortController.UpdateShortLink)
		protectedShort.GET("", shortController.ListShortLinks) // ✅ UNIVERSAL: Auto-detects role and filters accordingly
		protectedShort.GET("/:code/stats", shortController.GetShortLinkStats)
		protectedShort.GET("/:code/views", shortController.GetShortLinkViewsPaginated) // New route for paginated views
		protectedShort.DELETE("/:code", shortController.DeleteShortLink)
		protectedShort.GET("/short/stats", shortController.GetAllStatsShorts)
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
		protectedAdminShort.PUT("/:code", shortController.UpdateShortLink)            // Admin update any short link
		protectedAdminShort.POST("/:code/edotensei", shortController.ReviveShortLink) // Revive deleted short link
		protectedAdminShort.GET("/short/stats", shortController.GetAllStatsShorts)
	}
}
