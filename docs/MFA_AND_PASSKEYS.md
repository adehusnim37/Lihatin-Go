# Sign-in security, recovery codes, and passkeys

The account page is `/profile/security`, linked from the Profile Security tab. It keeps authenticator management, passkeys, and recovery code generation together. Sessions remain at `/profile/sessions`.

## Authentication policy

Passkeys replace the password step. Registration requires a discoverable credential and local user verification (device screen lock, biometrics, or security key PIN). WebAuthn verifies the RP ID, allowed origin, challenge, user handle, signature, and credential counter through go-webauthn. Private keys stay with the authenticator.

Accounts with enabled TOTP still complete TOTP (or a single-use recovery code) after a passkey assertion. The existing privileged-account TOTP enrollment policy remains enforced. Otherwise a verified passkey completes the same session/cookie and login-event path as other sign-in methods. Locked, disabled, unverified, or temporarily blocked accounts cannot complete passkey login.

Enrollment, removal, and recovery-code replacement require a short-lived, one-use verification token bound to the account and current session. Obtain it with an authenticator code when TOTP is enabled, the password when it is off, or an existing passkey assertion. Registration challenges are also session-bound; login challenges are discoverable and resolve the owner using the credential ID and opaque user handle. Challenges expire after five minutes and are consumed once even when verification fails.

A maximum of 10 passkeys is allowed per account. Deleting the last passkey requires an available password or verified Google sign-in method. Sign-in events include `passkey` and `recovery_code`. The existing Sessions attempt list also records passkey proof results (explicitly labelled when TOTP is still pending) and completed recovery-code sign-ins. Passkey additions and removals send security email using the existing mail service. New public bootstrap routes retain CSRF origin checks and are rate limited.

## Recovery codes

Codes are stored as bcrypt hashes and shown only when initially generated or regenerated. `GET /auth/totp/recovery-codes` now returns only `codes_count`, never hashes. `GET /auth/totp/status` returns enabled/verified state, friendly name, verification and last-use timestamps, and remaining count.

`POST /auth/totp/regenerate-recovery-codes` accepts `verification_token` (preferred) or a current `totp_code`; successful replacement immediately invalidates previous codes. The response is no-store. The UI offers copy/download and clears plaintext from component state when closed.

`POST /auth/verify-totp-login` accepts `pending_auth_token` and either `totp_code` or `recovery_code`. A recovery code accepts the saved `ABCD-1234` format or eight characters without the hyphen, case-insensitively. Verification and removal occur in one database transaction. Reuse fails. The pending login is atomically consumed before issuing a session.

If all factors are lost, the UI points to the existing `lost_2fa` support flow. A support request does not disable factors or grant account access; ownership review remains required.

## Passkey endpoints (relative to `/v1`)

- `GET /auth/passkeys`: own credential names, IDs, creation and last-use dates, plus configuration availability.
- `POST /auth/security/verify`: obtain a verification token using password or authenticator proof.
- `POST /auth/passkeys/register/begin`: `{ name, verification_token }` returns `{ challenge_id, options }`.
- `POST /auth/passkeys/register/finish`: `{ challenge_id, credential }` verifies browser attestation and stores the public credential record.
- `PATCH /auth/passkeys/:id`: `{ name }`, scoped to the signed-in account.
- `DELETE /auth/passkeys/:id`: `{ verification_token }`, scoped to the signed-in account.
- `POST /auth/passkeys/reauth/begin`, `/finish`: verify an existing passkey to obtain an action token.
- `POST /auth/passkeys/login/begin`, `/finish`: public WebAuthn assertion flow; returns either a pending TOTP response or the existing successful login response.

Finish bodies contain the browser credential encoded as base64url JSON. Credential records and key material are never returned by listing endpoints.

## Configuration and rollout

Set `WEBAUTHN_RP_ID` to the frontend host without a scheme/port (for example `lihat.in`), and `WEBAUTHN_ORIGINS` to comma-separated exact frontend origins (for example `https://lihat.in`). Defaults are derived from `FRONTEND_URL`; localhost HTTP is available for development. Production WebAuthn requires HTTPS. Do not derive these values from the request Origin header or change RP ID after enrollment.

The existing AutoMigrate runner creates `user_passkeys` with an account foreign key and credential-hash uniqueness. Passkey public records include credential IDs, public keys, transports, attestation, counter and backup flags as serialized library data. Keep the database protected and encrypted at rest under the deployment's existing storage controls. Existing TOTP secret storage is unchanged by this feature.
