package routes

import (
	"github.com/adehusnim37/lihatin-go/controllers/short-link"
	"github.com/adehusnim37/lihatin-go/middleware"
	"github.com/adehusnim37/lihatin-go/repositories"
	"github.com/gin-gonic/gin"
)

func RegisterShortRoutes(rg *gin.RouterGroup, shortController *shortlink.Controller, userRepo repositories.UserRepository) {
	shortGroup := rg.Group("/short")
	{
		shortGroup.POST("", shortController.Create, middleware.OptionalAuth(userRepo))
	}
}

