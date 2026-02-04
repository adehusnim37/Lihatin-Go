package routes

import (
	"github.com/adehusnim37/lihatin-go/controllers"
	"github.com/adehusnim37/lihatin-go/controllers/auth"
	"github.com/adehusnim37/lihatin-go/controllers/auth/admin"
	"github.com/adehusnim37/lihatin-go/controllers/auth/api"
	"github.com/adehusnim37/lihatin-go/controllers/auth/email"
	loginattempts "github.com/adehusnim37/lihatin-go/controllers/auth/login-attempts"
	"github.com/adehusnim37/lihatin-go/controllers/auth/totp"
	"github.com/adehusnim37/lihatin-go/middleware"
	"github.com/adehusnim37/lihatin-go/repositories"
	"github.com/gin-gonic/gin"
)

// RegisterAuthRoutes registers all authentication-related routes
func RegisterAuthRoutes(rg *gin.RouterGroup, authController *auth.Controller, userRepo repositories.UserRepository, loginAttemptRepo repositories.LoginAttemptRepository, emailController *email.Controller, totpController *totp.Controller, baseController *controllers.BaseController) {
	// Create API key controller
	apiKeyController := api.NewAPIKeyController(baseController)
	// Create admin controller
	adminController := admin.NewAdminController(baseController)
	// Create login attempts controller
	loginAttemptsController := loginattempts.NewLoginAttemptsController(baseController)
	// Public authentication routes (no auth required)
	authGroup := rg.Group("/auth")
	{
		// Basic authentication
		authGroup.POST("/login", middleware.RecordLoginAttempt(&loginAttemptRepo), authController.Login)
		authGroup.POST("/register", authController.Register)

		// TOTP Login verification (no auth required - this IS the auth step)
		authGroup.POST("/verify-totp-login", totpController.VerifyTOTPLogin)

		// Password reset flow
		authGroup.Use(middleware.RateLimitMiddleware(5)) // Limit to 5 requests per minute per IP
		authGroup.POST("/forgot-password", authController.ForgotPassword)
		authGroup.GET("/validate-reset", authController.ValidateResetToken) // Token passed as URL param to validate
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
		// Current user info (for cookie-based auth check)
		protectedAuth.GET("/me", authController.GetCurrentUser)

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
		protectedAuth.GET("/check-verification-email", emailController.CheckVerificationEmail)
		// TOTP (Two-Factor Authentication) management
		totpGroup := protectedAuth.Group("/totp")
		{
			totpGroup.POST("/setup", totpController.SetupTOTP)
			totpGroup.POST("/verify", totpController.VerifyTOTP)
			totpGroup.POST("/disable", totpController.DisableTOTP)
			totpGroup.GET("/recovery-codes", totpController.GetRecoveryCodes)
			totpGroup.POST("/regenerate-recovery-codes", totpController.RegenerateRecoveryCodes)
		}

		// Login Attempts - accessible by all authenticated users
		loginAttemptsGroup := protectedAuth.Group("/login-attempts")
		{
			loginAttemptsGroup.GET("", loginAttemptsController.GetLoginAttempts)
			loginAttemptsGroup.GET("/stats/:email_or_username/:days", loginAttemptsController.LoginAttemptsStats)
			loginAttemptsGroup.GET("/recent-activity", loginAttemptsController.GetRecentActivity)
			loginAttemptsGroup.GET("/attempts-by-hour", loginAttemptsController.GetAttemptsByHour)
			loginAttemptsGroup.GET("/:id", loginAttemptsController.GetLoginAttempt)
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

		// Admin can access all login attempts (not filtered by user)
		adminLoginAttemptsGroup := adminAuth.Group("/login-attempts")
		{
			adminLoginAttemptsGroup.GET("", loginAttemptsController.GetLoginAttempts)
			adminLoginAttemptsGroup.GET("/stats/:email_or_username/:days", loginAttemptsController.LoginAttemptsStats)
			adminLoginAttemptsGroup.GET("/recent-activity", loginAttemptsController.GetRecentActivity)
			adminLoginAttemptsGroup.GET("/attempts-by-hour", loginAttemptsController.GetAttemptsByHour)
			adminLoginAttemptsGroup.GET("/top-failed-ips", loginAttemptsController.GetTopFailedIPs)
			adminLoginAttemptsGroup.GET("/suspicious-activity", loginAttemptsController.GetSuspiciousActivity)
			adminLoginAttemptsGroup.GET("/:id", loginAttemptsController.GetLoginAttempt)
			// adminLoginAttemptsGroup.DELETE("/:id", loginAttemptsController.DeleteLoginAttempt)
			// adminLoginAttemptsGroup.DELETE("", loginAttemptsController.ClearLoginAttempts)
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
