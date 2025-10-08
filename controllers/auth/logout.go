package auth

import (
	"github.com/adehusnim37/lihatin-go/middleware"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

func (c *Controller) Logout(ctx *gin.Context) {
	sessionID := ctx.GetString("session_id")
	userID := ctx.GetString("user_id")

	if err := c.repo.GetUserAuthRepository().Logout(userID); err != nil {
		utils.Logger.Error("Failed to logout user",
			"user_id", userID,
			"session_id", sessionID,
			"error", err.Error(),
		)
		utils.HandleError(ctx, err, userID)
		return
	}

	// Invalidate session in Redis
	if err := middleware.DeleteAllUserSessions(ctx, userID); err != nil {
		utils.Logger.Error("Error invalidating user session", "user_id", userID, "session_id", sessionID, "error", err)
		utils.SendErrorResponse(ctx, 500, "Failed to invalidate session", "ERR_SESSION_INVALIDATION_FAILED", utils.ErrSessionInvalidationFailed.Error(), nil)
		return
	}

	utils.SendOKResponse(ctx, nil, "Logout successful")
}
