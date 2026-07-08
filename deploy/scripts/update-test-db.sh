#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

WPLINK_DEPLOY_TARGET="${WPLINK_DEPLOY_TARGET:-${DEPLOY_TARGET:-}}"
WPLINK_SSH_KEY="${WPLINK_SSH_KEY:-}"
SSH_PORT="${SSH_PORT:-22}"
REMOTE_CONFIG_DIR="${REMOTE_CONFIG_DIR:-/etc/wplink}"
REMOTE_TMP_ROOT="${REMOTE_TMP_ROOT:-/tmp}"
SEED_DEMO=0
CONFIRM_RESET="${WPLINK_CONFIRM_RESET:-0}"

usage() {
  cat <<'EOF'
Usage:
  WPLINK_DEPLOY_TARGET=root@your-test-server deploy/scripts/update-test-db.sh --yes

Options:
  --target USER@HOST              SSH target. Same as WPLINK_DEPLOY_TARGET.
  --ssh-key PATH                  SSH private key path. Same as WPLINK_SSH_KEY.
  --port PORT                     SSH port. Default: 22.
  --config-dir PATH               Remote config dir. Default: /etc/wplink.
  --seed-demo                     Import backend/scripts/seed_demo_data.sql after migrations.
  --yes                           Confirm destructive reset of the remote test database.
  -h, --help                      Show this help.

Environment variables:
  WPLINK_DEPLOY_TARGET            Required unless --target is set.
  WPLINK_SSH_KEY                  Optional SSH private key path.
  SSH_PORT                        Default: 22.
  REMOTE_CONFIG_DIR               Default: /etc/wplink.
  REMOTE_TMP_ROOT                 Default: /tmp.
  WPLINK_CONFIRM_RESET            Set to 1 as an alternative to --yes.

What this does on the remote test server:
  1. Reads DATABASE_URL from REMOTE_CONFIG_DIR/wplink.env.
  2. Backs up the current database with pg_dump to /tmp/wplink-db-backup-*.sql.
  3. Drops and recreates the public schema.
  4. Applies every backend/migrations/*.up.sql in sorted order.
  5. Optionally imports backend/scripts/seed_demo_data.sql with --seed-demo.

This script is intended for test servers only. It destroys current data after backup.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --target)
      WPLINK_DEPLOY_TARGET="${2:-}"
      shift 2
      ;;
    --ssh-key)
      WPLINK_SSH_KEY="${2:-}"
      shift 2
      ;;
    --port)
      SSH_PORT="${2:-}"
      shift 2
      ;;
    --config-dir)
      REMOTE_CONFIG_DIR="${2:-}"
      shift 2
      ;;
    --seed-demo)
      SEED_DEMO=1
      shift
      ;;
    --yes)
      CONFIRM_RESET=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      printf 'unknown argument: %s\n\n' "$1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ -z "$WPLINK_DEPLOY_TARGET" ]]; then
  printf 'WPLINK_DEPLOY_TARGET is required.\n\n' >&2
  usage >&2
  exit 2
fi

if [[ "$CONFIRM_RESET" != "1" ]]; then
  printf 'Refusing to reset the remote database without --yes or WPLINK_CONFIRM_RESET=1.\n\n' >&2
  usage >&2
  exit 2
fi

if [[ -n "$WPLINK_SSH_KEY" && ! -f "$WPLINK_SSH_KEY" ]]; then
  printf 'WPLINK_SSH_KEY file does not exist: %s\n' "$WPLINK_SSH_KEY" >&2
  exit 2
fi

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    printf 'required command not found: %s\n' "$1" >&2
    exit 1
  fi
}

shell_quote() {
  printf '%q' "$1"
}

require_command bash
require_command find
require_command sort
require_command tar
require_command ssh
require_command scp
require_command mktemp

migration_count="$(find "$ROOT_DIR/backend/migrations" -maxdepth 1 -type f -name '*.up.sql' | wc -l | tr -d '[:space:]')"
if [[ "$migration_count" == "0" ]]; then
  printf 'no migration files found in backend/migrations\n' >&2
  exit 1
fi

if [[ ! -f "$ROOT_DIR/backend/scripts/seed_demo_data.sql" ]]; then
  printf 'missing seed file: backend/scripts/seed_demo_data.sql\n' >&2
  exit 1
fi

timestamp="$(date +%Y%m%d%H%M%S)"
local_tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/wplink-test-db-${timestamp}.XXXXXX")"
bundle_path="$local_tmp_dir/update-test-db.tar.gz"
remote_bundle="${REMOTE_TMP_ROOT%/}/wplink-test-db-${timestamp}.tar.gz"

cleanup() {
  rm -rf "$local_tmp_dir"
}
trap cleanup EXIT

ssh_cmd=(ssh -p "$SSH_PORT")
scp_cmd=(scp -P "$SSH_PORT")
if [[ -n "$WPLINK_SSH_KEY" ]]; then
  ssh_cmd+=(-i "$WPLINK_SSH_KEY")
  scp_cmd+=(-i "$WPLINK_SSH_KEY")
fi

printf 'packaging migrations and optional seed files...\n'
tar -czf "$bundle_path" -C "$ROOT_DIR" backend/migrations backend/scripts/seed_demo_data.sql

printf 'uploading database reset bundle to %s:%s...\n' "$WPLINK_DEPLOY_TARGET" "$remote_bundle"
"${scp_cmd[@]}" "$bundle_path" "$WPLINK_DEPLOY_TARGET:$remote_bundle"

remote_env=(
  "REMOTE_BUNDLE=$(shell_quote "$remote_bundle")"
  "REMOTE_CONFIG_DIR=$(shell_quote "$REMOTE_CONFIG_DIR")"
  "SEED_DEMO=$(shell_quote "$SEED_DEMO")"
  "CONFIRM_RESET=$(shell_quote "$CONFIRM_RESET")"
)

"${ssh_cmd[@]}" "$WPLINK_DEPLOY_TARGET" "${remote_env[*]} bash -s" <<'REMOTE_SCRIPT'
set -euo pipefail

if [[ "$CONFIRM_RESET" != "1" ]]; then
  printf 'remote reset confirmation missing; aborting\n' >&2
  exit 2
fi

if [[ "$(id -u)" == "0" ]]; then
  SUDO=()
else
  SUDO=(sudo)
fi

need_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    printf 'remote required command not found: %s\n' "$1" >&2
    exit 1
  fi
}

need_command bash
need_command tar
need_command psql
need_command pg_dump
need_command mktemp
if [[ "${#SUDO[@]}" -gt 0 ]]; then
  need_command sudo
fi

if [[ ! -f "$REMOTE_BUNDLE" ]]; then
  printf 'remote bundle not found: %s\n' "$REMOTE_BUNDLE" >&2
  exit 1
fi

read_database_url() {
  "${SUDO[@]}" bash -c 'set -euo pipefail; set -a; source "$1"; set +a; printf "%s" "${DATABASE_URL:-}"' _ "$REMOTE_CONFIG_DIR/wplink.env"
}

psql_run() {
  local db_url="$1"
  shift
  "${SUDO[@]}" bash -c 'DATABASE_URL="$1"; shift; psql "$DATABASE_URL" "$@"' _ "$db_url" "$@"
}

pg_dump_run() {
  local db_url="$1"
  shift
  "${SUDO[@]}" bash -c 'DATABASE_URL="$1"; shift; pg_dump "$DATABASE_URL" "$@"' _ "$db_url" "$@"
}

database_url="$(read_database_url)"
if [[ -z "$database_url" ]]; then
  printf 'DATABASE_URL is empty in %s/wplink.env\n' "$REMOTE_CONFIG_DIR" >&2
  exit 1
fi

work_dir="$(mktemp -d /tmp/wplink-test-db.XXXXXX)"
remote_cleanup() {
  rm -rf "$work_dir" "$REMOTE_BUNDLE"
}
trap remote_cleanup EXIT

tar -xzf "$REMOTE_BUNDLE" -C "$work_dir"

backup_file="/tmp/wplink-db-backup-$(date +%Y%m%d%H%M%S).sql"
printf 'backing up current database to %s...\n' "$backup_file"
pg_dump_run "$database_url" -f "$backup_file"
"${SUDO[@]}" chmod 0600 "$backup_file" || true

printf 'dropping and recreating public schema...\n'
psql_run "$database_url" -v ON_ERROR_STOP=1 -q -c "DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public; GRANT ALL ON SCHEMA public TO public;"
psql_run "$database_url" -v ON_ERROR_STOP=1 -q -c "CREATE TABLE IF NOT EXISTS schema_migrations (version text PRIMARY KEY, name text NOT NULL, applied_at timestamptz NOT NULL DEFAULT now());"

migration_dir="$work_dir/backend/migrations"
for migration_file in "$migration_dir"/*.up.sql; do
  migration_base="$(basename "$migration_file")"
  migration_key="${migration_base%.up.sql}"
  migration_version="${migration_base%%_*}"
  printf 'applying migration: %s\n' "$migration_base"
  psql_run "$database_url" -v ON_ERROR_STOP=1 -f "$migration_file"
  psql_run "$database_url" -v ON_ERROR_STOP=1 -q -c "INSERT INTO schema_migrations (version, name) VALUES ('$migration_version', '$migration_key') ON CONFLICT (version) DO UPDATE SET name = EXCLUDED.name, applied_at = now();"
done

if [[ "$SEED_DEMO" == "1" ]]; then
  printf 'importing demo seed data...\n'
  psql_run "$database_url" -v ON_ERROR_STOP=1 -f "$work_dir/backend/scripts/seed_demo_data.sql"
else
  printf 'skipping demo seed data; pass --seed-demo to import it.\n'
fi

printf 'verifying unified resource demand direction schema...\n'
psql_run "$database_url" -v ON_ERROR_STOP=1 <<'SQL'
DO $$
DECLARE
  direction_column_count integer;
  demand_type_count integer;
BEGIN
  SELECT count(*)
  INTO direction_column_count
  FROM information_schema.columns
  WHERE table_schema = 'public'
    AND table_name IN ('resources', 'resource_type_configs')
    AND column_name = 'direction';

  IF direction_column_count <> 2 THEN
    RAISE EXCEPTION 'expected direction columns on resources and resource_type_configs, got %', direction_column_count;
  END IF;

  SELECT count(*)
  INTO demand_type_count
  FROM resource_type_configs
  WHERE direction = 'demand'
    AND type_code IN ('buy_goods', 'find_inventory', 'find_factory', 'find_service');

  IF demand_type_count < 4 THEN
    RAISE EXCEPTION 'expected 4 demand resource types, got %', demand_type_count;
  END IF;
END $$;
SQL

psql_run "$database_url" -v ON_ERROR_STOP=1 -c "SELECT type_code, type_name, direction FROM resource_type_configs WHERE type_code IN ('goods','inventory','factory','service','buy_goods','find_inventory','find_factory','find_service') ORDER BY direction, type_code;"

printf 'test database updated successfully. Backup: %s\n' "$backup_file"
REMOTE_SCRIPT
