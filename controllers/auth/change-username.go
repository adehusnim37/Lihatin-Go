package auth

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/gin-gonic/gin"
)

func (c *Controller) ChangeUsername(ctx *gin.Context) {
	userID := ctx.GetString("user_id")
	userEmail := ctx.GetString("email")

	var req dto.ChangeUsernameRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		httputil.HandleError(ctx, err, userID)
		return
	}

	oldUsername, err := c.repo.GetUserRepository().ChangeUsername(userID, req.NewUsername)
	if err != nil {
		httputil.HandleError(ctx, err, userID)
		return
	}

	// Send username change confirmation email (async, non-blocking)
	if userEmail != "" {
		go func(email, oldName, newName, uid string) {
			if err := c.emailService.SendUsernameChangedEmail(email, oldName, newName); err != nil {
				logger.Logger.Error("Failed to send username changed email",
					"user_id", uid,
					"email", email,
					"error", err.Error(),
				)
			}
		}(userEmail, oldUsername, req.NewUsername, userID)
	}

	httputil.SendSuccessResponse(ctx, http.StatusOK, gin.H{
		"old_username": oldUsername,
		"new_username": req.NewUsername,
	}, "Username changed successfully")
}
