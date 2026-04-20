package admin

import (
	"net/http"
	"strings"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
	"github.com/adehusnim37/lihatin-go/internal/pkg/disposable"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/gin-gonic/gin"
)

func (c *Controller) GetDisposableEmailPolicy(ctx *gin.Context) {
	response, err := c.buildDisposableEmailPolicyResponse()
	if err != nil {
		httputil.SendErrorResponse(
			ctx,
			http.StatusInternalServerError,
			"DISPOSABLE_EMAIL_POLICY_READ_FAILED",
			"Failed to retrieve disposable email policy",
			"policy",
		)
		return
	}

	httputil.SendOKResponse(ctx, response, "Disposable email policy retrieved successfully")
}

func (c *Controller) UpdateDisposableEmailPolicy(ctx *gin.Context) {
	var req dto.UpdateAdminDisposableEmailPolicyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		httputil.SendErrorResponse(
			ctx,
			http.StatusBadRequest,
			"INVALID_POLICY_PAYLOAD",
			"Enabled field is required",
			"enabled",
		)
		return
	}

	adminID := strings.TrimSpace(ctx.GetString("user_id"))
	if err := c.repo.GetSystemSettingRepository().SetBool(disposable.SettingKeyDisposableEmailEnabled, *req.Enabled, adminID); err != nil {
		logger.Logger.Error("Failed to update disposable email policy",
			"admin_id", adminID,
			"enabled", *req.Enabled,
			"error", err.Error(),
		)
		httputil.SendErrorResponse(
			ctx,
			http.StatusInternalServerError,
			"DISPOSABLE_EMAIL_POLICY_UPDATE_FAILED",
			"Failed to update disposable email policy",
			"policy",
		)
		return
	}

	response, err := c.buildDisposableEmailPolicyResponse()
	if err != nil {
		httputil.SendErrorResponse(
			ctx,
			http.StatusInternalServerError,
			"DISPOSABLE_EMAIL_POLICY_READ_FAILED",
			"Failed to retrieve updated disposable email policy",
			"policy",
		)
		return
	}

	httputil.SendOKResponse(ctx, response, "Disposable email policy updated successfully")
}

func (c *Controller) buildDisposableEmailPolicyResponse() (dto.AdminDisposableEmailPolicyResponse, error) {
	settingRepo := c.repo.GetSystemSettingRepository()
	enabled, err := settingRepo.GetBool(disposable.SettingKeyDisposableEmailEnabled, true)
	if err != nil {
		return dto.AdminDisposableEmailPolicyResponse{}, err
	}

	setting, err := settingRepo.GetByKey(disposable.SettingKeyDisposableEmailEnabled)
	if err != nil {
		return dto.AdminDisposableEmailPolicyResponse{}, err
	}

	effective := disposable.ShouldEnforce(config.GetEnvOrDefault(config.Env, "development"), enabled)

	return dto.AdminDisposableEmailPolicyResponse{
		Enabled:               enabled,
		EffectiveInCurrentEnv: effective,
		LastUpdatedBy:         setting.UpdatedBy,
		LastUpdatedAt:         &setting.UpdatedAt,
	}, nil
}
