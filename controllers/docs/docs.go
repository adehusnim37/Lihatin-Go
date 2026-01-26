package docs

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/gin-gonic/gin"
)

// Controller handles documentation-related requests
type Controller struct{}

// NewController creates a new docs controller
func NewController() *Controller {
	return &Controller{}
}

// GetPostmanCollection serves the Postman collection JSON file
func (c *Controller) GetPostmanCollection(ctx *gin.Context) {
	// Get the path to the Postman collection file
	// The file is located in examples/Lihat.in.postman_collection.json
	filePath := filepath.Join("examples", "Lihat.in.postman_collection.json")

	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		ctx.JSON(http.StatusNotFound, common.APIResponse{
			Success: false,
			Message: "Postman collection not found",
			Error:   map[string]string{"file": "Collection file does not exist"},
		})
		return
	}

	// Set headers for JSON response
	ctx.Header("Content-Type", "application/json")
	ctx.Header("Access-Control-Allow-Origin", "*")
	ctx.Header("Cache-Control", "public, max-age=3600") // Cache for 1 hour

	// Write raw JSON response
	ctx.Data(http.StatusOK, "application/json", data)
}

// GetAPIInfo returns basic API information
func (c *Controller) GetAPIInfo(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data: gin.H{
			"name":        "Lihatin API",
			"version":     "1.0.0",
			"description": "URL Shortener and Link Management API",
			"docs": gin.H{
				"postman": "/v1/docs/postman",
			},
		},
		Message: "API information retrieved successfully",
	})
}
