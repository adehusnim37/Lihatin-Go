package shortlink

import (
	"github.com/adehusnim37/lihatin-go/dto"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/gin-gonic/gin"
)

func (c *Controller) AdminUnbanShortLink(ctx *gin.Context) {
	var unbanData dto.CodeRequest

	if err := ctx.ShouldBindUri(&unbanData); err != nil {
		validator.SendValidationError(ctx, err, &unbanData)
		return
	}

	if err := c.repo.RestoreShortByAdmin(unbanData.Code); err != nil {
		httputil.HandleError(ctx, err, ctx.GetString("user_id"))
		return
	}

	httputil.SendOKResponse(ctx, unbanData, "Short link unbanned successfully")
}
