package utils

import (
	"encoding/json"
	"reflect"
	"regexp"
	"strings"

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

// GetJSONFieldName returns the JSON tag name for a struct field
func GetJSONFieldName(fieldName string, structPtr interface{}) string {
	structType := reflect.TypeOf(structPtr).Elem()
	field, found := structType.FieldByName(fieldName)
	if !found {
		return strings.ToLower(fieldName)
	}

	jsonTag := field.Tag.Get("json")
	if jsonTag == "" {
		return strings.ToLower(fieldName)
	}

	// Remove omitempty or other options from json tag
	parts := strings.Split(jsonTag, ",")
	return parts[0]
}

// GetIndonesianType converts Go type to Indonesian type name
func GetIndonesianType(goType string) string {
	switch goType {
	case "string":
		return "string"
	case "int", "int64", "int32":
		return "angka"
	case "float64", "float32":
		return "angka desimal"
	case "bool":
		return "boolean"
	default:
		return "format yang benar"
	}
}

// GetIndonesianValidationMessage converts validation error to Indonesian message
func GetIndonesianValidationMessage(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return "harus diisi"
	case "url":
		return "harus berupa URL yang valid"
	case "max":
		return "maksimal " + err.Param() + " karakter"
	case "min":
		return "minimal " + err.Param() + " karakter"
	case "len":
		return "harus tepat " + err.Param() + " karakter"
	case "alphanum":
		return "hanya boleh huruf dan angka"
	case "alpha":
		return "hanya boleh huruf"
	case "numeric":
		return "hanya boleh angka"
	case "email":
		return "harus berupa email yang valid"
	case "pwdcomplex":
		return "harus mengandung huruf, angka, dan simbol"
	case "username":
		return "username harus 5-20 karakter"
	default:
		return "tidak valid"
	}
}

// HandleJSONBindingError creates Indonesian error messages for JSON binding errors
func HandleJSONBindingError(err error, structPtr interface{}) map[string]string {
	bindingErrors := make(map[string]string)

	// Check if it's a JSON syntax error
	if _, ok := err.(*json.SyntaxError); ok {
		bindingErrors["json"] = "format JSON tidak valid"
	} else if jsonErr, ok := err.(*json.UnmarshalTypeError); ok {
		// Get the JSON field name from the struct field
		fieldName := GetJSONFieldName(jsonErr.Field, structPtr)
		bindingErrors[fieldName] = "harus bertipe " + GetIndonesianType(jsonErr.Type.String())
	} else {
		// Generic binding error
		bindingErrors["request"] = "format data tidak valid"
	}

	return bindingErrors
}

// HandleValidationError creates Indonesian error messages for validation errors
func HandleValidationError(err error, structPtr interface{}) map[string]string {
	validationErrors := make(map[string]string)

	if validationErrs, ok := err.(validator.ValidationErrors); ok {
		for _, validationErr := range validationErrs {
			fieldName := GetJSONFieldName(validationErr.Field(), structPtr)
			validationErrors[fieldName] = GetIndonesianValidationMessage(validationErr)
		}
	}

	return validationErrors
}
