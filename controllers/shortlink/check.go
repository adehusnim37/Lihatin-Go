package shortlink

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	httpPkg "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/gin-gonic/gin"
)

func (c *Controller) CheckShortLink(ctx *gin.Context) {
	var codeData dto.CodeRequest
	if err := ctx.ShouldBindUri(&codeData); err != nil {
		validator.SendValidationError(ctx, err, &codeData)
		return
	}

	exists, err := c.repo.CheckShortCode(&codeData)
	if err != nil {
		httpPkg.SendErrorResponse(ctx, http.StatusInternalServerError, "Failed to check short link", "Connection", "There's an error on our end. Please try again later.")
		return
	}

	// Check the boolean return value - if exists is false, the code doesn't exist
	if !exists {
		httpPkg.SendErrorResponse(ctx, http.StatusNotFound, "Short code does not exist.", "code", "Short code does not exist.")
		return
	}

	httpPkg.SendOKResponse(ctx, nil, "Short code exists.")
}
