# Admin-assisted credential recovery

The public support form never approves or executes a credential reset. Normal
`PUT /v1/auth/admin/support/tickets/:id` updates cannot reset credentials either.

## Review and approval

1. The requester submits a `lost_2fa` ticket for their existing account.
2. They verify the ticket through **email OTP** on the support access page. This
   records mailbox ownership for that ticket for 24 hours. An access code alone
   does not set this proof. Existing tickets must complete OTP verification again.
3. An administrator reviews ownership evidence and moves the ticket to
   **In Progress**. This status does not authorize recovery by itself.
4. Under **User Management**, choose **Reset Authenticator (TOTP)** or **Remove
   All Passkeys**. The confirmation also offers removal of both.
5. Record the ownership verification reason, explicitly approve recovery, type the
   ticket code, and reauthenticate using the administrator's own TOTP or passkey.
   Password-only verification is insufficient.

The admin must follow the organization's ownership review procedure. Mailbox
possession and a submitted ticket are not a replacement for that human review.
Never record passwords, OTPs, recovery codes, or identity documents in the reason.

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
It adds `support_tickets.requester_verified_at` and `support_credential_resets`.
Deploy the backend and UI together. No new environment variables are required.

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
