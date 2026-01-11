package routes

import (
	"github.com/adehusnim37/lihatin-go/controllers/logger"
	"github.com/gin-gonic/gin"
)

func RegisterLoggerRoutes(rg *gin.RouterGroup, loggerController *logger.LoggerController) {
	logs := rg.Group("/logs")
	logs.GET("/", loggerController.GetAllLogs)
	logs.GET("/user/:username", loggerController.GetLogsByUsername)
	logs.GET("/short/:code", loggerController.GetLogsByShortLink)
}
