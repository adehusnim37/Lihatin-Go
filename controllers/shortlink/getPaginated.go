package shortlink

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

func (c *Controller) GetShortLinkViewsPaginated(ctx *gin.Context) {
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
		ctx.JSON(http.StatusUnauthorized, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Unauthorized",
			Error:   map[string]string{"user": "User not authenticated"},
		})
		return
	}
	userIDStr := userID.(string)

	// Get pagination parameters from query string
	pageStr := ctx.DefaultQuery("page", "1")
	limitStr := ctx.DefaultQuery("limit", "10")
	sortStr := ctx.DefaultQuery("sort", "clicked_at")
	orderByStr := ctx.DefaultQuery("order_by", "desc")

	// Validate pagination parameters for views
	page, limit, sort, orderBy, vErrs := utils.PaginateValidate(pageStr, limitStr, sortStr, orderByStr, utils.Role(userRoleStr))
	if vErrs != nil {
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid pagination parameters",
			Error:   vErrs,
		})
		return
	}

	utils.Logger.Info("Fetching paginated short link views",
		"user_id", userIDStr,
		"user_role", userRoleStr,
		"code", req.Code,
		"page", page,
		"limit", limit,
		"sort", sort,
		"order_by", orderBy,
	)

	// Get paginated views with complete data
	paginatedData, err := c.repo.GetShortLinkViewsPaginated(req.Code, userIDStr, page, limit, sort, orderBy, userRoleStr)
	if err != nil {
		switch {
		case err == utils.ErrShortLinkNotFound:
			ctx.JSON(http.StatusNotFound, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Failed to retrieve short link views",
				Error:   map[string]string{"code": "Link dengan kode tersebut tidak ditemukan"},
			})
		case err == utils.ErrShortLinkUnauthorized:
			ctx.JSON(http.StatusForbidden, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Failed to retrieve short link views",
				Error:   map[string]string{"code": "Anda tidak memiliki akses ke link ini"},
			})
		default:
			ctx.JSON(http.StatusInternalServerError, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Failed to retrieve short link views",
				Error:   map[string]string{"code": "Terjadi kesalahan pada server"},
			})
		}
		return
	}

	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data:    paginatedData,
		Message: "Short link views with pagination retrieved successfully",
		Error:   nil,
	})
}
