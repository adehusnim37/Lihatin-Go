package routes

import (
	"github.com/adehusnim37/lihatin-go/controllers/user"
	"github.com/gin-gonic/gin"
)

func RegisterUserRoutes(rg *gin.RouterGroup, userController *user.Controller) {
	users := rg.Group("/users")
	users.GET("/", userController.GetAll)
	users.GET("/:id", userController.GetByID)
	users.POST("/", userController.Create)
	users.PUT("/:id", userController.Update)
	users.DELETE("/:id", userController.Delete)

	// Authentication routes
	auth := rg.Group("/auth")
	auth.POST("/login", userController.Login)
}
