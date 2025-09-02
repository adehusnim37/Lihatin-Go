package shortlink

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)


func (c *Controller) GetShortLinkStats(ctx *gin.Context) {
	// Implement the logic to retrieve and return short link statistics
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

	stats, err := c.repo.ShortLinkStats(req.Code)
	if err != nil {
		switch {
		case err == utils.ErrShortLinkNotFound:
			ctx.JSON(http.StatusNotFound, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Failed to retrieve short link stats",
				Error:   map[string]string{"code": "Link dengan kode tersebut tidak ditemukan"},
			})
		case err == utils.ErrShortLinkUnauthorized:
			ctx.JSON(http.StatusForbidden, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Failed to retrieve short link stats",
				Error:   map[string]string{"code": "Anda tidak memiliki akses ke link ini"},
			})
		default:
			ctx.JSON(http.StatusInternalServerError, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Failed to retrieve short link stats",
				Error:   map[string]string{"code": "Terjadi kesalahan pada server"},
			})
			return
		}
	}

	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data:    stats,
		Message: "Short link stats retrieved successfully",
		Error:   nil,
	})
}
