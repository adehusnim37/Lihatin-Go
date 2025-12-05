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

func (c *Controller) ReviveShortLink(ctx *gin.Context) {
	var codeData dto.CodeRequest

	if err := ctx.ShouldBindUri(&codeData); err != nil {
		validator.SendValidationError(ctx, err, &codeData)
		return
	}

	if err := c.repo.RestoreDeletedShortByAdmin(codeData.Code); err != nil {
		logger.Logger.Error("Failed to revive short link",
			"short_code", codeData.Code,
			"error", err.Error(),
		)
		switch err {
		case errors.ErrShortLinkNotFound:
			ctx.JSON(http.StatusNotFound, common.APIResponse{
				Success: false,
				Data:    nil,
			})
			return
		case errors.ErrShortIsNotDeleted:
			ctx.JSON(http.StatusBadRequest, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Failed to revive short link",
				Error:   map[string]string{"code": "Link dengan kode tersebut tidak dalam status dihapus"},
			})
			return
		default:
			ctx.JSON(http.StatusInternalServerError, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Failed to revive short link",
				Error:   map[string]string{"code": "Terjadi kesalahan pada server"},
			})
			return
		}
	}

	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data:    codeData,
		Message: "Short link revived successfully",
		Error:   nil,
	})
}
