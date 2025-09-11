package shortlink

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

func (c *Controller) ReviveShortLink(ctx *gin.Context) {
	var codeData dto.CodeRequest

	if err := ctx.ShouldBindUri(&codeData); err != nil {
		utils.SendValidationError(ctx, err, &codeData)
		return
	}

	if err := c.repo.RestoreDeletedShortByAdmin(codeData.Code); err != nil {
		utils.Logger.Error("Failed to revive short link",
			"short_code", codeData.Code,
			"error", err.Error(),
		)
		switch err {
		case utils.ErrShortLinkNotFound:
			ctx.JSON(http.StatusNotFound, common.APIResponse{
				Success: false,
				Data:    nil,
			})
		case utils.ErrShortIsNotDeleted:
			ctx.JSON(http.StatusBadRequest, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Failed to revive short link",
				Error:   map[string]string{"code": "Link dengan kode tersebut tidak dalam status dihapus"},
			})
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
