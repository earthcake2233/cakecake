# loadtest

轻量、可复现的 cakecake 后端压测工具（Go 编写，零外部依赖，Windows / Linux / macOS 通用）。

## 构建

```bash
go build -o loadtest ./cmd/loadtest
```

线上 CentOS 7 需静态编译：

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o loadtest ./cmd/loadtest
```

## HTTP 压测

```bash
loadtest http -url http://127.0.0.1:8080/api/v1/hot-search -c 50 -d 30s -out hot50.json
loadtest http -url http://127.0.0.1:8080/api/v1/videos -c 20 -d 30s -qps 500 -out vids500.json
```

输出 QPS、P50/P90/P99、错误数、状态码分布。

## WebSocket 弹幕压测

```bash
JWT_SECRET=<密钥> loadtest ws -url ws://127.0.0.1:8080/api/v1/ws/danmaku \
  -video 6 -clients 100 -sender-users 30 -send-interval 500ms -d 25s -out ws100.json
```

输出连接成功/失败数、消息吞吐、读错误分类（意外/超时/正常结束）、发送成功/失败数。
`JWT_SECRET` 仅用于铸造发送弹幕所需的用户 token，不在输出中暴露。

## 说明

- 压测机与服务器同机时，数字是保守下界；正式压测建议走 `127.0.0.1` 或同 VPC 内网，公网带宽会先于 CPU 成为瓶颈。
- 测 QPS 上限前关闭限流（本项目的限流开关是运行时配置 `rate_limit_enabled`，通过管理接口写入）。
