package routes

import (
	"github.com/adehusnim37/lihatin-go/controllers"
	"github.com/gin-gonic/gin"
)

func RegisterLoggerRoutes(rg *gin.RouterGroup, loggerController *controllers.LoggerController) {
	logs := rg.Group("/logs")
	logs.GET("/", loggerController.GetAllLogs)
	logs.GET("/user/:username", loggerController.GetLogsByUsername)
}
