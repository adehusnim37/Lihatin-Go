package totp

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/models/user"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

// DisableTOTP disables TOTP for user account
func (c *Controller) DisableTOTP(ctx *gin.Context) {
	userID := ctx.GetString("user_id")
	username := ctx.GetString("username")

	if userID == "" {
		utils.SendErrorResponse(ctx, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated", "auth", userID)
		return
	}

	// Bind request
	var req dto.DisableTOTPRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(ctx, err, &req)
		return
	}

	// Get user auth info
	userAuth, err := c.repo.GetUserAuthRepository().GetUserAuthByUserID(userID)
	if err != nil {
		utils.Logger.Error("Failed to get user auth info",
			"user_id", userID,
			"error", err.Error(),
		)
		utils.HandleError(ctx, err, userID)
		return
	}

	if !userAuth.IsTOTPEnabled {
		utils.Logger.Warn("TOTP disable attempted for user without TOTP enabled",
			"user_id", userID,
		)
		utils.SendErrorResponse(ctx, http.StatusBadRequest, "TOTP_NOT_ENABLED", "TOTP is not enabled for this account", "totp", userID)
		return
	}

	if err := utils.CheckPassword(userAuth.PasswordHash, req.Password); err != nil {
		utils.Logger.Warn("Invalid password provided for TOTP disable",
			"user_id", userID,
		)
		utils.SendErrorResponse(ctx, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid password provided", "auth", userID)
		return
	}


	// Get TOTP method
	totpMethod, err := c.repo.GetAuthMethodRepository().GetAuthMethodByType(userAuth.ID, user.AuthMethodTypeTOTP)
	if err != nil {
		utils.Logger.Error("Failed to get TOTP method",
			"user_id", userID,
			"error", err.Error(),
		)
		utils.HandleError(ctx, err, userID)
		return
	}

	// Disable TOTP
	if err := c.repo.GetAuthMethodRepository().DisableAuthMethod(totpMethod.ID); err != nil {
		utils.Logger.Error("Failed to disable TOTP",
			"user_id", userID,
			"error", err.Error(),
		)
		utils.SendErrorResponse(ctx, http.StatusInternalServerError, "TOTP_DISABLE_FAILED", "Failed to disable TOTP", "database", userID)
		return
	}

	utils.Logger.Info("TOTP disabled successfully",
		"user_id", userID,
		"username", username,
	)

	utils.SendOKResponse(ctx, map[string]interface{}{
		"user_id": userID,
		"message": "Two-factor authentication has been successfully disabled",
		"status":  "disabled",
	}, "TOTP disabled successfully")
}
