# Admin-assisted credential recovery

The public support form never approves or executes a credential reset. Normal
`PUT /v1/auth/admin/support/tickets/:id` updates cannot reset credentials either.

## Review and approval

1. The requester submits a `lost_2fa` ticket for their existing account.
2. An administrator reviews the request, sets the ticket to **In Progress**, and
   saves the update. This status never approves recovery by itself.
3. Under **User Management → Account recovery verification**, the administrator
   clicks **Send recovery OTP**. The code is sent to the linked account's stored,
   verified email. No OTP or challenge token is returned to the administrator.
4. The requester opens their existing protected ticket conversation using their
   ticket access code (or their signed-in account). They enter the email code in
   **Verify your recovery request** and confirm they requested account recovery.
   No standalone public verification page or secret URL parameter is used.
5. Successful verification records mailbox ownership for 24 hours. The admin
   panel refreshes while waiting for verification; the reset buttons then unlock.
6. The administrator selects **Reset Authenticator (TOTP)**, **Remove All
   Passkeys**, or both, records a meaningful ownership verification reason
   (20–1000 characters), explicitly approves, types the ticket code, and verifies
   using their own TOTP or passkey. Password-only verification is insufficient.

An existing successful support-access OTP remains valid as mailbox proof for
24 hours. Opening a ticket using its access code alone does not create this proof.

The admin must follow the organization's ownership review procedure. Mailbox
possession and a submitted ticket are not a replacement for that human review.
Never record passwords, OTPs, recovery codes, or identity documents in the reason.

## Recovery OTP endpoints

- Admin send: `POST /v1/auth/admin/support/tickets/:id/recovery/otp`.
- Protected public ticket resend: `POST /v1/support/tickets/:ticketCode/recovery/otp`.
- Protected public ticket verify: `POST /v1/support/tickets/:ticketCode/recovery/verify-otp`.
- Signed-in owner resend/verify: the equivalent routes under
  `/v1/auth/support/tickets/:id/recovery/`.

The requester submits `{ "otp_code": "123456", "confirmed": true }` in the
verification form. The server selects the active challenge for that exact ticket;
requesters cannot supply a challenge token, email destination, or another target.

Codes expire after 10 minutes, allow at most five incorrect attempts, and can be
used once. Sending is limited to three times per ticket per hour, with a one-minute
cooldown shared by admin sends and requester resends. Requesters can resend only
after an administrator has initiated recovery verification. Additional route and
verification attempt limits apply. All recovery endpoints fail closed when the
required Redis state is unavailable.

Verification requires an existing ticket-scoped support session or the linked
user's authenticated session, even if someone knows the ticket URL. Public routes
keep origin validation; authenticated routes also require the normal CSRF token.
No OTP/challenge token is placed in a URL, response, conversation message, or audit
note. Redis stores the hashed code with a separate `support_recovery` purpose;
the database stores only the challenge token hash and request/expiry timestamps.
A replacement challenge invalidates earlier codes for the ticket. Verification
rechecks the category, In Progress status, target role, active status, matching
email, challenge version, expiry, and absence of a completed reset.

The email links to the protected conversation and tells the requester not to send
the OTP to staff. Opening the link performs no verification or credential change.
Both public and signed-in conversations expose an inline verification form;
there is no admin-facing field for the requester's OTP. Request and verification
are recorded as internal audit notes.

While a visible Lost 2FA conversation is under review and lacks fresh mailbox
proof, it refreshes every 15 seconds to discover verification requests. An admin
detail awaiting a live challenge refreshes every 10 seconds. Polling stops after
verification or when the ticket is resolved; admin polling also stops at code
expiry. Normal tickets do not start recovery polling.

Mailbox possession is not conclusive proof of identity, particularly if email is
compromised. The original access code and the fresh OTP may both arrive through
email; they are not independent MFA factors. Human ownership review and the
administrator's enrolled MFA remain mandatory before any reset.

## Endpoint and server checks

`POST /v1/auth/admin/support/tickets/:id/credential-reset`

```json
{
  "method": "all",
  "approved": true,
  "reason": "Reviewed ownership evidence using the approved recovery procedure.",
  "ticket_code": "LHTK-ABC123",
  "verification_token": "<single-use MFA verification token>"
}
```

Methods: `totp`, `passkeys`, `all`. The existing security verification endpoints
issue a session-bound, five-minute proof. Only proofs from an enrolled
TOTP/passkey are accepted. The endpoint requires authenticated admin middleware,
verified email, CSRF protection for browser requests, and rate limiting.

The database transaction rechecks the admin's current role and active status,
the ticket category/status, recent requester OTP proof, exact ticket confirmation,
and the linked account's current email, verified email, role, and active status.
The target is derived exclusively from the stored ticket; there is no caller-
controlled target ID or fallback email lookup. Self resets and every privileged
account are rejected. Removing all passkeys requires an existing password or
verified Google sign-in so recovery does not remove the last login method.

Successful approval:

- Deletes selected TOTP secrets/recovery codes and/or all passkey credentials.
- Keeps the password unchanged and invalidates outstanding password reset links.
- Revokes user sessions and refresh tokens. A Redis error rolls back the database
  changes. If Redis revocation succeeded but the transaction subsequently fails,
  the user may already be signed out; credentials remain unchanged.
- Resolves the ticket and invalidates its public access code.
- Stores the approving admin, target, method, reason, IP, and timestamp; adds an
  internal conversation audit note.
- Inserts a security email into a transactional outbox addressed to the stored,
  verified account email.

Each ticket has at most one reset record, enforced by its primary key. Reopening
it or replaying the request cannot authorize another reset. A new recovery
requires a new ticket and review. Approval and reset are one atomic operation;
there is no reusable preapproval flag or background auto-accept path.

## Delivery and deployment

Apply the existing database migration flow before starting the updated backend.
It adds `support_tickets.requester_verified_at`,
`recovery_verification_requested_at`, `recovery_verification_expires_at`,
`recovery_verification_token_hash`, and `support_credential_resets`.
Deploy the backend and UI together. No new environment variables are required.
The backend also runs its existing AutoMigrate flow at startup. Recovery OTP email
requires configured SMTP credentials; failed delivery clears the active challenge
and returns an error rather than reporting success. `FRONTEND_URL` must use HTTPS
in production.

The registered scheduler job `support-reset-security-notifications` runs every
minute. SMTP credentials must be configured. Missing credentials or delivery
failures keep notifications pending and produce scheduler errors; retries back
off up to 60 minutes and survive restarts. Conditional leases prevent concurrent
workers from routinely sending the same record. Delivery is at least once: a
crash after SMTP accepts a message but before recording success can cause a
repeat notification. Monitor pending rows and scheduler failures.

The email identifies the ticket, removed factors, and UTC time and instructs the
user to register protection again and report unrequested changes. It contains no
credentials or verification tokens.

This flow follows [OWASP's MFA recovery and factor replacement guidance](https://cheatsheetseries.owasp.org/cheatsheets/Multifactor_Authentication_Cheat_Sheet.html).
