package loginattempts

import (
	"net/http"
	"strings"

	"github.com/adehusnim37/lihatin-go/dto"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/gin-gonic/gin"
)

func (c *Controller) LoginAttemptsStats(ctx *gin.Context) {
	logger.Logger.Info("LoginAttemptsStats called")

	var req dto.LoginAttemptsStatsRequest

	if err := ctx.ShouldBindUri(&req); err != nil {
		httputil.SendErrorResponse(ctx, http.StatusBadRequest, "INVALID_PARAMS", "Invalid parameters", "params", nil)
		return
	}

	// Authorization check: non-admin users can only view their own stats
		role := ctx.GetString("role")
		isAdmin := strings.EqualFold(role, "admin")
		
	if !isAdmin {
		userEmail := ctx.GetString("username")
		if req.EmailOrUsername != userEmail {
			httputil.SendErrorResponse(ctx, 403, "ACCESS_DENIED", "You can only view your own statistics", "", nil)
			return
		}
	}

	stats, err := c.repo.GetLoginAttemptRepository().GetLoginStats(req.EmailOrUsername, req.Days)
	if err != nil {
		httputil.HandleError(ctx, err, nil)
		return
	}

	// Enhance response with additional context
	response := map[string]any{
		"stats":             stats,
		"email_or_username": req.EmailOrUsername,
		"days":              req.Days,
		"period_start":      stats["period_start"],
		"period_end":        stats["period_end"],
	}

	httputil.SendOKResponse(ctx, response, "Successfully retrieved login statistics")
}
