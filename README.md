# Lihatin-Go

Backend API for Lihatin, built with Go and Gin.
This service handles authentication, short link management, analytics, support ticket flows, and operational logging.

## Project Status

- Active development
- Public repository
- Issues and pull requests are welcome

## Features

- Cookie-based authentication with refresh token flow
- Google OAuth login support
- Email OTP and TOTP (2FA) flows
- Short link create/read/update/delete and redirect
- API key management and API key-scoped short-link endpoints
- Login attempt tracking and security analytics
- User history/audit entries for account-related actions
- Support ticket system with public and authenticated flows
- Structured activity logging endpoints

## Tech Stack

- Go 1.26
- Gin Web Framework
- GORM + MySQL
- Redis (session and token-related flows)
- JWT
- SMTP email delivery

## Repository Links

- Backend: [github.com/adehusnim37/Lihatin-Go](https://github.com/adehusnim37/Lihatin-Go)
- Frontend: [github.com/adehusnim37/Lihatin-UI](https://github.com/adehusnim37/Lihatin-UI)

## Project Structure

```text
cmd/                  # Utility commands (migrations)
controllers/          # HTTP handlers
dto/                  # Request/response payload objects
internal/             # Core packages (auth, config, jobs, etc.)
middleware/           # Auth, rate-limit, logging, and security middleware
models/               # Database/domain models
repositories/         # Data-access layer
routes/               # Route registration and grouping
deploy/               # Production deployment files
```

## Requirements

- Go 1.26+
- MySQL 8+
- Redis/Valkey
- SMTP credentials (required for auth/support email flows)

## Quick Start (Local)

1. Clone repository

```bash
git clone https://github.com/adehusnim37/Lihatin-Go.git
cd Lihatin-Go
```

2. Install dependencies

```bash
go mod tidy
```

3. Create environment file

```bash
cp .env.example .env
```

4. Set required values in `.env`

- `DATABASE_URL`
- `REDIS_ADDR`
- `JWT_SECRET`
- `SESSION_SECRET`
- `PREMIUM_CODE_SECRET`
- `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASS`, `FROM_EMAIL`
- `APP_PORT` (default `:8080`)

5. Run application

```bash
go run main.go
```

Health check:

```bash
curl -fsS http://127.0.0.1:8080/v1/health
```

## Environment and Runtime Notes

- `ENV=production` enables production-only middleware behavior (such as CSRF middleware application rules).
- `GIN_MODE=release` should be set in production.
- Application runs database migrations on startup.

## API Surface (Summary)

Base path: `/v1`

### Auth

- `POST /auth/login`
- `POST /auth/oauth/google/start`
- `POST /auth/oauth/google/callback`
- `POST /auth/signup/start`
- `POST /auth/signup/resend-otp`
- `POST /auth/signup/verify-otp`
- `POST /auth/signup/complete`
- `POST /auth/login/email-otp/verify`
- `POST /auth/login/email-otp/resend`
- `POST /auth/verify-totp-login`
- `POST /auth/forgot-password`
- `GET /auth/validate-reset`
- `POST /auth/reset-password`
- `POST /auth/refresh-token`

### Short Links

- Public: `POST /short`, `GET /short/:code`
- User scoped: `/users/me/shorts/*`
- Admin scoped: `/admin/shorts/*`
- API key scoped: `/api/short/*`

### Support

- Public support and ticket access: `/support/*`
- Authenticated user support: `/auth/support/*`
- Admin support management: `/auth/admin/support/*`

### Logs and Analytics

- Activity logs: `/logs/*`
- Login attempts: `/auth/login-attempts/*` and `/auth/admin/login-attempts/*`
- User history: `/auth/history/*`

For detailed contracts and payloads, use:

- [OpenAPI spec](docs/openapi.yaml)
- [Postman collection](examples/Lihat.in.postman_collection.json)

## Security Notes

- Login and OTP endpoints include rate limiting.
- Session and refresh token flows depend on Redis availability.
- Keep production secrets in environment variables, never in source control.
- Prefer reverse-proxy deployment and bind API to localhost/private network.

## Deployment

Production compose files are available in `deploy/backend-prod`.

See deployment guide:

- [Backend deployment README](deploy/backend-prod/README.md)
- [DEPLOYMENT_GUIDE.md](DEPLOYMENT_GUIDE.md)

## Documentation

- [docs/AUTHENTICATION_SYSTEM_DOCS.md](docs/AUTHENTICATION_SYSTEM_DOCS.md)
- [docs/HTTP_ONLY_COOKIE_AUTH_IMPLEMENTATION.md](docs/HTTP_ONLY_COOKIE_AUTH_IMPLEMENTATION.md)
- [docs/LOGIN_ATTEMPTS_API_QUICK_REFERENCE.md](docs/LOGIN_ATTEMPTS_API_QUICK_REFERENCE.md)
- [docs/LOGGER_API_QUICK_REFERENCE.md](docs/LOGGER_API_QUICK_REFERENCE.md)
- [docs/EMAIL_CHANGE_FLOW.md](docs/EMAIL_CHANGE_FLOW.md)

## Testing

```bash
# Run package tests
go test ./...

# Run helper verification scripts
go run tests/system_verification.go
go run tests/auth_system_verification.go
```

## Contributing

1. Fork repository
2. Create feature branch
3. Commit changes
4. Open pull request

## License

This project is licensed under the Unlicense.
See [LICENSE](LICENSE).
