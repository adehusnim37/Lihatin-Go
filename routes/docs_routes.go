package routes

import (
	"github.com/adehusnim37/lihatin-go/controllers/docs"
	"github.com/gin-gonic/gin"
)

// RegisterDocsRoutes registers documentation routes
func RegisterDocsRoutes(router *gin.RouterGroup) {
	docsController := docs.NewController()

	// Documentation routes (public, no auth required)
	docsGroup := router.Group("/docs")
	{
		// API info endpoint
		docsGroup.GET("", docsController.GetAPIInfo)
		docsGroup.GET("/", docsController.GetAPIInfo)

		// Postman collection endpoint
		docsGroup.GET("/postman", docsController.GetPostmanCollection)
	}
}
