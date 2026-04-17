package auth

import (
	"net/http"
	"time"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/gin-gonic/gin"
)

// Login authenticates a user with email and password,
// then enforces second factor (TOTP or email OTP).
func (c *Controller) Login(ctx *gin.Context) {
	var loginReq dto.LoginRequest

	if err := ctx.ShouldBindJSON(&loginReq); err != nil {
		validator.SendValidationError(ctx, err, &loginReq)
		return
	}

	user, userAuth, err := c.repo.GetUserAuthRepository().GetUserForLogin(loginReq.EmailOrUsername)
	if err != nil {
		if user != nil {
			c.repo.GetUserAuthRepository().IncrementFailedLogin(user.ID)
		}
		httputil.HandleError(ctx, err, nil)
		return
	}

	isLocked, err := c.repo.GetUserAuthRepository().IsAccountLocked(user.ID)
	if err != nil {
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "LOGIN_FAILED", "An error occurred during login", "auth")
		return
	}

	if isLocked {
		httputil.SendErrorResponse(ctx, http.StatusForbidden, "ACCOUNT_LOCKED", "Your account is locked. Please try again later.", "auth")
		return
	}

	if !userAuth.IsActive {
		httputil.SendErrorResponse(ctx, http.StatusForbidden, "ACCOUNT_DEACTIVATED", "Your account has been deactivated. Please contact support.", "auth")
		return
	}

	if err := auth.CheckPassword(userAuth.PasswordHash, loginReq.Password); err != nil {
		c.repo.GetUserAuthRepository().IncrementFailedLogin(user.ID)
		httputil.SendErrorResponse(ctx, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email/username or password", "auth")
		return
	}

	if err := c.requireSecondFactor(ctx, user, userAuth, "Password verified"); err != nil {
		return
	}
}

// Helper function to safely format time pointer
func formatTimeOrEmpty(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02T15:04:05Z07:00")
}
