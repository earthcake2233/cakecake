#!/usr/bin/env bash
# Diagnose why the CakeCake monitoring stack is not visible.
# Read-only; never prints secret values.
set -uo pipefail
cd "$(dirname "$0")/.."

echo "=== 1) 监控容器状态 ==="
docker compose -f docker-compose.monitoring.yml ps prometheus alertmanager grafana 2>&1 || true
echo "(后端走 air 时不会出现在这里；检查宿主机 8080 即可)"

echo
echo "=== 2) 端口探测 ==="
for p in 3000 9090 9093; do
  code=$(curl -s -o /dev/null -m 2 -w '%{http_code}' "http://127.0.0.1:$p/" || true)
  echo "http://127.0.0.1:$p -> ${code:-closed}"
done

echo
echo "=== 3) 后端 /metrics 是否可抓 ==="
token=$(sed -n 's/^METRICS_TOKEN=//p' .env | head -1)
if [ -z "$token" ]; then
  echo "METRICS_TOKEN 为空 -> 先运行: python3 scripts/init_env.py --refresh"
else
  host_out=$(curl -s -m 3 -H "Authorization: Bearer $token" "http://127.0.0.1:8080/metrics" || true)
  if echo "$host_out" | grep -q "cakecake_llm_requests_total"; then
    echo "OK：宿主机 8080 /metrics 可抓，指标已暴露"
  else
    echo "FAIL：宿主机 8080 /metrics 抓不到（后端没起、端口不同，或 token 不匹配）"
  fi

  if docker compose -f docker-compose.monitoring.yml ps --services 2>/dev/null | grep -qx backend; then
    container_out=$(docker compose exec -T backend sh -c 'wget -qO- --header="Authorization: Bearer $METRICS_TOKEN" http://127.0.0.1:8080/metrics' 2>/dev/null || true)
    if echo "$container_out" | grep -q "cakecake_llm_requests_total"; then
      echo "OK：compose 内 backend:8080/metrics 可抓"
    else
      echo "FAIL：compose 内抓不到 -> 检查 .env 的 METRICS_TOKEN 是否同时注入 backend 与 prometheus"
    fi
  fi
fi

echo
echo "=== 3.5) 云服务器 /metrics（https://chengzisoft.top/metrics）==="
cloud_token=$(sed -n 's/^METRICS_TOKEN_CLOUD=//p' .env | head -1)
[ -z "$cloud_token" ] && cloud_token="$token"
if [ -n "$cloud_token" ]; then
  noauth=$(curl -s -o /dev/null -m 8 -w '%{http_code}' https://chengzisoft.top/metrics || true)
  code=$(curl -s -o /dev/null -m 8 -w '%{http_code}' -H "Authorization: Bearer $cloud_token" https://chengzisoft.top/metrics || true)
  echo "https://chengzisoft.top/metrics no-auth -> ${noauth:-closed}, with-token -> ${code:-closed}"
  if [ "$noauth" = "200" ]; then
    echo "WARN：云上 /metrics 未启用 Bearer 鉴权（服务器 METRICS_TOKEN 未配置或后端未重启）"
  elif [ "$code" = "200" ]; then
    echo "OK：云上 /metrics 可抓"
  else
    echo "FAIL：云上 /metrics 不可抓（401=token 不一致；404=nginx 未加 /metrics 反代；000=网络/证书）"
  fi
else
  echo "METRICS_TOKEN 为空，跳过云上检查"
fi

echo
echo "=== 4) Prometheus 目标状态 ==="
if docker compose -f docker-compose.monitoring.yml ps --services 2>/dev/null | grep -qx prometheus; then
  curl -s -m 3 http://127.0.0.1:9090/api/v1/targets | python3 -c "import json,sys; d=json.load(sys.stdin); [print(t['labels'].get('job'), t['labels'].get('instance'), t['health']) for t in d['data']['activeTargets']]" || true
else
  echo "prometheus 容器未运行 -> 先执行: docker compose -f docker-compose.monitoring.yml up -d"
fi

echo
echo "=== 5) 看板入口 ==="
echo "Grafana     : http://127.0.0.1:3000  用户 admin，密码在 .env 的 GRAFANA_ADMIN_PASSWORD"
echo "Prometheus  : http://127.0.0.1:9090  Targets 页确认 cakecake-backend UP"
echo "Alertmanager: http://127.0.0.1:9093"
