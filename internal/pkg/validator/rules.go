package validator

import (
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

var (
	upperLetterRegex    = regexp.MustCompile(`[A-Z]`)
	lowerLetterRegex    = regexp.MustCompile(`[a-z]`)
	decimalDigitRegex   = regexp.MustCompile(`\d`)
	identifierRegex     = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	noSpecialRegex      = regexp.MustCompile(`^[a-zA-Z0-9\s]+$`)
	secretCodeRegex     = regexp.MustCompile(`^[A-Za-z0-9\-\s]{20,80}$`)
	passwordSymbolRegex = regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?~` + "`" + `]`)
)

// validatePasswordComplexity validates password complexity
func validatePasswordComplexity(fl validator.FieldLevel) bool {
	password := fl.Field().String()

	if len(password) < 8 {
		return false
	}

	hasUpper := upperLetterRegex.MatchString(password)
	hasLower := lowerLetterRegex.MatchString(password)
	hasNumber := decimalDigitRegex.MatchString(password)
	hasSymbol := passwordSymbolRegex.MatchString(password)

	return hasUpper && hasLower && hasNumber && hasSymbol
}

// validateUsername validates username format
func validateUsername(fl validator.FieldLevel) bool {
	username := fl.Field().String()
	return identifierRegex.MatchString(username)
}

// validateNoSpace validates that field contains no spaces
func validateNoSpace(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	return !strings.Contains(value, " ")
}

func validateNoSpecial(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	return noSpecialRegex.MatchString(value)
}

func validateSaveUrlShort(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	return identifierRegex.MatchString(value)
}

// validateSixDigit validates that integer has exactly 6 digits
func validateSixDigit(fl validator.FieldLevel) bool {
	value := fl.Field().Int()
	// Check if value is exactly 6 digits (100000 <= value <= 999999)
	return value >= 100000 && value <= 999999
}

// validateNotSameDigit validates that a passcode is not composed of one repeated digit.
func validateNotSameDigit(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true
	}

	first := value[0]
	for i := 1; i < len(value); i++ {
		if value[i] != first {
			return true
		}
	}

	return false
}

// validateSecretCode validates premium secret code format (base32-like, optional separator)
func validateSecretCode(fl validator.FieldLevel) bool {
	secretCode := fl.Field().String()
	return secretCodeRegex.MatchString(secretCode)
}
