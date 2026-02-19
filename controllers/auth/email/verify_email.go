package email

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/models/user"
	"github.com/gin-gonic/gin"
)

// VerifyEmail verifies email with the provided token
func (c *Controller) VerifyEmail(ctx *gin.Context) {
	var response dto.VerifyEmailResponse
	token := ctx.Query("token")
	frontendURL := config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000")

	if token == "" {
		logger.Logger.Warn("Email verification attempted without token")
		// Redirect to login with error query param
		ctx.Redirect(http.StatusFound, frontendURL+"/auth/login?error=token_required")
		return
	}

	// Verify token
	response, err := c.repo.GetUserAuthRepository().VerifyEmail(token)
	if err != nil {
		logger.Logger.Error("Email verification failed",
			"token", token,
			"error", err.Error(),
		)
		// Redirect to login with error query param
		ctx.Redirect(http.StatusFound, frontendURL+"/auth/login?error=verification_failed")
		return
	}

	// Send appropriate notification based on verification source
	switch response.Source {
	case user.EmailSourceSignup:
		// For signup, send success verification email
		if err := c.emailService.SendSuccessEmailVerification(response.Email, response.Username); err != nil {
			logger.Logger.Error("Failed to send verification email",
				"username", response.Username,
				"email", response.Email,
				"error", err.Error(),
			)
			httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "EMAIL_SEND_FAILED", "Failed to send verification email", "email", response.Email)
			return
		}
	case user.EmailSourceChange:
		// For email change, send notification to OLD email with revoke link
		if response.OldEmail != "" && response.Token != "" {
			if err := c.emailService.SendChangeOldEmailNotification(response.OldEmail, response.Email, response.Username, response.Token); err != nil {
				logger.Logger.Error("Failed to send change email notification to old email",
					"username", response.Username,
					"old_email", response.OldEmail,
					"new_email", response.Email,
					"error", err.Error(),
				)
				// Don't fail the verification if notification fails, just log it
				logger.Logger.Warn("Email verification succeeded but notification to old email failed")
			}
		}

		// Also send success notification to NEW email
		if err := c.emailService.SendSuccessEmailVerification(response.Email, response.Username); err != nil {
			logger.Logger.Error("Failed to send success verification to new email",
				"username", response.Username,
				"email", response.Email,
				"error", err.Error(),
			)
			// Don't fail, just log
			logger.Logger.Warn("Email verification succeeded but success notification failed")
		}
	}

	logger.Logger.Info("Email verified successfully", "token", token)

	// Redirect to frontend success page
	redirectURL := frontendURL + "/auth/success-verify-email"

	logger.Logger.Info("Redirecting to success page", "redirect_url", redirectURL)
	ctx.Redirect(http.StatusFound, redirectURL)
}
