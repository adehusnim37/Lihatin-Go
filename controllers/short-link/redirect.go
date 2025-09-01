package shortlink

import (
	"net/http"
	"strconv"

	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Redirect handles short link redirection and tracking
func (c *Controller) Redirect(ctx *gin.Context) {
	shortCode := ctx.Param("code")
	passcodeStr := ctx.Query("passcode")

	// Parse passcode if provided, default to 0 if not provided or invalid
	passcode := 0
	if passcodeStr != "" {
		if parsedPasscode, err := strconv.Atoi(passcodeStr); err == nil {
			passcode = parsedPasscode
		}
		// If parsing fails, passcode remains 0 (no passcode)
	}

	if shortCode == "" {
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Short code is required",
			Error:   map[string]string{"code": "Short code tidak boleh kosong"},
		})
		return
	}

	// Capture user data for tracking
	ipAddress := ctx.ClientIP()
	userAgent := ctx.Request.UserAgent()
	referer := ctx.Request.Referer()
	if referer == "" {
		referer = ctx.Request.Header.Get("URL")
	}

	// Get short link and track the view
	link, err := c.repo.RedirectByShortCode(shortCode, ipAddress, userAgent, referer, passcode)
	if err != nil {

		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Short link not found or expired",
				Error:   map[string]string{"details": "Link tidak ditemukan, tidak aktif, expired, atau telah mencapai batas klik"},
			})
			return
		}

		// Handle passcode errors specifically
		if err.Error() == "incorrect or missing passcode" {
			ctx.JSON(http.StatusUnauthorized, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Passcode required",
				Error:   map[string]string{"passcode": "Passcode yang diberikan salah atau tidak ada"},
			})
			return
		}

		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to retrieve short link",
			Error:   map[string]string{"details": err.Error()},
		})
		return
	}

	// This should only be reached if no error occurred
	// Redirect to original URL
	ctx.Redirect(http.StatusMovedPermanently, link.OriginalURL)
}
