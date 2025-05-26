package user

import (
	"database/sql"
	"net/http"

	"github.com/adehusnim37/lihatin-go/models"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// Login authenticates a user with email/username and password
func (c *Controller) Login(ctx *gin.Context) {
	var loginReq models.LoginRequest

	// Bind and validate the request body
	if err := ctx.ShouldBindJSON(&loginReq); err != nil {
		ctx.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid input",
			Error:   map[string]string{"input": "Invalid input format or missing fields, please check your request body"},
		})
		return
	}

	// Validate the login request
	if err := c.Validate.Struct(loginReq); err != nil {
		errorMap := make(map[string]string)
		for _, err := range err.(validator.ValidationErrors) {
			field := err.Field()
			switch err.Tag() {
			case "required":
				errorMap[field] = field + " is required"
			case "min":
				errorMap[field] = field + " must be at least " + err.Param() + " characters"
			case "max":
				errorMap[field] = field + " must not exceed " + err.Param() + " characters"
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

	// Get user from database
	user, err := c.repo.GetUserForLogin(loginReq.EmailOrUsername)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusUnauthorized, models.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Invalid credentials",
				Error:   map[string]string{"auth": "Invalid email/username or password"},
			})
			return
		}

		ctx.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Login failed",
			Error:   map[string]string{"error": "An error occurred during login, please try again later"},
		})
		return
	}

	// Check password
	if err := utils.CheckPassword(user.Password, loginReq.Password); err != nil {
		ctx.JSON(http.StatusUnauthorized, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid credentials",
			Error:   map[string]string{"auth": "Invalid email/username or password"},
		})
		return
	}

	// Create user profile (without sensitive information)
	userProfile := models.UserProfile{
		ID:        user.ID,
		Username:  user.Username,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email,
		Avatar:    user.Avatar,
		IsPremium: user.IsPremium,
		CreatedAt: user.CreatedAt,
	}

	// Set user context for logging
	ctx.Set("username", user.Username)
	ctx.Set("user_id", user.ID)
	ctx.Set("role", map[bool]string{true: "premium", false: "regular"}[user.IsPremium])

	// Create login response
	loginResponse := models.LoginResponse{
		User:    userProfile,
		Message: "Login successful",
		// Token: generateJWTToken(user), // Uncomment if implementing JWT
	}

	ctx.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    loginResponse,
		Message: "Login successful",
		Error:   nil,
	})
}
