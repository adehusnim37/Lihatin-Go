package utils

import (
	"encoding/json"
	"reflect"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
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

// Indonesian error messages mapping
var indonesianMessages = map[string]string{
	"required":     "%s wajib diisi",
	"min":          "%s minimal %s karakter/item",
	"max":          "%s maksimal %s karakter/item",
	"len":          "%s harus %s karakter/item",
	"email":        "%s harus berupa email yang valid",
	"url":          "%s harus berupa URL yang valid",
	"alphanum":     "%s hanya boleh berisi huruf dan angka",
	"alpha":        "%s hanya boleh berisi huruf",
	"lowercase":    "%s harus berupa huruf kecil",
	"uppercase":    "%s harus berupa huruf besar",
	"numeric":      "%s harus berupa angka",
	"oneof":        "%s harus salah satu dari: %s",
	"matches":      "%s format tidak valid",
	"unique":       "%s tidak boleh ada yang duplikat",
	"no_space":     "%s tidak boleh mengandung spasi",
	"no_special":   "%s tidak boleh mengandung karakter khusus",
	"saveurlshort": "%s hanya boleh berisi huruf, angka, underscore, dan hyphen",
	"eqfield":      "%s harus sama dengan %s",
	"nefield":      "%s tidak boleh sama dengan %s",
	"pwdcomplex":   "%s harus mengandung minimal 8 karakter, huruf besar, huruf kecil, angka, dan simbol",
	"username":     "%s hanya boleh berisi huruf, angka, underscore, dan hyphen",
	"gte":          "%s minimal %s",
	"lte":          "%s maksimal %s",
	"gt":           "%s harus lebih dari %s",
	"lt":           "%s harus kurang dari %s",
	"dive":         "item dalam %s",
	"six_digit":    "%s harus tepat 6 digit angka",
	"datetime":     "%s harus berupa tanggal dan waktu yang valid dengan format %s",
	"required_if":  "%s wajib diisi ketika %s adalah %s",
	"excluded_if":  "%s tidak boleh diisi ketika %s adalah %s",
	"excluded":     "%s tidak boleh diisi",
	"omitempty":    "%s tidak valid",
	"bool":         "%s harus berupa boolean",
	"array":        "%s harus berupa array",
	"slice":        "%s harus berupa array",
	"map":          "%s harus berupa peta",
	"set":          "%s harus berupa set",
}

// Type mapping for Indonesian error messages
var typeMapping = map[string]string{
	"string":   "teks",
	"int":      "angka",
	"int64":    "angka",
	"int32":    "angka",
	"float64":  "angka desimal",
	"float32":  "angka desimal",
	"bool":     "boolean",
	"[]string": "array teks",
	"[]int":    "array angka",
	"array":    "array",
	"slice":    "array",
	"link":     "%s harus berupa payload link tunggal yang valid",
	"links":    "%s harus berupa payload link jamak yang valid",
}

// GetFieldLabel extracts the label from struct tag or defaults to field name
func GetFieldLabel(fieldName string, structPtr interface{}) string {
	if structPtr == nil {
		return fieldName
	}

	val := reflect.ValueOf(structPtr)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return fieldName
	}

	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Name == fieldName {
			if label := field.Tag.Get("label"); label != "" {
				return label
			}
			if jsonTag := field.Tag.Get("json"); jsonTag != "" {
				parts := strings.Split(jsonTag, ",")
				if parts[0] != "" && parts[0] != "-" {
					return parts[0]
				}
			}
			break
		}
	}

	return fieldName
}

// GetIndonesianType converts Go type to Indonesian type name
func GetIndonesianType(goType string) string {
	if msg, exists := typeMapping[goType]; exists {
		return msg
	}
	return "format yang benar"
}

// HandleJSONBindingError handles JSON syntax and type mismatch errors
func HandleJSONBindingError(err error, structPtr interface{}) ValidationResponse {
	details := []DetailError{}
	errorMap := make(map[string]string)

	if syntaxErr, ok := err.(*json.SyntaxError); ok {
		details = append(details, DetailError{
			Field:   "json",
			Message: "Format JSON tidak valid pada posisi " + strings.Split(syntaxErr.Error(), " ")[len(strings.Split(syntaxErr.Error(), " "))-1],
		})
		errorMap["json"] = "Format JSON tidak valid"
	} else if typeErr, ok := err.(*json.UnmarshalTypeError); ok {
		fieldLabel := GetFieldLabel(typeErr.Field, structPtr)
		message := fieldLabel + " harus bertipe " + GetIndonesianType(typeErr.Type.String())

		details = append(details, DetailError{
			Field:   typeErr.Field,
			Message: message,
		})
		errorMap[typeErr.Field] = message
	} else {
		// Handle other JSON errors
		details = append(details, DetailError{
			Field:   "request",
			Message: "Format data request tidak valid",
		})
		errorMap["request"] = "Format data request tidak valid"
	}

	return ValidationResponse{
		Status:   400,
		Details:  details,
		ErrorMap: errorMap,
		Message:  "Validasi gagal",
		Success:  false,
	}
}

// HandleValidationError handles validator.v10 validation errors
func HandleValidationError(err error, structPtr interface{}) ValidationResponse {
	details := []DetailError{}
	errorMap := make(map[string]string)

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, validationErr := range validationErrors {
			fieldLabel := GetFieldLabel(validationErr.Field(), structPtr)
			message := formatIndonesianMessage(validationErr, fieldLabel)

			details = append(details, DetailError{
				Field:   validationErr.Field(),
				Message: message,
			})
			errorMap[validationErr.Field()] = message
		}
	}

	return ValidationResponse{
		Status:   400,
		Details:  details,
		ErrorMap: errorMap,
		Message:  "Validasi gagal",
		Success:  false,
	}
}

// formatIndonesianMessage formats validation error message in Indonesian
func formatIndonesianMessage(err validator.FieldError, fieldLabel string) string {
	tag := err.Tag()
	param := err.Param()

	if template, exists := indonesianMessages[tag]; exists {
		if param != "" {
			// Handle special cases
			switch tag {
			case "oneof":
				return strings.Replace(template, "%s", strings.ReplaceAll(param, " ", ", "), 1)
			default:
				// For tags with parameters like min, max, len
				if strings.Contains(template, "%s") {
					template = strings.Replace(template, "%s", fieldLabel, 1)
					template = strings.Replace(template, "%s", param, 1)
				}
				return template
			}
		} else {
			// For tags without parameters
			return strings.Replace(template, "%s", fieldLabel, 1)
		}
	}

	// Fallback for unknown validation tags
	return fieldLabel + " tidak valid"
}

// SendValidationError sends formatted validation error response
func SendValidationError(c *gin.Context, err error, structPtr interface{}) {
	var result ValidationResponse

	// Check if it's a JSON binding error or validation error
	if _, ok := err.(*json.SyntaxError); ok {
		result = HandleJSONBindingError(err, structPtr)
	} else if _, ok := err.(*json.UnmarshalTypeError); ok {
		result = HandleJSONBindingError(err, structPtr)
	} else if _, ok := err.(validator.ValidationErrors); ok {
		result = HandleValidationError(err, structPtr)
	} else {
		// Handle custom error messages (from business logic)
		errorMessage := err.Error()
		result = ValidationResponse{
			Status:  400,
			Details: []DetailError{{Field: "request", Message: errorMessage}},
			ErrorMap: map[string]string{
				"request": errorMessage,
			},
			Message: "Validasi gagal",
			Success: false,
		}
	}

	c.JSON(result.Status, gin.H{
		"success": result.Success,
		"message": result.Message,
		"errors":  result.ErrorMap,
		"details": result.Details,
	})
}

// Custom validation functions

// validatePasswordComplexity validates password complexity
func validatePasswordComplexity(fl validator.FieldLevel) bool {
	password := fl.Field().String()

	if len(password) < 8 {
		return false
	}

	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`\d`).MatchString(password)
	hasSymbol := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?~` + "`" + `]`).MatchString(password)

	return hasUpper && hasLower && hasNumber && hasSymbol
}

// validateUsername validates username format
func validateUsername(fl validator.FieldLevel) bool {
	username := fl.Field().String()
	return regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(username)
}

// validateUnique validates that slice has no duplicate values
func validateUnique(fl validator.FieldLevel) bool {
	slice := fl.Field()

	if slice.Kind() != reflect.Slice && slice.Kind() != reflect.Array {
		return true
	}

	seen := make(map[string]bool)
	for i := 0; i < slice.Len(); i++ {
		value := slice.Index(i).String()
		if seen[value] {
			return false
		}
		seen[value] = true
	}

	return true
}

// validateNoSpace validates that field contains no spaces
func validateNoSpace(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	return !strings.Contains(value, " ")
}

func validateNoSpecial(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	return regexp.MustCompile(`^[a-zA-Z0-9\s]+$`).MatchString(value)
}

func validateSaveUrlShort(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	return regexp.MustCompile(`^[a-zA-Z0-9-_]+$`).MatchString(value)
}

// validateSixDigit validates that integer has exactly 6 digits
func validateSixDigit(fl validator.FieldLevel) bool {
	value := fl.Field().Int()
	// Check if value is exactly 6 digits (100000 <= value <= 999999)
	return value >= 100000 && value <= 999999
}

// SetupCustomValidators registers custom validation rules
func SetupCustomValidators(v *validator.Validate) {
	v.RegisterValidation("pwdcomplex", validatePasswordComplexity)
	v.RegisterValidation("username", validateUsername)
	v.RegisterValidation("unique", validateUnique)
	v.RegisterValidation("no_space", validateNoSpace)
	v.RegisterValidation("no_special", validateNoSpecial)
	v.RegisterValidation("saveurlshort", validateSaveUrlShort)
	v.RegisterValidation("six_digit", validateSixDigit)
}
