package shortlink

import (
	"net/http"
	"strconv"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/internal/pkg/errors"
	httpPkg "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/gin-gonic/gin"
)

func (c *Controller) DeleteShortLink(ctx *gin.Context) {
	var deleteData dto.DeleteRequest

	if err := ctx.ShouldBindUri(&deleteData); err != nil {
		validator.SendValidationError(ctx, err, &deleteData)
		return
	}

	if err := ctx.ShouldBindJSON(&deleteData); err != nil && err.Error() != "EOF" {
		validator.SendValidationError(ctx, err, &deleteData)
		return
	}

	shortCode := ctx.Param("code")
	var passcode int
	if deleteData.Passcode != "" {
		var err error
		passcode, err = strconv.Atoi(deleteData.Passcode)
		if err != nil {
			logger.Logger.Error("Invalid passcode format",
				"passcode", deleteData.Passcode,
				"error", err.Error(),
			)
			httpPkg.SendErrorResponse(ctx, http.StatusBadRequest, "Invalid passcode format", "passcode", "Passcode harus berupa angka")
			return
		}
	}

	if err := c.repo.DeleteShortLink(shortCode, ctx.GetString("user_id"), passcode, ctx.GetString("role")); err != nil {
		logger.Logger.Error("Failed to delete short link",
			"short_code", shortCode,
			"error", err.Error(),
		)
		switch err {
		case errors.ErrShortLinkNotFound:
			httpPkg.SendErrorResponse(ctx, http.StatusNotFound, "Failed to delete short link", "code", "Link dengan kode tersebut tidak ditemukan")
		case errors.ErrShortLinkUnauthorized:
			httpPkg.SendErrorResponse(ctx, http.StatusForbidden, "Failed to delete short link", "code", "Anda tidak memiliki akses ke link ini")
		case errors.ErrShortLinkAlreadyDeleted:
			httpPkg.SendErrorResponse(ctx, http.StatusGone, "Failed to delete short link", "code", "Link dengan kode tersebut sudah dihapus/tidak ditemukan")
		default:
			httpPkg.SendErrorResponse(ctx, http.StatusInternalServerError, "Failed to delete short link", "code", "Terjadi kesalahan pada server")
		}
		return
	}

	httpPkg.SendOKResponse(ctx, nil, "Short link deleted successfully")
}
