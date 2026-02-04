package admin

import (
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/gin-gonic/gin"
)

// LockUser locks a user account (admin only)
func (c *Controller) LockUser(ctx *gin.Context) {
	userID := ctx.Param("id")
	if userID == "" {
		httputil.SendErrorResponse(ctx, http.StatusBadRequest, "USER_ID_REQUIRED", "User ID is required", "user_id", userID)
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
		httputil.HandleError(ctx, err, userID)
		return
	}

	// Check if user is already locked
	if user.IsLocked {
		httputil.SendErrorResponse(ctx, http.StatusConflict, "USER_ALREADY_LOCKED", "User account is already locked", "user_id", userID)
		return
	}

	// Lock the user
	if err := c.repo.GetUserAdminRepository().LockUser(userID, req.Reason); err != nil {
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "LOCK_USER_FAILED", "Failed to lock user account, please try again later", "user_id", userID)
		return
	}

	logger.Logger.Info("User locked successfully", "user_id", userID, "reason", req.Reason)

	httputil.SendOKResponse(ctx, nil, "User account locked successfully")
}
