package auth

import (
	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

// GetProfile retrieves the profile of the authenticated user
func (c *Controller) GetProfile(ctx *gin.Context) {
	userID := ctx.GetString("user_id")

	// Get user information
	user, err := c.repo.GetUserRepository().GetUserByID(userID)
	if err != nil {
		utils.HandleError(ctx, err, userID)
		return
	}

	// Get user auth information
	_, err = c.repo.GetUserAuthRepository().GetUserAuthByUserID(user.ID)
	if err != nil {
		utils.HandleError(ctx, err, userID)
		return
	}

	// Create profile response
	profile := dto.UserProfile{
		ID:        user.ID,
		Username:  user.Username,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email,
		Avatar:    user.Avatar,
		IsPremium: user.IsPremium,
		CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	utils.SendOKResponse(ctx, profile, "Profile retrieved successfully")
}
