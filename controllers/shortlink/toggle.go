package shortlink

import (
	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/gin-gonic/gin"
)

func (c *Controller) SwitchActiveInActiveShort(ctx *gin.Context) {
	var codeData dto.CodeRequest
	userID := ctx.GetString("user_id")
	role := ctx.GetString("role")

	if err := ctx.ShouldBindUri(&codeData); err != nil {
		validator.SendValidationError(ctx, err, &codeData)
		return
	}

	if err := c.repo.ToggleActiveInActiveShort(codeData.Code, userID, role); err != nil {
		http.HandleError(ctx, err, userID)
		return
	}	

	http.SendOKResponse(ctx, nil, "Short link toggled successfully")
}