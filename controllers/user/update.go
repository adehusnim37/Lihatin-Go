package user

import (
	"database/sql"
	"net/http"

	"github.com/adehusnim37/lihatin-go/models"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// Update memperbarui user yang ada
func (c *Controller) Update(ctx *gin.Context) {
	id := ctx.Param("id")
	var user models.UpdateUser
	if err := ctx.ShouldBindJSON(&user); err != nil {
		ctx.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid input",
			Error:   map[string]string{"input": "Invalid input format or missing fields, please check your request body"},
		})
		return
	}

	// Validate the user input
	if err := c.Validate.Struct(user); err != nil {
		errorMap := make(map[string]string)
		for _, err := range err.(validator.ValidationErrors) {
			field := err.Field()
			switch err.Tag() {
			case "required":
				errorMap[field] = field + " is required"
			case "email":
				errorMap[field] = "Invalid email format"
			case "min":
				errorMap[field] = field + " must be at least " + err.Param() + " characters"
			case "max":
				errorMap[field] = field + " must not exceed " + err.Param() + " characters"
			case "pwdcomplex":
				errorMap[field] = "Password must contain at least one letter, one number, and one special character"
			case "url":
				errorMap[field] = "Invalid URL format"
			default:
				errorMap[field] = "Invalid value for " + field
			}
		}

		ctx.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Validation failed",
			Error:   errorMap,
		})
		return
	}

	_, err := c.repo.GetUserByID(id) // Check if user exists
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, models.APIResponse{
				Success: false,
				Data:    nil,
				Message: "User not found",
				Error:   map[string]string{"id": "User with the specified ID does not exist"},
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to retrieve user",
			Error:   map[string]string{"error": "An error occurred while retrieving the user, please try again later"},
		})
		return
	}

	if err := c.repo.UpdateUser(id, &user); err != nil {
		// Check if it's an email duplication error
		if err.Error() == "email already exists" {
			ctx.JSON(http.StatusConflict, models.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Email already exists",
				Error:   map[string]string{"email": "Email already exists, please use a different email"},
			})
			return
		}

		ctx.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to update user",
			Error:   map[string]string{"error": "Failed to update user, please try again later"},
		})
		return
	}
	ctx.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    user,
		Message: "User updated successfully",
		Error:   nil,
	})
}
