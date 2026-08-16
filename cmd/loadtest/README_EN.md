# loadtest

A lightweight, reproducible load generator for the cakecake backend (Go,
zero external dependencies; Windows / Linux / macOS).

## Build

```bash
go build -o loadtest ./cmd/loadtest
```

Static build for CentOS 7:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o loadtest ./cmd/loadtest
```

## HTTP load test

```bash
loadtest http -url http://127.0.0.1:8080/api/v1/hot-search -c 50 -d 30s -out hot50.json
loadtest http -url http://127.0.0.1:8080/api/v1/videos -c 20 -d 30s -qps 500 -out vids500.json
```

Reports QPS, P50/P90/P99, error count and status-code distribution.

## WebSocket danmaku load test

```bash
JWT_SECRET=<secret> loadtest ws -url ws://127.0.0.1:8080/api/v1/ws/danmaku \
  -video 6 -clients 100 -sender-users 30 -send-interval 500ms -d 25s -out ws100.json
```

Reports connection success/failure, message throughput, read-error breakdown
(unexpected / timeout / clean end) and send success/failure counts.
`JWT_SECRET` is only used to mint danmaku-sender tokens; it is never printed.

## Notes

- When the generator runs on the same host as the server, numbers are a
  conservative lower bound; prefer `127.0.0.1` or same-VPC internal network,
  because public bandwidth becomes the bottleneck before CPU.
- Disable rate limiting before measuring the QPS ceiling (this project exposes
  it as the runtime config `rate_limit_enabled`, toggled via the admin API).
