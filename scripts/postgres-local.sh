#!/bin/sh
set -eu
project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cluster="$project_root/.local/postgres"
port="${LOCAL_POSTGRES_PORT:-55432}"
case "$port" in ''|*[!0-9]*) echo 'LOCAL_POSTGRES_PORT must be numeric' >&2; exit 1;; esac
if ! command -v pg_ctl >/dev/null 2>&1; then
  echo 'PostgreSQL tools are missing. Add Postgres.app/bin to PATH or install PostgreSQL yourself.' >&2
  exit 1
fi
if [ "${1:-start}" = stop ]; then
  pg_ctl -D "$cluster" -m fast stop
  exit
fi
mkdir -p "$project_root/.local"
if [ ! -f "$cluster/PG_VERSION" ]; then
  initdb -D "$cluster" -A trust --no-locale -E UTF8
fi
if ! pg_ctl -D "$cluster" status >/dev/null 2>&1; then
  pg_ctl -D "$cluster" -l "$project_root/.local/postgres.log" -o "-h 127.0.0.1 -p $port -k /private/tmp -c shared_buffers=64MB -c max_connections=30" start
fi
existing=$(psql -h 127.0.0.1 -p "$port" -d postgres -Atqc "SELECT 1 FROM pg_database WHERE datname = 'jobhunter'")
if [ "$existing" != 1 ]; then
  createdb -h 127.0.0.1 -p "$port" jobhunter
fi
printf 'Native PostgreSQL is ready. Set DATABASE_URL in .env to:\npostgres://%s@127.0.0.1:%s/jobhunter?sslmode=disable\n' "$(id -un)" "$port"
