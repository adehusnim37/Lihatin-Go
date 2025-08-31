package shortlink

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/models/shortlink"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (c *Controller) Create(ctx *gin.Context) {
	var req shortlink.CreateShortLinkRequest
	var userID string
	if userIDVal, exists := ctx.Get("user_id"); exists {
		userID = userIDVal.(string)
	}

	if userID == "" {
		userID = req.UserID
	}

	req.UserID = userID

	if err := ctx.ShouldBindJSON(&req); err != nil {
		// Use global function to handle JSON binding errors
		bindingErrors := utils.HandleJSONBindingError(err, &req)

		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid request body",
			Error:   bindingErrors,
		})
		return
	}

	// Validate the request using the validator
	if err := c.Validate.Struct(&req); err != nil {
		// Use global function to handle validation errors
		validationErrors := utils.HandleValidationError(err, &req)

		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Validation failed",
			Error:   validationErrors,
		})
		return
	}

	link := &shortlink.CreateShortLinkRequest{
		UserID:      req.UserID,
		OriginalURL: req.OriginalURL,
		Title:       req.Title,
		Description: req.Description,
		CustomCode:  req.CustomCode,
		Passcode:    req.Passcode, // 🎯 ADD THIS MISSING LINE!
		ExpiresAt:   req.ExpiresAt,
	}

	if err := c.repo.CreateShortLink(link); err != nil {
		if err == gorm.ErrDuplicatedKey {
			ctx.JSON(http.StatusConflict, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Short link already exists",
				Error:   map[string]string{"details": err.Error()},
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to create short link",
			Error:   map[string]string{"details": err.Error()},
		})
		return
	}

	ctx.JSON(http.StatusCreated, common.APIResponse{
		Success: true,
		Data:    link,
		Message: "Short link created successfully",
		Error:   nil,
	})
}
