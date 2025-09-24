package clientip

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

func GetDeviceAndIPInfo(ctx *gin.Context) (deviceID *string, lastIP *string) {
    // Try to get device ID from header
    device := ctx.GetHeader("X-Device-ID")
    
    // If no device ID, create fingerprint
    if device == "" {
        userAgent := ctx.GetHeader("User-Agent")
        acceptLang := ctx.GetHeader("Accept-Language")
        acceptEnc := ctx.GetHeader("Accept-Encoding")
        
        // Simple fingerprint (you can make this more sophisticated)
        fingerprint := fmt.Sprintf("%s_%s_%s", 
            hashString(userAgent), 
			hashString(acceptLang), 
			hashString(acceptEnc),
        )
        device = "fp_" + fingerprint
    }
    
    // Get real IP (considering proxies)
    ip := getRealClientIP(ctx)
    
    // Return pointers (nil if empty)
    if device != "" {
        deviceID = &device
    }
    if ip != "" {
        lastIP = &ip
    }
    
    return
}

// Get real client IP considering proxies
func getRealClientIP(ctx *gin.Context) string {
    // Check X-Forwarded-For header first
    if xForwardedFor := ctx.GetHeader("X-Forwarded-For"); xForwardedFor != "" {
        // Get first IP from comma-separated list
        ips := strings.Split(xForwardedFor, ",")
        if len(ips) > 0 {
            return strings.TrimSpace(ips[0])
        }
    }
    
    // Check X-Real-IP header
    if xRealIP := ctx.GetHeader("X-Real-IP"); xRealIP != "" {
        return xRealIP
    }
    
    // Fallback to RemoteAddr
    return ctx.ClientIP()
}

func hashString(s string) string {
	// Simple hash function (you can replace with a proper hash like SHA256)
	var hash int
	for _, char := range s {
		hash += int(char)
	}
	return fmt.Sprintf("%x", hash)
}