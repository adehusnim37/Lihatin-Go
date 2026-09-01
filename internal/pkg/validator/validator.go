package validator

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/pemistahl/lingua-go"
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

// Indonesian error messages mapping.
// #nosec G101 -- These are validation messages, not credentials.
var indonesianMessages = map[string]string{
	"required":        "%s wajib diisi",
	"min":             "%s minimal %s",
	"max":             "%s maksimal %s",
	"len":             "%s harus %s",
	"email":           "%s harus berupa email yang valid",
	"url":             "%s harus berupa URL yang valid",
	"alphanum":        "%s hanya boleh berisi huruf dan angka",
	"alpha":           "%s hanya boleh berisi huruf",
	"lowercase":       "%s harus berupa huruf kecil",
	"uppercase":       "%s harus berupa huruf besar",
	"numeric":         "%s harus berupa angka",
	"oneof":           "%s harus salah satu dari: %s",
	"matches":         "%s format tidak valid",
	"unique":          "%s tidak boleh ada yang duplikat",
	"no_space":        "%s tidak boleh mengandung spasi",
	"no_special":      "%s tidak boleh mengandung karakter khusus",
	"saveurlshort":    "%s hanya boleh berisi huruf, angka, underscore, dan hyphen",
	"eqfield":         "%s harus sama dengan %s",
	"nefield":         "%s tidak boleh sama dengan %s",
	"pwdcomplex":      "%s harus mengandung minimal 8 karakter, huruf besar, huruf kecil, angka, dan simbol",
	"username":        "%s hanya boleh berisi huruf, angka, underscore, dan hyphen",
	"gte":             "%s minimal %s",
	"lte":             "%s maksimal %s",
	"gt":              "%s harus lebih dari %s",
	"lt":              "%s harus kurang dari %s",
	"dive":            "item dalam %s",
	"six_digit":       "%s harus tepat 6 digit angka",
	"datetime":        "%s harus berupa tanggal dan waktu yang valid dengan format %s",
	"required_if":     "%s wajib diisi ketika %s adalah %s",
	"excluded_if":     "%s tidak boleh diisi ketika %s adalah %s",
	"excluded":        "%s tidak boleh diisi",
	"omitempty":       "%s tidak valid",
	"bool":            "%s harus berupa boolean",
	"array":           "%s harus berupa array",
	"slice":           "%s harus berupa array",
	"map":             "%s harus berupa peta",
	"set":             "%s harus berupa set",
	"secret_code":     "%s harus berupa secret code yang valid",
	"not_same_digit":  "%s tidak boleh terdiri dari angka yang sama semua",
	"meaningful_text": "%s tidak boleh berupa teks acak/tidak bermakna",
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

					// Add dynamic suffix based on tag and type
					switch tag {
					case "min", "max", "len":
						switch err.Kind() {
						case reflect.String:
							template += " karakter"
						case reflect.Slice, reflect.Map, reflect.Array:
							template += " item"
						}
					}
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
// Uses common.APIResponse format for consistency with other API responses
func SendValidationError(c *gin.Context, err error, structPtr any) {
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

	// Use common.APIResponse format for consistency
	c.JSON(result.Status, gin.H{
		"success": result.Success,
		"data":    nil,
		"message": result.Message,
		"error":   result.ErrorMap, // Use 'error' instead of 'errors' to match common.APIResponse
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
	return regexp.MustCompile(`^[A-Za-z0-9\-\s]{20,80}$`).MatchString(secretCode)
}

var (
	meaningfulTextTokenRegex = regexp.MustCompile(`[^\p{L}]+`)
	technicalTokenRegex      = regexp.MustCompile(`(?i)(https?://|www\.|[/\\]|(?:err|error|exception|status|code|api|http|sql|json|jwt|uuid|otp|timeout|refused|failed)\b)`)

	// Lingua is used as a language signal, not as the sole accept/reject rule.
	// Support tickets commonly mix Indonesian and English, so the detector is
	// intentionally restricted to those two languages.
	meaningfulTextDetector = lingua.NewLanguageDetectorBuilder().
				FromLanguages(lingua.Indonesian, lingua.English).
				Build()
)

// validateMeaningfulText rejects obvious gibberish/random keyboard-mashing
// while allowing normal Indonesian/English support text and technical terms.
func validateMeaningfulText(fl validator.FieldLevel) bool {
	return IsMeaningfulText(fl.Field().String())
}

// IsMeaningfulText exposes the same gibberish/keyboard-mashing detection used
// by the "meaningful_text" struct tag so it can be reused outside of
// struct-tag validation (e.g. multipart form fields like support message
// bodies).
func IsMeaningfulText(raw string) bool {
	value := strings.TrimSpace(raw)
	if value == "" {
		return true
	}

	metrics := analyzeMeaningfulText(value)
	if metrics.letterCount == 0 {
		return true
	}

	score := meaningfulTextGibberishScore(metrics, value)
	if score < 2 {
		return true
	}

	// A low-vowel score can happen with valid abbreviation-heavy support text.
	// Keep that one signal permissive when Lingua confidently recognizes the
	// text as English or Indonesian; multiple independent signals still reject.
	return score == 2 && metrics.lowVowelRatio && metrics.wordCount <= 3 && isConfidentEnglishOrIndonesian(value)
}

type meaningfulTextMetrics struct {
	words          []string
	wordCount      int
	shortWordCount int
	letterCount    int
	vowelCount     int
	uniqueLetters  int
	maxConsonants  int
	lowVowelRatio  bool
	repeatedChunks bool
}

func analyzeMeaningfulText(value string) meaningfulTextMetrics {
	metrics := meaningfulTextMetrics{
		words: meaningfulTextTokenRegex.Split(strings.ToLower(value), -1),
	}
	uniqueLetters := make(map[rune]struct{})
	for _, word := range metrics.words {
		if word == "" {
			continue
		}
		metrics.wordCount++
		if len([]rune(word)) <= 3 {
			metrics.shortWordCount++
		}
		metrics.repeatedChunks = metrics.repeatedChunks || hasRepeatedChunk(word)

		consonants := 0
		for _, r := range word {
			if !unicode.In(r, unicode.Latin) {
				continue
			}
			metrics.letterCount++
			uniqueLetters[r] = struct{}{}
			if strings.ContainsRune("aeiou", r) {
				metrics.vowelCount++
				consonants = 0
				continue
			}
			consonants++
			if consonants > metrics.maxConsonants {
				metrics.maxConsonants = consonants
			}
		}
	}
	metrics.uniqueLetters = len(uniqueLetters)
	metrics.lowVowelRatio = metrics.letterCount >= 12 &&
		float64(metrics.vowelCount)/float64(metrics.letterCount) < 0.20
	return metrics
}

func meaningfulTextGibberishScore(metrics meaningfulTextMetrics, value string) int {
	score := 0
	if metrics.letterCount >= 15 && metrics.wordCount == 1 && !technicalTokenRegex.MatchString(value) {
		score += 2
	}
	if metrics.maxConsonants >= 6 {
		score += 2
	}
	if metrics.lowVowelRatio && metrics.wordCount >= 3 {
		score += 2
	}
	if metrics.lowVowelRatio && metrics.wordCount >= 4 && metrics.shortWordCount >= metrics.wordCount-1 {
		score++
	}
	if metrics.repeatedChunks {
		score += 2
	}
	if metrics.letterCount >= 16 && float64(metrics.uniqueLetters)/float64(metrics.letterCount) < 0.25 {
		score++
	}
	return score
}

func hasRepeatedChunk(word string) bool {
	runes := []rune(word)
	if len(runes) < 8 {
		return false
	}

	for chunkLength := 1; chunkLength <= 4; chunkLength++ {
		for start := 0; start+chunkLength*3 <= len(runes); start++ {
			chunk := string(runes[start : start+chunkLength])
			repeats := 1
			for next := start + chunkLength; next+chunkLength <= len(runes); next += chunkLength {
				if string(runes[next:next+chunkLength]) != chunk {
					break
				}
				repeats++
			}
			if repeats >= 3 {
				return true
			}
		}
	}
	return false
}

func isConfidentEnglishOrIndonesian(value string) bool {
	language, detected := meaningfulTextDetector.DetectLanguageOf(value)
	if !detected || (language != lingua.English && language != lingua.Indonesian) {
		return false
	}

	return meaningfulTextDetector.ComputeLanguageConfidence(value, language) >= 0.65
}

// SetupCustomValidators registers custom validation rules
func SetupCustomValidators(v *validator.Validate) error {
	registrations := []struct {
		tag string
		fn  validator.Func
	}{
		{tag: "pwdcomplex", fn: validatePasswordComplexity},
		{tag: "username", fn: validateUsername},
		{tag: "unique", fn: validateUnique},
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
