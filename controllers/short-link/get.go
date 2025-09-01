package shortlink

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/gin-gonic/gin"
)

type DTO struct {
	Code string `json:"code" binding:"required,min=1,max=100,alphanum" uri:"code"`
}

func (c *Controller) GetShortLink(ctx *gin.Context) {
	var dto DTO

	if err := ctx.ShouldBindUri(&dto); err != nil {
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid request",
			Error:   map[string]string{"code": "Invalid short code format"},
		})
		return
	}

	shortLink, err := c.repo.GetShortLinkByShortCode(dto.Code)

	if err != nil {
		ctx.JSON(http.StatusNotFound, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Short link not found",
			Error:   map[string]string{"id": "Short link not found"},
		})
		return
	}

	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data:    shortLink,
		Message: "Short link retrieved successfully",
		Error:   nil,
	})
}
