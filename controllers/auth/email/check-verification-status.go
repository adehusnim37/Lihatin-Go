package email

import (
	"errors"
	"net/http"

	apperrors "github.com/adehusnim37/lihatin-go/internal/pkg/errors"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/gin-gonic/gin"
)

// CheckVerificationStatusByIdentifier checks email verification status using base64url identifier.
// Public endpoint so unverified users can poll status from check-email page.
func (c *Controller) CheckVerificationStatusByIdentifier(ctx *gin.Context) {
	encodedIdentifier := ctx.Query("identifier")
	if encodedIdentifier == "" {
		httputil.SendErrorResponse(ctx, http.StatusBadRequest, "IDENTIFIER_REQUIRED", "Identifier is required", "identifier")
		return
	}

	emailOrUsername, err := decodeIdentifier(encodedIdentifier)
	if err != nil {
		httputil.SendErrorResponse(ctx, http.StatusBadRequest, "INVALID_IDENTIFIER", "Invalid identifier format", "identifier")
		return
	}

	_, userAuth, err := c.repo.GetUserAuthRepository().GetUserForLogin(emailOrUsername)
	if err != nil {
		if errors.Is(err, apperrors.ErrUserNotFound) {
			httputil.SendOKResponse(ctx, map[string]any{
				"verified": false,
			}, "EMAIL_NOT_VERIFIED")
			return
		}

		logger.Logger.Error("Failed to check email verification status",
			"identifier", emailOrUsername,
			"error", err.Error(),
		)
		httputil.HandleError(ctx, err, emailOrUsername)
		return
	}

	httputil.SendOKResponse(ctx, map[string]any{
		"verified": userAuth.IsEmailVerified,
	}, "EMAIL_VERIFICATION_STATUS_RETRIEVED")
}
