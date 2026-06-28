#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
OUTPUT_DIR="${OUTPUT_DIR:-$ROOT_DIR/dist/release}"

cd "$ROOT_DIR"

node backend/scripts/prepare_admin_embed.mjs

mkdir -p "$OUTPUT_DIR"

(
  cd backend
  go build -trimpath -ldflags="-s -w" -o "$OUTPUT_DIR/wplink-api" ./app
)

cp backend/etc/app.production.yaml.example "$OUTPUT_DIR/app.yaml.example"
cp deploy/wplink.env.example "$OUTPUT_DIR/wplink.env.example"
cp deploy/systemd/wplink-api.service "$OUTPUT_DIR/wplink-api.service"
cp deploy/nginx/wplink.conf "$OUTPUT_DIR/wplink.nginx.conf"

printf 'release files written to %s\n' "$OUTPUT_DIR"
