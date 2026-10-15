# Email notifications

Optional email is disabled by default. Security, authentication, account,
premium-access, and support emails are transactional and do not use these
preferences.

## User preferences

- `GET /v1/notifications/preferences`
- `PATCH /v1/notifications/preferences`

Example:

```json
{
  "weekly_summary_email": true,
  "promotional_email": false
}
```

Weekly summaries cover the previous complete Monday-Sunday interval. The
default scheduler runs on Monday at 08:00, 12:00, and 16:00 in the process
timezone. Its delivery ledger ensures that successful messages are only sent
once. Override the schedule with `WEEKLY_SUMMARY_CRON`.

## Promotional campaigns

Campaign endpoints require an admin or super-admin account:

The web administration page is available at
`/main/admin/email-campaigns`.

- `GET /v1/auth/admin/promotional-campaigns`
- `POST /v1/auth/admin/promotional-campaigns`
- `POST /v1/auth/admin/promotional-campaigns/image` (multipart field `image`)
- `GET /v1/auth/admin/promotional-campaigns/:id`
- `GET /v1/auth/admin/promotional-campaigns/:id/deliveries`
- `PUT /v1/auth/admin/promotional-campaigns/:id`
- `POST /v1/auth/admin/promotional-campaigns/:id/schedule`
- `POST /v1/auth/admin/promotional-campaigns/:id/cancel`
- `DELETE /v1/auth/admin/promotional-campaigns/:id`

The campaign list accepts `page`, `limit`, and an optional `status` (`draft`,
`scheduled`, `sending`, `completed`, `cancelled`, or `failed`). `total` and
`total_pages` describe the filtered result, while `status_counts` includes all
campaigns across every page and status. The admin summary uses these global
counts rather than treating the current page as the complete campaign history.

Create a draft:

```json
{
  "name": "Premium launch",
  "subject": "Meet the new Lihatin Premium",
  "preheader": "More analytics and higher limits.",
  "body": "Premium gives you deeper insight into every link.",
  "image_url": "https://cdn.example.com/premium-launch.jpg",
  "image_alt": "Lihatin Premium analytics dashboard",
  "cta_label": "Explore Premium",
  "cta_url": "https://lihat.in/main"
}
```

The hero image is optional. Admins can paste a public HTTP(S) URL or upload a
verified JPG, PNG, WebP, or GIF image up to 5 MB. Uploads use the configured
S3-compatible object storage and return `image_url` plus `object_key`. Email
copy should still make sense when a recipient's mail client blocks remote
images.

Schedule it immediately with an empty JSON object:

```json
{}
```

Or provide an RFC 3339 timestamp:

```json
{
  "scheduled_at": "2026-08-03T08:00:00+07:00"
}
```

The worker claims due campaigns once per minute. It selects only verified,
active users who explicitly opted into promotional email, rechecks consent
immediately before delivery, and records a unique delivery per campaign and
user. Failed campaigns can be scheduled again; successful recipients are
skipped during retries.

Both weekly and promotional messages include signed unsubscribe links and
`List-Unsubscribe`/one-click unsubscribe headers.

## In-app announcements

In-app announcements are product notices shown inside the dashboard. They do
not depend on the email preferences above: every active account sees a
published announcement until that account dismisses it.

Reader endpoints:

- `GET /v1/notifications/in-app` — pending announcements for the current account,
  used by the dashboard's one-at-a-time announcement dialog
- `POST /v1/notifications/in-app/:id/read` — mark one announcement as read

Inbox endpoints (`/v1/notifications/in-app`):

- `GET /inbox?page=1&limit=20` — paginated announcements visible to the current
  account, including both read and unread items. The response includes
  `unread_count`; reading an item does not remove it from the inbox.
- `GET /unread-count` — current unread badge count (`unread_count`)
- `POST /read-all` — mark all currently visible unread announcements as read;
  returns `marked_count` and is safe to repeat
- `POST /:id/read` — mark one announcement as read
- `POST /:id/unread` — return a read announcement to unread, so it can be read
  again from the inbox

The inbox response's announcement items include `is_read` and `read_at` as well
as the announcement content. The `read` and `unread` actions are account-scoped.

Administration endpoints (`admin` or `super_admin`):

- `GET /v1/auth/admin/in-app-announcements` — paginated list, accepts `page`,
  `limit`, and `status` (`draft`, `published`, or `archived`). The response
  includes `status_counts` totals and a per-announcement `read_count`.
- `POST /v1/auth/admin/in-app-announcements`
- `GET /v1/auth/admin/in-app-announcements/:id`
- `PUT /v1/auth/admin/in-app-announcements/:id`
- `DELETE /v1/auth/admin/in-app-announcements/:id`
- `POST /v1/auth/admin/in-app-announcements/:id/publish`
- `POST /v1/auth/admin/in-app-announcements/:id/archive`
- `GET /v1/auth/admin/in-app-announcements/:id/audience?status=read&page=1&limit=10`
  — paginated account list for `status=read` or `status=unread`. The response
  includes `read_count`, `unread_count`, `total_users`, and `readers`; each row
  contains the account's identity, `is_read`, `read_at`, and `first_seen_at`.
  `read` lists accounts with a read record; `unread` lists accounts with an
  existing unread record (for example, one returned to unread with the inbox's
  “Mark as unread” action). “Read again” opens the announcement dialog and keeps
  the announcement read. Accounts with no record are not included in either
  list. `total_users` is the count of all verified, active accounts, not a
  count scoped to this announcement; `unread_count` counts existing unread
  records.

The web administration page is available at
`/main/admin/in-app-notifications`, next to the email campaigns under the
**Notifications** sidebar group.

Create an announcement:

```json
{
  "title": "API request limits have changed",
  "body": "Free accounts now allow 60 requests per minute.",
  "href": "https://lihat.in/main/api-integrations/docs",
  "status": "published",
  "start_at": "2026-10-15T20:00:00Z",
  "ends_at": "2026-10-22T20:00:00Z"
}
```

`title` is limited to 100 characters and `body` to 150 characters, because the
user-facing dialog renders both without scrolling. `href` is optional but must
be a full HTTP(S) URL when present.

A published announcement reaches an account only while both hold:

- `start_at` is empty or already passed, and
- `ends_at` is empty or still in the future.

Publishing without `start_at` sets `published_at` immediately. Archiving is
reversible with the publish endpoint; deletion permanently removes the
announcement along with every dismissal record.
