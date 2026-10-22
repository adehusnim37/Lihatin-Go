package validator

import (
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
)

// New returns a validator with the application's custom tags registered.
// Use it for structs with validate tags; reuse the instance across requests.
func New() (*validator.Validate, error) {
	v := validator.New()
	if err := Register(v); err != nil {
		return nil, err
	}
	return v, nil
}

// Register adds the application's custom tags to an existing validator.
// Call it once during startup, before using v from concurrent requests.
// It preserves the instance's tag name (for example Gin's "binding").
func Register(v *validator.Validate) error {
	if v == nil {
		return errors.New("validator: cannot register rules on a nil validator")
	}
	registrations := []struct {
		tag string
		fn  validator.Func
	}{
		{tag: "pwdcomplex", fn: validatePasswordComplexity},
		{tag: "username", fn: validateUsername},
		{tag: "no_space", fn: validateNoSpace},
		{tag: "no_special", fn: validateNoSpecial},
		{tag: "saveurlshort", fn: validateSaveUrlShort},
		{tag: "six_digit", fn: validateSixDigit},
		{tag: "secret_code", fn: validateSecretCode},
		{tag: "not_same_digit", fn: validateNotSameDigit},
		{tag: "meaningful_text", fn: validateMeaningfulText},
	}

	for _, registration := range registrations {
		if err := v.RegisterValidation(registration.tag, registration.fn); err != nil {
			return fmt.Errorf("failed to register validation %q: %w", registration.tag, err)
		}
	}

	return nil
}

// SetupCustomValidators registers the application's custom rules.
// Deprecated: use Register for an existing instance or New for a new one.
func SetupCustomValidators(v *validator.Validate) error { return Register(v) }
