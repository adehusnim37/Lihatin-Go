package auth

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/gin-gonic/gin"
)

func (c *Controller) ResetPassword(ctx *gin.Context) {
	var req dto.ResetPasswordRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		validator.SendValidationError(ctx, err, &req)
		return
	}

	// Validate reset token
	user, err := c.repo.GetUserAuthRepository().ValidatePasswordResetToken(req.Token)
	if err != nil {
		httputil.HandleError(ctx, err, nil)
		return
	}

	// Hash new password
	hashedPassword, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Password reset failed",
			Error:   map[string]string{"error": "Failed to process new password"},
		})
		return
	}

	// Reset password
	if err := c.repo.GetUserAuthRepository().ResetPassword(req.Token, hashedPassword); err != nil {
		httputil.HandleError(ctx, err, nil)
		return
	}

	if err := c.emailService.SuccessfulPasswordChangeEmail(user.Email, user.Username); err != nil {
		logger.Logger.Error("Failed to send successful password change email",
			"email", user.Email,
			"error", err.Error(),
		)
	}

	httputil.SendOKResponse(ctx, nil, "Successfully Changed The Password")
}
