# Minibili AI 网关（消息中心助手）

## 功能

- 每位登录用户在「我的消息」中自动拥有与 **Minibili AI** 的固定会话（`kind=agent`）。
- 用户消息走现有 `POST /api/v1/dm/conversations/:id/messages`；服务端异步调用 **DeepSeek**，助手回复落库后经 **WebSocket**（`/api/v1/ws/chat`）推送。
- 短期上下文保存在 **Redis**（`mb:agent:hist:{conversationId}`），日配额 `mb:agent:quota:{userId}:{date}`。

## 架构

```text
Vue MbDmChatPanel
  → POST .../dm/.../messages
  → Go handler（鉴权、落库、WS 推送用户消息）
  → goroutine → internal/aigateway（DeepSeek HTTP）
  → 助手消息落库 → ChatHub.PushJSON
```

## 运营后台配置

登录运营中心 → **AI 角色**（`/admin/agent`）：

- **多角色卡片**：每个角色独立名称、头像、人设、欢迎语库
- **欢迎语库**：可配置多句，用户**首次**与该角色建立会话时**随机**抽取一句
- 每个启用角色在消息中心对应**独立会话**（不同系统账号）
- 最多 12 个角色；停用后不再为新用户创建会话，已有会话保留

## 环境变量

| 变量 | 说明 |
|------|------|
| `DEEPSEEK_API_KEY` | DeepSeek API Key（必填才启用回复） |
| `DEEPSEEK_BASE_URL` | 默认 `https://api.deepseek.com` |
| `DEEPSEEK_MODEL` | 默认 `deepseek-chat` |
| `AGENT_BOT_USERNAME` | 系统账号用户名，默认 `minibili_ai` |
| `AGENT_MAX_HISTORY` | Redis 上下文轮数上限 |
| `AGENT_HISTORY_TTL` | Redis 上下文过期时间（Go duration，默认 `720h` 即 30 天） |
| `AGENT_DAILY_QUOTA` | 每用户每日调用次数 |

## 面试可强调点

1. **网关职责**：鉴权、敏感词、配额、超时、模型适配，与业务 API 解耦。
2. **复用 IM**：同一套 DM 表、分页、WS，降低前端成本。
3. **异步回复**：用户请求快速返回，LLM 在后台 goroutine，结果 push。
4. **可观测扩展位**：`trace_id`、Prometheus、流式首 token 延迟（当前为非流式完整回复）。

## 相关代码

- `internal/aigateway/` — DeepSeek 客户端与 Redis 上下文
- `internal/service/agent.go` — 编排、配额、落库
- `internal/handler/dm.go` — agent 会话分支
- `internal/data/agent_seed.go` — 系统用户与会话初始化
