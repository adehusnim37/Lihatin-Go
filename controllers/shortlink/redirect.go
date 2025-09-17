package shortlink

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

// Redirect handles short link redirection and tracking
func (c *Controller) Redirect(ctx *gin.Context) {
	// 1. Bind URI parameter untuk code (required)
	var codeData dto.CodeRequest
	if err := ctx.ShouldBindUri(&codeData); err != nil {
		utils.SendValidationError(ctx, err, &codeData)
		return
	}

	// 2. Bind query parameter untuk passcode (optional)
	var passcodeData dto.PasscodeRequest
	if err := ctx.ShouldBindQuery(&passcodeData); err != nil {
		passcodeData.Passcode = 0
		utils.SendValidationError(ctx, err, &passcodeData)
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
		switch err {
		case utils.ErrShortLinkNotFound:
			ctx.JSON(http.StatusNotFound, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Short link not found",
				Error:   map[string]string{"details": err.Error()},
			})
		case utils.ErrPasscodeRequired:
			ctx.JSON(http.StatusUnauthorized, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Passcode required",
				Error:   map[string]string{"details": err.Error()},
			})
		case utils.ErrInvalidPasscode:
			ctx.JSON(http.StatusUnauthorized, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Invalid passcode",
				Error:   map[string]string{"details": err.Error()},
			})
		case utils.ErrShortLinkUnauthorized:
			ctx.JSON(http.StatusForbidden, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Unauthorized access",
				Error:   map[string]string{"details": err.Error()},
			})
		case utils.ErrShortLinkInactive:
			ctx.JSON(http.StatusGone, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Short link is inactive",
				Error:   map[string]string{"details": err.Error()},
			})
		case utils.ErrShortLinkExpired:
			ctx.JSON(http.StatusGone, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Short link has expired",
				Error:   map[string]string{"details": err.Error()},
			})
		case utils.ErrLinkIsBanned:
			ctx.JSON(http.StatusForbidden, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "The original URL is banned",
				Error:   map[string]string{"details": err.Error()},
			})
		case utils.ErrShortLinkAlreadyDeleted:
			ctx.JSON(http.StatusGone, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Short link has already been deleted",
				Error:   map[string]string{"details": err.Error()},
			})
		case utils.ErrClickLimitReached:
			ctx.JSON(http.StatusTooManyRequests, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Click limit reached",
				Error:   map[string]string{"details": err.Error()},
			})
		default:
			ctx.JSON(http.StatusInternalServerError, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Failed to retrieve short link",
				Error:   map[string]string{"details": err.Error()},
			})
		}
		return
	}

	// This should only be reached if no error occurred
	// Redirect to original URL
	ctx.Redirect(http.StatusMovedPermanently, link.OriginalURL)
}
