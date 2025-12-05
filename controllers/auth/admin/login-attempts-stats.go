package admin

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/gin-gonic/gin"
)

func (c *Controller) LoginAttemptsStats(ctx *gin.Context) {
	logger.Logger.Info("LoginAttemptsStats called")

	var req dto.LoginAttemptsStatsRequest

	if err := ctx.ShouldBindUri(&req); err != nil {
		httputil.SendErrorResponse(ctx, http.StatusBadRequest, "USER_ID_REQUIRED", "User ID is required", "user_id", nil)
		return
	}

	stats, err := c.repo.GetLoginAttemptRepository().GetLoginStats(req.EmailOrUsername, req.Days)
	if err != nil {
		httputil.HandleError(ctx, err, nil)
		return
	}

	httputil.SendOKResponse(ctx, stats, "Successfull Get Login Stats.")
}
