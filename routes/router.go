package routes

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/controllers"
	"github.com/adehusnim37/lihatin-go/controllers/auth"
	"github.com/adehusnim37/lihatin-go/controllers/auth/email"
	"github.com/adehusnim37/lihatin-go/controllers/auth/totp"
	"github.com/adehusnim37/lihatin-go/controllers/logger"
	"github.com/adehusnim37/lihatin-go/controllers/shortlink"
	pkgauth "github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
	"github.com/adehusnim37/lihatin-go/internal/pkg/csrf"
	"github.com/adehusnim37/lihatin-go/middleware"
	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/repositories/authrepo"
	"github.com/adehusnim37/lihatin-go/repositories/loggerrepo"
	"github.com/adehusnim37/lihatin-go/repositories/userrepo"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func SetupRouter(validate *validator.Validate) *gin.Engine {
	// Inisialisasi router Gin default (sudah include logger & recovery middleware)
	r := gin.Default()

	// Apply CORS middleware first
	r.Use(pkgauth.CORSMiddleware())

	// Apply CSRF middleware (only in production)
	env := config.GetEnvOrDefault(config.Env, "development")
	if env == "production" {
		csrfOpts := csrf.DefaultOptions()
		// Skip CSRF untuk public routes yang tidak memerlukan session
		csrfOpts.SkipPaths = []string{
			// Webhook & Public API
			"/v1/webhook",
			"/v1/public",
			"/v1/health",
			// Public Auth Endpoints (no session yet)
			"/v1/auth/register",
			"/v1/auth/login",
			"/v1/auth/forgot-password",
			"/v1/auth/reset-password",
			"/v1/auth/verify-email",
			"/v1/auth/resend-verification-email",
			"/v1/auth/revoke-email-change",
			"/v1/auth/refresh-token",
			"/v1/auth/verify-totp-login",
			"/v1/auth/validate-reset",
		}
		r.Use(csrf.Middleware(csrfOpts))
	}

	// Initialize GORM for new auth repositories
	dsn := config.GetRequiredEnv(config.EnvDatabaseURL)
	gormDB, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to database with GORM: " + err.Error())
	}

	// Create base controller with GORM for auth
	baseController := controllers.NewBaseControllerWithGorm(gormDB, validate)

	// Inisialisasi controller spesifik
	authController := auth.NewAuthController(baseController)
	loggerController := logger.NewLoggerController(baseController)
	shortController := shortlink.NewController(baseController)
	emailController := email.NewController(baseController)
	totpController := totp.NewController(baseController)

	// Setup repositories for middleware
	loggerRepo := loggerrepo.NewLoggerRepository(gormDB)
	userRepo := userrepo.NewUserRepository(gormDB) // Changed to use GORM
	userAuthRepo := authrepo.NewUserAuthRepository(gormDB)
	loginAttemptRepo := authrepo.NewLoginAttemptRepository(gormDB)
	authRepo := authrepo.NewAuthRepository(gormDB) // Add auth repository for API key middleware

	// Apply global middleware for activity logging
	r.Use(middleware.ActivityLogger(loggerRepo))

	// Definisikan route untuk user, auth, dan logger
	v1 := r.Group("/v1")

	// CSRF Token endpoint (untuk SPA fetch token)
	v1.GET("/csrf-token", func(c *gin.Context) {
		token := csrf.GetMaskedToken(c)
		c.JSON(http.StatusOK, common.APIResponse{
			Success: true,
			Data: gin.H{
				"csrfToken": token,
			},
			Message: "CSRF token generated",
			Error:   nil,
		})
	})

	RegisterAuthRoutes(v1, authController, userRepo, userAuthRepo, *loginAttemptRepo, emailController, totpController, baseController)
	RegisterLoggerRoutes(v1, userRepo, userAuthRepo, loggerController)
	RegisterShortRoutes(v1, shortController, userRepo, userAuthRepo, authRepo)

	// Route health check
	v1.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, common.APIResponse{
			Success: true,
			Data:    "OK",
			Message: "Health check passed",
			Error:   nil,
		})
	})

	return r
}
