package helpers

import (
	"strconv"
)

// StringToInt converts a string to an integer, returning 0 if the conversion fails.
func StringToInt(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return i
}

// PtrToString converts a pointer to string, returning empty string if nil.
func PtrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
