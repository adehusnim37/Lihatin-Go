package shortlink

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/internal/pkg/errors"
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
		switch err {
		case errors.ErrShortLinkNotFound:
			ctx.JSON(http.StatusNotFound, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Short link not found",
				Error:   map[string]string{"details": err.Error()},
			})
		case errors.ErrPasscodeRequired:
			ctx.JSON(http.StatusUnauthorized, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Passcode required",
				Error:   map[string]string{"details": err.Error()},
			})
		case errors.ErrInvalidPasscode:
			ctx.JSON(http.StatusUnauthorized, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Invalid passcode",
				Error:   map[string]string{"details": err.Error()},
			})
		case errors.ErrShortLinkUnauthorized:
			ctx.JSON(http.StatusForbidden, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Unauthorized access",
				Error:   map[string]string{"details": err.Error()},
			})
		case errors.ErrShortLinkInactive:
			ctx.JSON(http.StatusGone, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Short link is inactive",
				Error:   map[string]string{"details": err.Error()},
			})
		case errors.ErrShortLinkExpired:
			ctx.JSON(http.StatusGone, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Short link has expired",
				Error:   map[string]string{"details": err.Error()},
			})
		case errors.ErrLinkIsBanned:
			ctx.JSON(http.StatusForbidden, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "The original URL is banned",
				Error:   map[string]string{"details": err.Error()},
			})
		case errors.ErrShortLinkAlreadyDeleted:
			ctx.JSON(http.StatusGone, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Short link has already been deleted",
				Error:   map[string]string{"details": err.Error()},
			})
		case errors.ErrClickLimitReached:
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
