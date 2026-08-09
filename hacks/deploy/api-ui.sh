#!/usr/bin/env bash
# 一键构建并启动：API + 内嵌 UI（单服务单端口，同源无 CORS）
#
# 用法:
#   ./api-ui.sh                 # 默认端口 8000
#   PORT=9000 ./api-ui.sh
set -euo pipefail
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DIR"

PORT="${PORT:-8000}"

echo "▶ 构建前端 ..."
(cd ui && npm run build)

echo "▶ 同步前端产物到 cmd/api-server/web/ ..."
rm -rf cmd/api-server/web
mkdir -p cmd/api-server/web
cp -r ui/dist/. cmd/api-server/web/

echo "▶ 启动服务 (端口 $PORT) ..."
echo ""
echo "============================================"
echo "  UI + API: http://127.0.0.1:${PORT}"
echo "  Ctrl+C 停止"
echo "============================================"
echo ""

go run ./cmd/api-server --port "$PORT"
