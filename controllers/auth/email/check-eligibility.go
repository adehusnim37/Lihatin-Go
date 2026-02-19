package email

import (
	"github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/gin-gonic/gin"
)

// CheckEmailChangeEligibility checks if user can change email
func (c *Controller) CheckEmailChangeEligibility(ctx *gin.Context) {
	userID := ctx.GetString("user_id")

	eligible, daysRemaining, err := c.repo.GetHistoryUserRepository().CheckEmailChangeEligibility(userID)
	if err != nil {
		logger.Logger.Error("Failed to check email change eligibility",
			"user_id", userID,
			"error", err.Error(),
		)
		http.HandleError(ctx, err, nil)
		return
	}

	response := map[string]any{
		"eligible":       eligible,
		"days_remaining": daysRemaining,
	}

	if !eligible {
		response["message"] = "You must wait before changing your email again"
		response["reason"] = "rate_limit"

		if daysRemaining > 0 {
			response["retry_after_days"] = daysRemaining
		}
	} else {
		response["message"] = "You are eligible to change your email"
	}

	http.SendOKResponse(ctx, response, "Email change eligibility checked")
}

// GetEmailChangeHistory returns user's email change history
func (c *Controller) GetEmailChangeHistory(ctx *gin.Context) {
	userID := ctx.GetString("user_id")

	// Get last 90 days of history
	history, err := c.repo.GetHistoryUserRepository().GetEmailChangeHistory(userID, 90)
	if err != nil {
		logger.Logger.Error("Failed to get email change history",
			"user_id", userID,
			"error", err.Error(),
		)
		http.HandleError(ctx, err, nil)
		return
	}

	http.SendOKResponse(ctx, map[string]interface{}{
		"history": history,
		"total":   len(history),
	}, "Email change history retrieved successfully")
}
