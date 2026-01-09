package shortlink

import (
	"github.com/adehusnim37/lihatin-go/dto"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/gin-gonic/gin"
)

func (c *Controller) UpdateShortLink(ctx *gin.Context) {
	var updateData dto.UpdateShortLinkRequest
	shortCode := ctx.Param("code")
	if err := ctx.ShouldBindJSON(&updateData); err != nil {
		validator.SendValidationError(ctx, err, &updateData)
		return
	}

	if err := ctx.ShouldBindUri(&shortCode); err != nil {
		validator.SendValidationError(ctx, err, &shortCode)
		return
	}

	if err := c.repo.UpdateShortLink(shortCode, ctx.GetString("user_id"), ctx.GetString("user_role"), &updateData); err != nil {
		httputil.HandleError(ctx, err, ctx.GetString("user_id"))
		return
	}

	httputil.SendOKResponse(ctx, updateData, "Short link updated successfully")
}
