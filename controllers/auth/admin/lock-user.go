package admin

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/gin-gonic/gin"
)

// LockUser locks a user account (admin only)
func (c *Controller) LockUser(ctx *gin.Context) {
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

	var req dto.AdminLockUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		validator.SendValidationError(ctx, err, &req)
		return
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

	// Check if user is already locked
	if user.IsLocked {
		ctx.JSON(http.StatusConflict, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "User already locked",
			Error:   map[string]string{"status": "User account is already locked"},
		})
		return
	}

	// Lock the user
	if err := c.repo.GetUserAdminRepository().LockUser(userID, req.Reason); err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to lock user",
			Error:   map[string]string{"error": "Failed to lock user account, please try again later"},
		})
		return
	}

	logger.Logger.Info("User locked successfully", "user_id", userID, "reason", req.Reason)

	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data:    nil,
		Message: "User account locked successfully",
		Error:   nil,
	})
}
