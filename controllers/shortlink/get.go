package shortlink

import (
	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/internal/pkg/errors"
	"github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/gin-gonic/gin"
)

func (c *Controller) GetShortLink(ctx *gin.Context) {
	// Parse URI parameters
	var req dto.CodeRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		validator.SendValidationError(ctx, err, &req)
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
		http.HandleError(ctx, errors.ErrUserUnauthorized, nil)
		return
	}
	userIDStr := userID.(string)

	// Get paginated views with complete data
	paginatedData, err := c.repo.GetShortLink(req.Code, userIDStr, userRoleStr)
	if err != nil {
		http.HandleError(ctx, err, userID)
		return
	}

	http.SendOKResponse(ctx, paginatedData, "Short link views with pagination retrieved successfully")
}
