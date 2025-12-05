package shortlink

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/internal/pkg/errors"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/gin-gonic/gin"
)

func (c *Controller) GetShortLinkStats(ctx *gin.Context) {
	// Parse URI parameters
	var req dto.CodeRequest

	if err := ctx.ShouldBindUri(&req); err != nil {
		validator.SendValidationError(ctx, err, &req)
		return
	}

	stats, err := c.repo.GetStatsShortLink(req.Code, ctx.GetString("user_id"), ctx.GetString("role"))
	if err != nil {
		switch {
		case err == errors.ErrShortLinkNotFound:
			ctx.JSON(http.StatusNotFound, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Failed to retrieve short link stats",
				Error:   map[string]string{"code": "Link dengan kode tersebut tidak ditemukan"},
			})
		case err == errors.ErrShortLinkUnauthorized:
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
		}
		return
	}

	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data:    stats,
		Message: "Short link stats retrieved successfully",
		Error:   nil,
	})
}
