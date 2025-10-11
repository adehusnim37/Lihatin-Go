package email

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/models/user"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

// UndoChangeEmail reverts the email change process using the provided token
func (c *Controller) UndoChangeEmail(ctx *gin.Context) {
	token := ctx.Query("token")

	