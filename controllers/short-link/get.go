package shortlink

import (
	"net/http"
	"strconv"

	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (c *Controller) GetShortLink(ctx *gin.Context) {
	id := ctx.Param("id")
	passcodeStr := ctx.Query("passcode")
	
	passcode, err := strconv.Atoi(passcodeStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid passcode format",
			Error:   map[string]string{"passcode": "Passcode must be a valid integer"},
		})
		return
	}
	shortLink, err := c.repo.GetShortLinkByShortCode(id, ctx.ClientIP(), ctx.Request.UserAgent(), ctx.Request.Referer(), passcode)


	if err != nil && err != gorm.ErrRecordNotFound {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to retrieve short link. Looks like the link is not active or has expired.",
			Error:   map[string]string{"id": "Failed to retrieve short link. Looks like the link is not active or has expired."},
		})
		return
	}

	if err != nil {
		ctx.JSON(http.StatusNotFound, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Short link not found",
			Error:   map[string]string{"id": "Short link not found"},
		})
		return
	}

	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data:    shortLink,
		Message: "Short link retrieved successfully",
		Error:   nil,
	})
}
