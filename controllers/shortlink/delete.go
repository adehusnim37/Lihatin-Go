package shortlink

import (
	"net/http"
	"strconv"

	"github.com/adehusnim37/lihatin-go/dto"
	httpPkg "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/gin-gonic/gin"
)

func (c *Controller) DeleteShortLink(ctx *gin.Context) {
	var deleteData dto.DeleteRequest

	if err := ctx.ShouldBindUri(&deleteData); err != nil {
		validator.SendValidationError(ctx, err, &deleteData)
		return
	}

	if err := ctx.ShouldBindJSON(&deleteData); err != nil && err.Error() != "EOF" {
		validator.SendValidationError(ctx, err, &deleteData)
		return
	}

	shortCode := ctx.Param("code")
	var passcode int
	if deleteData.Passcode != "" {
		var err error
		passcode, err = strconv.Atoi(deleteData.Passcode)
		if err != nil {
			logger.Logger.Error("Invalid passcode format",
				"passcode", deleteData.Passcode,
				"error", err.Error(),
			)
			httpPkg.SendErrorResponse(ctx, http.StatusBadRequest, "Invalid passcode format", "passcode", "Passcode harus berupa angka")
			return
		}
	}

	if err := c.repo.DeleteShortLink(shortCode, ctx.GetString("user_id"), passcode, ctx.GetString("role")); err != nil {
		httpPkg.HandleError(ctx, err, ctx.GetString("user_id"))
		return
	}

	httpPkg.SendOKResponse(ctx, nil, "Short link deleted successfully")
}
