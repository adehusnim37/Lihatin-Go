package email

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

// VerifyEmail verifies email with the provided token
func (c *Controller) VerifyEmail(ctx *gin.Context) {
    token := ctx.Query("token")
    if token == "" {
        utils.Logger.Warn("Email verification attempted without token")
        utils.SendErrorResponse(ctx, http.StatusBadRequest, "TOKEN_REQUIRED", "Verification token is required", "token", nil)
        return
    }

    // Verify token
    err := c.repo.GetUserAuthRepository().VerifyEmail(token)
    if err != nil {
        utils.Logger.Error("Email verification failed",
            "token", token,
            "error", err.Error(),
        )
        utils.HandleError(ctx, err, nil)
        return
    }

    utils.Logger.Info("Email verified successfully", "token", token)

    // Send success response
    utils.SendOKResponse(ctx, map[string]interface{}{
        "message": "Email verification completed successfully",
    }, "Email verified successfully")
}
