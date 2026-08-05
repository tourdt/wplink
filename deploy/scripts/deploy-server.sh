#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

WPLINK_DEPLOY_TARGET="${WPLINK_DEPLOY_TARGET:-${DEPLOY_TARGET:-}}"
WPLINK_SSH_KEY="${WPLINK_SSH_KEY:-}"
SSH_PORT="${SSH_PORT:-22}"
REMOTE_APP_DIR="${REMOTE_APP_DIR:-/opt/wplink}"
REMOTE_CONFIG_DIR="${REMOTE_CONFIG_DIR:-/etc/wplink}"
REMOTE_RELEASES_DIR="${REMOTE_RELEASES_DIR:-/opt/wplink/releases}"
REMOTE_SERVICE="${REMOTE_SERVICE:-wplink-api}"
REMOTE_NGINX_CONF="${REMOTE_NGINX_CONF:-/etc/nginx/conf.d/wplink.conf}"
SERVICE_USER="${SERVICE_USER:-wplink}"
SERVICE_GROUP="${SERVICE_GROUP:-$SERVICE_USER}"
HEALTH_BASE_URL="${HEALTH_BASE_URL:-http://127.0.0.1:4000}"
RUN_MIGRATIONS="${RUN_MIGRATIONS:-1}"
MARK_MIGRATIONS_APPLIED="${MARK_MIGRATIONS_APPLIED:-0}"
INSTALL_NGINX="${INSTALL_NGINX:-0}"
CONFIRM_NO_LEGACY_SCHEDULERS=0

MIGRATION_FILES=()
while IFS= read -r migration_file; do
  [[ -n "$migration_file" ]] && MIGRATION_FILES+=("$migration_file")
done < <(find "$ROOT_DIR/backend/migrations" -maxdepth 1 -type f -name '*.up.sql' -exec basename {} \; | sort)

if [[ ${#MIGRATION_FILES[@]} -eq 0 ]]; then
  printf 'no migration files found under backend/migrations\n' >&2
  exit 1
fi

usage() {
  cat <<'EOF'
Usage:
  WPLINK_DEPLOY_TARGET=root@your-server deploy/scripts/deploy-server.sh

Options:
  --target USER@HOST              SSH target. Same as WPLINK_DEPLOY_TARGET.
  --ssh-key PATH                  SSH private key path. Same as WPLINK_SSH_KEY.
  --port PORT                     SSH port. Default: 22.
  --skip-migrations               Do not run migrations. Only allowed when the audit lease
                                  schema is already compatible.
  --mark-migrations-applied       Record migrations without SQL. Only allowed when the
                                  audit lease schema is already compatible.
  --confirm-no-legacy-schedulers  Confirm every old API/Scheduler on every host is stopped
                                  before the first audit lease protocol migration.
  --install-nginx                 Install deploy/nginx/wplink.conf to Nginx and reload it.
  -h, --help                      Show this help.

Environment variables:
  WPLINK_SSH_KEY                  Optional SSH private key path.
  REMOTE_APP_DIR                  Default: /opt/wplink
  REMOTE_CONFIG_DIR               Default: /etc/wplink
  REMOTE_RELEASES_DIR             Default: /opt/wplink/releases
  REMOTE_SERVICE                  Default: wplink-api
  REMOTE_NGINX_CONF               Default: /etc/nginx/conf.d/wplink.conf
  SERVICE_USER                    Default: wplink
  SERVICE_GROUP                   Default: same as SERVICE_USER
  HEALTH_BASE_URL                 Default: http://127.0.0.1:4000
  RUN_MIGRATIONS                  Default: 1
  MARK_MIGRATIONS_APPLIED         Default: 0
  INSTALL_NGINX                   Default: 0

Notes:
  - Run this script on your local machine from the repository.
  - The SSH user must be root or have passwordless sudo for systemd, /opt, and /etc.
  - On the first deploy, missing /etc/wplink/app.yaml or wplink.env are created from
    templates, then the script stops so you can fill production secrets and rerun.
  - Existing databases without audit_lease_until require a maintenance window. Stop
    every legacy API/Scheduler first, then rerun with --confirm-no-legacy-schedulers.
    The flag cannot inspect or stop services on other hosts; see docs/deployment.md.
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
    --skip-migrations)
      RUN_MIGRATIONS=0
      shift
      ;;
    --mark-migrations-applied)
      MARK_MIGRATIONS_APPLIED=1
      shift
      ;;
    --confirm-no-legacy-schedulers)
      CONFIRM_NO_LEGACY_SCHEDULERS=1
      shift
      ;;
    --install-nginx)
      INSTALL_NGINX=1
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
require_command tar
require_command ssh
require_command scp

for migration_file in "${MIGRATION_FILES[@]}"; do
  if [[ ! -f "$ROOT_DIR/backend/migrations/$migration_file" ]]; then
    printf 'missing migration file: backend/migrations/%s\n' "$migration_file" >&2
    exit 1
  fi
  if [[ ! "$migration_file" =~ ^[0-9]{6}_[a-z0-9_]+\.up\.sql$ ]]; then
    printf 'invalid migration file name: backend/migrations/%s\n' "$migration_file" >&2
    exit 1
  fi
done

printf 'building release package...\n'
bash "$ROOT_DIR/deploy/scripts/build-release.sh"

required_release_files=(
  "$ROOT_DIR/dist/release/wplink-api"
  "$ROOT_DIR/dist/release/wplink-api.service"
  "$ROOT_DIR/dist/release/wplink.nginx.conf"
  "$ROOT_DIR/dist/release/app.yaml.example"
  "$ROOT_DIR/dist/release/wplink.env.example"
)

for release_file in "${required_release_files[@]}"; do
  if [[ ! -f "$release_file" ]]; then
    printf 'missing release file: %s\n' "$release_file" >&2
    exit 1
  fi
done

migration_manifest="$ROOT_DIR/dist/release/migrations.manifest"
printf '%s\n' "${MIGRATION_FILES[@]}" > "$migration_manifest"

release_name="wplink-$(date +%Y%m%d%H%M%S)"
local_bundle="$ROOT_DIR/dist/$release_name.tar.gz"
remote_tmp="/tmp/$release_name"
bundle_items=(
  "dist/release/wplink-api"
  "dist/release/wplink-api.service"
  "dist/release/wplink.nginx.conf"
  "dist/release/app.yaml.example"
  "dist/release/wplink.env.example"
  "dist/release/migrations.manifest"
)

for migration_file in "${MIGRATION_FILES[@]}"; do
  bundle_items+=("backend/migrations/$migration_file")
done

printf 'creating upload bundle: %s\n' "$local_bundle"
tar -C "$ROOT_DIR" -czf "$local_bundle" "${bundle_items[@]}"

ssh_cmd=(ssh -p "$SSH_PORT")
scp_cmd=(scp -P "$SSH_PORT")
if [[ -n "$WPLINK_SSH_KEY" ]]; then
  ssh_cmd+=(-i "$WPLINK_SSH_KEY" -o IdentitiesOnly=yes)
  scp_cmd+=(-i "$WPLINK_SSH_KEY" -o IdentitiesOnly=yes)
fi

printf 'creating remote temp dir: %s\n' "$remote_tmp"
"${ssh_cmd[@]}" "$WPLINK_DEPLOY_TARGET" "mkdir -p $(shell_quote "$remote_tmp")"

printf 'uploading bundle to %s...\n' "$WPLINK_DEPLOY_TARGET"
"${scp_cmd[@]}" "$local_bundle" "$WPLINK_DEPLOY_TARGET:$remote_tmp/release.tar.gz"

remote_env=(
  "REMOTE_TMP=$(shell_quote "$remote_tmp")"
  "RELEASE_NAME=$(shell_quote "$release_name")"
  "REMOTE_APP_DIR=$(shell_quote "$REMOTE_APP_DIR")"
  "REMOTE_CONFIG_DIR=$(shell_quote "$REMOTE_CONFIG_DIR")"
  "REMOTE_RELEASES_DIR=$(shell_quote "$REMOTE_RELEASES_DIR")"
  "REMOTE_SERVICE=$(shell_quote "$REMOTE_SERVICE")"
  "REMOTE_NGINX_CONF=$(shell_quote "$REMOTE_NGINX_CONF")"
  "SERVICE_USER=$(shell_quote "$SERVICE_USER")"
  "SERVICE_GROUP=$(shell_quote "$SERVICE_GROUP")"
  "HEALTH_BASE_URL=$(shell_quote "$HEALTH_BASE_URL")"
  "RUN_MIGRATIONS=$(shell_quote "$RUN_MIGRATIONS")"
  "MARK_MIGRATIONS_APPLIED=$(shell_quote "$MARK_MIGRATIONS_APPLIED")"
  "CONFIRM_NO_LEGACY_SCHEDULERS=$(shell_quote "$CONFIRM_NO_LEGACY_SCHEDULERS")"
  "INSTALL_NGINX=$(shell_quote "$INSTALL_NGINX")"
)

printf 'installing release on remote server...\n'
"${ssh_cmd[@]}" "$WPLINK_DEPLOY_TARGET" "${remote_env[*]} bash -s" <<'REMOTE_SCRIPT'
set -euo pipefail

SUDO=()
if [[ "$(id -u)" -ne 0 ]]; then
  SUDO=(sudo)
fi

need_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    printf 'remote required command not found: %s\n' "$1" >&2
    exit 1
  fi
}

need_command bash
need_command curl
need_command tar
if [[ "${#SUDO[@]}" -gt 0 ]]; then
  need_command sudo
fi
# 即使显式跳过 migration，也必须查询 schema 协议状态，避免在 clean/legacy 数据库安装新二进制。
need_command psql

extract_dir="$REMOTE_TMP/extract"
mkdir -p "$extract_dir"
tar -xzf "$REMOTE_TMP/release.tar.gz" -C "$extract_dir"

if ! getent group "$SERVICE_GROUP" >/dev/null 2>&1; then
  "${SUDO[@]}" groupadd --system "$SERVICE_GROUP"
fi

if ! id "$SERVICE_USER" >/dev/null 2>&1; then
  nologin_shell="$(command -v nologin || true)"
  if [[ -z "$nologin_shell" ]]; then
    nologin_shell="/bin/false"
  fi
  "${SUDO[@]}" useradd --system --gid "$SERVICE_GROUP" --home-dir "$REMOTE_APP_DIR" --shell "$nologin_shell" "$SERVICE_USER"
fi

"${SUDO[@]}" mkdir -p "$REMOTE_APP_DIR" "$REMOTE_CONFIG_DIR" "$REMOTE_RELEASES_DIR/$RELEASE_NAME" "$REMOTE_APP_DIR/logs"

if [[ -f "$REMOTE_APP_DIR/wplink-api" ]]; then
  "${SUDO[@]}" cp "$REMOTE_APP_DIR/wplink-api" "$REMOTE_RELEASES_DIR/$RELEASE_NAME/wplink-api.previous"
fi

created_config=0
if [[ ! -f "$REMOTE_CONFIG_DIR/app.yaml" ]]; then
  "${SUDO[@]}" install -m 0640 -o "$SERVICE_USER" -g "$SERVICE_GROUP" "$extract_dir/dist/release/app.yaml.example" "$REMOTE_CONFIG_DIR/app.yaml"
  created_config=1
fi

if [[ ! -f "$REMOTE_CONFIG_DIR/wplink.env" ]]; then
  "${SUDO[@]}" install -m 0640 -o "$SERVICE_USER" -g "$SERVICE_GROUP" "$extract_dir/dist/release/wplink.env.example" "$REMOTE_CONFIG_DIR/wplink.env"
  created_config=1
fi

"${SUDO[@]}" chown "$SERVICE_USER:$SERVICE_GROUP" "$REMOTE_CONFIG_DIR/app.yaml" "$REMOTE_CONFIG_DIR/wplink.env"
"${SUDO[@]}" chmod 0640 "$REMOTE_CONFIG_DIR/app.yaml" "$REMOTE_CONFIG_DIR/wplink.env"

service_file="$REMOTE_TMP/$REMOTE_SERVICE.service"
cat > "$service_file" <<SERVICE_UNIT
[Unit]
Description=Wplink API Service
After=network.target

[Service]
Type=simple
User=$SERVICE_USER
Group=$SERVICE_GROUP
WorkingDirectory=$REMOTE_APP_DIR
EnvironmentFile=$REMOTE_CONFIG_DIR/wplink.env
ExecStart=$REMOTE_APP_DIR/wplink-api -f $REMOTE_CONFIG_DIR/app.yaml
Restart=always
RestartSec=5
LimitNOFILE=65535
NoNewPrivileges=true

[Install]
WantedBy=multi-user.target
SERVICE_UNIT

"${SUDO[@]}" install -m 0644 "$service_file" "/etc/systemd/system/$REMOTE_SERVICE.service"
"${SUDO[@]}" systemctl daemon-reload

if [[ "$created_config" == "1" ]]; then
  cat <<CONFIG_MESSAGE
Created initial config templates:
  $REMOTE_CONFIG_DIR/app.yaml
  $REMOTE_CONFIG_DIR/wplink.env

Fill production database, JWT, WeChat, SMS, and Qiniu values, then rerun this deploy script.
CONFIG_MESSAGE
  exit 2
fi

read_database_url() {
  "${SUDO[@]}" bash -c 'set -euo pipefail; set -a; source "$1"; set +a; printf "%s" "${DATABASE_URL:-}"' _ "$REMOTE_CONFIG_DIR/wplink.env"
}

psql_run() {
  local db_url="$1"
  shift
  "${SUDO[@]}" bash -c 'DATABASE_URL="$1"; shift; psql "$DATABASE_URL" "$@"' _ "$db_url" "$@"
}

database_url="$(read_database_url)"
if [[ -z "$database_url" ]]; then
  printf 'DATABASE_URL is empty in %s\n' "$REMOTE_CONFIG_DIR/wplink.env" >&2
  exit 1
fi

# 不依赖 schema_migrations 判断协议代际：历史库可能从未使用该登记表。
# 所有发布模式都必须先完成检测，skip/mark 不能绕过新二进制所依赖的 schema 前置条件。
protocol_state="$(psql_run "$database_url" -X -A -t -q -v ON_ERROR_STOP=1 -c "
SELECT CASE
  WHEN to_regclass('public.resources') IS NULL THEN 'clean'
  WHEN EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = 'public'
      AND table_name = 'resources'
      AND column_name = 'audit_lease_until'
  ) THEN 'compatible'
  ELSE 'legacy'
END;
")"
protocol_state="${protocol_state//[[:space:]]/}"

case "$protocol_state" in
  clean)
    if [[ "$RUN_MIGRATIONS" != "1" ]]; then
      printf '干净新库不能跳过 migration；必须先创建完整 schema，已拒绝安装新二进制。\n' >&2
      exit 1
    fi
    if [[ "$MARK_MIGRATIONS_APPLIED" == "1" ]]; then
      printf '干净新库不能只标记 migration；必须实际执行完整 SQL，已拒绝安装新二进制。\n' >&2
      exit 1
    fi
    printf '干净新库将执行完整 migration，首次协议升级无需 legacy 确认。\n'
    ;;
  compatible)
    printf '数据库已具备审核租约字段，允许按当前 migration 参数继续发布。\n'
    ;;
  legacy)
    if [[ "$RUN_MIGRATIONS" != "1" ]]; then
      printf '旧协议数据库不能跳过 migration；必须先完成 000035 租约升级，已拒绝安装新二进制。\n' >&2
      exit 1
    fi
    if [[ "$MARK_MIGRATIONS_APPLIED" == "1" ]]; then
      printf '旧协议数据库不能只标记 migration；必须实际执行 000035 租约升级，已拒绝安装新二进制。\n' >&2
      exit 1
    fi
    if [[ "$CONFIRM_NO_LEGACY_SCHEDULERS" != "1" ]]; then
      cat >&2 <<'PROTOCOL_GATE_MESSAGE'
检测到已有 resources 表但尚无 audit_lease_until，默认拒绝在旧调度协议运行期间执行 migration。
必须先停止所有旧 API/Scheduler，并确认所有主机上都没有旧任务仍在执行，然后使用 --confirm-no-legacy-schedulers 重试。
脚本只能检查并停止当前目标主机，无法自动确认其他服务器；完整维护窗口步骤见 docs/deployment.md。
PROTOCOL_GATE_MESSAGE
      exit 1
    fi

    printf '确认参数只表示运维已在所有主机消除旧 Scheduler；脚本不会也不能自动确认其他服务器。\n'
    if "${SUDO[@]}" systemctl is-active --quiet "$REMOTE_SERVICE"; then
      printf '首次协议升级：migration 前停止当前目标主机服务 %s。\n' "$REMOTE_SERVICE"
      "${SUDO[@]}" systemctl stop "$REMOTE_SERVICE"
    else
      printf '首次协议升级：当前目标主机服务 %s 未运行。\n' "$REMOTE_SERVICE"
    fi
    ;;
  *)
    printf '无法识别数据库调度协议状态，已拒绝执行 migration 和安装新二进制。\n' >&2
    exit 1
    ;;
esac

if [[ "$RUN_MIGRATIONS" == "1" || "$MARK_MIGRATIONS_APPLIED" == "1" ]]; then
  migration_files=()
  while IFS= read -r migration_file; do
    [[ -n "$migration_file" ]] && migration_files+=("$migration_file")
  done < "$extract_dir/dist/release/migrations.manifest"

  if [[ ${#migration_files[@]} -eq 0 ]]; then
    printf 'migration manifest is empty\n' >&2
    exit 1
  fi

  migration_batch="$REMOTE_TMP/run-migrations.sql"
  {
    printf '\\set ON_ERROR_STOP on\n'
    printf 'BEGIN;\n'
    # 事务级 advisory lock 保证多次部署并发触发时仍只有一个迁移执行者。
    printf "SELECT pg_advisory_xact_lock(hashtext('wplink_schema_migrations'));\n"
    printf "CREATE TABLE IF NOT EXISTS schema_migrations (version text PRIMARY KEY, name text NOT NULL, applied_at timestamptz NOT NULL DEFAULT now());\n"

    for migration_file in "${migration_files[@]}"; do
      if [[ ! "$migration_file" =~ ^[0-9]{6}_[a-z0-9_]+\.up\.sql$ ]]; then
        printf 'invalid migration file name in manifest: %s\n' "$migration_file" >&2
        exit 1
      fi

      migration_key="${migration_file%.up.sql}"
      migration_version="${migration_file%%_*}"
      printf "SELECT NOT EXISTS (SELECT 1 FROM schema_migrations WHERE version = '%s') AS should_apply \\gset\n" "$migration_version"
      printf '\\if :should_apply\n'
      if [[ "$MARK_MIGRATIONS_APPLIED" == "1" ]]; then
        printf '\\echo marking migration as applied without execution: %s\n' "$migration_file"
      else
        printf '\\echo applying migration: %s\n' "$migration_file"
        printf '\\ir %s\n' "$extract_dir/backend/migrations/$migration_file"
      fi
      printf "INSERT INTO schema_migrations (version, name) VALUES ('%s', '%s');\n" "$migration_version" "$migration_key"
      printf '\\else\n'
      printf '\\echo migration already applied: %s\n' "$migration_file"
      printf '\\endif\n'
    done

    printf 'COMMIT;\n'
  } > "$migration_batch"

  # SQL 变更与 schema_migrations 记录在同一事务提交，任一版本失败都会整体回滚。
  psql_run "$database_url" -q -f "$migration_batch"
fi

# 首次协议升级必须先完成 legacy 闸门、停旧服务和 migration，再切换依赖新字段的二进制。
# 否则 fail-closed 后旧服务若被 systemd 拉起，会在旧 schema 上误启新版本。
"${SUDO[@]}" install -m 0755 "$extract_dir/dist/release/wplink-api" "$REMOTE_APP_DIR/wplink-api"
"${SUDO[@]}" chown -R "$SERVICE_USER:$SERVICE_GROUP" "$REMOTE_APP_DIR"

if [[ "$INSTALL_NGINX" == "1" ]]; then
  "${SUDO[@]}" install -m 0644 "$extract_dir/dist/release/wplink.nginx.conf" "$REMOTE_NGINX_CONF"
  "${SUDO[@]}" nginx -t
  "${SUDO[@]}" systemctl reload nginx
fi

"${SUDO[@]}" systemctl enable "$REMOTE_SERVICE"
"${SUDO[@]}" systemctl restart "$REMOTE_SERVICE"

for path in /healthz /readyz; do
  ok=0
  for _ in $(seq 1 30); do
    if curl -fsS "$HEALTH_BASE_URL$path" >/dev/null; then
      ok=1
      break
    fi
    sleep 2
  done

  if [[ "$ok" != "1" ]]; then
    printf 'health check failed: %s%s\n' "$HEALTH_BASE_URL" "$path" >&2
    "${SUDO[@]}" systemctl status "$REMOTE_SERVICE" --no-pager || true
    exit 1
  fi
  printf 'health check ok: %s%s\n' "$HEALTH_BASE_URL" "$path"
done

printf 'remote deploy finished: %s\n' "$RELEASE_NAME"
REMOTE_SCRIPT

printf 'deploy finished: %s\n' "$release_name"
