package auth

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

func (c *Controller) ValidateResetToken(ctx *gin.Context) {
	var req dto.ForgotPasswordToken

	// Bind and validate request
	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.SendValidationError(ctx, err, &req)
		return
	}
	// Validate the token
	if _, err := c.repo.GetUserAuthRepository().ValidatePasswordResetToken(req.Token); err != nil {
		utils.SendErrorResponse(
			ctx,
			http.StatusUnauthorized,
			"INVALID_TOKEN",
			"Password reset token is invalid or expired",
			"URL parameter",
			nil,
		)
		return
	}

	utils.SendSuccessResponse(ctx, http.StatusOK, nil, "Token is valid")
}
