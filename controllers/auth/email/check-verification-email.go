package email

import (
	"net/http"

	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/gin-gonic/gin"
)

// CheckVerificationEmail checks the verification email status

func (c *Controller) CheckVerificationEmail(ctx *gin.Context) {
	userID := ctx.GetString("user_id")

	// Add validation for required fields
	if userID == "" {
		httputil.SendErrorResponse(ctx, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated", "auth", userID)
		return
	}

	// Check if user email is already verified
	userAuth, err := c.repo.GetUserAuthRepository().GetUserAuthByUserID(userID)
	if err != nil {
		logger.Logger.Error("Failed to get user auth info",
			"user_id", userID,
			"error", err.Error(),
		)
		httputil.HandleError(ctx, err, userID)
		return
	}

	if userAuth.IsEmailVerified {
		logger.Logger.Info("Email is already verified",
			"user_id", userID,
		)
		httputil.SendOKResponse(
			ctx,
			"VERIFIED",
			"EMAIL_VERIFIED",
		)
		return
	}

	httputil.SendErrorResponse(ctx, http.StatusBadRequest, "EMAIL_NOT_VERIFIED", "Email is not verified", "email", userID)
}
