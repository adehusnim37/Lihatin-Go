package admin

import (
	"net/http"

	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/gin-gonic/gin"
)

// UnlockUser unlocks a user account (admin only)
func (c *Controller) UnlockUser(ctx *gin.Context) {
	userID := ctx.Param("id")
	if userID == "" {
		httputil.SendErrorResponse(ctx, http.StatusBadRequest, "USER_ID_REQUIRED", "User ID is required", "user_id", userID)
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
		httputil.SendErrorResponse(ctx, http.StatusNotFound, "USER_NOT_FOUND", "User not found", "user_id", userID)
		return
	}

	// Check if user is actually locked
	if !user.IsLocked {
		httputil.SendErrorResponse(ctx, http.StatusConflict, "USER_NOT_LOCKED", "User not locked", "user_id", userID)
		return
	}

	// Unlock the user
	if err := c.repo.GetUserAdminRepository().UnlockUser(userID, req.Reason); err != nil {
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "UNLOCK_USER_FAILED", "Failed to unlock user account, please try again later", "user_id", userID)
		return
	}

	logger.Logger.Info("User unlocked successfully", "user_id", userID, "reason", req.Reason)

	httputil.SendOKResponse(ctx, nil, "User account unlocked successfully")
}
