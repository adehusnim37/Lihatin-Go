package utils

import (
    "strings"
)

type DeviceInfo struct {
    Device  string
    Browser string
    OS      string
}

// ParseUserAgent extracts device, browser, and OS information from user agent string
func ParseUserAgent(userAgent string) DeviceInfo {
    if userAgent == "" {
        return DeviceInfo{
            Device:  "Unknown",
            Browser: "Unknown", 
            OS:      "Unknown",
        }
    }

    userAgent = strings.ToLower(userAgent)
    
    // Detect OS
    var os string
    switch {
    case strings.Contains(userAgent, "windows"):
        os = "Windows"
    case strings.Contains(userAgent, "mac"):
        os = "macOS"
    case strings.Contains(userAgent, "linux"):
        os = "Linux"
    case strings.Contains(userAgent, "android"):
        os = "Android"
    case strings.Contains(userAgent, "iphone") || strings.Contains(userAgent, "ipad"):
        os = "iOS"
    default:
        os = "Unknown"
    }

    // Detect Browser
    var browser string
    switch {
    case strings.Contains(userAgent, "chrome") && !strings.Contains(userAgent, "edge"):
        browser = "Chrome"
    case strings.Contains(userAgent, "firefox"):
        browser = "Firefox"
    case strings.Contains(userAgent, "safari") && !strings.Contains(userAgent, "chrome"):
        browser = "Safari"
    case strings.Contains(userAgent, "edge"):
        browser = "Edge"
    case strings.Contains(userAgent, "opera"):
        browser = "Opera"
    default:
        browser = "Unknown"
    }

    // Detect Device Type
    var device string
    switch {
    case strings.Contains(userAgent, "mobile"):
        device = "Mobile"
    case strings.Contains(userAgent, "tablet") || strings.Contains(userAgent, "ipad"):
        device = "Tablet"
    default:
        device = "Desktop"
    }

    return DeviceInfo{
        Device:  device,
        Browser: browser,
        OS:      os,
    }
}