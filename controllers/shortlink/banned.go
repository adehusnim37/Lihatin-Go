package shortlink

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/internal/pkg/errors"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/gin-gonic/gin"
)

func (c *Controller) AdminBannedShortLink(ctx *gin.Context) {
	var banData dto.BannedRequest
	var codeData dto.CodeRequest

	if err := ctx.ShouldBindUri(&codeData); err != nil {
		validator.SendValidationError(ctx, err, &codeData)
		return
	}
	// Bind JSON
	if err := ctx.ShouldBindJSON(&banData); err != nil {
		validator.SendValidationError(ctx, err, &banData)
		return
	}

	// Set data dari param dan JWT

	if err := c.repo.BannedShortByAdmin(&banData, ctx.GetString("user_id"), &codeData); err != nil {
		logger.Logger.Error("Failed to banned short link",
			"short_code", codeData.Code,
			"error", err.Error(),
		)
		switch err {
		case errors.ErrShortLinkNotFound:
			ctx.JSON(http.StatusNotFound, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Failed to banned short link",
				Error:   map[string]string{"code": "Link dengan kode tersebut tidak ditemukan"},
			})
		case errors.ErrShortLinkUnauthorized:
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
		Data:    banData,
		Message: "Short link banned successfully",
		Error:   nil,
	})
}
