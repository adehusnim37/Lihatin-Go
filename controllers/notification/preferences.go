package notification

import (
	"net/http"
	"strings"

	"github.com/adehusnim37/lihatin-go/dto"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/models/user"
	"github.com/gin-gonic/gin"
)

func (c *Controller) GetPreferences(ctx *gin.Context) {
	userID := strings.TrimSpace(ctx.GetString("user_id"))
	preference, err := c.preferences.Get(userID)
	if err != nil {
		httputil.SendErrorResponse(
			ctx,
			http.StatusInternalServerError,
			"NOTIFICATION_PREFERENCES_READ_FAILED",
			"Failed to retrieve notification preferences",
			"notifications",
		)
		return
	}

	httputil.SendOKResponse(ctx, toPreferenceResponse(preference), "Notification preferences retrieved successfully")
}

func (c *Controller) UpdatePreferences(ctx *gin.Context) {
	var request dto.UpdateNotificationPreferenceRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		httputil.SendErrorResponse(
			ctx,
			http.StatusBadRequest,
			"INVALID_NOTIFICATION_PREFERENCES",
			"Invalid notification preference payload",
			"notifications",
		)
		return
	}
	if request.WeeklySummaryEmail == nil && request.PromotionalEmail == nil {
		httputil.SendErrorResponse(
			ctx,
			http.StatusBadRequest,
			"EMPTY_NOTIFICATION_PREFERENCES",
			"At least one notification preference is required",
			"notifications",
		)
		return
	}

	userID := strings.TrimSpace(ctx.GetString("user_id"))
	preference, err := c.preferences.Update(
		userID,
		request.WeeklySummaryEmail,
		request.PromotionalEmail,
		"account_settings",
	)
	if err != nil {
		httputil.SendErrorResponse(
			ctx,
			http.StatusInternalServerError,
			"NOTIFICATION_PREFERENCES_UPDATE_FAILED",
			"Failed to update notification preferences",
			"notifications",
		)
		return
	}

	httputil.SendOKResponse(ctx, toPreferenceResponse(preference), "Notification preferences updated successfully")
}

func toPreferenceResponse(preference *user.NotificationPreference) dto.NotificationPreferenceResponse {
	return dto.NotificationPreferenceResponse{
		SecurityAlertsEmail: true,
		WeeklySummaryEmail:  preference.WeeklySummaryEmail,
		PromotionalEmail:    preference.PromotionalEmail,
		WeeklySummaryOptIn:  preference.WeeklySummaryOptInAt,
		PromotionalOptIn:    preference.PromotionalOptInAt,
	}
}
