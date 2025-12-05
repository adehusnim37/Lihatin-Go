package http

import (
	"errors"
	"net/http"

	apperrors "github.com/adehusnim37/lihatin-go/internal/pkg/errors"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/gin-gonic/gin"
)

// HandleError provides universal error handling - now much simpler!
func HandleError(ctx *gin.Context, err error, userID any) {
	// Try to cast to AppError
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		// It's an AppError, use its metadata directly
		logger.Logger.Error("API Error",
			"status_code", appErr.StatusCode,
			"error_code", appErr.Code,
			"message", appErr.Message,
			"field", appErr.Field,
			"user_id", userID,
		)

		ctx.JSON(appErr.StatusCode, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: appErr.Message,
			Error:   map[string]string{appErr.Field: appErr.Message},
		})
		return
	}

	// Fallback for unknown errors
	logger.Logger.Error("Unhandled error occurred",
		"error", err.Error(),
		"user_id", userID,
	)
	ctx.JSON(http.StatusInternalServerError, common.APIResponse{
		Success: false,
		Data:    nil,
		Message: "Internal server error",
		Error:   map[string]string{"server": err.Error()},
	})
}

// SendSuccessResponse sends a standardized success response
func SendSuccessResponse(ctx *gin.Context, statusCode int, data interface{}, message string) {
	ctx.JSON(statusCode, common.APIResponse{
		Success: true,
		Data:    data,
		Message: message,
		Error:   nil,
	})
}

// SendCreatedResponse sends a standardized 201 Created response
func SendCreatedResponse(ctx *gin.Context, data interface{}, message string) {
	SendSuccessResponse(ctx, http.StatusCreated, data, message)
}

// SendOKResponse sends a standardized 200 OK response
func SendOKResponse(ctx *gin.Context, data interface{}, message string) {
	SendSuccessResponse(ctx, http.StatusOK, data, message)
}

// SendNoContentResponse sends a standardized 204 No Content response
func SendNoContentResponse(ctx *gin.Context, message string) {
	ctx.JSON(http.StatusNoContent, common.APIResponse{
		Success: true,
		Data:    nil,
		Message: message,
		Error:   nil,
	})
}

func SendErrorResponse(ctx *gin.Context, statusCode int, errorCode, message, field string, userID ...any) {
	var uid any
	if len(userID) > 0 {
		uid = userID[0]
	}

	logArgs := []any{
		"status_code", statusCode,
		"error_code", errorCode,
		"message", message,
		"field", field,
	}
	if uid != nil {
		logArgs = append(logArgs, "user_id", uid)
	}

	logger.Logger.Error("API Error", logArgs...)

	ctx.JSON(statusCode, common.APIResponse{
		Success: false,
		Data:    nil,
		Message: message,
		Error:   map[string]string{field: message},
	})
}
