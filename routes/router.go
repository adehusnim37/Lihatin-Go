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
	"github.com/adehusnim37/lihatin-go/repositories"
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
	env := config.GetEnvOrDefault("ENV", "development")
	if env == "production" {
		csrfOpts := csrf.DefaultOptions()
		// Skip CSRF untuk webhook dan API key authenticated routes
		csrfOpts.SkipPaths = []string{
			"/v1/webhook",
			"/v1/public",
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
	loggerRepo := repositories.NewLoggerRepository(gormDB)
	userRepo := repositories.NewUserRepository(gormDB) // Changed to use GORM
	loginAttemptRepo := repositories.NewLoginAttemptRepository(gormDB)
	authRepo := repositories.NewAuthRepository(gormDB) // Add auth repository for API key middleware

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

	RegisterAuthRoutes(v1, authController, userRepo, *loginAttemptRepo, emailController, totpController, baseController)
	RegisterLoggerRoutes(v1, loggerController)
	RegisterShortRoutes(v1, shortController, userRepo, authRepo)

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
