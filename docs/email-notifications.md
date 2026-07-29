# Email notifications

Optional email is disabled by default. Security, authentication, account,
premium-status, and support emails are transactional and do not use these
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
