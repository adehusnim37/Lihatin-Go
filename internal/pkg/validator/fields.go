package validator

import (
	"reflect"
	"strings"
)

// GetFieldLabel returns a field's label tag, JSON name, or Go name, in that order.
// input may be a struct or a pointer to a struct, including a nil pointer.
func GetFieldLabel(fieldName string, input any) string {
	typ := reflect.TypeOf(input)
	for typ != nil && typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ == nil || typ.Kind() != reflect.Struct {
		return fieldName
	}
	field, ok := typ.FieldByName(fieldName)
	if !ok {
		// JSON type errors report JSON names, while validator reports Go names.
		for i := 0; i < typ.NumField(); i++ {
			candidate := typ.Field(i)
			name, _, _ := strings.Cut(candidate.Tag.Get("json"), ",")
			if name == fieldName && name != "-" {
				field, ok = candidate, true
				break
			}
		}
	}
	if !ok {
		return fieldName
	}
	if label := field.Tag.Get("label"); label != "" {
		return label
	}
	name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
	if name != "" && name != "-" {
		return name
	}
	return fieldName
}
