package shortlink

import (
	"errors"
	"net/http"

	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

type DTO struct {
	Code string `json:"code" binding:"required,min=1,max=100,alphanum" uri:"code"`
}

func (c *Controller) GetShortLink(ctx *gin.Context) {
	var dto DTO

	if err := ctx.ShouldBindUri(&dto); err != nil {
		bindingErrors := utils.HandleJSONBindingError(err, &dto)
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid request",
			Error:   bindingErrors,
		})
		return
	}

	shortLink, err := c.repo.GetShortLinkByShortCode(dto.Code, ctx.GetString("user_id"))

	if err != nil {
		switch {
		case errors.Is(err, utils.ErrShortLinkNotFound):
			ctx.JSON(http.StatusNotFound, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Short link not found",
				Error:   map[string]string{"code": "Link dengan kode tersebut tidak ditemukan"},
			})
		case errors.Is(err, utils.ErrShortLinkUnauthorized):
			ctx.JSON(http.StatusForbidden, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Access denied",
				Error:   map[string]string{"code": "Anda tidak memiliki akses ke link ini"},
			})
		default:
			ctx.JSON(http.StatusInternalServerError, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Internal server error",
				Error:   map[string]string{"code": "Terjadi kesalahan pada server"},
			})
		}
		return
	}

	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data:    shortLink,
		Message: "Short link retrieved successfully",
		Error:   nil,
	})
}
