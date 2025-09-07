package shortlink

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

func (c *Controller) GetShortLink(ctx *gin.Context) {
	// Parse URI parameters
	var req dto.CodeRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		bindingErrors := utils.HandleJSONBindingError(err, &req)
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid request",
			Error:   bindingErrors,
		})
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

	// Get paginated views with complete data
	paginatedData, err := c.repo.GetShortLink(req.Code, userIDStr, userRoleStr)
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
