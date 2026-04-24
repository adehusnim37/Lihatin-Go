package routes

import (
	"github.com/adehusnim37/lihatin-go/controllers"
	supportcontroller "github.com/adehusnim37/lihatin-go/controllers/support"
	"github.com/adehusnim37/lihatin-go/middleware"
	"github.com/adehusnim37/lihatin-go/repositories/authrepo"
	"github.com/adehusnim37/lihatin-go/repositories/userrepo"
	"github.com/gin-gonic/gin"
)

func RegisterSupportRoutes(rg *gin.RouterGroup, userRepo userrepo.UserRepository, userAuthRepo *authrepo.UserAuthRepository, baseController *controllers.BaseController) {
	supportController := supportcontroller.NewController(baseController)

	supportPublic := rg.Group("/support")
	{
		supportPublic.POST("/tickets", middleware.RateLimitMiddleware(10, 24, 0), supportController.CreateTicket)
		supportPublic.GET("/track", middleware.RateLimitMiddleware(20, 0, 1), supportController.TrackTicket)
	}

	supportAdmin := rg.Group("/auth/admin/support")
	supportAdmin.Use(middleware.AuthMiddleware(userRepo, userAuthRepo), middleware.AdminAuth(), middleware.RequireEmailVerification())
	{
		supportAdmin.GET("/tickets", supportController.ListTickets)
		supportAdmin.GET("/tickets/:id", supportController.GetTicket)
		supportAdmin.PUT("/tickets/:id", supportController.UpdateTicket)
	}
}
