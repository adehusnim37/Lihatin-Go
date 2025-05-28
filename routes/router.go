package routes

import (
	"database/sql"

	"github.com/adehusnim37/lihatin-go/controllers"
	"github.com/adehusnim37/lihatin-go/controllers/auth"
	"github.com/adehusnim37/lihatin-go/controllers/user"
	"github.com/adehusnim37/lihatin-go/middleware"
	"github.com/adehusnim37/lihatin-go/repositories"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func SetupRouter(db *sql.DB, validate *validator.Validate) *gin.Engine {
	// Inisialisasi router Gin default (sudah include logger & recovery middleware)
	r := gin.Default()

	// Initialize GORM for new auth repositories
	dsn := "adehusnim:ryugamine123A@tcp(localhost:3306)/LihatinGo?charset=utf8mb4&parseTime=True&loc=Local"
	gormDB, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to database with GORM: " + err.Error())
	}

	// Buat base controller yang akan digunakan oleh semua controller
	baseController := controllers.NewBaseController(db, validate)

	// Create base controller with GORM for auth
	baseAuthController := controllers.NewBaseControllerWithGorm(db, gormDB, validate)

	// Inisialisasi controller spesifik
	userController := user.NewController(baseController)
	authController := auth.NewAuthController(baseAuthController)
	loggerController := controllers.NewLoggerController(baseController)

	// Setup logger repository for middleware
	loggerRepo := repositories.NewLoggerRepository(db)

	// Setup user repository for auth middleware
	userRepo := repositories.NewUserRepository(db)

	// Apply global middleware for activity logging
	r.Use(middleware.ActivityLogger(loggerRepo))

	// Definisikan route untuk user, auth, dan logger
	v1 := r.Group("/v1")
	RegisterUserRoutes(v1, userController)
	RegisterAuthRoutes(v1, authController, userRepo)
	RegisterLoggerRoutes(v1, loggerController)

	return r
}
