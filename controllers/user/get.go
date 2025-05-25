package user

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/models"
	"github.com/gin-gonic/gin"
)

// GetAll mengembalikan semua user
func (c *Controller) GetAll(ctx *gin.Context) {
	users, err := c.repo.GetAllUsers()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to retrieve users",
			Error:   map[string]string{"error": "Failed to retrieve users, please try again later"},
		})
		return
	}
	ctx.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    users,
		Message: "Users retrieved successfully",
		Error:   nil,
	})
}

// GetByID mengembalikan user berdasarkan ID
func (c *Controller) GetByID(ctx *gin.Context) {
	id := ctx.Param("id")
	user, err := c.repo.GetUserByID(id)
	if err != nil {
		ctx.JSON(http.StatusNotFound, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "User not found",
			Error:   map[string]string{"id": "User with this ID does not exist"},
		})
		return
	}
	ctx.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    user,
		Message: "User found",
		Error:   nil,
	})
}
