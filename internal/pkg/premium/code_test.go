package premium

import (
	"testing"
	"time"
)

const testGoodSecret = "test-premium-secret-material-for-lifetime-test"

func TestBuildAndVerifyCodeTimeBased(t *testing.T) {
	t.Setenv("PREMIUM_CODE_SECRET", testGoodSecret)

	validUntil := time.Now().UTC().Add(30 * 24 * time.Hour)
	code, err := BuildSecretCode(validUntil)
	if err != nil {
		t.Fatalf("BuildSecretCode failed: %v", err)
	}

	now := time.Now().UTC()
	expiry, digest, err := VerifyCode(code, now)
	if err != nil {
		t.Fatalf("VerifyCode failed: %v", err)
	}
	if expiry.IsZero() {
		t.Fatal("expected a non-zero expiry for time-based code")
	}
	if !expiry.Equal(validUntil.Truncate(time.Second)) {
		t.Fatalf("expiry mismatch: got %v want %v", expiry, validUntil.Truncate(time.Second))
	}
	if digest == "" {
		t.Fatal("expected a non-empty digest")
	}
}

func TestBuildAndVerifyCodeLifetime(t *testing.T) {
	t.Setenv("PREMIUM_CODE_SECRET", testGoodSecret)

	// Zero time -> lifetime code.
	code, err := BuildSecretCode(time.Time{})
	if err != nil {
		t.Fatalf("BuildSecretCode failed: %v", err)
	}

	// Verify far in the future: must still pass (never expires).
	expiry, digest, err := VerifyCode(code, time.Now().UTC().AddDate(100, 0, 0))
	if err != nil {
		t.Fatalf("lifetime code should never expire, got: %v", err)
	}
	if !expiry.IsZero() {
		t.Fatalf("expected zero expiry for lifetime code, got %v", expiry)
	}
	if digest == "" {
		t.Fatal("expected a non-empty digest")
	}
}

func TestLifetimeSentinel(t *testing.T) {
	if !IsLifetimeUnix(0xFFFFFFFF) {
		t.Fatal("expected 0xFFFFFFFF to be treated as lifetime sentinel")
	}
	if IsLifetimeUnix(0) {
		t.Fatal("expected 0 to NOT be treated as lifetime sentinel")
	}
}

func TestExpiredCodeRejected(t *testing.T) {
	t.Setenv("PREMIUM_CODE_SECRET", testGoodSecret)

	validUntil := time.Now().UTC().Add(-2 * time.Hour) // already expired
	code, err := BuildSecretCode(validUntil)
	if err != nil {
		t.Fatalf("BuildSecretCode failed: %v", err)
	}

	if _, _, err := VerifyCode(code, time.Now().UTC()); err != ErrCodeExpired {
		t.Fatalf("expected ErrCodeExpired, got %v", err)
	}
}

func TestTamperedCodeRejected(t *testing.T) {
	t.Setenv("PREMIUM_CODE_SECRET", testGoodSecret)

	validUntil := time.Now().UTC().Add(30 * 24 * time.Hour)
	code, err := BuildSecretCode(validUntil)
	if err != nil {
		t.Fatalf("BuildSecretCode failed: %v", err)
	}

	// Tamper by decoding, flipping a nonce payload byte, and re-encoding.
	// This changes the payload while leaving the received tag untouched, so the
	// signature check must reject it. (Version byte kept intact.)
	raw, err := base32Encoding.DecodeString(normalizeCode(code))
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	raw[5] ^= 0x01 // flip a nonce byte -> payload changed, version unchanged
	tampered := base32Encoding.EncodeToString(raw)

	if _, _, err := VerifyCode(tampered, time.Now().UTC()); err != ErrCodeSignature {
		t.Fatalf("expected ErrCodeSignature for tampered payload, got %v", err)
	}
}