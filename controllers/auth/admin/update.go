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

	var req dto.AdminUpdateUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		validator.SendValidationError(ctx, err, &req)
		return
	}

	if err := c.repo.GetUserAdminRepository().UpdateUserByAdmin(userID.ID, req); err != nil {
		httputil.HandleError(ctx, err, userID)
		return
	}

	updatedUser, err := c.repo.GetUserAdminRepository().GetUserDetailByID(userID.ID)
	if err != nil {
		httputil.HandleError(ctx, err, userID)
		return
	}

	httputil.SendOKResponse(ctx, updatedUser, "User updated successfully")
}
