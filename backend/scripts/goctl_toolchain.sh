#!/usr/bin/env bash

# 统一构建并校验 API 生成所需的 goctl，避免调用方误用系统中的其他版本。
set -euo pipefail

readonly REQUIRED_GOCTL_VERSION='1.7.5'
readonly GOCTL_MODULE='github.com/zeromicro/go-zero/tools/goctl'
command_name="${1:---install}"
install_dir="${2:-${GOCTL_INSTALL_DIR:-$(pwd)/.bin}}"
goctl_bin="${GOCTL_BIN:-${install_dir}/goctl}"

verify_goctl() {
  if [[ ! -x "$goctl_bin" ]]; then
    echo "未找到 goctl 可执行文件: $goctl_bin" >&2
    return 1
  fi

  local version_output
  version_output="$($goctl_bin --version)"
  if [[ ! "$version_output" =~ goctl[[:space:]]version[[:space:]]${REQUIRED_GOCTL_VERSION}([[:space:]]|$) ]]; then
    echo "goctl 版本不匹配，需要 ${REQUIRED_GOCTL_VERSION}，当前为: ${version_output}" >&2
    return 1
  fi
  echo "$version_output"
}

install_goctl() {
  mkdir -p "$install_dir"
  local build_dir
  build_dir="$(mktemp -d)"
  trap "rm -rf -- $(printf '%q' "$build_dir")" EXIT

  # 以临时模块解析精确 tag 后构建，避开在不同 Go 环境下不稳定的 go install 入口。
  (
    cd "$build_dir"
    go mod init wplink/goctl-toolchain >/dev/null
    go get "${GOCTL_MODULE}@v${REQUIRED_GOCTL_VERSION}" >/dev/null 2>&1
    go build -o "$goctl_bin" "$GOCTL_MODULE"
  )
  verify_goctl
}

case "$command_name" in
  --install)
    install_goctl
    ;;
  --verify)
    verify_goctl
    ;;
  *)
    echo "用法: $0 --install [安装目录] | --verify" >&2
    exit 2
    ;;
esac
