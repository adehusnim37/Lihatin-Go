package admin

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/gin-gonic/gin"
)

func (c *Controller) UpdateUser(ctx *gin.Context) {
	var userID dto.UserIDGenericRequest
	if err := ctx.ShouldBindUri(&userID); err != nil {
		httputil.SendErrorResponse(ctx, http.StatusBadRequest, "USER_ID_REQUIRED", "User ID is required", "user_id", userID)
		return
	}

	var req dto.UpdateProfileRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		validator.SendValidationError(ctx, err, &req)
		return
	}

	if err := c.repo.GetUserRepository().UpdateUser(userID.ID, req); err != nil {
		httputil.HandleError(ctx, err, userID)
		return
	}

	updatedUser, err := c.repo.GetUserRepository().GetUserByID(userID.ID)
	if err != nil {
		httputil.HandleError(ctx, err, userID)
		return
	}

	httputil.SendOKResponse(ctx, toAdminUserResponse(*updatedUser), "User updated successfully")
}
