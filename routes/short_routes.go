package routes

import (
	shortlink "github.com/adehusnim37/lihatin-go/controllers/shortlink"
	"github.com/adehusnim37/lihatin-go/middleware"
	"github.com/adehusnim37/lihatin-go/repositories/authrepo"
	"github.com/adehusnim37/lihatin-go/repositories/userrepo"
	"github.com/gin-gonic/gin"
)

func RegisterShortRoutes(rg *gin.RouterGroup, shortController *shortlink.Controller, userRepo userrepo.UserRepository, userAuthRepo *authrepo.UserAuthRepository, authRepo *authrepo.AuthRepository) {
	shortGroup := rg.Group("/short")
	{
		shortGroup.Use(middleware.RateLimitMiddleware(25, 00, 30)) // Limit to 25 requests per minute for public access
		shortGroup.Use(middleware.OptionalAuth(userRepo))
		shortGroup.POST("", shortController.Create)
		shortGroup.GET("/:code", shortController.Redirect)
		shortGroup.GET("check/:code", shortController.CheckShortLink)
		shortGroup.GET("check/:code/:passcode", shortController.CheckShortLink)
	}

	// ✅ API ROUTES: Accessible by API key authentication (service-to-service)
	apiShort := rg.Group("api/short")
	{
		apiShort.Use(middleware.AuthRepositoryAPIKeyMiddleware(authRepo))
		// All API keys on an account share its tier limit. Check permissions first,
		// then reserve the key's total-use quota only after the rate check passes.
		apiRateLimit := middleware.PremiumRateLimitMiddleware(50, 100)
		apiUsage := middleware.APIKeyUsageMiddleware(authRepo)
		apiShort.POST("", middleware.CheckPermissionAPIKey(authRepo, []string{"write"}, false), apiRateLimit, apiUsage, shortController.Create)
		apiShort.GET("/:code", middleware.CheckPermissionAPIKey(authRepo, []string{"read"}, false), apiRateLimit, apiUsage, shortController.GetShortLink)
		apiShort.PUT("/:code", middleware.CheckPermissionAPIKey(authRepo, []string{"update"}, false), apiRateLimit, apiUsage, shortController.UpdateShortLink)
		apiShort.GET("", middleware.CheckPermissionAPIKey(authRepo, []string{"read"}, false), apiRateLimit, apiUsage, shortController.ListShortLinks)
		apiShort.GET("/:code/stats", middleware.CheckPermissionAPIKey(authRepo, []string{"read"}, false), apiRateLimit, apiUsage, shortController.GetShortLinkStats)
		apiShort.GET("/:code/views", middleware.CheckPermissionAPIKey(authRepo, []string{"read"}, false), apiRateLimit, apiUsage, shortController.GetShortLinkViewsPaginated)
		apiShort.DELETE("/:code", middleware.CheckPermissionAPIKey(authRepo, []string{"delete"}, false), apiRateLimit, apiUsage, shortController.DeleteShortLink)
		apiShort.GET("/stats", middleware.CheckPermissionAPIKey(authRepo, []string{"read"}, false), apiRateLimit, apiUsage, shortController.GetAllStatsShorts)
	}

	// ✅ PROTECTED ROUTES: Accessible by authenticated users (user or admin)

	protectedShort := rg.Group("users/me/shorts")
	{
		protectedShort.Use(middleware.AuthMiddleware(userRepo, userAuthRepo))
		protectedShort.Use(middleware.RateLimitMiddleware(100, 0, 60)) // Limit to 100 requests per minute for authenticated users
		protectedShort.POST("", shortController.Create)
		protectedShort.GET("", shortController.ListShortLinks) // ✅ UNIVERSAL: Auto-detects role and filters accordingly
		protectedShort.GET("/stats", shortController.GetAllStatsShorts)
		protectedShort.GET("/:code/stats", shortController.GetShortLinkStats)
		protectedShort.GET("/:code", shortController.GetShortLink)
		protectedShort.PUT("/:code", shortController.UpdateShortLink)
		protectedShort.DELETE("/:code", shortController.DeleteShortLink)
		protectedShort.GET("/:code/views", shortController.GetShortLinkViewsPaginated) // New route for paginated views
		protectedShort.POST("/:code/toggle-active-inactive", shortController.SwitchActiveInActiveShort)
		protectedShort.DELETE("/:code/passcode", shortController.RemovePasscode)

	}

	// ✅ ADMIN ROUTES: Only accessible by admin users
	protectedAdminShort := rg.Group("admin/shorts")
	{
		protectedAdminShort.Use(middleware.AuthMiddleware(userRepo, userAuthRepo))
		protectedAdminShort.Use(middleware.RequireRole("admin")) // Ensures only admin access
		// ✅ UNIVERSAL ENDPOINT: Same endpoint, but admin gets all data
		protectedAdminShort.GET("", shortController.ListShortLinks)              // Will return all short links for admin
		protectedAdminShort.GET("users/:userID", shortController.ListShortLinks) // Get list of short links for specific user
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
