package auth

import (
	"errors"
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/gin-gonic/gin"
)

// SignupCompletionStatus validates a pending signup token without consuming it.
// It intentionally does not return the email associated with the token.
func (c *Controller) SignupCompletionStatus(ctx *gin.Context) {
	var req dto.SignupCompletionStatusRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		httputil.SendErrorResponse(ctx, http.StatusUnauthorized, "SIGNUP_TOKEN_INVALID", "Signup session expired or invalid. Please restart signup.", "signup_token")
		return
	}

	email, err := auth.GetSignupCompletionTokenEmail(ctx.Request.Context(), req.SignupToken)
	if err != nil {
		if !errors.Is(err, auth.ErrSignupCompletionTokenInvalid) {
			logger.Logger.Warn("Failed to validate signup completion token",
				"error", err.Error(),
			)
		}
		httputil.SendErrorResponse(ctx, http.StatusUnauthorized, "SIGNUP_TOKEN_INVALID", "Signup session expired or invalid. Please restart signup.", "signup_token")
		return
	}

	existingUser, _ := c.repo.GetUserRepository().GetUserByEmailOrUsername(email)
	if existingUser != nil {
		_ = auth.DeleteSignupCompletionToken(ctx.Request.Context(), req.SignupToken)
		httputil.SendErrorResponse(ctx, http.StatusUnauthorized, "SIGNUP_TOKEN_INVALID", "Signup session expired or invalid. Please restart signup.", "signup_token")
		return
	}

	httputil.SendOKResponse(
		ctx,
		dto.SignupCompletionStatusResponse{Valid: true},
		"Signup session is valid.",
	)
}
