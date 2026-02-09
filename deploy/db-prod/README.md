# Production DB Stack (MariaDB + Valkey)

Production-ready Docker stack for a VPS with `2 vCPU / 2GB RAM`.
This stack is tuned for Lihatin-Go and intended for **database-only** usage.

## Included

- MariaDB 11.4 with conservative memory settings
- Valkey 8 with AOF persistence and memory cap
- Health checks, restart policy, resource limits
- Backup and restore scripts for MariaDB + Valkey

## 1) Prepare Server

- Ubuntu 22.04/24.04 recommended.
- Install Docker + Compose plugin.
- Open firewall only for trusted sources.

Example UFW (replace `<BACKEND_IP>`):

```bash
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow OpenSSH
sudo ufw allow from <BACKEND_IP> to any port 3306 proto tcp
sudo ufw allow from <BACKEND_IP> to any port 6379 proto tcp
sudo ufw enable
```

If your backend runs on the same host, keep `DB_BIND_IP=127.0.0.1` and do **not** open 3306/6379 publicly.

## 2) Configure

```bash
cd deploy/db-prod
cp .env.example .env
```

Edit `.env` with strong passwords (24+ chars).

## 3) Start

```bash
docker compose pull
docker compose up -d
```

Check health:

```bash
docker compose ps
docker compose logs -f mariadb
docker compose logs -f valkey
```

## 4) App Connection Strings

MariaDB DSN (`DATABASE_URL`):

```txt
<MARIADB_USER>:<MARIADB_PASSWORD>@tcp(<DB_HOST>:3306)/<MARIADB_DATABASE>?charset=utf8mb4&parseTime=true&loc=Local
```

Valkey (Redis-compatible):

- `REDIS_ADDR=<DB_HOST>:6379`
- `REDIS_PASSWORD=<REDIS_PASSWORD>`
- `REDIS_DB=0`

`<DB_HOST>` can be private IP of your Tencent VPS.

## 5) Backups

Run manual backup:

```bash
cd deploy/db-prod
./scripts/backup.sh
```

Create daily cron at 03:10:

```bash
crontab -e
```

```cron
10 3 * * * cd /path/to/Lihatin-Go/deploy/db-prod && ./scripts/backup.sh >> ./backups/backup.log 2>&1
```

Restore:

```bash
# MariaDB
./scripts/restore.sh mariadb ./backups/mariadb/mariadb_lihatin_go_YYYYMMDD_HHMMSS.sql.gz

# Valkey
./scripts/restore.sh valkey ./backups/valkey/valkey_YYYYMMDD_HHMMSS.rdb
```

## 6) Hardening Checklist

- Use strong random passwords.
- Keep `DB_BIND_IP=127.0.0.1` unless remote backend requires network access.
- Restrict 3306/6379 by firewall IP allowlist only.
- Do not expose Valkey publicly without strict firewall.
- Keep Docker images updated monthly: `docker compose pull && docker compose up -d`.
- Monitor disk (`40GB`) and backup size.

## 7) Capacity Notes for 2GB VPS

- MariaDB memory cap in compose: `1300m`
- Valkey memory cap in compose: `320m`
- Valkey dataset configured with `maxmemory 256mb` + `allkeys-lru`
- Enough for small-to-medium workloads with proper indexing and query tuning.
