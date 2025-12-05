package routes

import (
	"net/http"
	"strings"

	"github.com/adehusnim37/lihatin-go/controllers"
	"github.com/adehusnim37/lihatin-go/controllers/auth"
	"github.com/adehusnim37/lihatin-go/controllers/auth/email"
	"github.com/adehusnim37/lihatin-go/controllers/auth/totp"
	"github.com/adehusnim37/lihatin-go/controllers/logger"
	"github.com/adehusnim37/lihatin-go/controllers/shortlink"
	"github.com/adehusnim37/lihatin-go/middleware"
	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/repositories"
	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// CORS middleware
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get allowed origins from environment or use default
		allowedOrigins := config.GetEnvOrDefault(config.EnvAllowedOrigins, "http://localhost:3000,http://localhost:3001")
		origins := strings.Split(allowedOrigins, ",")

		origin := c.Request.Header.Get("Origin")

		// Check if origin is allowed
		originAllowed := false
		for _, allowedOrigin := range origins {
			if strings.TrimSpace(allowedOrigin) == origin {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				originAllowed = true
				break
			}
		}

		// If no specific origin matched but we have origins configured, still allow for development
		if !originAllowed && origin != "" {
			// For development, allow localhost origins
			if strings.Contains(origin, "localhost") || strings.Contains(origin, "127.0.0.1") {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			}
		}

		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-API-Key")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func SetupRouter(validate *validator.Validate) *gin.Engine {
	// Inisialisasi router Gin default (sudah include logger & recovery middleware)
	r := gin.Default()

	// Apply CORS middleware first
	r.Use(CORSMiddleware())

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

	// Initialize login attempts cleanup scheduler
	// log.Println("Starting login attempts cleanup scheduler...")
	// go jobs.RunCleanupScheduler(context.Background(), loginAttemptRepo, 24*time.Hour, 0)
	// log.Println("✅ Login attempts cleanup scheduler succeeded.")

	// Definisikan route untuk user, auth, dan logger
	v1 := r.Group("/v1")
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
