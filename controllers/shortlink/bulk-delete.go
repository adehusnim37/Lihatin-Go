package shortlink

import (
	"github.com/adehusnim37/lihatin-go/dto"
	httpPkg "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/gin-gonic/gin"
)

func (c *Controller) AdminBulkDeleteShortLinks(ctx *gin.Context) {
	var reqs *dto.BulkDeleteRequest

	if err := ctx.ShouldBindJSON(&reqs); err != nil {
		validator.SendValidationError(ctx, err, &reqs)
		return
	}

	if err := c.repo.DeleteShortsLink(reqs); err != nil {
		userID := ctx.GetString("user_id")
		httpPkg.HandleError(ctx, err, userID)
		return
	}

	httpPkg.SendOKResponse(ctx, nil, "Short links deleted successfully")
}
