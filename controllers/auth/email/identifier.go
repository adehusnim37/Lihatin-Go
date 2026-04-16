package email

import (
	"encoding/base64"
	"errors"
	"strings"
)

func decodeIdentifier(encoded string) (string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		return "", err
	}

	identifier := strings.TrimSpace(string(raw))
	if len(identifier) < 3 || len(identifier) > 100 {
		return "", errors.New("decoded identifier out of allowed bounds")
	}

	return identifier, nil
}
