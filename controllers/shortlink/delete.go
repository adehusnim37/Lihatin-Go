package shortlink

import (
	"net/http"
	"strconv"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

func (c *Controller) DeleteShortLink(ctx *gin.Context) {
	var deleteData dto.DeleteRequest

	if err := ctx.ShouldBindUri(&deleteData); err != nil {
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid URI parameter",
			Error:   map[string]string{"code": "Invalid short code"},
		})
		return
	}

	if err := ctx.ShouldBindJSON(&deleteData); err != nil {
		bindingErrors := utils.HandleJSONBindingError(err, &deleteData)
		utils.Logger.Error("Failed to bind JSON",
			"error", err.Error(),
		)
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid request",
			Error:   bindingErrors,
		})
		return
	}

	shortCode := ctx.Param("code")
	passcode, err := strconv.Atoi(deleteData.Passcode)
	if err != nil {
		utils.Logger.Error("Invalid passcode format",
			"passcode", deleteData.Passcode,
			"error", err.Error(),
		)
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid passcode format",
			Error:   map[string]string{"passcode": "Passcode harus berupa angka"},
		})
		return
	}

	if err := c.repo.DeleteShortLink(shortCode, ctx.GetString("user_id"), passcode); err != nil {
		utils.Logger.Error("Failed to delete short link",
			"short_code", shortCode,
			"error", err.Error(),
		)
		switch {
		case err == utils.ErrShortLinkNotFound:
			ctx.JSON(http.StatusNotFound, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Failed to delete short link",
				Error:   map[string]string{"code": "Link dengan kode tersebut tidak ditemukan"},
			})
		case err == utils.ErrShortLinkUnauthorized:
			ctx.JSON(http.StatusForbidden, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Failed to delete short linkq",
				Error:   map[string]string{"code": "Anda tidak memiliki akses ke link ini"},
			})
		case err == utils.ErrShortLinkAlreadyDeleted:
			ctx.JSON(http.StatusGone, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Failed to delete short link",
				Error:   map[string]string{"code": "Link dengan kode tersebut sudah dihapus/tidak ditemukan."},
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
		Data:    deleteData,
		Message: "Short link updated successfully",
		Error:   nil,
	})
}
