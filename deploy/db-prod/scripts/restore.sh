#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

if [[ $# -lt 2 ]]; then
  echo "Usage: $0 <mariadb|mysql|valkey|redis> <backup_file>"
  exit 1
fi

if [[ ! -f .env ]]; then
  echo "[ERROR] .env not found in $ROOT_DIR"
  exit 1
fi

TYPE="$1"
FILE="$2"

if [[ ! -f "$FILE" ]]; then
  echo "[ERROR] backup file not found: $FILE"
  exit 1
fi

set -a
source .env
set +a

case "$TYPE" in
  mariadb|mysql)
    echo "[WARN] Restoring MariaDB from $FILE"
    gunzip -c "$FILE" | docker compose exec -T mariadb sh -lc 'exec mariadb -uroot -p"$MARIADB_ROOT_PASSWORD"'
    ;;
  valkey|redis)
    echo "[WARN] Restoring Valkey from $FILE"
    docker compose stop valkey
    docker compose cp "$FILE" valkey:/data/dump.rdb
    docker compose start valkey
    ;;
  *)
    echo "[ERROR] invalid type: $TYPE (expected mariadb|mysql|valkey|redis)"
    exit 1
    ;;
esac

echo "[OK] Restore completed"
