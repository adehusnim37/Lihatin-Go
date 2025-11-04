package admin

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

func (c *Controller) LoginAttemptsStats(ctx *gin.Context) {
	utils.Logger.Info("LoginAttemptsStats called")

	var req dto.LoginAttemptsStatsRequest

	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.SendErrorResponse(ctx, http.StatusBadRequest, "USER_ID_REQUIRED", "User ID is required", "user_id", nil)
		return
	}

	stats, err := c.repo.GetLoginAttemptRepository().GetLoginStats(req.EmailOrUsername, req.Days)
	if err != nil {
		utils.HandleError(ctx, err, nil)
		return
	}

	utils.SendOKResponse(ctx, stats, "Successfull Get Login Stats.")
}
