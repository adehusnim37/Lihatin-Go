package user

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/models"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// Create membuat user baru
func (c *Controller) Create(ctx *gin.Context) {
	var user models.User
	if err := ctx.ShouldBindJSON(&user); err != nil {
		ctx.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid input",
			Error:   map[string]string{"input": "Invalid input format or missing fields, please check your request body"},
		})
		return
	}

	// Validate the user input based on the validation rules
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

	// Check if the email already exists
	existingUser, err := c.repo.GetUserByEmailOrUsername(user.Email)
	if err == nil && existingUser != nil {
		ctx.JSON(http.StatusConflict, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Email already exists",
			Error:   map[string]string{"email": "Email already exists, please use a different email"},
		})
		return
	}

	// Check if the username already exists
	existingUser, err = c.repo.GetUserByEmailOrUsername(user.Username)
	if err == nil && existingUser != nil {
		ctx.JSON(http.StatusConflict, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Username already exists",
			Error:   map[string]string{"username": "Username already exists, please use a different username"},
		})
		return
	}

	if err := c.repo.CreateUser(&user); err != nil {
		ctx.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to create user",
			Error:   map[string]string{"error": "Failed to create user, please try again later"},
		})
		return
	}
	ctx.JSON(http.StatusCreated, models.APIResponse{
		Success: true,
		Data:    user,
		Message: "User created successfully",
		Error:   nil,
	})
}
