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