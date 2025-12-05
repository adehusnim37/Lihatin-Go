package admin

import (
	"net/http"
	
	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/models/common"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/gin-gonic/gin"
)

// GetUserByID retrieves a single user by ID (admin only)
func (c *Controller) GetUserByID(ctx *gin.Context) {
	logger.Logger.Info("Admin GetUserByID called")
	// Bind URI parameters
	var userID dto.UserIDGenericRequest
	if err := ctx.ShouldBindUri(&userID); err != nil {
		httputil.SendErrorResponse(ctx, http.StatusBadRequest, "USER_ID_REQUIRED", "User ID is required", "user_id", userID)
		return
	}

	logger.Logger.Info("Fetching user by ID", "user_id", userID)

	// Get user from repository
	user, err := c.repo.GetUserRepository().GetUserByID(userID.ID)
	if err != nil {
		httputil.HandleError(ctx, err, userID)
		return
	}

	// Convert to admin response format (remove password)
	adminUser := dto.AdminUserResponse{
		ID:           user.ID,
		Username:     user.Username,
		Email:        user.Email,
		FirstName:    user.FirstName,
		LastName:     user.LastName,
		Role:         user.Role,
		IsPremium:    user.IsPremium,
		IsLocked:     user.IsLocked,
		LockedAt:     user.LockedAt,
		LockedReason: user.LockedReason,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}

	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data:    adminUser,
		Message: "User retrieved successfully",
		Error:   nil,
	})
}
