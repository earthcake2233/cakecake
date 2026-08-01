#!/usr/bin/env bash
# Minibili 一键部署脚本（WSL / Linux）
#
# 流程：本地测试（无 bug 门禁）→ 构建 → 二次确认 → 上传云服务器 → 重启 → 健康检查
#
# 用法：
#   ./scripts/deploy.sh                 # 完整流程（测试 + 构建 + 确认 + 部署）
#   ./scripts/deploy.sh --skip-tests    # 跳过测试（仅构建 + 确认 + 部署）
#   ./scripts/deploy.sh --dry-run       # 只做测试 + 构建，演练本地部分（不上传）
#   ./scripts/deploy.sh --deploy-only   # 用上次 --dry-run 的产物直接部署（仍会二次确认）
#   ./scripts/deploy.sh --yes           # 跳过交互确认（自动化场景慎用）
#
# 配置：复制 scripts/deploy.local.env.example 为 scripts/deploy.local.env 并填写；
#       或直接 export DEPLOY_HOST / DEPLOY_USER / DEPLOY_KEY / DEPLOY_PORT。

set -euo pipefail

SKIP_TESTS=0
DRY_RUN=0
DEPLOY_ONLY=0
AUTO_YES=0
for arg in "$@"; do
  case "$arg" in
    --skip-tests) SKIP_TESTS=1 ;;
    --dry-run)    DRY_RUN=1 ;;
    --deploy-only) DEPLOY_ONLY=1 ;;
    --yes)        AUTO_YES=1 ;;
    *)
      echo "未知参数: $arg（支持 --skip-tests / --dry-run / --deploy-only / --yes）" >&2
      exit 2
      ;;
  esac
done

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

# ---- 安全加载 KEY=VALUE 配置文件（.env 含括号/特殊字符，不能直接 source）----
load_kv_file() {
  local file="$1" allow="${2:-}"
  [[ -f "$file" ]] || return 0
  local key value
  while IFS='=' read -r key value; do
    [[ -z "$key" ]] && continue
    [[ "$key" == \#* ]] && continue
    if [[ -z "$allow" ]] || [[ " $allow " == *" $key "* ]]; then
      export "$key=$value"
    fi
  done < "$file"
}

# 应用 .env 中测试需要的变量（避免污染其他默认值测试）
load_kv_file "$ROOT/.env" "RABBITMQ_URL"

# 部署配置
load_kv_file "$ROOT/scripts/deploy.local.env"

DEPLOY_HOST="${DEPLOY_HOST:-}"
DEPLOY_USER="${DEPLOY_USER:-}"
DEPLOY_KEY="${DEPLOY_KEY:-}"
DEPLOY_PORT="${DEPLOY_PORT:-22}"
REMOTE_DIR="${REMOTE_DIR:-/opt/minibili}"
SERVICE_NAME="${SERVICE_NAME:-minibili}"

if [[ "$DRY_RUN" -eq 0 ]]; then
  if [[ -z "$DEPLOY_HOST" || -z "$DEPLOY_USER" || -z "$DEPLOY_KEY" ]]; then
    echo "缺少部署配置：请复制 scripts/deploy.local.env.example 为 scripts/deploy.local.env 并填写 DEPLOY_HOST / DEPLOY_USER / DEPLOY_KEY。"
    echo "（或先用 --dry-run 只演练本地测试与构建部分）" >&2
    exit 2
  fi
  [[ -f "$DEPLOY_KEY" ]] || { echo "SSH 私钥不存在: $DEPLOY_KEY" >&2; exit 2; }
fi

WORK="/tmp/minibili-deploy-latest"

if [[ "$DEPLOY_ONLY" -eq 1 ]]; then
  if [[ ! -f "$WORK/mini-bili" || ! -f "$WORK/www.tar.gz" ]]; then
    echo "找不到上次构建产物（$WORK），请先运行 ./scripts/deploy.sh --dry-run 生成。" >&2
    exit 2
  fi
else
  rm -rf "$WORK"
  mkdir -p "$WORK"
fi

echo "==> 项目目录: $ROOT"
echo "==> 部署目标: ${DEPLOY_USER:-?}@${DEPLOY_HOST:-?}:${DEPLOY_PORT:-22} -> $REMOTE_DIR"

# ---- [1/5] 测试门禁 ----
if [[ "$DEPLOY_ONLY" -eq 1 ]]; then
  echo "==> [1/5] 跳过测试（--deploy-only，复用上次构建）"
elif [[ "$SKIP_TESTS" -eq 1 ]]; then
  echo "==> [1/5] 跳过测试（--skip-tests）"
else
  echo "==> [1/5] 运行测试（无 bug 门禁）..."

  unformatted="$(git ls-files -z '*.go' | xargs -0 gofmt -l 2>/dev/null || true)"
  if [[ -n "$unformatted" ]]; then
    echo "!! gofmt 未通过，以下文件需要 gofmt -w 格式化:" >&2
    echo "$unformatted" >&2
    exit 1
  fi
  echo "    gofmt: 通过"

  echo "    后端测试 (go test -tags=integration ./...)..."
  go test -tags=integration -count=1 -timeout 300s ./...

  echo "    前端测试 (npm test)..."
  (cd cakecake-vue/bilibili-vue && npm test)
fi

# ---- [2/5] 构建 ----
echo "==> [2/5] 构建产物..."
if [[ "$DEPLOY_ONLY" -eq 1 ]]; then
  echo "    复用已有产物: $WORK/mini-bili + $WORK/www.tar.gz"
else
  echo "    后端 (linux/amd64)..."
  go build -buildvcs=false -ldflags="-s -w" -o "$WORK/mini-bili" ./cmd/mini-bili

  echo "    前端 dist..."
  if [[ ! -f cakecake-vue/bilibili-vue/.env.production ]]; then
    cp cakecake-vue/bilibili-vue/.env.production.example cakecake-vue/bilibili-vue/.env.production
    echo "    提示: 已从 .env.production.example 生成 .env.production"
  fi
  (cd cakecake-vue/bilibili-vue && npm run build)
  tar -czf "$WORK/www.tar.gz" -C cakecake-vue/bilibili-vue/dist .
  dirty="no"
  git status --porcelain | grep -q . && dirty="yes"
  printf 'commit=%s branch=%s dirty=%s built=%s\n' \
    "$(git rev-parse --short HEAD 2>/dev/null || echo '?')" \
    "$(git branch --show-current 2>/dev/null || echo '?')" \
    "$dirty" \
    "$(date '+%Y-%m-%d %H:%M:%S')" > "$WORK/deploy-info.txt"
fi

BIN_SIZE="$(du -h "$WORK/mini-bili" | cut -f1)"
DIST_SIZE="$(du -h "$WORK/www.tar.gz" | cut -f1)"

# ---- [3/5] 二次确认 ----
echo "==> [3/5] 部署确认"
echo "    后端二进制: $BIN_SIZE ($WORK/mini-bili)"
echo "    前端资源:   $DIST_SIZE (www.tar.gz)"
if [[ -f "$WORK/deploy-info.txt" ]]; then
  echo "    构建信息:   $(cat "$WORK/deploy-info.txt")"
else
  echo "    当前提交:   $(git rev-parse --short HEAD 2>/dev/null || echo '?') ($(git branch --show-current 2>/dev/null || echo '?'))"
  git status --porcelain | grep -q . && echo "    注意: 工作区有未提交改动，将部署当前工作区内容"
fi

if [[ "$DRY_RUN" -eq 1 ]]; then
  echo "    [dry-run] 演练模式：不上传。"
  echo "==> 演练完成 ✓（测试与构建均通过，产物在 $WORK）"
  exit 0
fi

if [[ "$AUTO_YES" -ne 1 ]]; then
  read -r -p "    确认以上测试通过、无 bug？输入 yes 开始部署: " answer
  [[ "$answer" == "yes" ]] || { echo "    已取消。"; exit 1; }
fi

# ---- [4/5] 上传并重启 ----
echo "==> [4/5] 上传到 $DEPLOY_HOST 并重启服务..."
SSH=(ssh -i "$DEPLOY_KEY" -p "$DEPLOY_PORT" -o BatchMode=yes -o StrictHostKeyChecking=accept-new -o ConnectTimeout=15)
SCP=(scp -i "$DEPLOY_KEY" -P "$DEPLOY_PORT" -o BatchMode=yes -o StrictHostKeyChecking=accept-new -o ConnectTimeout=15)
DEST="${DEPLOY_USER}@${DEPLOY_HOST}"

"${SCP[@]}" "$WORK/mini-bili" "$WORK/www.tar.gz" "${DEST}:/tmp/"
"${SSH[@]}" "$DEST" "set -e
install -m 755 /tmp/mini-bili $REMOTE_DIR/bin/mini-bili
rm -rf $REMOTE_DIR/www/*
mkdir -p $REMOTE_DIR/www
tar xzf /tmp/www.tar.gz -C $REMOTE_DIR/www
rm -f /tmp/mini-bili /tmp/www.tar.gz
systemctl restart $SERVICE_NAME
ok=0
for _ in \$(seq 1 15); do
  sleep 2
  if curl -fsS --max-time 3 http://127.0.0.1:8080/api/v1/health >/dev/null 2>&1; then
    ok=1
    break
  fi
done
if [ \"\$ok\" != \"1\" ]; then
  echo 'health check failed, rolling back...' >&2
  systemctl stop $SERVICE_NAME || true
  sleep 1
  if [ -f $REMOTE_DIR/bin/mini-bili.prev ]; then
    install -m 755 $REMOTE_DIR/bin/mini-bili.prev $REMOTE_DIR/bin/mini-bili
  fi
  systemctl start $SERVICE_NAME || true
  exit 1
fi
nginx -t && nginx -s reload
curl -fsS http://127.0.0.1:8080/api/v1/health | head -c 200
echo
echo 'Minibili deploy OK'"

# ---- [5/5] 公网健康检查 ----
echo "==> [5/5] 健康检查"
sleep 2
if curl -fsS --max-time 15 "https://${DEPLOY_HOST}/api/v1/health" >/dev/null 2>&1; then
  echo "    公网健康检查通过: https://${DEPLOY_HOST}/api/v1/health"
else
  echo "    警告: 公网健康检查未通过（可能该路径未对外暴露或域名有缓存），请自行用浏览器确认。" >&2
fi

rm -rf "$WORK"
echo "==> 部署完成 ✓"
