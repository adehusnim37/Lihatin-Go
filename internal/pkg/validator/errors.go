package validator

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
)

// ValidationResponse contains the parsed validation error
type ValidationResponse struct {
	Status   int               `json:"status"`
	Details  []DetailError     `json:"details"`
	ErrorMap map[string]string `json:"error_map"`
	Message  string            `json:"message"`
	Success  bool              `json:"success"`
}

// DetailError represents a single field validation error
type DetailError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// FormatError converts binding, validation, or application errors to a response.
// Wrapped JSON and validation errors are recognized using errors.As.
// Callers should pass a non-nil error from binding or validation.
func FormatError(err error, input any) ValidationResponse {
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	var validationErrors validator.ValidationErrors
	switch {
	case errors.As(err, &syntaxErr), errors.As(err, &typeErr):
		return HandleJSONBindingError(err, input)
	case errors.As(err, &validationErrors):
		return HandleValidationError(err, input)
	default:
		message := "Format data request tidak valid"
		if err != nil {
			message = err.Error()
		}
		return newValidationResponse([]DetailError{{Field: "request", Message: message}})
	}
}

// HandleJSONBindingError formats JSON syntax and type mismatch errors.
func HandleJSONBindingError(err error, input any) ValidationResponse {
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	switch {
	case errors.As(err, &syntaxErr):
		response := newValidationResponse([]DetailError{{
			Field: "json", Message: fmt.Sprintf("Format JSON tidak valid pada posisi %d", syntaxErr.Offset),
		}})
		response.ErrorMap["json"] = "Format JSON tidak valid"
		return response
	case errors.As(err, &typeErr):
		label := GetFieldLabel(typeErr.Field, input)
		return newValidationResponse([]DetailError{{
			Field: typeErr.Field, Message: label + " harus bertipe " + GetIndonesianType(typeErr.Type.String()),
		}})
	default:
		return newValidationResponse([]DetailError{{Field: "request", Message: "Format data request tidak valid"}})
	}
}

// HandleValidationError formats field errors using label and json tags.
func HandleValidationError(err error, input any) ValidationResponse {
	details := make([]DetailError, 0)
	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) {
		for _, fieldErr := range validationErrors {
			details = append(details, DetailError{
				Field:   fieldErr.Field(),
				Message: formatIndonesianMessage(fieldErr, GetFieldLabel(fieldErr.Field(), input)),
			})
		}
	}
	return newValidationResponse(details)
}

func newValidationResponse(details []DetailError) ValidationResponse {
	errorMap := make(map[string]string, len(details))
	for _, detail := range details {
		errorMap[detail.Field] = detail.Message
	}
	return ValidationResponse{
		Status: http.StatusBadRequest, Details: details, ErrorMap: errorMap,
		Message: "Validasi gagal", Success: false,
	}
}
