package email

import (
	"net/http"
	"time"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/models/user"
	"github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/gin-gonic/gin"
)

func (c *Controller) ChangeEmail(ctx *gin.Context) {
	UserId := ctx.GetString("user_id")
	username := ctx.GetString("username")
	userAuth := c.repo.GetUserAuthRepository()
	ip := ctx.ClientIP()                     // Gets the real client IP, considering X-Forwarded-For etc.
	userAgent := ctx.GetHeader("User-Agent") // Gets the User-Agent string

	var req dto.ChangeEmailRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		validator.SendValidationError(ctx, err, &req)
		return
	}

	err := userAuth.ChangeEmail(UserId, req.NewEmail, ip, userAgent)
	if err != nil {
		httputil.HandleError(ctx, err, nil)
		return
	}

	// Generate verification token
	token, err := auth.GenerateVerificationToken()
	if err != nil {
		logger.Logger.Error("Failed to generate verification token",
			"user_id", UserId,
			"error", err.Error(),
		)
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "TOKEN_GENERATION_FAILED", "Failed to generate verification token", "token", UserId)
		return
	}

	// Save token to database
	if err := userAuth.SetEmailVerificationToken(UserId, token, user.EmailSourceChange, time.Now().Add(1*time.Hour)); err != nil {
		logger.Logger.Error("Failed to save verification token",
			"user_id", UserId,
			"error", err.Error(),
		)
		httputil.HandleError(ctx, err, UserId)
		return
	}

	// Send verification email using injected service
	if err := c.emailService.SendChangeEmailConfirmation(req.NewEmail, username, token); err != nil {
		logger.Logger.Error("Failed to send verification email",
			"user_id", UserId,
			"email", req.NewEmail,
			"error", err.Error(),
		)
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "EMAIL_SEND_FAILED", "Failed to send verification email", "email", UserId)
		return
	}

	// Process the request
	httputil.SendSuccessResponse(ctx, http.StatusOK, req.NewEmail, "Change email request processed. Please verify your new email address as soon as possible.")
}
