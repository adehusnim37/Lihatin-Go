package auth

import (
	"net/http"
	"time"

	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

// SendVerificationEmail sends email verification
func (c *Controller) SendVerificationEmail(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Authentication required",
			Error:   map[string]string{"auth": "Please login to access this feature"},
		})
		return
	}

	// Get user information
	user, err := c.repo.GetUserRepository().GetUserByID(userID.(string))
	if err != nil {
		ctx.JSON(http.StatusNotFound, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "User not found",
			Error:   map[string]string{"user": "User not found"},
		})
		return
	}

	// Check if user is already verified
	userAuth, err := c.repo.GetUserAuthRepository().GetUserAuthByUserID(user.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to check verification status",
			Error:   map[string]string{"error": "Internal server error"},
		})
		return
	}

	if userAuth.IsEmailVerified {
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Email already verified",
			Error:   map[string]string{"verification": "Your email is already verified"},
		})
		return
	}

	// Generate verification token
	token, err := utils.GenerateVerificationToken()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to generate verification token",
			Error:   map[string]string{"error": "Internal server error"},
		})
		return
	}

	// Set token expiry (24 hours from now)
	expiresAt := time.Now().Add(24 * time.Hour)

	// Save token to database
	if err := c.repo.GetUserAuthRepository().SetEmailVerificationToken(user.ID, token, expiresAt); err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to save verification token",
			Error:   map[string]string{"error": "Internal server error"},
		})
		return
	}

	// Send verification email
	if err := c.emailService.SendVerificationEmail(user.Email, user.FirstName, token); err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to send verification email",
			Error:   map[string]string{"email": "Could not send verification email"},
		})
		return
	}

	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data:    map[string]interface{}{"sent_to": user.Email},
		Message: "Verification email sent successfully",
		Error:   nil,
	})
}

// VerifyEmail verifies email with token
func (c *Controller) VerifyEmail(ctx *gin.Context) {
	token := ctx.Query("token")
	if token == "" {
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Verification token required",
			Error:   map[string]string{"token": "Missing verification token"},
		})
		return
	}

	// Verify email with token
	if err := c.repo.GetUserAuthRepository().VerifyEmail(token); err != nil {
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Email verification failed",
			Error:   map[string]string{"token": "Invalid or expired verification token"},
		})
		return
	}

	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data:    nil,
		Message: "Email verified successfully",
		Error:   nil,
	})
}

// ForgotPassword initiates password reset
func (c *Controller) ForgotPassword(ctx *gin.Context) {
	var req struct {
		Email string `json:"email" validate:"required,email"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid input",
			Error:   map[string]string{"input": "Invalid email format"},
		})
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

// ResetPassword resets password with token
func (c *Controller) ResetPassword(ctx *gin.Context) {
	var req struct {
		Token    string `json:"token" validate:"required"`
		Password string `json:"password" validate:"required,min=8,max=50,pwdcomplex"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid input",
			Error:   map[string]string{"input": "Invalid input format"},
		})
		return
	}

	//if err := c.Validate.Struct(req); err != nil {
	//	ctx.JSON(http.StatusBadRequest, common.APIResponse{
	//		Success: false,
	//		Data:    nil,
	//		Message: "Validation failed",
	//		Error:   map[string]string{"password": "Password must be 8-50 characters with letters, numbers, and symbols"},
	//	})
	//	return
	//}

	// Validate reset token
	_, err := c.repo.GetUserAuthRepository().ValidatePasswordResetToken(req.Token)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Password reset failed",
			Error:   map[string]string{"token": "Invalid or expired reset token"},
		})
		return
	}

	// Hash new password
	hashedPassword, err := utils.HashPassword(req.Password)
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
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Password reset failed",
			Error:   map[string]string{"error": "Failed to update password"},
		})
		return
	}

	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data:    nil,
		Message: "Password reset successfully",
		Error:   nil,
	})
}

// RefreshToken generates new access token from refresh token
func (c *Controller) RefreshToken(ctx *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" validate:"required"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid input",
			Error:   map[string]string{"input": "Refresh token is required"},
		})
		return
	}

	// Validate refresh token
	userID, err := utils.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid refresh token",
			Error:   map[string]string{"token": "Invalid or expired refresh token"},
		})
		return
	}

	// Get user data
	user, err := c.repo.GetUserRepository().GetUserByID(userID)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "User not found",
			Error:   map[string]string{"user": "Invalid user"},
		})
		return
	}

	// Get user auth data
	userAuth, err := c.repo.GetUserAuthRepository().GetUserAuthByUserID(userID)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "User auth not found",
			Error:   map[string]string{"auth": "Invalid user authentication"},
		})
		return
	}

	// Generate new access token
	role := map[bool]string{true: "premium", false: "regular"}[user.IsPremium]
	newToken, err := utils.GenerateJWT(user.ID, *userAuth.SessionID, *userAuth.DeviceID, *userAuth.LastIP, user.Username, user.Email, role, user.IsPremium, userAuth.IsEmailVerified)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to generate token",
			Error:   map[string]string{"error": "Token generation failed"},
		})
		return
	}

	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"token": newToken,
		},
		Message: "Token refreshed successfully",
		Error:   nil,
	})
}

// Logout invalidates user session (placeholder for token blacklisting)
func (c *Controller) Logout(ctx *gin.Context) {
	// In a real implementation, you would:
	// 1. Add the token to a blacklist (Redis/database)
	// 2. Clear any session data
	// 3. Optionally revoke refresh tokens

	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data:    nil,
		Message: "Logged out successfully",
		Error:   nil,
	})
}

// ChangePassword changes user password (requires current password)
func (c *Controller) ChangePassword(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Authentication required",
			Error:   map[string]string{"auth": "Please login to access this feature"},
		})
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password" validate:"required"`
		NewPassword     string `json:"new_password" validate:"required,min=8,max=50,pwdcomplex"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid input",
			Error:   map[string]string{"input": "Invalid input format"},
		})
		return
	}

	if err := c.Validate.Struct(req); err != nil {
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Validation failed",
			Error:   map[string]string{"password": "New password must be 8-50 characters with letters, numbers, and symbols"},
		})
		return
	}

	// Get user auth data
	userAuth, err := c.repo.GetUserAuthRepository().GetUserAuthByUserID(userID.(string))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to get user data",
			Error:   map[string]string{"error": "Internal server error"},
		})
		return
	}

	// Verify current password
	if err := utils.CheckPassword(userAuth.PasswordHash, req.CurrentPassword); err != nil {
		ctx.JSON(http.StatusUnauthorized, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid current password",
			Error:   map[string]string{"password": "Current password is incorrect"},
		})
		return
	}

	// Hash new password
	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Password change failed",
			Error:   map[string]string{"error": "Failed to process new password"},
		})
		return
	}

	// Update password
	if err := c.repo.GetUserAuthRepository().UpdatePassword(userID.(string), hashedPassword); err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Password change failed",
			Error:   map[string]string{"error": "Failed to update password"},
		})
		return
	}

	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data:    nil,
		Message: "Password changed successfully",
		Error:   nil,
	})
}
