package shortlink

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/gin-gonic/gin"
)

// Redirect handles short link redirection and tracking
func (c *Controller) Redirect(ctx *gin.Context) {
	// 1. Bind URI parameter untuk code (required)
	var codeData dto.CodeRequest
	if err := ctx.ShouldBindUri(&codeData); err != nil {
		validator.SendValidationError(ctx, err, &codeData)
		return
	}

	// 2. Bind query parameter untuk passcode (optional)
	var passcodeData dto.PasscodeRequest
	if err := ctx.ShouldBindQuery(&passcodeData); err != nil {
		passcodeData.Passcode = 0
		validator.SendValidationError(ctx, err, &passcodeData)
		return
	}

	ipAddress := ctx.ClientIP()
	userAgent := ctx.Request.UserAgent()
	referer := ctx.Request.Referer()
	if referer == "" {
		referer = ctx.Request.Header.Get("URL")
	}

	// Get short link and track the view
	link, err := c.repo.RedirectByShortCode(codeData.Code, ipAddress, userAgent, referer, passcodeData.Passcode)
	if err != nil {
		httputil.HandleError(ctx, err, nil)
		return
	}

	// This should only be reached if no error occurred
	// Redirect to original URL
	ctx.Redirect(http.StatusMovedPermanently, link.OriginalURL)
}
