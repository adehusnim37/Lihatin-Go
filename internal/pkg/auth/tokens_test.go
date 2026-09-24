package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestGeneratedAPIKeySecretValidates(t *testing.T) {
	id, secret, stored, _, err := GenerateAPIKeyPair("")
	if err != nil {
		t.Fatal(err)
	}
	if !ValidateAPIKeyIDFormat(id) {
		t.Fatalf("generated ID has invalid format: %q", id)
	}
	if !ValidateAPISecretKey(secret, stored) {
		t.Fatal("newly generated API key secret does not validate")
	}
	if ValidateAPISecretKey(secret+"wrong", stored) {
		t.Fatal("wrong secret was accepted")
	}
	parts := strings.Split(stored, ":")
	if len(parts) != 2 || len(parts[0]) != 32 || len(parts[1]) != 64 {
		t.Fatal("generated hash is not salt:derived-key")
	}
}

func TestLegacyAndMalformedAPIKeyHashes(t *testing.T) {
	legacy := sha256.Sum256([]byte("legacy-secret"))
	if !ValidateAPISecretKey("legacy-secret", hex.EncodeToString(legacy[:])) {
		t.Fatal("previously issued SHA-256 key is no longer usable")
	}
	if ValidateAPISecretKey("wrong", hex.EncodeToString(legacy[:])) {
		t.Fatal("wrong legacy secret was accepted")
	}
	for _, stored := range []string{"", "not-hex", "short:salt", strings.Repeat("0", 32) + ":" + strings.Repeat("x", 64)} {
		if ValidateAPISecretKey("secret", stored) {
			t.Fatalf("malformed hash %q was accepted", stored)
		}
	}
}
