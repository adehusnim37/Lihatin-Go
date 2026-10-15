package auth

import "testing"

func TestRecoveryCodeBcryptVerificationAndNormalization(t *testing.T) {
	hashes := HashRecoveryCodes([]string{"ABCD-1234", "EFGH-5678"})
	if !ValidateRecoveryCode("abcd1234", hashes) {
		t.Fatal("correct code did not match bcrypt hash")
	}
	if ValidateRecoveryCode("ABCD-9999", hashes) {
		t.Fatal("wrong code accepted")
	}
	remaining := RemoveRecoveryCode(hashes, "ABCD1234")
	if len(remaining) != 1 || ValidateRecoveryCode("ABCD-1234", remaining) || !ValidateRecoveryCode("EFGH-5678", remaining) {
		t.Fatal("removal did not preserve only unused codes")
	}
}
