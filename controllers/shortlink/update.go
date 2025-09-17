package shortlink

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

func (c *Controller) UpdateShortLink(ctx *gin.Context) {
	var updateData dto.UpdateShortLinkRequest
	shortCode := ctx.Param("code")
	if err := ctx.ShouldBindJSON(&updateData); err != nil {
		utils.SendValidationError(ctx, err, &updateData)
		return
	}

	if err := ctx.ShouldBindUri(&shortCode); err != nil {
		utils.SendValidationError(ctx, err, &shortCode)
		return
	}

	if err := c.repo.UpdateShortLink(shortCode, ctx.GetString("user_id"), ctx.GetString("user_role"), &updateData); err != nil {
		utils.Logger.Error("Failed to update short link",
			"short_code", shortCode,
			"error", err.Error(),
		)
		switch err {
		case utils.ErrShortLinkNotFound:
			ctx.JSON(http.StatusNotFound, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Failed to update short link",
				Error:   map[string]string{"code": "Link dengan kode tersebut tidak ditemukan"},
			})
		case utils.ErrShortLinkUnauthorized:
			ctx.JSON(http.StatusForbidden, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Failed to update short link",
				Error:   map[string]string{"code": "Anda tidak memiliki akses ke link ini"},
			})
		default:
			ctx.JSON(http.StatusInternalServerError, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Failed to update short link",
				Error:   map[string]string{"code": "Terjadi kesalahan pada server"},
			})
		}
		return
	}

	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data:    updateData,
		Message: "Short link updated successfully",
		Error:   nil,
	})
}
