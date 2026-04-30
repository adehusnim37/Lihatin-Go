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
		supportPublic.POST("/access/request-otp", middleware.RateLimitMiddleware(15, 0, 10), supportController.RequestAccessOTP)
		supportPublic.POST("/access/resend-otp", middleware.RateLimitMiddleware(20, 0, 10), supportController.ResendAccessOTP)
		supportPublic.POST("/access/verify-otp", middleware.RateLimitMiddleware(20, 0, 10), supportController.VerifyAccessOTP)
		supportPublic.POST("/access/verify-code", middleware.RateLimitMiddleware(30, 0, 10), supportController.VerifyAccessCode)
		supportPublic.GET("/tickets/:ticketCode/messages", middleware.RateLimitMiddleware(40, 0, 1), supportController.ListPublicConversation)
		supportPublic.POST("/tickets/:ticketCode/messages", middleware.RateLimitMiddleware(25, 0, 10), supportController.SendPublicMessage)
		supportPublic.GET("/tickets/:ticketCode/attachments/:attachmentID", middleware.RateLimitMiddleware(60, 0, 1), supportController.DownloadPublicAttachment)
	}

	supportUser := rg.Group("/auth/support")
	supportUser.Use(middleware.AuthMiddleware(userRepo, userAuthRepo), middleware.RequireEmailVerification())
	{
		supportUser.GET("/tickets", supportController.ListUserTickets)
		supportUser.GET("/tickets/:id/messages", supportController.ListUserConversation)
		supportUser.POST("/tickets/:id/messages", supportController.SendUserMessage)
		supportUser.GET("/attachments/:attachmentID", supportController.DownloadUserAttachment)
	}

	supportAdmin := rg.Group("/auth/admin/support")
	supportAdmin.Use(middleware.AuthMiddleware(userRepo, userAuthRepo), middleware.AdminAuth(), middleware.RequireEmailVerification())
	{
		supportAdmin.GET("/tickets", supportController.ListTickets)
		supportAdmin.GET("/tickets/:id", supportController.GetTicket)
		supportAdmin.GET("/tickets/:id/messages", supportController.ListAdminConversation)
		supportAdmin.POST("/tickets/:id/messages", supportController.SendAdminMessage)
		supportAdmin.GET("/attachments/:attachmentID", supportController.DownloadAdminAttachment)
		supportAdmin.PUT("/tickets/:id", supportController.UpdateTicket)
	}
}
