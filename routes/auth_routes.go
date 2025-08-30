package routes

import (
	"github.com/adehusnim37/lihatin-go/controllers/auth"
	"github.com/adehusnim37/lihatin-go/middleware"
	"github.com/adehusnim37/lihatin-go/repositories"
	"github.com/gin-gonic/gin"
)

// RegisterAuthRoutes registers all authentication-related routes
func RegisterAuthRoutes(rg *gin.RouterGroup, authController *auth.Controller, userRepo repositories.UserRepository, loginAttemptRepo repositories.LoginAttemptRepository) {
	// Public authentication routes (no auth required)
	authGroup := rg.Group("/auth")
	{
		// Basic authentication
		authGroup.POST("/login", authController.Login, middleware.RecordLoginAttempt(&loginAttemptRepo))
		authGroup.POST("/register", authController.Register)

		// Password reset flow
		authGroup.POST("/forgot-password", authController.ForgotPassword)
		authGroup.POST("/reset-password", authController.ResetPassword)

		// Email verification
		authGroup.GET("/verify-email", authController.VerifyEmail)

		// Token refresh
		authGroup.POST("/refresh-token", authController.RefreshToken)
	}

	// Protected authentication routes (require JWT auth)
	protectedAuth := rg.Group("/auth")
	protectedAuth.Use(middleware.AuthMiddleware(userRepo))
	{
		// Account management
		protectedAuth.POST("/logout", authController.Logout)
		protectedAuth.POST("/change-password", authController.ChangePassword)
		protectedAuth.GET("/profile", authController.GetProfile)

		// TOTP (Two-Factor Authentication) management
		totpGroup := protectedAuth.Group("/totp")
		{
			totpGroup.POST("/setup", authController.SetupTOTP)
			totpGroup.POST("/verify", authController.VerifyTOTP)
			totpGroup.POST("/disable", authController.DisableTOTP)
			totpGroup.GET("/recovery-codes", authController.GetRecoveryCodes)
			totpGroup.POST("/regenerate-recovery-codes", authController.RegenerateRecoveryCodes)
		}

		// Email verification for authenticated users
		protectedAuth.POST("/send-verification-email", authController.SendVerificationEmail)
	}

	// Email verification routes that require email verification
	emailVerifiedGroup := rg.Group("/auth")
	emailVerifiedGroup.Use(middleware.AuthMiddleware(userRepo), middleware.RequireEmailVerification())
	{
		// Premium features that require email verification
		emailVerifiedGroup.GET("/premium-status", authController.GetPremiumStatus)
	}

	// Admin-only routes (admin and super_admin access)
	adminAuth := rg.Group("/auth/admin")
	adminAuth.Use(middleware.AuthMiddleware(userRepo), middleware.AdminAuth())
	{
		adminAuth.GET("/users", authController.GetAllUsers)
		adminAuth.POST("/users/:id/lock", authController.LockUser)
		adminAuth.POST("/users/:id/unlock", authController.UnlockUser)
		adminAuth.GET("/login-attempts", authController.GetLoginAttempts)
	}

	// Super Admin-only routes (super_admin access only)
	superAdminAuth := rg.Group("/auth/super-admin")
	superAdminAuth.Use(middleware.AuthMiddleware(userRepo), middleware.SuperAdminAuth())
	{
		// Add super admin specific routes here if needed
		// Example: superAdminAuth.POST("/create-admin", authController.CreateAdmin)
	}

	// API Key management
	apiKeyGroup := rg.Group("/api-keys")
	apiKeyGroup.Use(middleware.AuthMiddleware(userRepo))
	{
		apiKeyGroup.GET("/", authController.GetAPIKeys)
		apiKeyGroup.POST("/", authController.CreateAPIKey)
		apiKeyGroup.DELETE("/:id", authController.RevokeAPIKey)
		apiKeyGroup.PUT("/:id", authController.UpdateAPIKey)
	}
}
