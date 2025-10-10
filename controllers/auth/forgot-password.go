package auth

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/utils"
)

func (c *Controller) ForgotPassword(ctx *gin.Context) {
	var req dto.ForgotPasswordRequest
	userID := ctx.GetString("user_id")

	if (req.Email == "" && req.Username == "") || (req.Email != "" && req.Username != "") {
		utils.SendErrorResponse(
			ctx,
			http.StatusBadRequest,
			"INVALID_INPUT",
			"Either email or username must be provided, but not both",
			"Request body",
			userID,
		)
		return
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(ctx, err, &req)
		return
	}

	if err := c.Validate.Struct(req); err != nil {
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Validation failed",
			Error:   map[string]string{"email": "Valid email address is required"},
		})
		return
	}

	// Get user by email
	user, err := c.repo.GetUserRepository().GetUserByEmailOrUsername(req.Email)
	if err != nil {
		// Always return success to prevent email enumeration
		ctx.JSON(http.StatusOK, common.APIResponse{
			Success: true,
			Data:    nil,
			Message: "If an account with that email exists, a password reset link has been sent",
			Error:   nil,
		})
		return
	}

	// Generate reset token
	token, err := utils.GeneratePasswordResetToken()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to generate reset token",
			Error:   map[string]string{"error": "Internal server error"},
		})
		return
	}

	// Set token expiry (1 hour from now)
	expiresAt := time.Now().Add(1 * time.Hour)

	// Save token to database
	if err := c.repo.GetUserAuthRepository().SetPasswordResetToken(user.ID, token, expiresAt); err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to save reset token",
			Error:   map[string]string{"error": "Internal server error"},
		})
		return
	}

	// Send reset email
	if err := c.emailService.SendPasswordResetEmail(user.Email, user.FirstName, token); err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to send reset email",
			Error:   map[string]string{"email": "Could not send password reset email"},
		})
		return
	}

	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data:    nil,
		Message: "If an account with that email exists, a password reset link has been sent",
		Error:   nil,
	})
}
