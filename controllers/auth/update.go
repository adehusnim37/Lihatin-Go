package auth

import (
	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/gin-gonic/gin"
)

func (c *Controller) UpdateProfile(ctx *gin.Context) {
	var req dto.UpdateProfileRequest

	userID := ctx.GetString("user_id")

	// Bind and validate request
	if err := ctx.ShouldBindJSON(&req); err != nil {
		validator.SendValidationError(ctx, err, &req)
		return
	}

	// Update user profile
	err := c.repo.GetUserRepository().UpdateUser(userID, req)
	if err != nil {
		http.HandleError(ctx, err, userID)
		return
	}

	// Retrieve updated user information
	updatedUser, err := c.repo.GetUserRepository().GetUserByID(userID)
	if err != nil {
		http.HandleError(ctx, err, userID)
		return
	}

	// Send updated user information in response
	http.SendOKResponse(ctx, updatedUser, "Profile updated successfully")
}
