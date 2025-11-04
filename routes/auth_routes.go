package routes

import (
	"github.com/adehusnim37/lihatin-go/controllers"
	"github.com/adehusnim37/lihatin-go/controllers/auth"
	"github.com/adehusnim37/lihatin-go/controllers/auth/admin"
	"github.com/adehusnim37/lihatin-go/controllers/auth/api"
	"github.com/adehusnim37/lihatin-go/controllers/auth/email"
	"github.com/adehusnim37/lihatin-go/middleware"
	"github.com/adehusnim37/lihatin-go/repositories"
	"github.com/gin-gonic/gin"
)

// RegisterAuthRoutes registers all authentication-related routes
func RegisterAuthRoutes(rg *gin.RouterGroup, authController *auth.Controller, userRepo repositories.UserRepository, loginAttemptRepo repositories.LoginAttemptRepository, emailController *email.Controller, baseController interface{}) {
	// Create API key controller
	apiKeyController := api.NewAPIKeyController(baseController.(*controllers.BaseController))
	// Create admin controller
	adminController := admin.NewAdminController(baseController.(*controllers.BaseController))
	// Public authentication routes (no auth required)
	authGroup := rg.Group("/auth")
	{
		// Basic authentication
		authGroup.POST("/login", middleware.RecordLoginAttempt(&loginAttemptRepo), authController.Login)
		authGroup.POST("/register", authController.Register)

		// Password reset flow
		authGroup.Use(middleware.RateLimitMiddleware(5)) // Limit to 5 requests per minute per IP
		authGroup.POST("/forgot-password", authController.ForgotPassword)
		authGroup.GET("/validate-reset/:token", authController.ValidateResetToken) // Token passed as URL param to validate
		authGroup.POST("/reset-password", authController.ResetPassword)

		// Email verification
		authGroup.GET("/verify-email", emailController.VerifyEmail) // Token passed as query param
		authGroup.POST("/resend-verification-email", emailController.ResendVerificationEmail)
		authGroup.GET("/revoke-email-change", emailController.UndoChangeEmail) // Token passed as query param

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
		protectedAuth.PUT("/profile", authController.UpdateProfile)
		protectedAuth.DELETE("/delete-account", authController.DeleteAccount, middleware.RequireEmailVerification())
		protectedAuth.POST("/send-verification-email", emailController.SendVerificationEmails)
		protectedAuth.POST("/change-email", emailController.ChangeEmail)
		protectedAuth.GET("/check-email-change-eligibility", emailController.CheckEmailChangeEligibility)
		protectedAuth.GET("/email-change-history", emailController.GetEmailChangeHistory)
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
		adminAuth.GET("/users", adminController.GetAllUsers)
		adminAuth.GET("/users/:id", adminController.GetUserByID)
		adminAuth.POST("/users/:id/lock", adminController.LockUser)
		adminAuth.POST("/users/:id/unlock", adminController.UnlockUser)
		{
			adminAuth.GET("/login-attempts", adminController.GetLoginAttempts)
			adminAuth.GET("/login-attempts/stats/:email_or_username/:days", adminController.LoginAttemptsStats)
		}
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
	{
		apiKeyGroup.Use(middleware.AuthMiddleware(userRepo))
		apiKeyGroup.GET("/", apiKeyController.GetAPIKeys)
		apiKeyGroup.GET("/:id", apiKeyController.GetAPIKey)
		apiKeyGroup.POST("/", apiKeyController.CreateAPIKey)
		apiKeyGroup.DELETE("/:id", apiKeyController.RevokeAPIKey)
		apiKeyGroup.PUT("/:id", apiKeyController.UpdateAPIKey)
		apiKeyGroup.POST("/:id/refresh", apiKeyController.RefreshAPIKey)
		apiKeyGroup.POST("/:id/activate", apiKeyController.ActivateAPIKey)
		apiKeyGroup.POST("/:id/deactivate", apiKeyController.DeactivateAPIKey)
		apiKeyGroup.GET("/:id/usage", apiKeyController.GetAPIKeyActivityLogs)
	}
}
