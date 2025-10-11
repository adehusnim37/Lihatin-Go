package email

import (
	"net/http"
	"time"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/models/user"
	"github.com/adehusnim37/lihatin-go/utils"
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
		utils.SendValidationError(ctx, err, &req)
		return
	}

	err := userAuth.ChangeEmail(UserId, req.NewEmail, ip, userAgent)
	if err != nil {
		utils.HandleError(ctx, err, nil)
		return
	}

	// Generate verification token
	token, err := utils.GenerateVerificationToken()
	if err != nil {
		utils.Logger.Error("Failed to generate verification token",
			"user_id", UserId,
			"error", err.Error(),
		)
		utils.SendErrorResponse(ctx, http.StatusInternalServerError, "TOKEN_GENERATION_FAILED", "Failed to generate verification token", "token", UserId)
		return
	}

	// Save token to database
	if err := userAuth.SetEmailVerificationToken(UserId, token, user.EmailSourceChange, time.Now().Add(1*time.Hour)); err != nil {
		utils.Logger.Error("Failed to save verification token",
			"user_id", UserId,
			"error", err.Error(),
		)
		utils.HandleError(ctx, err, UserId)
		return
	}

	// Send verification email using injected service
	if err := c.emailService.SendChangeEmailConfirmation(req.NewEmail, username, token); err != nil {
		utils.Logger.Error("Failed to send verification email",
			"user_id", UserId,
			"email", req.NewEmail,
			"error", err.Error(),
		)
		utils.SendErrorResponse(ctx, http.StatusInternalServerError, "EMAIL_SEND_FAILED", "Failed to send verification email", "email", UserId)
		return
	}

	// Process the request
	utils.SendSuccessResponse(ctx, http.StatusOK, nil, "Change email request processed. Please verify your new email address as soon as possible.")
}
