<p align="center">
  <strong><img src="https://img.shields.io/badge/🇨🇳中文-00a1d6?style=flat-square" alt="中文"></strong>
  <a href="monitoring_EN.md">
    <img src="https://img.shields.io/badge/🇬🇧English-999999?style=flat-square" alt="English">
  </a>
</p>

# 可观测性：Prometheus + Alertmanager + Grafana

## 一、架构与部署方式

本项目采用「**监控留本地，云服务器只出数据**」的部署方式：

- **本地开发机**跑 Prometheus + Alertmanager + Grafana（Docker，仅绑定 `127.0.0.1`）；
- 后端用 `air`（开发）/ systemd 二进制（生产）跑在宿主机，`/metrics` 由后端自身暴露；
- **云服务器不装 Docker、不跑监控组件**，只通过 Nginx 把 `/metrics` 反代出去，
  Bearer 鉴权由后端 `METRICS_TOKEN` 强制，内存零开销；
- 本地 Prometheus 同时抓两个 job：本地后端（`host.docker.internal:8080`）和
  云服务器（`https://chengzisoft.top/metrics`），一个看板看本地 + 生产。

> 仓库根的 `docker-compose.yml` 也内置了监控三件套，那是「一条命令本地全栈」的演示模式；
> 日常 air / npm 开发与生产部署请使用 `docker-compose.monitoring.yml`。

## 二、组件与指标

| 组件 | 容器 | 端口（默认，仅绑定 127.0.0.1） | 职责 |
| --- | --- | --- | --- |
| Prometheus | `cakecake-prometheus` | 9090 | 抓取 `/metrics`，存储 15 天时序数据，评估告警规则 |
| Alertmanager | `cakecake-alertmanager` | 9093 | 告警去重、分组、抑制、路由到 webhook |
| Grafana | `cakecake-grafana` | 3000 | 可视化看板（数据源与看板自动 provisioning） |

后端暴露的指标（`cakecake_*`）：

- `cakecake_llm_requests_total{status}` — LLM 请求成功/失败次数（错误率来源）
- `cakecake_llm_first_token_seconds` — 流式首 token 延迟 histogram
- `cakecake_llm_tokens_total{type}` — prompt / completion token 用量
- `cakecake_llm_cost_usd_total{user,date}` — 按用户/日估算成本
- `cakecake_agent_tool_calls_total{tool,status}` — 工具调用次数与失败率
- `cakecake_agent_tool_call_seconds{tool}` — 工具调用耗时
- `cakecake_agent_controls_total{type}` — 暂停 / 继续 / 重新生成次数

> 注意：`cakecake_*` 是进程内计数器，后端重启会清零；需要长期成本账目时另做落库或 OTel。

> 成本口径：`cakecake_llm_cost_usd_total` 按 DeepSeek V4-Flash 官方价估算（2026-08：
> 输入缓存命中 0.02 元 / 缓存未命中 1 元 / 输出 2 元，每百万 tokens），按约 7.14
> 汇率折算为 USD；未上报缓存拆分时按全部未命中保守计费。仅供估算，不是官方账单。

## 三、本地快速开始

```bash
# 1) 初始化/刷新本地 .env（保留已有值，只补缺失密钥；同时生成 .secrets/）
python3 scripts/init_env.py --refresh

# 2) 启动监控容器（只跑监控，不依赖 MySQL/Redis/后端 compose）
docker compose -f docker-compose.monitoring.yml up -d

# 3) 后端用 air、前端用 npm 照常启动（后端必须监听 8080）
```

访问入口：

- Grafana：http://127.0.0.1:3000（用户 `admin`，密码在 `.env` 的 `GRAFANA_ADMIN_PASSWORD`）
- Prometheus：http://127.0.0.1:9090（Targets / Alerts / Graph）
- Alertmanager：http://127.0.0.1:9093
- 裸指标：`curl -H "Authorization: Bearer $METRICS_TOKEN" http://127.0.0.1:8080/metrics`

Grafana 启动后自动出现数据源 `Prometheus` 和看板 **CakeCake AI Gateway**，无需手工导入。

## 四、云服务器接入（生产）

生产部署走 GitHub Actions（`deploy.yml`），后端是 systemd 二进制，**不跑 Docker**。
云上只需要开一条受保护的 `/metrics` 数据通道：

1. **Nginx 反代**：把 `deploy/nginx-minibili.conf` 里的 `location = /metrics` 合并进服务器
   `/etc/nginx/conf.d/minibili.conf`，然后：

   ```bash
   sudo nginx -t && sudo systemctl reload nginx
   ```

   该 location 只反代到 `127.0.0.1:8080` 并透传 `Authorization`；后端用
   `METRICS_TOKEN` 强制 Bearer 鉴权。**8080 不要对外开端口。**

2. **服务器 `.env`**：`/opt/minibili/.env` 里必须有 `METRICS_TOKEN`。
   GitHub Actions 每次部署会自动把仓库最新的 `deploy/env.production.example`
   合并进服务器 `.env`（`scripts/merge_env.sh`）：
   - 已有键值（含密钥/token）**一律不覆盖**；
   - 只追加模板新增键；写入前备份为 `.env.prev`；
   - 新密钥第一次以空值补入，部署日志只提示键名，不会打印值。

3. **本地抓取云上数据**：本地 Prometheus 已内置 job `cakecake-backend-cloud`
   （`https://chengzisoft.top/metrics`）。默认使用与本地相同的
   `METRICS_TOKEN`；如需独立 token，在本地 `.env` 填 `METRICS_TOKEN_CLOUD=` 后重跑
   `python3 scripts/init_env.py --refresh`，会生成 `.secrets/metrics_token_cloud`。

4. **本地验证**：

   ```bash
   token=$(grep '^METRICS_TOKEN=' .env | cut -d= -f2)
   curl -s -H "Authorization: Bearer $token" https://chengzisoft.top/metrics | grep cakecake_ | head
   ```

   通过后 15 秒内 Prometheus 的 `cakecake-backend-cloud` 变 UP，看板用
   `instance=host:8080` / `instance=cloud` 区分本地与生产。

## 五、安全基线

1. `METRICS_TOKEN` 是 `/metrics` 的 Bearer 鉴权令牌；`init_env.py` 生成强随机值，
   只写入 gitignored 的 `.env` / `.secrets/`。未设置时端点保持开放（仅本地开发）。
2. Prometheus / Grafana / Alertmanager 端口只绑定 `127.0.0.1`。云上需要看板时用
   SSH 隧道：`ssh -L 3000:127.0.0.1:3000 root@<服务器>`，不要裸开 3000/9090/9093。
3. Grafana 关闭匿名注册（`GF_USERS_ALLOW_SIGN_UP=false`），首次登录后更换密码。
4. `cakecake_llm_cost_usd_total` 携带用户 ID 标签，属于隐私数据；`/metrics` 必须走
   HTTPS + Bearer，不能裸奔公网。
5. 告警出口默认是占位 webhook（`http://127.0.0.1:9999/webhook`），接入真实值班通道前
   修改 `deploy/monitoring/alertmanager.yml` 并重建 Alertmanager。

## 六、告警规则（deploy/monitoring/rules/cakecake-ai.yml）

| 告警 | 触发条件 | 严重级别 |
| --- | --- | --- |
| `CakecakeBackendDown` | `up == 0` 持续 2 分钟（本地或云上 job） | critical |
| `CakecakeHighLLMErrorRate` | LLM 错误率 > 10% 持续 10 分钟 | warning |
| `CakecakeHighFirstTokenLatency` | P95 首 token 延迟 > 5s 持续 10 分钟 | warning |
| `CakecakeHighToolFailureRate` | 单工具失败率 > 20% 持续 10 分钟 | warning |
| `CakecakeSlowToolCalls` | 单工具 P95 耗时 > 10s 持续 10 分钟 | warning |
| `CakecakeDailyCostSpike` | 1 小时估算成本增量 > $10 | warning |

修改规则后热加载：

```bash
docker compose -f docker-compose.monitoring.yml exec prometheus kill -HUP 1
```

## 七、看板说明

**CakeCake AI Gateway** 看板包含：

- LLM 请求速率与错误率；
- 首 token 延迟 p50/p95/p99；
- token 吞吐（prompt / completion 分开）；
- 工具调用速率、失败率、P95 耗时；
- 24h 暂停/继续/重新生成计数；
- 按用户 24h 成本 Top10 与成本速率；
- Go 运行时（堆内存、goroutine）。

## 八、排查指南

1. **看不到监控**：先确认容器起来了：

   ```bash
   docker compose -f docker-compose.monitoring.yml ps
   bash scripts/monitoring-check.sh
   ```

2. **Grafana 登录不进去**：用户 `admin`，密码在 `.env` 的 `GRAFANA_ADMIN_PASSWORD`
   （`init_env.py` 生成，不是 `admin`）。

3. **本地 target DOWN**：`http://127.0.0.1:9090/targets` 看 `cakecake-backend`；
   后端必须监听 8080（air 已启动但端口不同时改 `prometheus-host.yml` 的 targets）。

4. **云上 target DOWN**：错误信息区分原因——`401`=token 不一致；
   `404`=Nginx 没配 `/metrics` 反代；`000`=网络/证书。用
   `bash scripts/monitoring-check.sh` 的「云服务器 /metrics」一节直接验证。

5. **面板全是 No data**：`cakecake_*` 是「零样本不输出」，先让 AI 回一条消息
   （或调一次工具），等 15~30 秒再看。
