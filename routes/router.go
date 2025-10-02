package routes

import (
	"database/sql"
	"net/http"

	"github.com/adehusnim37/lihatin-go/controllers"
	"github.com/adehusnim37/lihatin-go/controllers/auth"
	"github.com/adehusnim37/lihatin-go/controllers/auth/email"
	"github.com/adehusnim37/lihatin-go/controllers/logger"
	"github.com/adehusnim37/lihatin-go/controllers/shortlink"
	"github.com/adehusnim37/lihatin-go/controllers/user"
	"github.com/adehusnim37/lihatin-go/middleware"
	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/repositories"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func SetupRouter(db *sql.DB, validate *validator.Validate) *gin.Engine {
	// Inisialisasi router Gin default (sudah include logger & recovery middleware)
	r := gin.Default()

	// Initialize GORM for new auth repositories
	dsn := utils.GetRequiredEnv(utils.EnvDatabaseURL)
	gormDB, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to database with GORM: " + err.Error())
	}

	// Create base controller with GORM for auth
	baseController := controllers.NewBaseControllerWithGorm(db, gormDB, validate)

	// Inisialisasi controller spesifik
	userController := user.NewController(baseController)
	authController := auth.NewAuthController(baseController)
	loggerController := logger.NewLoggerController(baseController)
	shortController := shortlink.NewController(baseController)
	emailController := email.NewController(baseController)

	// Setup repositories for middleware
	loggerRepo := repositories.NewLoggerRepository(gormDB)
	userRepo := repositories.NewUserRepository(gormDB) // Changed to use GORM
	loginAttemptRepo := repositories.NewLoginAttemptRepository(gormDB)
	authRepo := repositories.NewAuthRepository(gormDB) // Add auth repository for API key middleware

	// Apply global middleware for activity logging
	r.Use(middleware.ActivityLogger(loggerRepo))

	// Definisikan route untuk user, auth, dan logger
	v1 := r.Group("/v1")
	RegisterUserRoutes(v1, userController)
	RegisterAuthRoutes(v1, authController, userRepo, *loginAttemptRepo, emailController, baseController)
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
