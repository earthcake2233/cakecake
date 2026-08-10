<p align="center">
  <a href="monitoring.md">
    <img src="https://img.shields.io/badge/🇨🇳中文-999999?style=flat-square" alt="中文">
  </a>
  <strong><img src="https://img.shields.io/badge/🇬🇧English-00a1d6?style=flat-square" alt="English"></strong>
</p>

# Observability: Prometheus + Alertmanager + Grafana

## 1. Architecture & Deployment Model

This project uses a "**monitoring stays local, the cloud server only exports data**"
deployment model:

- **Local dev machine** runs Prometheus + Alertmanager + Grafana (Docker, bound to
  `127.0.0.1` only);
- The backend runs on the host (`air` for development / systemd binary in
  production) and exposes `/metrics` itself;
- **The cloud server does not run Docker or any monitoring component.** Nginx
  only reverse-proxies `/metrics`, and Bearer auth is enforced by the backend
  `METRICS_TOKEN` — zero extra memory on the server;
- Local Prometheus scrapes two jobs: the local backend
  (`host.docker.internal:8080`) and the cloud server
  (`https://chengzisoft.top/metrics`), so one dashboard covers local + production.

> The root `docker-compose.yml` also bundles the monitoring trio for the
> "one-command local full stack" demo. For daily `air` / `npm` development and
> production, use `docker-compose.monitoring.yml`.

## 2. Components & Metrics

| Component | Container | Port (default, bound to 127.0.0.1) | Role |
| --- | --- | --- | --- |
| Prometheus | `cakecake-prometheus` | 9090 | Scrapes `/metrics`, stores 15 days of TSDB data, evaluates alert rules |
| Alertmanager | `cakecake-alertmanager` | 9093 | Alert dedup, grouping, inhibition, routing to webhook |
| Grafana | `cakecake-grafana` | 3000 | Dashboards (datasource & dashboard auto-provisioned) |

Backend metrics (`cakecake_*`):

- `cakecake_llm_requests_total{status}` — LLM request success/failure count (error-rate source)
- `cakecake_llm_first_token_seconds` — streaming first-token latency histogram
- `cakecake_llm_tokens_total{type}` — prompt / completion token usage
- `cakecake_llm_cost_usd_total{user,date}` — estimated cost per user/day
- `cakecake_agent_tool_calls_total{tool,status}` — tool call count & failure rate
- `cakecake_agent_tool_call_seconds{tool}` — tool call duration
- `cakecake_agent_controls_total{type}` — pause / continue / regenerate counters

> Note: `cakecake_*` are in-process counters and reset when the backend
> restarts; use a persistent ledger or OTel when long-term accounting is needed.

## 3. Local Quick Start

```bash
# 1) Initialize/refresh the local .env (keeps existing values, fills missing
#    secrets, and writes .secrets/ files)
python3 scripts/init_env.py --refresh

# 2) Start the monitoring containers (monitoring only; no MySQL/Redis/backend deps)
docker compose -f docker-compose.monitoring.yml up -d

# 3) Start the backend with air and the frontend with npm as usual
#    (the backend must listen on 8080)
```

Entry points:

- Grafana: http://127.0.0.1:3000 (user `admin`, password from `GRAFANA_ADMIN_PASSWORD` in `.env`)
- Prometheus: http://127.0.0.1:9090 (Targets / Alerts / Graph)
- Alertmanager: http://127.0.0.1:9093
- Raw metrics: `curl -H "Authorization: Bearer $METRICS_TOKEN" http://127.0.0.1:8080/metrics`

After Grafana starts, the `Prometheus` datasource and the **CakeCake AI Gateway**
dashboard appear automatically — no manual import.

## 4. Cloud Server Integration (Production)

Production deploys via GitHub Actions (`deploy.yml`): the backend is a systemd
binary and **does not run Docker**. The cloud side only needs one protected
`/metrics` data path:

1. **Nginx reverse proxy**: merge the `location = /metrics` block from
   `deploy/nginx-minibili.conf` into `/etc/nginx/conf.d/minibili.conf`, then:

   ```bash
   sudo nginx -t && sudo systemctl reload nginx
   ```

   The location proxies only to `127.0.0.1:8080` and forwards `Authorization`;
   the backend enforces Bearer auth via `METRICS_TOKEN`. **Never expose 8080
   publicly.**

2. **Server `.env`**: `/opt/minibili/.env` must contain `METRICS_TOKEN`. Every
   GitHub Actions deploy auto-merges the latest `deploy/env.production.example`
   into the server `.env` (`scripts/merge_env.sh`):
   - Existing keys (including secrets/tokens) are **never overwritten**;
   - Only new template keys are appended; the previous file is backed up as
     `.env.prev`;
   - New secret keys are added empty on first deploy; logs print key names only,
     never values.

3. **Scrape the cloud from local**: local Prometheus already has the
   `cakecake-backend-cloud` job (`https://chengzisoft.top/metrics`). By default it
   uses the same `METRICS_TOKEN` as local. For a separate token, set
   `METRICS_TOKEN_CLOUD=` in the local `.env`, re-run
   `python3 scripts/init_env.py --refresh`, and
   `.secrets/metrics_token_cloud` is generated.

4. **Local verification**:

   ```bash
   token=$(grep '^METRICS_TOKEN=' .env | cut -d= -f2)
   curl -s -H "Authorization: Bearer $token" https://chengzisoft.top/metrics | grep cakecake_ | head
   ```

   Once it returns 200, `cakecake-backend-cloud` turns UP within 15 seconds.
   The dashboard distinguishes `instance=host:8080` (local) from
   `instance=cloud` (production).

## 5. Security Baseline

1. `METRICS_TOKEN` guards `/metrics` with Bearer auth; `init_env.py` generates
   strong random values written only to gitignored `.env` / `.secrets/`. When
   unset, the endpoint stays open (local development only).
2. Prometheus / Grafana / Alertmanager ports bind to `127.0.0.1` only. To view
   the dashboard from the cloud, use an SSH tunnel
   (`ssh -L 3000:127.0.0.1:3000 root@<server>`) — never open 3000/9090/9093.
3. Grafana disables anonymous sign-up (`GF_USERS_ALLOW_SIGN_UP=false`); change
   the password after first login.
4. `cakecake_llm_cost_usd_total` carries user IDs and is privacy-sensitive;
   `/metrics` must stay behind HTTPS + Bearer auth.
5. The alert webhook defaults to a harmless placeholder
   (`http://127.0.0.1:9999/webhook`); edit
   `deploy/monitoring/alertmanager.yml` and recreate Alertmanager before wiring
   a real on-call channel.

## 6. Alert Rules (deploy/monitoring/rules/cakecake-ai.yml)

| Alert | Condition | Severity |
| --- | --- | --- |
| `CakecakeBackendDown` | `up == 0` for 2 minutes (local or cloud job) | critical |
| `CakecakeHighLLMErrorRate` | LLM error rate > 10% for 10 minutes | warning |
| `CakecakeHighFirstTokenLatency` | P95 first-token latency > 5s for 10 minutes | warning |
| `CakecakeHighToolFailureRate` | Per-tool failure rate > 20% for 10 minutes | warning |
| `CakecakeSlowToolCalls` | Per-tool P95 duration > 10s for 10 minutes | warning |
| `CakecakeDailyCostSpike` | Estimated cost increase > $10 in 1 hour | warning |

Reload rules after editing:

```bash
docker compose -f docker-compose.monitoring.yml exec prometheus kill -HUP 1
```

## 7. Dashboard Overview

The **CakeCake AI Gateway** dashboard includes:

- LLM request rate & error rate;
- First-token latency p50/p95/p99;
- Token throughput (prompt / completion separately);
- Tool call rate, failure rate, P95 duration;
- 24h pause/continue/regenerate counters;
- Per-user 24h cost Top10 & cost rate;
- Go runtime (heap memory, goroutines).

## 8. Troubleshooting

1. **Monitoring not visible**: confirm the containers are up:

   ```bash
   docker compose -f docker-compose.monitoring.yml ps
   bash scripts/monitoring-check.sh
   ```

2. **Cannot log into Grafana**: user `admin`, password from
   `GRAFANA_ADMIN_PASSWORD` in `.env` (generated by `init_env.py`, not `admin`).

3. **Local target DOWN**: open `http://127.0.0.1:9090/targets` and check
   `cakecake-backend`; the backend must listen on 8080 (change the targets in
   `prometheus-host.yml` if air uses a different port).

4. **Cloud target DOWN**: the error explains the cause — `401` = token mismatch;
   `404` = Nginx `/metrics` proxy missing; `000` = network/TLS. Use the
   "Cloud /metrics" section of `bash scripts/monitoring-check.sh` to verify.

5. **Dashboard panels show No data**: `cakecake_*` counters emit nothing until
   observed. Ask the AI a question (or trigger a tool call), then wait 15–30 s.
