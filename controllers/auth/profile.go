package auth

import (
	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/middleware"
	"github.com/gin-gonic/gin"
)

// GetProfile retrieves the profile of the authenticated user
func (c *Controller) GetProfile(ctx *gin.Context) {
	userID := ctx.GetString("user_id")

	// Get user information
	user, err := c.repo.GetUserRepository().GetUserByID(userID)
	if err != nil {
		http.HandleError(ctx, err, userID)
		return
	}

	// Get user auth information
	userAuth, err := c.repo.GetUserAuthRepository().GetUserAuthByUserID(user.ID)
	if err != nil {
		http.HandleError(ctx, err, userID)
		return
	}

	var currentSession *dto.CurrentSessionResponse
	var previousLogin *dto.LoginEventResponse
	sessionID := ctx.GetString("session_id")
	if sessionID != "" {
		sessionData, sessionErr := middleware.GetSession(ctx.Request.Context(), sessionID)
		if sessionErr != nil {
			logger.Logger.Warn("Failed to load current session for profile",
				"user_id", userID,
				"error", sessionErr.Error(),
			)
		} else {
			currentSession = &dto.CurrentSessionResponse{
				IPAddress: sessionData.IPAddress,
				UserAgent: sessionData.UserAgent,
				DeviceID:  sessionData.DeviceID,
				CreatedAt: sessionData.CreatedAt,
				LastSeen:  sessionData.LastSeen,
				ExpiresAt: sessionData.ExpiresAt,
			}

			event, eventErr := c.repo.GetLoginEventRepository().GetPreviousLoginForSession(
				userID,
				sessionID,
				sessionData.CreatedAt,
			)
			if eventErr != nil {
				logger.Logger.Warn("Failed to load previous login for profile",
					"user_id", userID,
					"error", eventErr.Error(),
				)
			} else {
				previousLogin = dto.NewLoginEventResponse(event)
			}
		}
	}

	profile := dto.ProfileAuthResponse{
		User: dto.UserProfile{
			ID:            user.ID,
			Username:      user.Username,
			FirstName:     user.FirstName,
			LastName:      user.LastName,
			Email:         user.Email,
			Avatar:        user.Avatar,
			PremiumAccess: dto.NewPremiumAccessResponse(user.PremiumAccess),
			CreatedAt:     user.CreatedAt.Format("2006-01-02 15:04:05"),
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
			LoginBlockedUntil:   userAuth.LoginBlockedUntil,
			AccountStatus:       string(userAuth.AccountStatus),
			TOTPEnabled:         userAuth.HasEnabledTOTP(),
			CreatedAt:           userAuth.CreatedAt,
			UpdatedAt:           userAuth.UpdatedAt,
			PreviousLogin:       previousLogin,
			CurrentSession:      currentSession,
		},
	}
	http.SendOKResponse(ctx, profile, "Profile retrieved successfully")
}
