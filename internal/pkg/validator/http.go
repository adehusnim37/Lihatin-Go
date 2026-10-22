package validator

import "github.com/gin-gonic/gin"

// SendValidationError writes the application's validation error envelope.
// Pass the original request struct to use its label and json tags in messages.
func SendValidationError(c *gin.Context, err error, input any) {
	result := FormatError(err, input)
	c.JSON(result.Status, gin.H{
		"success": result.Success,
		"data":    nil,
		"message": result.Message,
		"error":   result.ErrorMap,
	})
}
