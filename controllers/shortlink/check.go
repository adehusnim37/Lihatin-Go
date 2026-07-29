package shortlink

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	httpPkg "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/gin-gonic/gin"
)

func (c *Controller) CheckShortLink(ctx *gin.Context) {
	var codeData dto.CodeRequest
	if err := ctx.ShouldBindUri(&codeData); err != nil {
		validator.SendValidationError(ctx, err, &codeData)
		return
	}

	preview, err := c.repo.CheckShortCode(&codeData)
	if err != nil {
		httpPkg.HandleError(ctx, err, nil)
		return
	}

	if preview == nil {
		httpPkg.SendErrorResponse(ctx, http.StatusNotFound, "Short code does not exist.", "code", "Short code does not exist.")
		return
	}

	httpPkg.SendOKResponse(ctx, preview, "Short code exists.")
}
