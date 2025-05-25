package routes

import (
	"database/sql"

	"github.com/adehusnim37/lihatin-go/controllers"
	"github.com/adehusnim37/lihatin-go/controllers/user"
	"github.com/adehusnim37/lihatin-go/middleware"
	"github.com/adehusnim37/lihatin-go/repositories"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func SetupRouter(db *sql.DB, validate *validator.Validate) *gin.Engine {
	// Inisialisasi router Gin default (sudah include logger & recovery middleware)
	r := gin.Default()

	// Buat base controller yang akan digunakan oleh semua controller
	baseController := controllers.NewBaseController(db, validate)

	// Inisialisasi controller spesifik
	userController := user.NewController(baseController)
	loggerController := controllers.NewLoggerController(baseController)

	// Setup logger repository for middleware
	loggerRepo := repositories.NewLoggerRepository(db)

	// Apply global middleware for activity logging
	r.Use(middleware.ActivityLogger(loggerRepo))

	// Definisikan route untuk user dan logger
	v1 := r.Group("/v1")
	RegisterUserRoutes(v1, userController)
	RegisterLoggerRoutes(v1, loggerController)

	return r
}
