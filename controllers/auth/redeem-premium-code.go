package auth

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/adehusnim37/lihatin-go/dto"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/premium"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/adehusnim37/lihatin-go/middleware"
	"github.com/gin-gonic/gin"
)

// RedeemPremiumCode upgrades authenticated user account to premium using one-time code
func (c *Controller) RedeemPremiumCode(ctx *gin.Context) {
	userID := ctx.GetString("user_id")
	if userID == "" {
		httputil.SendErrorResponse(ctx, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", "auth")
		return
	}

	var req dto.RedeemPremiumCodeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		validator.SendValidationError(ctx, err, &req)
		return
	}

	currentUser, err := c.repo.GetUserRepository().GetUserByID(userID)
	if err != nil {
		httputil.HandleError(ctx, err, userID)
		return
	}

	if currentUser.IsPremium {
		httputil.SendErrorResponse(ctx, http.StatusConflict, "ALREADY_PREMIUM", "Account is already premium", "secret_code")
		return
	}

	manager := middleware.GetSessionManager()
	if manager == nil || manager.GetRedisClient() == nil {
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "PREMIUM_CODE_REDEEM_FAILED", "Premium code service is unavailable", "secret_code")
		return
	}

	_, _, err = premium.RedeemOneTimeCode(
		context.Background(),
		manager.GetRedisClient(),
		req.SecretCode,
		userID,
		time.Now(),
	)
	if err != nil {
		switch {
		case errors.Is(err, premium.ErrCodeFormat), errors.Is(err, premium.ErrCodeSignature):
			httputil.SendErrorResponse(ctx, http.StatusBadRequest, "PREMIUM_CODE_INVALID", "Secret code is invalid", "secret_code")
		case errors.Is(err, premium.ErrCodeExpired):
			httputil.SendErrorResponse(ctx, http.StatusBadRequest, "PREMIUM_CODE_EXPIRED", "Secret code is expired", "secret_code")
		case errors.Is(err, premium.ErrCodeUsed):
			httputil.SendErrorResponse(ctx, http.StatusConflict, "PREMIUM_CODE_ALREADY_USED", "Secret code has already been used", "secret_code")
		case errors.Is(err, premium.ErrSecretKeyMissing):
			httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "PREMIUM_CODE_CONFIG_INVALID", "Premium code is not configured", "secret_code")
		default:
			httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "PREMIUM_CODE_REDEEM_FAILED", "Failed to redeem premium code", "secret_code")
		}
		return
	}

	if err := c.repo.GetUserRepository().SetPremium(userID, true); err != nil {
		httputil.HandleError(ctx, err, userID)
		return
	}

	httputil.SendOKResponse(ctx, map[string]any{
		"user_id":    userID,
		"is_premium": true,
	}, "Premium code redeemed successfully")
}
