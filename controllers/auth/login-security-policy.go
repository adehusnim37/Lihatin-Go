package auth

import (
	"strings"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
)

func isPrivilegedRole(role string) bool {
	normalizedRole := strings.ToLower(strings.TrimSpace(role))
	return normalizedRole == "admin" || normalizedRole == "super_admin"
}

func isPrivilegedTOTPEnforced() bool {
	return config.GetEnvAsBool(config.EnvAuthEnforceTOTPForPrivileged, true)
}
