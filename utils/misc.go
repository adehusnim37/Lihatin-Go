package utils

import (
	"crypto/rand"
	"fmt"
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

func GenerateUUIDV7() string {
    // Generate 16 random bytes
    bytes := make([]byte, 16)
    if _, err := rand.Read(bytes); err != nil {
        return ""
    }

    // Set the version to 7 (time-based)
    bytes[6] = (bytes[6] & 0x0F) | (0x70) // Version 7
    bytes[8] = (bytes[8] & 0x3F) | 0x80   // Variant is 10
     
    // Convert to UUID string format
    uuid := fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
        bytes[0:4],
        bytes[4:6],
        bytes[6:8],
        bytes[8:10],
        bytes[10:16],
    )

    return uuid
}

func ValidateUUIDV7(uuid string) bool {
    if len(uuid) != 36 {
        return false
    }
    // Check version (should be '7' at position 14)
    if uuid[14] != '7' {
        return false
    }
    // Check variant (should be one of '8', '9', 'a', or 'b' at position 19)
    if uuid[19] != '8' && uuid[19] != '9' && uuid[19] != 'a' && uuid[19] != 'b' {
        return false
    }
    return true
}

func PtrToString(s *string) string {
    if s == nil {
        return ""
    }
    return *s
}
