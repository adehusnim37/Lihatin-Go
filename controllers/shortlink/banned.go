package shortlink

import (
	"github.com/adehusnim37/lihatin-go/dto"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
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
		httputil.HandleError(ctx, err, ctx.GetString("user_id"))
		return
	}

	httputil.SendOKResponse(ctx, banData, "Short link banned successfully")
}
