package admin

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/gin-gonic/gin"
)

func (c *Controller) GetUserDetailByID(ctx *gin.Context) {
	logger.Logger.Info("Admin GetUserDetailByID called")

	var userID dto.UserIDGenericRequest
	if err := ctx.ShouldBindUri(&userID); err != nil {
		httputil.SendErrorResponse(ctx, http.StatusBadRequest, "USER_ID_REQUIRED", "User ID is required", "user_id", userID)
		return
	}

	logger.Logger.Info("Fetching detailed user by ID", "user_id", userID.ID)

	detail, err := c.repo.GetUserAdminRepository().GetUserDetailByID(userID.ID)
	if err != nil {
		httputil.HandleError(ctx, err, userID)
		return
	}

	httputil.SendOKResponse(ctx, detail, "User detailed profile retrieved successfully")
}
