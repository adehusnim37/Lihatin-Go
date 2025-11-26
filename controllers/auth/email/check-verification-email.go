package email

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

// CheckVerificationEmail checks the verification email status

func (c *Controller) CheckVerificationEmail(ctx *gin.Context) {
	userID := ctx.GetString("user_id")

	// Add validation for required fields
	if userID == "" {
		utils.SendErrorResponse(ctx, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated", "auth", userID)
		return
	}

	// Check if user email is already verified
	userAuth, err := c.repo.GetUserAuthRepository().GetUserAuthByUserID(userID)
	if err != nil {
		utils.Logger.Error("Failed to get user auth info",
			"user_id", userID,
			"error", err.Error(),
		)
		utils.HandleError(ctx, err, userID)
		return
	}

	if userAuth.IsEmailVerified {
		utils.Logger.Info("Email is already verified",
			"user_id", userID,
		)
		utils.SendOKResponse(
			ctx,
			"VERIFIED",
			"EMAIL_VERIFIED",
		)
		return
	}

	utils.SendErrorResponse(ctx, http.StatusBadRequest, "EMAIL_NOT_VERIFIED", "Email is not verified", "email", userID)
}
