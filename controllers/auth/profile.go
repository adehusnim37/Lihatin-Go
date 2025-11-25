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
	userAuth, err := c.repo.GetUserAuthRepository().GetUserAuthByUserID(user.ID)
	if err != nil {
		utils.HandleError(ctx, err, userID)
		return
	}

	profile := dto.ProfileAuthResponse{
		User: dto.UserProfile{
			ID:        user.ID,
			Username:  user.Username,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Email:     user.Email,
			Avatar:    user.Avatar,
			IsPremium: user.IsPremium,
			CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
		},
		Auth: dto.CompletedUserAuthResponse{
			ID:                  userAuth.ID,
			UserID:              userAuth.UserID,
			IsEmailVerified:     userAuth.IsEmailVerified,
			DeviceID:            userAuth.DeviceID,
			LastIP:              userAuth.LastIP,
			LastLoginAt:         userAuth.LastLoginAt,
			LastLogoutAt:        userAuth.LastLogoutAt,
			FailedLoginAttempts: userAuth.FailedLoginAttempts,
			PasswordChangedAt:   userAuth.PasswordChangedAt,
			LockoutUntil:        userAuth.LockoutUntil,
			IsActive:            userAuth.IsActive,
			IsTOTPEnabled:       userAuth.IsTOTPEnabled,
			CreatedAt:           userAuth.CreatedAt,
			UpdatedAt:           userAuth.UpdatedAt,
		},
	}
	utils.SendOKResponse(ctx, profile, "Profile retrieved successfully")
}
