package notification

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/notifications"
	"github.com/gin-gonic/gin"
)

func (c *Controller) Unsubscribe(ctx *gin.Context) {
	token := strings.TrimSpace(ctx.Query("token"))
	userID, category, err := notifications.VerifyUnsubscribeToken(token)
	if err != nil {
		c.finishUnsubscribe(ctx, "", false)
		return
	}

	// Human-facing links use GET and must confirm in the frontend. This avoids
	// security scanners unsubscribing users merely by prefetching email links.
	if ctx.Request.Method == http.MethodGet {
		frontendURL := strings.TrimRight(config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000"), "/")
		query := url.Values{}
		query.Set("token", token)
		query.Set("category", category)
		ctx.Redirect(http.StatusFound, frontendURL+"/email-preferences/unsubscribe?"+query.Encode())
		return
	}

	if err := c.preferences.Unsubscribe(userID, category); err != nil {
		httputil.SendErrorResponse(
			ctx,
			http.StatusInternalServerError,
			"EMAIL_UNSUBSCRIBE_FAILED",
			"Failed to update email subscription",
			"token",
		)
		return
	}

	c.finishUnsubscribe(ctx, category, true)
}

func (c *Controller) finishUnsubscribe(ctx *gin.Context, category string, success bool) {
	if ctx.Request.Method == http.MethodGet {
		frontendURL := strings.TrimRight(config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000"), "/")
		query := url.Values{}
		query.Set("status", map[bool]string{true: "success", false: "invalid"}[success])
		if category != "" {
			query.Set("category", category)
		}
		ctx.Redirect(http.StatusFound, frontendURL+"/email-preferences/unsubscribed?"+query.Encode())
		return
	}

	if !success {
		httputil.SendErrorResponse(
			ctx,
			http.StatusBadRequest,
			"INVALID_UNSUBSCRIBE_TOKEN",
			"The unsubscribe link is invalid",
			"token",
		)
		return
	}

	httputil.SendOKResponse(ctx, gin.H{"category": category}, "Email subscription disabled successfully")
}
