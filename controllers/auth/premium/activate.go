package premium

import (
	"errors"
	"net/http"
	"time"

	"github.com/adehusnim37/lihatin-go/dto"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/internal/pkg/premium"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/gin-gonic/gin"
)

func (c *Controller) ActivatePremium(ctx *gin.Context) {

	userID := ctx.GetString("user_id")
	username := ctx.GetString("username")
	userEmail := ctx.GetString("email")

	var req dto.RedeemPremiumCodeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		validator.SendValidationError(ctx, err, &req)
		return
	}

	// 1. Manual Verification (Stateless)
	if _, _, err := premium.VerifyCode(req.SecretCode, time.Now()); err != nil {
		switch {
		case errors.Is(err, premium.ErrCodeFormat), errors.Is(err, premium.ErrCodeSignature):
			httputil.SendErrorResponse(ctx, http.StatusBadRequest, "PREMIUM_CODE_INVALID", "Secret code is invalid", "secret_code")
		case errors.Is(err, premium.ErrCodeExpired):
			httputil.SendErrorResponse(ctx, http.StatusBadRequest, "PREMIUM_CODE_EXPIRED", "Secret code is expired", "secret_code")
		case errors.Is(err, premium.ErrSecretKeyMissing):
			httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "PREMIUM_CODE_CONFIG_INVALID", "Premium code is not configured", "secret_code")
		default:
			httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "PREMIUM_CODE_VERIFY_FAILED", "Failed to verify premium code", "secret_code")
		}
		return
	}

	// 1.5 Check if user is already premium
	isPremium, err := c.userRepo.CheckPremiumByUsernameOrEmail(username)
	if err != nil {
		httputil.HandleError(ctx, err, nil)
		return
	}

	if isPremium {
		httputil.SendErrorResponse(ctx, http.StatusBadRequest, "USER_ALREADY_PREMIUM", "User is already premium", "user_id")
		return
	}

	// 2. Database Verification
	if err := c.premiumRepo.RedeemPremiumCode(req.SecretCode, userID); err != nil {
		httputil.HandleError(ctx, err, nil)
		return
	}

	// 3. Send premium activation email (async, non-blocking)
	if userEmail != "" {
		go func(email, name, code, uid string) {
			if err := c.emailSvc.SendPremiumRedeemedEmail(email, name, code); err != nil {
				logger.Logger.Error("Failed to send premium redeemed email",
					"user_id", uid,
					"email", email,
					"error", err.Error(),
				)
			}
		}(userEmail, username, req.SecretCode, userID)
	}

	httputil.SendOKResponse(ctx, dto.RedeemPremiumCodeResponse{
		SecretCode: req.SecretCode,
		UpdatedAt:  time.Now(),
	}, "Premium code activated successfully")
}
