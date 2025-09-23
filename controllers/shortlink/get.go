package shortlink

import (
	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

func (c *Controller) GetShortLink(ctx *gin.Context) {
	// Parse URI parameters
	var req dto.CodeRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.SendValidationError(ctx, err, &req)
		return
	}

	// Get user role and ID
	userRole, roleExists := ctx.Get("role")
	if !roleExists {
		userRole = "user"
	}
	userRoleStr := userRole.(string)

	userID, userExists := ctx.Get("user_id")
	if !userExists {
		utils.HandleError(ctx, utils.ErrUserUnauthorized, nil)
		return
	}
	userIDStr := userID.(string)

	// Get paginated views with complete data
	paginatedData, err := c.repo.GetShortLink(req.Code, userIDStr, userRoleStr)
	if err != nil {
		utils.HandleError(ctx, err, userID)
		return
	}

	utils.SendOKResponse(ctx, paginatedData, "Short link views with pagination retrieved successfully")
}
