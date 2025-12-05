package shortlink

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
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
		logger.Logger.Error("Failed to bulk delete short links",
			"short_codes", reqs.Codes,
			"error", err.Error(),
		)
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to delete short links",
			Error:   map[string]string{"error": "Gagal menghapus beberapa short link, silakan coba lagi nanti"},
		})
		return
	}

	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data:    nil,
		Message: "Short links deleted successfully",
		Error:   nil,
	})
}
