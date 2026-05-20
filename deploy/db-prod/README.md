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

Create shared Docker network (once per host):

```bash
docker network create lihatin-backbone
```

If you use a custom name, set `SHARED_NETWORK_NAME` in both:

- `deploy/db-prod/.env`
- `deploy/backend-prod/.env`

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

Use exact same database name and credentials in backend `.env` (case-sensitive).

Valkey (Redis-compatible):

- `REDIS_ADDR=<DB_HOST>:6379`
- `REDIS_PASSWORD=<REDIS_PASSWORD>`
- `REDIS_DB=0`

`<DB_HOST>` can be private IP of your Tencent VPS.
If backend runs in same host using shared Docker network, use:

- `DATABASE_URL=...@tcp(lihatin-mariadb:3306)/...`
- `REDIS_ADDR=lihatin-valkey:6379`

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

## 8) Optional: Remote DB Access via Tailscale/WireGuard (Safer than Public Port)

Use this only if you need desktop tools (Navicat, RedisInsight) from outside VPS.
Recommended: keep `DB_BIND_IP=127.0.0.1` and use tunnel over private mesh VPN.

### Option A: Tailscale + SSH Tunnel (recommended)

1. Install Tailscale on VPS and client, login with same account.
2. Keep DB stack local-only:

```env
DB_BIND_IP=127.0.0.1
```

3. From your laptop, create tunnel to MariaDB and Valkey:

```bash
ssh -L 3307:127.0.0.1:3306 -L 6380:127.0.0.1:6379 root@<TAILSCALE_VPS_IP>
```

4. Connect tools to local forwarded ports:

- Navicat: host `127.0.0.1`, port `3307`
- RedisInsight/redis-cli: host `127.0.0.1`, port `6380`

Redis CLI example:

```bash
redis-cli -h 127.0.0.1 -p 6380 -a <REDIS_PASSWORD> PING
```

### Option B: Direct over Tailscale/WireGuard subnet

If you need direct connection (no SSH tunnel):

1. Set `DB_BIND_IP=0.0.0.0`
2. Restart DB stack:

```bash
docker compose up -d
```

3. Allow firewall only private VPN range, never public world:

```bash
# Example Tailscale subnet
sudo ufw allow from 100.64.0.0/10 to any port 3306 proto tcp
sudo ufw allow from 100.64.0.0/10 to any port 6379 proto tcp
```

4. Connect using VPS private VPN IP (`100.x.x.x` for Tailscale).

### Security Notes

- Do not expose `3306`/`6379` to `0.0.0.0/0`.
- Use dedicated read/write user for tools (avoid root for daily use).
- Rotate DB/Redis credentials if secrets ever shared in chat/logs.
