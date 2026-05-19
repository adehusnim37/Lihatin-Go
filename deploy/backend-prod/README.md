# Backend Deploy (Lihatin-Go)

Production Docker deployment for API server.

## Prerequisites

- Docker Engine + Compose plugin installed.
- DB stack (`deploy/db-prod`) already running.
- DNS/Reverse proxy prepared.
- Shared Docker network already created (default: `lihatin-backbone`).

## 1) Configure

```bash
docker network create lihatin-backbone
cd deploy/backend-prod
cp .env.example .env
nano .env
```

Set all secrets and connection values.
For same-host deployment with split compose projects:

- `SHARED_NETWORK_NAME` must match value in `deploy/db-prod/.env`
- `DATABASE_URL` should target `lihatin-mariadb:3306`
- DB name in `DATABASE_URL` must match `MARIADB_DATABASE` from db stack exactly (default: `lihatin_go`, lowercase)
- DB user/password in `DATABASE_URL` must match `MARIADB_USER`/`MARIADB_PASSWORD` from db stack
- `REDIS_ADDR` should target `lihatin-valkey:6379`

If you use optional features, also set related vars:

- Google OAuth: `GOOGLE_OAUTH_*`
- Support captcha: `TURNSTILE_SECRET_KEY`
- Support notifications: `SUPPORT_ALERT_EMAILS`
- Disposable policy source override: `DISPOSABLE_EMAIL_BLOCK_LIST_URL`
- Object storage avatars: `OSS_*`

## 2) Build and Run

```bash
docker compose build --no-cache
docker compose up -d
```

Check status:

```bash
docker compose ps
docker compose logs -f api
```

Health endpoint:

```bash
curl -fsS http://127.0.0.1:8080/v1/health
```

## 3) Upgrade

```bash
git pull
docker compose build --no-cache
docker compose up -d
```

## 4) Rollback (quick)

```bash
docker image ls | grep lihatin-go
# run previous tag manually if needed
```

## Notes

- App runs DB migrations on startup. Run only one API instance per DB unless you add migration locking.
- Keep `APP_BIND_IP=127.0.0.1` and expose via reverse proxy for production.
- If API is on separate host, use private DB IP and lock firewall/security group strictly.
