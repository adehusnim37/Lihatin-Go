package admin

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/gin-gonic/gin"
)

// UnlockUser unlocks a user account (admin only)
func (c *Controller) UnlockUser(ctx *gin.Context) {
	userID := ctx.Param("id")
	if userID == "" {
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "User ID is required",
			Error:   map[string]string{"user_id": "User ID parameter is required"},
		})
		return
	}

	var req dto.AdminUnlockUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		// Allow empty body for unlock requests
		req = dto.AdminUnlockUserRequest{}
	}

	// Check if user exists
	user, err := c.repo.GetUserRepository().GetUserByID(userID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "User not found",
			Error:   map[string]string{"user_id": "User with this ID does not exist"},
		})
		return
	}

	// Check if user is actually locked
	if !user.IsLocked {
		ctx.JSON(http.StatusConflict, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "User not locked",
			Error:   map[string]string{"status": "User account is not currently locked"},
		})
		return
	}

	// Unlock the user
	if err := c.repo.GetUserAdminRepository().UnlockUser(userID, req.Reason); err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to unlock user",
			Error:   map[string]string{"error": "Failed to unlock user account, please try again later"},
		})
		return
	}

	logger.Logger.Info("User unlocked successfully", "user_id", userID, "reason", req.Reason)

	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data:    nil,
		Message: "User account unlocked successfully",
		Error:   nil,
	})
}
