# User Account Schema V2

This document is the source of truth for the normalized user, authentication,
account-state, and premium-access schema.

## Design rules

1. `users` stores identity and profile data only.
2. `user_auth` stores the one-to-one authentication record and is the only
   place that stores `password_hash`.
3. `user_premium_access` stores the current premium entitlement. Premium is not
   a role and is not stored on `users`.
4. `premium_status_events` is the append-only premium audit history.
5. `auth_methods` is the source of truth for TOTP and OAuth methods. TOTP
   availability is derived, not cached in `user_auth`.
6. Optional email subscriptions live in `notification_preferences` and do not
   affect security or transactional email.

## Tables

### `users`

Profile and authorization identity:

- `id`: UUIDv7 string
- `username`, `email`: unique login identifiers
- `first_name`, `last_name`, `avatar`: profile data
- `role`: authorization role (`user`, `admin`, `super_admin`)
- `username_changed`: username-change policy state
- timestamps and soft deletion

The table does **not** contain a password, premium state, or account lock state.

### `user_auth`

Exactly one row per user:

- `id`: UUIDv7 string
- `user_id`: unique foreign key to `users`
- `password_hash`: the only stored password hash
- `account_status`: `active`, `disabled`, or `locked`
- `status_changed_at`, `status_changed_by`, `status_reason`: persistent account
  status audit context
- `failed_login_attempts`, `login_blocked_until`: temporary brute-force defense
- email verification and password-reset state
- last login/logout, device, and IP metadata

Authentication responses expose `account_status` directly. There is no
`is_active` alias.

### `user_premium_access`

At most one current entitlement row per user:

- `user_id`: primary key and foreign key to `users`
- `status`: `active` or `revoked`
- `tier`: currently `premium`
- `source`: where the entitlement came from, such as a premium code
- `granted_at`, optional `expires_at`
- `revoke_type`: `temporary` or `permanent`
- status change actor, time, and reason

No row means the user is on the free tier. An entitlement is effective only
when its status is active and it has not expired.

API responses expose this relation as `premium_access`. There is no flattened
`is_premium`, premium revoke/reactivation alias, or premium claim in JWTs.

## Lock semantics

These states intentionally do not share a column:

| State | Storage | Can log in? | Ends how? | Typical response |
| --- | --- | --- | --- | --- |
| Active | `account_status=active` | Yes | N/A | Normal login |
| Disabled | `account_status=disabled` | No | Explicit activation | `403 ACCOUNT_DEACTIVATED` |
| Persistently locked | `account_status=locked` | No | Explicit admin unlock | `403 USER_LOCKED` |
| Temporarily blocked | future `login_blocked_until` | No new login until expiry | Automatically or clear failed-login block | `429 LOGIN_TEMPORARILY_BLOCKED` |

An admin unlock changes only the persistent status. Clearing the failed-login
block changes only `failed_login_attempts` and `login_blocked_until`. A support
workflow may intentionally perform both actions.

## Identifier choice

New user and user-auth IDs use UUIDv7. UUIDv7 is time ordered and standardized;
ULID would not make the IDs secret or materially safer. Existing MySQL string
column widths are preserved in this migration to avoid mixing a foreign-key
storage conversion with the account redesign. Converting UUID strings to
`BINARY(16)` can be evaluated later as a separate, benchmarked migration.

## Migration behavior

`RunMigrations` creates the normalized columns/table, backfills data, validates
that every active legacy user has an auth record and every auth record has a
password hash, and only then removes redundant columns.

Legacy data mapping:

- Prefer `user_auth.password_hash`; fall back to `users.password`.
- `users.is_locked=true` becomes `account_status=locked`.
- Otherwise `user_auth.is_active=false` becomes `account_status=disabled`.
- Failed-login `lockout_until` becomes `login_blocked_until`.
- Existing premium/revocation fields become one `user_premium_access` row.
- TOTP state is derived from verified, enabled `auth_methods`.
- Redundant premium audit role-transition columns are removed; audit events use
  `actor_id` and `actor_role`.

The migration drops legacy columns and must first be run against a recent
production snapshot in staging. Take a database backup, inspect row counts and
constraints, then schedule production execution. Do not treat application unit
tests as a substitute for that database rehearsal.
