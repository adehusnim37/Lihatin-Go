package shortlink

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

func (c *Controller) Create(ctx *gin.Context) {
	var req dto.CreateShortLinkRequest
	
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

	var userID string
    if userIDVal, exists := ctx.Get("user_id"); exists {
        userID = userIDVal.(string)
        utils.Logger.Info("User authenticated", "user_id", userID)
    }

    // 3. THIRD: Use authenticated user ID or fall back to request UserID (for anonymous)
    if userID == "" {
        userID = req.UserID // This could be empty for anonymous users
        utils.Logger.Info("No authenticated user, using request UserID", "user_id", userID)
    }

    // 4. Override request UserID with the final userID (authenticated takes priority)
    req.UserID = userID


	link := dto.CreateShortLinkRequest{
		UserID:      userID,
		OriginalURL: req.OriginalURL,
		Title:       req.Title,
		Description: req.Description,
		CustomCode:  req.CustomCode,
		Passcode:    req.Passcode, // 🎯 ADD THIS MISSING LINE!
		ExpiresAt:   req.ExpiresAt,
	}

	if err := c.repo.CreateShortLink(&link); err != nil {
		utils.Logger.Error("Failed to create short link",
			"error", err.Error(),
		)
		switch {
		case err == utils.ErrShortLinkNotFound:
			ctx.JSON(http.StatusNotFound, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Failed to retrieve short link stats",
				Error:   map[string]string{"code": "Link dengan kode tersebut tidak ditemukan"},
			})
		case err == utils.ErrShortLinkUnauthorized:
			ctx.JSON(http.StatusForbidden, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Failed to retrieve short link stats",
				Error:   map[string]string{"code": "Anda tidak memiliki akses ke link ini"},
			})
		default:
			ctx.JSON(http.StatusInternalServerError, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Failed to retrieve short link stats",
				Error:   map[string]string{"code": "Terjadi kesalahan pada server"},
			})
		}
		return
	}

	ctx.JSON(http.StatusCreated, common.APIResponse{
		Success: true,
		Data:    link,
		Message: "Short link created successfully",
		Error:   nil,
	})
}
