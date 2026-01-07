package shortlink

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/adehusnim37/lihatin-go/models/common"
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
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to check short link",
			Error:   map[string]string{"details": "Internal server error"},
		})
		return
	}

	// Check the boolean return value - if exists is false, the code doesn't exist
	if !exists {
		ctx.JSON(http.StatusNotFound, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Short code does not exist.",
			Error:   map[string]string{"details": "short link not found"},
		})
		return
	}

	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data:    nil,
		Message: "Short code exists.",
		Error:   nil,
	})
}
