package shortlink

import (
	"github.com/adehusnim37/lihatin-go/dto"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
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
		httputil.HandleError(ctx, err, nil)
		return
	}

	httputil.SendOKResponse(ctx, codeData, "Short link revived successfully")
}
