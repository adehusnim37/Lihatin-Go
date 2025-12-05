package admin

import (
	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/gin-gonic/gin"
)

// GetLoginAttempt retrieves a login attempt by its ID
func (c *Controller) GetLoginAttempt(ctx *gin.Context) {
	logger.Logger.Info("Admin GetLoginAttempt called")
	var req dto.IDGenericRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		validator.SendValidationError(ctx, err, &req)
		return
	}

	loginAttempts, err := c.repo.GetLoginAttemptRepository().GetLoginAttemptByID(req.ID)
	if err != nil {
		http.HandleError(ctx, err, nil)
		return
	}

	http.SendOKResponse(ctx, common.APIResponse{
		Success: true,
		Data:    loginAttempts,
		Message: "Successfully retrieved login attempt.",
		Error:   nil,
	}, "Successfully retrieved login attempt.")
}
