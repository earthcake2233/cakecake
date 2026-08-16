#!/usr/bin/env bash
# Reproduce every coverage / load-test number claimed in the resume.
#
# Usage (run from repo root):
#   ./scripts/loadtest/reproduce.sh coverage   # 单元 + 含集成标签覆盖率（任意机器）
#   ./scripts/loadtest/reproduce.sh bench      # 热榜前后对比 + WS 强压（在 ECS 上运行）
#   ./scripts/loadtest/reproduce.sh all        # 两者
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
TS="$(date +%Y%m%d%H%M%S)"
OUT_DIR="${REPRO_OUT_DIR:-$ROOT/docs/loadtest-results/repro/$TS}"
ENV_FILE="/opt/minibili/.env"
LT="/tmp/loadtest"
BASE="http://127.0.0.1:8080"

coverage() {
  echo "== unit coverage (go test ./... -cover) =="
  go test ./... -count=1 -coverprofile=/tmp/cov_unit_repro.out -covermode=atomic | tail -1
  go tool cover -func=/tmp/cov_unit_repro.out | tail -1
  echo
  echo "== total coverage (go test -tags=integration ./... -cover) =="
  go test -tags=integration ./... -count=1 -coverprofile=/tmp/cov_total_repro.out -covermode=atomic | tail -1
  go tool cover -func=/tmp/cov_total_repro.out | tail -1
}

admin_token() {
  local u p login
  u="$(grep '^ADMIN_SEED_USERNAME=' "$ENV_FILE" | cut -d= -f2-)"
  p="$(grep '^ADMIN_SEED_PASSWORD=' "$ENV_FILE" | cut -d= -f2-)"
  login="$(curl -sS -m 10 -X POST "$BASE/api/v1/admin/auth/login" \
    -H 'Content-Type: application/json' \
    -d "{\"username\":\"$u\",\"password\":\"$p\"}")"
  printf '%s' "$login" | grep -o '"access_token":"[^"]*"' | head -1 | cut -d'"' -f4
}

set_cfg() { # key value token
  curl -sS -m 10 -X PUT "$BASE/api/v1/admin/system-configs" \
    -H "Authorization: Bearer $3" -H 'Content-Type: application/json' \
    -d "{\"configs\":{\"$1\":\"$2\"}}" > /dev/null
}

bench() {
  [ -f "$ENV_FILE" ] || { echo "bench 需要在 ECS 上运行（依赖 /opt/minibili/.env）"; exit 1; }
  [ -x "$LT" ] || {
    echo "缺少 $LT，先构建："
    echo "  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $LT ./cmd/loadtest"
    exit 1
  }
  mkdir -p "$OUT_DIR"
  local token
  token="$(admin_token)"
  [ -n "$token" ] || { echo "admin 登录失败"; exit 1; }

  set_cfg rate_limit_enabled false "$token"
  echo "== 基线（缓存关）c40 25s =="
  set_cfg hotsearch_cache_enabled false "$token"
  sleep 1
  "$LT" http -url "$BASE/api/v1/hot-search" -c 40 -d 25s -out "$OUT_DIR/baseline_c40.json"

  echo "== 优化后（缓存开）c40 25s =="
  set_cfg hotsearch_cache_enabled true "$token"
  sleep 1
  "$LT" http -url "$BASE/api/v1/hot-search" -c 40 -d 25s -out "$OUT_DIR/after_c40.json"

  echo "== WS 100 连接 + 真实发送 =="
  local js
  js="$(grep '^JWT_SECRET=' "$ENV_FILE" | cut -d= -f2-)"
  export JWT_SECRET=$js
  "$LT" ws -url "ws://127.0.0.1:8080/api/v1/ws/danmaku" \
    -video 6 -clients 100 -sender-users 30 -send-interval 500ms -d 25s \
    -out "$OUT_DIR/ws100.json"

  set_cfg rate_limit_enabled true "$token"
  echo "health: $(curl -sS -m 5 "$BASE/api/v1/health" | head -c 60)"
  echo "结果目录: $OUT_DIR"
}

cmd="${1:-all}"
case "$cmd" in
  coverage) coverage ;;
  bench) bench ;;
  all) coverage; bench ;;
  *) echo "usage: $0 [coverage|bench|all]"; exit 2 ;;
esac
