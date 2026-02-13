package routes

import (
	"github.com/adehusnim37/lihatin-go/controllers/logger"
	"github.com/adehusnim37/lihatin-go/middleware"
	"github.com/adehusnim37/lihatin-go/repositories/userrepo"
	"github.com/gin-gonic/gin"
)

// RegisterLoggerRoutes registers logger routes with query parameters for pagination and filtering
// All endpoints support: page, limit, sort, order_by query parameters
// Filter endpoint also supports: username, action, method, route, level, status_code, ip_address, date_from, date_to, api_key
func RegisterLoggerRoutes(rg *gin.RouterGroup, userRepo userrepo.UserRepository, loggerController *logger.LoggerController) {
	logs := rg.Group("/logs")
	{
		logs.Use(middleware.AuthMiddleware(userRepo))

		// Get logs with ID
		logs.GET("/:id", loggerController.GetLogByID)

		// GET /logs?page=1&limit=10&sort=created_at&order_by=desc
		logs.GET("/all", loggerController.GetAllLogs)

		// Get logs stats
		logs.GET("/stats", loggerController.GetLogsStats)

		//Get All Counted Logs Get,Post,Put,Patch,Delete
		logs.GET("/all/count", loggerController.GetAllCountedLogs)

		// GET /logs/filter?page=1&limit=10&sort=created_at&order_by=desc&username=john&level=error&status_code=500
		logs.GET("/filter", loggerController.GetLogsWithFilter)

		// GET /logs/user/:username?page=1&limit=10&sort=created_at&order_by=desc
		logs.GET("/user/:username", loggerController.GetLogsByUsername)

		// GET /logs/short/:code?page=1&limit=10&sort=created_at&order_by=desc
		logs.GET("/short/:code", loggerController.GetLogsByShortLink)

	}
}
