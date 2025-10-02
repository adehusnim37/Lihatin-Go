package email

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

// Controller provides all handlers for email authentication operations
func (c *Controller) SendVerificationEmail(ctx *gin.Context) {
	userID := ctx.GetString("user_id")
	userEmail := ctx.GetString("email")
	userFirstName := ctx.GetString("username")

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
		
	// Save token to database
	if err := c.repo.GetUserAuthRepository().SetEmailVerificationToken(userID, token); err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to save verification token. Please try again later.",
			Error:   map[string]string{"error": "Internal server error"},
		})
		return
	}

	// Send verification email
	if err := c.emailService.SendVerificationEmail(userEmail, userFirstName, token); err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to send verification email",
			Error:   map[string]string{"email": "Could not send verification email"},
		})
		return
	}

	utils.SendCreatedResponse(ctx, map[string]interface{}{"sent_to": userEmail}, "Verification email sent successfully")
}