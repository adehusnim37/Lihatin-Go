package auth

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/gin-gonic/gin"
)

func (c *Controller) ValidateResetToken(ctx *gin.Context) {
	var req dto.ForgotPasswordToken

	// Bind and validate request
	if err := ctx.ShouldBindQuery(&req); err != nil {
		validator.SendValidationError(ctx, err, &req)
		return
	}
	// Validate the token
	if _, err := c.repo.GetUserAuthRepository().ValidatePasswordResetToken(req.Token); err != nil {
		httputil.SendErrorResponse(
			ctx,
			http.StatusUnauthorized,
			"INVALID_TOKEN",
			"Password reset token is invalid or expired",
			"URL parameter",
			nil,
		)
		return
	}

	httputil.SendSuccessResponse(ctx, http.StatusOK, nil, "Token is valid")
}
