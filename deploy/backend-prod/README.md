# Backend Deploy (Lihatin-Go)

Production Docker deployment for API server.

## Prerequisites

- Docker Engine + Compose plugin installed.
- External MariaDB and Valkey already running.
- DNS/Reverse proxy prepared.

## 1) Configure

```bash
cd deploy/backend-prod
cp .env.example .env
nano .env
```

Set all secrets and connection values.

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
- If API is on separate host, allow only proxy IP in firewall/security group.
