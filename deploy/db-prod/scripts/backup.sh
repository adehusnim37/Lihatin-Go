#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

if [[ ! -f .env ]]; then
  echo "[ERROR] .env not found in $ROOT_DIR"
  exit 1
fi

set -a
source .env
set +a

RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-14}"
STAMP="$(date +%Y%m%d_%H%M%S)"
MYSQL_BACKUP_DIR="$ROOT_DIR/backups/mysql"
VALKEY_BACKUP_DIR="$ROOT_DIR/backups/valkey"
mkdir -p "$MYSQL_BACKUP_DIR" "$VALKEY_BACKUP_DIR"

MYSQL_FILE="$MYSQL_BACKUP_DIR/mysql_${MYSQL_DATABASE}_${STAMP}.sql.gz"
VALKEY_FILE="$VALKEY_BACKUP_DIR/valkey_${STAMP}.rdb"

echo "[INFO] MySQL backup -> $MYSQL_FILE"
docker compose exec -T mysql sh -lc \
  'exec mysqldump -uroot -p"$MYSQL_ROOT_PASSWORD" --single-transaction --quick --routines --events --databases "$MYSQL_DATABASE"' \
  | gzip -c > "$MYSQL_FILE"

echo "[INFO] Valkey backup -> $VALKEY_FILE"
docker compose exec -T valkey sh -lc \
  'exec valkey-cli -a "$REDIS_PASSWORD" --rdb /tmp/dump.rdb >/dev/null'

docker compose cp valkey:/tmp/dump.rdb "$VALKEY_FILE" >/dev/null
docker compose exec -T valkey rm -f /tmp/dump.rdb >/dev/null

find "$MYSQL_BACKUP_DIR" -type f -name '*.sql.gz' -mtime +"$RETENTION_DAYS" -delete
find "$VALKEY_BACKUP_DIR" -type f -name '*.rdb' -mtime +"$RETENTION_DAYS" -delete

echo "[OK] Backup completed"
