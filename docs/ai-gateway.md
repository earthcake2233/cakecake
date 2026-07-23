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
## Tool Use / Function Calling

### 架构

```text
User Message
   → AgentService.GenerateReply
     → Gateway.CompleteUserTurnWithTools (最多 5 轮)
       → LLM.CompleteWithTools(messages, tools)
       → finish_reason == "tool_calls"?
         → Toolkit.ExecuteToolCalls (并行执行)
         → Push tool_call_start/end via WebSocket
         → Append tool results → loop back to LLM
       → finish_reason == "stop"?
         → Persist full history (含 tool 中间消息) → Redis
         → Return text reply → WebSocket push
```

### 新增 WebSocket 协议

**tool_call_start** — 开始执行某个工具
```json
{
  "type": "tool_call_start",
  "body": {
    "trace_id": "a1b2c3d4",
    "span_id": "a1b2c3d4-t0",
    "parent_span_id": "a1b2c3d4",
    "tool_name": "search_videos",
    "arguments": { "keyword": "golang" },
    "started_at": "2026-07-23T10:00:00Z"
  }
}
```

**tool_call_end** — 工具执行完成
```json
{
  "type": "tool_call_end",
  "body": {
    "trace_id": "a1b2c3d4",
    "span_id": "a1b2c3d4-t0",
    "tool_name": "search_videos",
    "duration_ms": 42,
    "result_summary": "found 3 results"
  }
}
```

**tool_result_data** — 工具返回的结构化结果数据（用于前端渲染卡片）
```json
{
  "type": "tool_result_data",
  "body": {
    "trace_id": "a1b2c3d4",
    "span_id": "a1b2c3d4-t0",
    "tool_name": "search_videos",
    "items": [
      {
        "id": 1,
        "title": "某科学的超电磁炮",
        "author": "earthcake",
        "plays": 32,
        "cover": "https://...",
        "duration": "5:24"
      }
    ]
  }
}
```

前端在收到 `tool_call_start` / `tool_call_end` 后，会收到对应的 `tool_result_data`。前端将 `items` 数组渲染为视频卡片、评论卡片或弹幕列表。

### 已定义工具

| Tool | 参数 | 说明 |
|------|------|------|
| `search_videos` | `keyword`(必填), `page`, `page_size` | 关键词搜索视频，优先走 ES，回退 DB LIKE |
| `get_video_detail` | `video_id`(必填) | 视频详情 + UP 主信息 + 标签 |
| `get_trending` | `limit` | 热门视频排行榜（按播放量） |
| `get_video_comments` | `video_id`(必填), `page`, `page_size` | 视频评论列表 |
| `get_video_danmaku` | `video_id`(必填), `limit` | 视频弹幕样本 |

### Admin 开关

每个工具可通过 RuntimeConfig 独立启用/禁用，key 格式：`tool_{name}_enabled`，默认 true。

| key | 说明 |
|-----|------|
| `tool_search_videos_enabled` | 搜索视频 |
| `tool_get_video_detail_enabled` | 视频详情 |
| `tool_get_trending_enabled` | 排行榜 |
| `tool_get_video_comments_enabled` | 评论 |
| `tool_get_video_danmaku_enabled` | 弹幕 |

### 面试可强调的点

1. **Tool Schema 设计**：每个 tool 的 description 写详细，parameters 标注 required，帮助模型准确选择
2. **多轮编排**：5 轮上限 + 超限降级，防止死循环
3. **并行执行**：同轮 tool_calls 用 goroutine 并行执行，提升响应速度
4. **防滥用三层**：配额检查 → 敏感词入参/出参过滤 → 每工具独立 RuntimeConfig 开关
5. **trace_id 贯穿**：同一 trace_id 既写 Zap 日志又推前端，后端调试和前端展示共用同一链路 ID