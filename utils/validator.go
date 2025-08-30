package utils

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

// SetupCustomValidators adds custom validation rules to the validator
func SetupCustomValidators(v *validator.Validate) {
	// Register custom validation for password complexity
	v.RegisterValidation("pwdcomplex", validatePasswordComplexity)
	v.RegisterValidation("username", ValidateUserRegistration)
}

// validatePasswordComplexity checks if a password has alphanumeric characters and symbols
func validatePasswordComplexity(fl validator.FieldLevel) bool {
	password := fl.Field().String()

	// Check for at least one letter
	hasLetter := regexp.MustCompile(`[a-zA-Z]`).MatchString(password)

	// Check for at least one digit
	hasDigit := regexp.MustCompile(`\d`).MatchString(password)

	// Check for at least one symbol
	hasSymbol := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`).MatchString(password)

	// Return true only if all conditions are met
	return hasLetter && hasDigit && hasSymbol
}

func ValidateUserRegistration(fl validator.FieldLevel) bool {
	username := fl.Field().String()

	minLength := 5
	maxLength := 20
	return username != "" && username != "null" && len(username) >= minLength && len(username) <= maxLength
}
