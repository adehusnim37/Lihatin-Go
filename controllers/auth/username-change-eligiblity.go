package auth

import (
	"net/http"

	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/gin-gonic/gin"
)

func (c *Controller) CheckUsernameChangeEligibility(ctx *gin.Context) {
	userID := ctx.GetString("user_id")

	// Check if user is eligible to change username
	if err := c.repo.GetUserRepository().CheckUsernameChangeEligibility(userID); err != nil {
		httputil.HandleError(ctx, err, userID)
		return
	}

	httputil.SendSuccessResponse(ctx, http.StatusOK, nil, "User is eligible to change username")
}
