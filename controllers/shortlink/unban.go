package shortlink

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

func (c *Controller) AdminUnbanShortLink(ctx *gin.Context) {
	var unbanData dto.CodeRequest

	if err := ctx.ShouldBindUri(&unbanData); err != nil {
		utils.SendValidationError(ctx, err, &unbanData)
		return
	}

	if err := c.repo.RestoreShortByAdmin(unbanData.Code); err != nil {
		utils.Logger.Error("Failed to banned short link",
			"short_code", unbanData.Code,
			"error", err.Error(),
		)
		switch err {
		case utils.ErrShortLinkNotFound:
			ctx.JSON(http.StatusNotFound, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Failed to banned short link",
				Error:   map[string]string{"code": "Link dengan kode tersebut tidak ditemukan"},
			})
		case utils.ErrShortLinkUnauthorized:
			ctx.JSON(http.StatusForbidden, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Failed to banned short link",
				Error:   map[string]string{"code": "Anda tidak memiliki akses ke link ini"},
			})
		default:
			ctx.JSON(http.StatusInternalServerError, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Failed to banned short link",
				Error:   map[string]string{"code": "Terjadi kesalahan pada server"},
			})
		}
		return
	}

	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data:    unbanData,
		Message: "Short link unbanned successfully",
		Error:   nil,
	})
}
