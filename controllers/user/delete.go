package user

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/models"
	"github.com/gin-gonic/gin"
)

// Delete menghapus user berdasarkan ID
func (c *Controller) Delete(ctx *gin.Context) {
	id := ctx.Param("id")
	var err error
	var user *models.User

	// Check if the user exists
	user, err = c.repo.GetUserByID(id)
	if err != nil {
		ctx.JSON(http.StatusNotFound, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "User not found",
			Error:   map[string]string{"id": "User with this ID does not exist"},
		})
		return
	}

	// Check if the user is premium
	premium, err := c.repo.CheckPremiumByUsernameOrEmail(user.Username)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to check user premium status",
			Error:   map[string]string{"error": "Failed to check user premium status, please try again later"},
		})
		return
	}

	if premium.IsPremium {
		ctx.JSON(http.StatusForbidden, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Cannot delete premium user",
			Error:   map[string]string{"premium": "Cannot delete a premium user, please contact support"},
		})
		return
	}

	// Create a struct to bind the confirmation username
	type DeleteConfirmation struct {
		Username string `json:"username"`
	}

	var confirmation DeleteConfirmation
	if err := ctx.ShouldBindJSON(&confirmation); err != nil {
		ctx.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid input",
			Error:   map[string]string{"input": "Please provide the username to confirm deletion. Only the user who created this account can delete it"},
		})
		return
	}

	if confirmation.Username != user.Username {
		ctx.JSON(http.StatusForbidden, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Username does not match",
			Error:   map[string]string{"username": "Username does not match, please provide the correct username to confirm deletion"},
		})
		return
	}

	if err = c.repo.DeleteUserPermanent(id); err != nil {
		ctx.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to delete user",
			Error:   map[string]string{"error": "Failed to delete user, please try again later"},
		})
		return
	}
	ctx.JSON(http.StatusNoContent, models.APIResponse{
		Success: true,
		Data:    nil,
		Message: "User deleted successfully. Thank you for using our service and have a great day! Any refund will not be processed if you delete your account",
		Error:   nil,
	})
}
