#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TEST_DB_TARGET="${WPLINK_DEPLOY_TARGET:-root@124.223.186.63}"
TEST_DB_SSH_KEY="${WPLINK_SSH_KEY:-${HOME}/.ssh/ebyby.pem}"

# 固定测试环境的常用参数，实际的备份、重置、迁移和演示数据导入仍由通用脚本统一维护。
exec "$ROOT_DIR/deploy/scripts/update-test-db.sh" \
  --target "$TEST_DB_TARGET" \
  --ssh-key "$TEST_DB_SSH_KEY" \
  --yes \
  --seed-demo \
  "$@"
