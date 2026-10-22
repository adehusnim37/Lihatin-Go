package validator

import (
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

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

// GetIndonesianType returns a display name for a Go type.
func GetIndonesianType(goType string) string {
	if name, ok := typeMapping[goType]; ok {
		return name
	}
	return "format yang benar"
}

func formatIndonesianMessage(err validator.FieldError, label string) string {
	template, ok := indonesianMessages[err.Tag()]
	if !ok {
		return label + " tidak valid"
	}
	message := strings.Replace(template, "%s", label, 1)
	param := err.Param()
	switch err.Tag() {
	case "oneof":
		param = strings.ReplaceAll(param, " ", ", ")
	case "required_if", "excluded_if":
		if field, value, ok := strings.Cut(param, " "); ok {
			message = strings.Replace(message, "%s", field, 1)
			param = value
		}
	}
	message = strings.Replace(message, "%s", param, 1)
	switch err.Tag() {
	case "min", "max", "len":
		switch err.Kind() {
		case reflect.String:
			message += " karakter"
		case reflect.Slice, reflect.Map, reflect.Array:
			message += " item"
		}
	}
	return message
}
