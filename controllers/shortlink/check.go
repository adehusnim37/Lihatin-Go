package shortlink

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/internal/pkg/errors"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/gin-gonic/gin"
)

func (c *Controller) CheckShortLink(ctx *gin.Context) {
	var codeData dto.CodeRequest
	if err := ctx.ShouldBindUri(&codeData); err != nil {
		validator.SendValidationError(ctx, err, &codeData)
		return
	}

	_, err := c.repo.CheckShortCode(&codeData)
	if err != nil {
		switch err {
		case errors.ErrShortLinkNotFound:
			ctx.JSON(http.StatusNotFound, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Short code not exist. You can create it.",
				Error:   map[string]string{"details": err.Error()},
			})
		default:
			ctx.JSON(http.StatusInternalServerError, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Failed to check short link",
				Error:   map[string]string{"details": "Internal server error"},
			})
		}
		return
	}

	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data:    nil,
		Message: "Short code exists.",
		Error:   nil,
	})
}
