package premium

import (
	"errors"
	"net/http"
	"time"

	"github.com/adehusnim37/lihatin-go/dto"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	premiumpkg "github.com/adehusnim37/lihatin-go/internal/pkg/premium"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/adehusnim37/lihatin-go/models/user"
	"github.com/gin-gonic/gin"
)

func (c *Controller) GeneratePremiumCode(ctx *gin.Context) {
	var req dto.GeneratePremiumCodeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		validator.SendValidationError(ctx, err, &req)
		return
	}

	if req.ValidUntil.IsZero() || !req.ValidUntil.After(time.Now()) {
		httputil.SendErrorResponse(
			ctx,
			http.StatusBadRequest,
			"INVALID_VALID_UNTIL",
			"Valid until must be in the future",
			"valid_until",
		)
		return
	}

	total := 1
	if req.IsBulk {
		if req.Amount < 1 {
			httputil.SendErrorResponse(
				ctx,
				http.StatusBadRequest,
				"INVALID_BULK_AMOUNT",
				"Amount must be at least 1 for bulk generation",
				"amount",
			)
			return
		}
		total = req.Amount
	}

	validUntil := req.ValidUntil.UTC()
	limitUsage := int64(req.LimitUsage)

	// Pre-generate secret codes to avoid code duplication
	secretCodes := make([]string, total)
	for i := 0; i < total; i++ {
		code, err := premiumpkg.BuildSecretCode(validUntil)
		if err != nil {
			switch {
			case errors.Is(err, premiumpkg.ErrSecretKeyMissing):
				httputil.SendErrorResponse(
					ctx,
					http.StatusInternalServerError,
					"PREMIUM_CODE_CONFIG_INVALID",
					"Premium code is not configured",
					"secret_code",
				)
			default:
				httputil.SendErrorResponse(
					ctx,
					http.StatusInternalServerError,
					"PREMIUM_CODE_GENERATION_FAILED",
					"Failed to generate premium code",
					"secret_code",
				)
			}
			return
		}
		secretCodes[i] = code
	}

	if !req.IsBulk {
		premiumKey := user.PremiumKey{
			Code:       secretCodes[0],
			ValidUntil: &validUntil,
			LimitUsage: &limitUsage,
		}

		if err := c.premiumRepo.CreateUserPremiumKey(&premiumKey); err != nil {
			httputil.HandleError(ctx, err, nil)
			return
		}

		httputil.SendCreatedResponse(ctx, dto.GeneratePremiumCodeResponse{
			ID:         premiumKey.ID,
			SecretCode: premiumKey.Code,
			ValidUntil: premiumKey.ValidUntil,
			LimitUsage: premiumKey.LimitUsage,
			UsageCount: premiumKey.UsageCount,
			CreatedAt:  premiumKey.CreatedAt,
			UpdatedAt:  premiumKey.UpdatedAt,
		}, "Premium code generated successfully")
		return
	}

	premiumKeys := make([]user.PremiumKey, total)
	for i, code := range secretCodes {
		premiumKeys[i] = user.PremiumKey{
			Code:       code,
			ValidUntil: &validUntil,
			LimitUsage: &limitUsage,
		}
	}

	createdKeys, err := c.premiumRepo.CreateManyUserPremiumKeys(premiumKeys)
	if err != nil {
		httputil.HandleError(ctx, err, nil)
		return
	}

	items := make([]dto.GeneratePremiumCodeResponse, total)
	for i := range createdKeys {
		items[i] = dto.GeneratePremiumCodeResponse{
			ID:         createdKeys[i].ID,
			SecretCode: createdKeys[i].Code,
			ValidUntil: createdKeys[i].ValidUntil,
			LimitUsage: createdKeys[i].LimitUsage,
			UsageCount: createdKeys[i].UsageCount,
			CreatedAt:  createdKeys[i].CreatedAt,
			UpdatedAt:  createdKeys[i].UpdatedAt,
		}
	}

	httputil.SendCreatedResponse(ctx, dto.GeneratePremiumCodeListResponse{
		IsBulk: req.IsBulk,
		Total:  len(items),
		Items:  items,
	}, "Premium code generated successfully")
}
