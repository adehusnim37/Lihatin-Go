package totp

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/models/user"
	"github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/gin-gonic/gin"
)

// DisableTOTP disables TOTP for user account
// Requires either password OR valid TOTP code for verification
func (c *Controller) DisableTOTP(ctx *gin.Context) {
	userID := ctx.GetString("user_id")
	username := ctx.GetString("username")

	// Bind request
	var req dto.DisableTOTPRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		validator.SendValidationError(ctx, err, &req)
		return
	}

	// Get user auth info
	userAuth, err := c.repo.GetUserAuthRepository().GetUserAuthByUserID(userID)
	if err != nil {
		logger.Logger.Error("Failed to get user auth info",
			"user_id", userID,
			"error", err.Error(),
		)
		httputil.HandleError(ctx, err, userID)
		return
	}

	// Check if TOTP is enabled
	if !userAuth.IsTOTPEnabled {
		logger.Logger.Warn("TOTP disable attempted for user without TOTP enabled",
			"user_id", userID,
		)
		httputil.SendErrorResponse(ctx, http.StatusBadRequest, "TOTP_NOT_ENABLED", "TOTP is not enabled for this account", "totp", userID)
		return
	}

	// Must provide either password OR TOTP code
	if req.Password == "" && req.TOTPCode == "" {
		httputil.SendErrorResponse(ctx, http.StatusBadRequest, "VERIFICATION_REQUIRED", "Please provide password or TOTP code to disable 2FA", "auth", userID)
		return
	}

	// Verify with password if provided
	if req.Password != "" {
		if err := auth.CheckPassword(userAuth.PasswordHash, req.Password); err != nil {
			logger.Logger.Warn("Invalid password provided for TOTP disable",
				"user_id", userID,
			)
			c.repo.GetUserAuthRepository().IncrementFailedLogin(userID)
			httputil.SendErrorResponse(ctx, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid password provided", "auth", userID)
			return
		}
		// Password valid - proceed to disable
	} else if req.TOTPCode != "" {
		// Verify with TOTP code
		encryptedSecret, err := c.repo.GetAuthMethodRepository().GetTOTPSecret(userAuth.ID)
		if err != nil {
			logger.Logger.Error("Failed to get TOTP secret",
				"user_id", userID,
				"error", err.Error(),
			)
			httputil.HandleError(ctx, err, userID)
			return
		}

		secret, err := auth.DecryptTOTPSecret(encryptedSecret)
		if err != nil {
			logger.Logger.Error("Failed to decrypt TOTP secret",
				"user_id", userID,
				"error", err.Error(),
			)
			httputil.HandleError(ctx, err, userID)
			return
		}

		if !auth.ValidateTOTPCodeWithWindow(secret, req.TOTPCode, 1) {
			logger.Logger.Warn("Invalid TOTP code provided for disable",
				"user_id", userID,
			)
			httputil.SendErrorResponse(ctx, http.StatusBadRequest, "INVALID_TOTP", "Invalid TOTP code", "totp")
			return
		}
		// TOTP valid - proceed to disable
	}

	// Get TOTP method
	totpMethod, err := c.repo.GetAuthMethodRepository().GetAuthMethodByType(userAuth.ID, user.AuthMethodTypeTOTP)
	if err != nil {
		logger.Logger.Error("Failed to get TOTP method",
			"user_id", userID,
			"error", err.Error(),
		)
		httputil.HandleError(ctx, err, userID)
		return
	}

	// Disable TOTP
	if err := c.repo.GetAuthMethodRepository().DisableAuthMethod(totpMethod.ID); err != nil {
		logger.Logger.Error("Failed to disable TOTP",
			"user_id", userID,
			"error", err.Error(),
		)
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "TOTP_DISABLE_FAILED", "Failed to disable TOTP", "database", userID)
		return
	}

	logger.Logger.Info("TOTP disabled successfully",
		"user_id", userID,
		"username", username,
	)

	httputil.SendOKResponse(ctx, map[string]any{
		"user_id": userID,
		"message": "Two-factor authentication has been successfully disabled",
		"status":  "disabled",
	}, "TOTP disabled successfully")
}
