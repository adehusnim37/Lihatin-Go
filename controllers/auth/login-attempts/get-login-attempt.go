package loginattempts

import (
	"strings"

	"github.com/adehusnim37/lihatin-go/dto"
	httpPkg "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/gin-gonic/gin"
)

// GetLoginAttempt retrieves a login attempt by its ID
func (c *Controller) GetLoginAttempt(ctx *gin.Context) {
	logger.Logger.Info("GetLoginAttempt called")
	var req dto.IDGenericRequest

	role := ctx.GetString("role")
	isAdmin := strings.EqualFold(role, "admin")

	if err := ctx.ShouldBindUri(&req); err != nil {
		validator.SendValidationError(ctx, err, &req)
		return
	}

	loginAttempt, err := c.repo.GetLoginAttemptRepository().GetLoginAttemptByID(req.ID)
	if err != nil {
		httpPkg.HandleError(ctx, err, nil)
		return
	}

	// Authorization check: non-admin users can only view their own attempts
	if !isAdmin {
		userEmail := ctx.GetString("username")
		if loginAttempt.EmailOrUsername != userEmail {
			httpPkg.SendErrorResponse(ctx, 403, "ACCESS_DENIED", "You can only view your own login attempts", "", nil)
			return
		}
	}

	httpPkg.SendOKResponse(ctx, loginAttempt, "Login attempt retrieved successfully")
}
