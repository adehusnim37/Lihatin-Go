package auth

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/adehusnim37/lihatin-go/models/user"
	"github.com/fxamacker/cbor/v2"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"net/http/httptest"
	"testing"
)

func TestPasskeyConfigPinsOriginRPAndUserVerification(t *testing.T) {
	t.Setenv("FRONTEND_URL", "https://lihat.example")
	t.Setenv("WEBAUTHN_RP_ID", "lihat.example")
	t.Setenv("WEBAUTHN_ORIGINS", "https://lihat.example,https://app.lihat.example")
	engine, err := passkeyEngine()
	if err != nil {
		t.Fatal(err)
	}
	if engine.Config.RPID != "lihat.example" || len(engine.Config.RPOrigins) != 2 || engine.Config.AuthenticatorSelection.UserVerification != protocol.VerificationRequired || engine.Config.AuthenticatorSelection.ResidentKey != protocol.ResidentKeyRequirementRequired {
		t.Fatal("passkey policy changed")
	}
	t.Setenv("WEBAUTHN_ORIGINS", "https://lihat.example,not-an-origin")
	if _, err = passkeyEngine(); err == nil {
		t.Fatal("invalid origin accepted")
	}
}

// Exercise our RP configuration and serialized credential storage with a software
// authenticator, without creating a credential in the developer's real account.
func TestPasskeyRegistrationAndAssertionRoundTrip(t *testing.T) {
	t.Setenv("FRONTEND_URL", "https://lihat.example")
	t.Setenv("WEBAUTHN_RP_ID", "lihat.example")
	t.Setenv("WEBAUTHN_ORIGINS", "https://lihat.example")
	engine, err := passkeyEngine()
	if err != nil {
		t.Fatal(err)
	}
	owner := &passkeyUser{account: &user.User{ID: "test-user", Email: "test@example.com", Username: "test-user"}}
	_, registration, err := engine.BeginRegistration(owner)
	if err != nil {
		t.Fatal(err)
	}
	private, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	public, err := cbor.Marshal(map[int]any{1: 2, 3: -7, -1: 1, -2: private.X.FillBytes(make([]byte, 32)), -3: private.Y.FillBytes(make([]byte, 32))})
	if err != nil {
		t.Fatal(err)
	}
	credentialID := make([]byte, 32)
	if _, err = rand.Read(credentialID); err != nil {
		t.Fatal(err)
	}
	rp := sha256.Sum256([]byte("lihat.example"))
	authData := append([]byte{}, rp[:]...)
	authData = append(authData, 0x45, 0, 0, 0, 0)
	authData = append(authData, make([]byte, 16)...)
	authData = append(authData, 0, byte(len(credentialID)))
	authData = append(authData, credentialID...)
	authData = append(authData, public...)
	attestation, err := cbor.Marshal(map[string]any{"fmt": "none", "attStmt": map[string]any{}, "authData": authData})
	if err != nil {
		t.Fatal(err)
	}
	enc := base64.RawURLEncoding.EncodeToString
	client, _ := json.Marshal(map[string]any{"type": "webauthn.create", "challenge": registration.Challenge, "origin": "https://lihat.example"})
	creation, _ := json.Marshal(map[string]any{"id": enc(credentialID), "rawId": enc(credentialID), "type": "public-key", "clientExtensionResults": map[string]any{}, "response": map[string]any{"clientDataJSON": enc(client), "attestationObject": enc(attestation), "transports": []string{"internal"}}})
	credential, err := engine.FinishRegistration(owner, *registration, httptest.NewRequest("POST", "/", bytes.NewReader(creation)))
	if err != nil {
		t.Fatalf("registration: %v", err)
	}
	stored, err := json.Marshal(credential)
	if err != nil {
		t.Fatal(err)
	}
	var restored webauthn.Credential
	if err = json.Unmarshal(stored, &restored); err != nil {
		t.Fatal(err)
	}
	owner.credentials = []webauthn.Credential{restored}
	_, assertion, err := engine.BeginDiscoverableLogin(webauthn.WithUserVerification(protocol.VerificationRequired))
	if err != nil {
		t.Fatal(err)
	}
	makeAssertion := func(origin string) []byte {
		client, _ := json.Marshal(map[string]any{"type": "webauthn.get", "challenge": assertion.Challenge, "origin": origin})
		authData := append([]byte{}, rp[:]...)
		authData = append(authData, 0x05, 0, 0, 0, 1)
		clientHash := sha256.Sum256(client)
		signed := append(append([]byte{}, authData...), clientHash[:]...)
		digest := sha256.Sum256(signed)
		signature, e := ecdsa.SignASN1(rand.Reader, private, digest[:])
		if e != nil {
			t.Fatal(e)
		}
		body, _ := json.Marshal(map[string]any{"id": enc(credentialID), "rawId": enc(credentialID), "type": "public-key", "clientExtensionResults": map[string]any{}, "response": map[string]any{"clientDataJSON": enc(client), "authenticatorData": enc(authData), "signature": enc(signature), "userHandle": enc(owner.WebAuthnID())}})
		return body
	}
	lookup := func(raw, handle []byte) (webauthn.User, error) {
		if !bytes.Equal(raw, credentialID) || !bytes.Equal(handle, owner.WebAuthnID()) {
			return nil, errors.New("wrong owner")
		}
		return owner, nil
	}
	if _, err = engine.FinishDiscoverableLogin(lookup, *assertion, httptest.NewRequest("POST", "/", bytes.NewReader(makeAssertion("https://attacker.example")))); err == nil {
		t.Fatal("untrusted origin accepted")
	}
	signed, err := engine.FinishDiscoverableLogin(lookup, *assertion, httptest.NewRequest("POST", "/", bytes.NewReader(makeAssertion("https://lihat.example"))))
	if err != nil {
		t.Fatalf("assertion: %v", err)
	}
	if signed.Authenticator.SignCount != 1 || signed.Authenticator.CloneWarning {
		t.Fatal("credential counter did not persist correctly")
	}
}
