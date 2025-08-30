package routes

import (
	"database/sql"
	"log"
	"os"

	"github.com/adehusnim37/lihatin-go/controllers"
	"github.com/adehusnim37/lihatin-go/controllers/auth"
	"github.com/adehusnim37/lihatin-go/controllers/user"
	"github.com/adehusnim37/lihatin-go/middleware"
	"github.com/adehusnim37/lihatin-go/repositories"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func SetupRouter(db *sql.DB, validate *validator.Validate) *gin.Engine {
	// Inisialisasi router Gin default (sudah include logger & recovery middleware)
	r := gin.Default()

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	// Initialize GORM for new auth repositories
	dsn := os.Getenv("DATABASE_URL") // Ambil dari environment variable
	if dsn == "" {
		log.Fatal("DATABASE_URL config is required")
	}
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
	loggerRepo := repositories.NewLoggerRepository(gormDB)
	userRepo := repositories.NewUserRepository(db)
	loginAttemptRepo := repositories.NewLoginAttemptRepository(gormDB)

	// Apply global middleware for activity logging
	r.Use(middleware.ActivityLogger(loggerRepo))

	// Definisikan route untuk user, auth, dan logger
	v1 := r.Group("/v1")
	RegisterUserRoutes(v1, userController)
	RegisterAuthRoutes(v1, authController, userRepo, *loginAttemptRepo)
	RegisterLoggerRoutes(v1, loggerController)

	return r
}
