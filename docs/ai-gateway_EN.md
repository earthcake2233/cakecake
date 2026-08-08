<p align="center">
  <a href="docs/ai-gateway.md">
    <img src="https://img.shields.io/badge/🇨🇳中文-999999?style=flat-square" alt="中文">
  </a>
  <strong><img src="https://img.shields.io/badge/🇬🇧English-00a1d6?style=flat-square" alt="English"></strong>
</p>

# cakecake AI Gateway (Message Center Assistant)

## Features

- Every logged-in user automatically gets a fixed **cakecake AI** conversation (`kind=agent`) in "My Messages."
- User messages go through existing `POST /api/v1/dm/conversations/:id/messages`; server **streams** **DeepSeek**, text deltas are pushed in real time via **WebSocket** (`/api/v1/ws/chat`), and the final reply is persisted then pushed as a full message; supports pause / continue / regenerate.
- Short-term context stored in **Redis** (`mb:agent:hist:{conversationId}`), daily quota `mb:agent:quota:{userId}:{date}`.

## Architecture

```mermaid
sequenceDiagram
    participant V as "Vue MbDmChatPanel"
    participant Go as "Go Handler"
    participant AI as "internal/aigateway"
    participant WS as "ChatHub (WebSocket)"

    V->>Go: POST .../dm/.../messages
    Go->>Go: Auth, persist, WS push user message
    Go-->>AI: goroutine -> DeepSeek streaming request
    AI-->>WS: Stream deltas (agent_delta) -> PushJSON
    AI-->>WS: Final reply persisted -> dm_message
```

## Admin Configuration

Login to admin panel -> **AI Roles** (`/admin/agent`):

- **Multi-role cards**: Each role has independent name, avatar, persona, welcome message.
- **Welcome message pool**: Multiple welcome messages configurable; when user **first** establishes conversation with a role, **randomly** selects one.
- Each enabled role corresponds to a **separate conversation** (different system account).
- Max 12 roles; after disabling, new users won't get conversations but existing ones are preserved.

## Environment Variables

| Variable | Description |
|----------|-------------|
| `DEEPSEEK_API_KEY` | DeepSeek API Key (required for replies) |
| `DEEPSEEK_BASE_URL` | Default `https://api.deepseek.com` |
| `DEEPSEEK_MODEL` | DeepSeek model name, default `deepseek-v4-flash` |
| `AGENT_BOT_USERNAME` | System account username, default `minibili_ai` |
| `AGENT_MAX_HISTORY` | Redis context round limit |
| `AGENT_HISTORY_TTL` | Redis context expiry (Go duration, default `720h` = 30 days) |
| `AGENT_DAILY_QUOTA` | Per-user daily call limit |

## Interview Highlights

1. **Gateway responsibilities**: Auth, sensitive words, quotas, timeouts, model adaptation -- decoupled from business APIs.
2. **Reuses IM**: Same DM tables, pagination, WS -- reducing frontend costs.
3. **Async streaming replies**: User requests return quickly; LLM streams in a background goroutine, deltas are pushed in real time, final reply persisted.
4. **Observability**: `trace_id` spans backend logs and frontend; streaming first-token latency is now observable (Prometheus metrics remain an extension point).

## Related Code

- `internal/aigateway/` -- DeepSeek client, streaming orchestration & Redis context
- `internal/service/agent/agent.go` -- Orchestration, pause/continue state machine, quotas, persistence
- `internal/handler/agent_direct_message.go` -- resume/regenerate, async suggestions
- `internal/handler/direct_message_ws.go` -- WS control frames
- `internal/data/agent_seed.go` -- System user & conversation initialization

## Tool Use / Function Calling

### Architecture

```mermaid
flowchart TD
    U[User Message]
    AS[AgentService.GenerateReply]
    GW[Gateway.CompleteUserTurnWithTools<br/>Max 5 rounds]
    LLM[LLM.CompleteWithTools]
    TC{"finish_reason == 'tool_calls'?"}
    TK[Toolkit.ExecuteToolCalls<br/>Parallel execution]
    WS1["Push tool_call_start/end via WebSocket"]
    LOOP["Append tool results, loop back to LLM"]
    STOP{"finish_reason == 'stop'?"}
    REDIS["Persist full history + tool messages to Redis"]
    WS2["Return text reply via WebSocket push"]

    U --> AS
    AS --> GW
    GW --> LLM
    LLM --> TC
    TC -->|Yes| TK
    TK --> WS1
    WS1 --> LOOP
    LOOP --> LLM
    LLM --> STOP
    STOP -->|Yes| REDIS
    REDIS --> WS2
```

### New WebSocket Protocol

**tool_call_start** -- Begin executing a tool:
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

**tool_call_end** -- Tool execution complete:
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

**tool_result_data** -- Structured results for frontend card rendering:
```json
{
  "type": "tool_result_data",
  "body": {
    "trace_id": "a1b2c3d4",
    "span_id": "a1b2c3d4-t0",
    "tool_name": "search_videos",
    "items": [
      { "id": 1, "title": "Super Electromagnetic Cannon", "author": "earthcake", "plays": 32, "cover": "https://...", "duration": "5:24" }
    ]
  }
}
```

### Defined Tools

| Tool | Parameters | Description |
|------|------------|-------------|
| `search_videos` | `keyword`(required), `page`, `page_size` | Keyword search, ES primary, DB LIKE fallback |
| `get_video_detail` | `video_id`(required) | Video detail + uploader info + tags |
| `get_trending` | `limit` | Hot video leaderboard (by play count) |
| `get_video_comments` | `video_id`(required), `page`, `page_size` | Video comment list |
| `get_video_danmaku` | `video_id`(required), `limit` | Video danmaku samples |

### Admin Toggles

Each tool can be independently enabled/disabled via RuntimeConfig, key format: `tool_{name}_enabled`, default `true`.

| Key | Description |
|-----|-------------|
| `tool_search_videos_enabled` | Search videos |
| `tool_get_video_detail_enabled` | Video detail |
| `tool_get_trending_enabled` | Leaderboard |
| `tool_get_video_comments_enabled` | Comments |
| `tool_get_video_danmaku_enabled` | Danmaku |

### Interview Highlights

1. **Tool Schema Design**: Each tool's description is detailed, parameters marked required, helping the model select accurately.
2. **Multi-round Orchestration**: 5-round cap + degradation on overflow, preventing dead loops.
3. **Parallel Execution**: Same-round tool_calls execute in goroutines, improving response speed.
4. **Three-layer Anti-abuse**: Quota check -> sensitive word input/output filter -> per-tool RuntimeConfig toggle.
5. **trace_id??**: Same trace_id in both Zap logs and frontend push, shared debugging chain.

---

## Streaming / Pause / Continue / Regenerate: Pitfalls & Key Technical Points

> This section records the pitfalls encountered while evolving the assistant from a non-streaming full reply into a streaming typewriter with pause/continue, regenerate, and follow-up suggestions. It is reorganized as "symptom → why it's hard → how we fixed it" so it is easier to follow.

### The Big Picture (Plain English)

An AI reply doesn't arrive all at once; it streams in small pieces (deltas). The product requires three things:

1. **Typewriter effect**: show text as it is generated;
2. **Stop ≠ cancel**: clicking "stop" only hides new content; the backend LLM keeps running. Clicking "continue" replays the buffered text at typewriter pace, then resumes live output;
3. **Regenerate**: ask the model again for the same user question and replace the old reply in place instead of creating a new bubble.

Why it's hard: the stream is asynchronous (the backend generates while the frontend receives), users can act concurrently (rapid stop/continue), and the WebSocket can drop and reconnect. Combined, these produce bugs that look fine at first glance but leak in edge cases.

### Glossary

| Term | Plain-English meaning |
|------|-----------------------|
| delta | A small piece of text emitted by the model during streaming |
| pause | Stop showing new content; the backend LLM keeps running and new deltas go into a buffer |
| buffer | Deltas accumulated while paused; replayed at typewriter pace on continue |
| resume (continue) | Replay the buffer first, then resume live deltas |
| regenerate | Re-ask the model for the same turn and replace the old reply in place |
| genID | Generation counter; incremented on each regenerate, used to discard stale streams |
| version bubble | Frontend merges multiple replies for the same question into one bubble with a `‹ n / n ›` switcher |

### 1. Delta Push Target: the DM `UserLow/UserHigh` Trap

**Symptom**: the final message arrives, but the typewriter never moves; debugging shows every `agent_delta` was pushed to the bot (nobody connected).

**Why it's hard**: DM conversations store both participant IDs sorted as `(min, max)` in `user_low/user_high`. The bot ID can be smaller than the user's (e.g. bot=14, user=18), so `conv.UserLow` is the **bot**, not the user.

**Fix**: `humanPeerForConversation(conv, botUserID)` explicitly returns the non-bot participant and is locked down by a unit test.

### 2. Global Callback vs Per-Call Callback (Cross-User Mixup)

**Symptom**: user A's reply contains fragments from user B.

**Why it's hard**: the gateway originally exposed a single `OnTextDelta` field overwritten before each generation. With two concurrent generations, the later closure replaced the earlier one.

**Fix**: `CompleteUserTurnStream` / `CompleteUserTurnWithToolsStream` / `ContinueTurnStream` now accept a **per-call** `onDelta func(string)`; every call holds its own closure (including its own `humanID`).

### 3. Pause/Resume State Machine (The Most Leak-Prone Part)

**Design goal**: clicking "stop" keeps the LLM stream running in the background and buffers deltas; clicking "continue" replays the buffer at typewriter pace and then resumes live deltas.

Core state:

| Field | Meaning |
|-------|---------|
| `paused` | Paused (deltas go to the buffer, not pushed) |
| `buffer []string` | Deltas buffered while paused |
| `dropped` | Superseded by a newer generation; discard later deltas |
| `genID` | Generation id; an old goroutine can never clear newer state |
| `pauseSeq` | Stop sequence, incremented on every pause |
| `resuming` | Replay-in-progress flag; serializes concurrent continues |

Three pitfalls hit:

- **Unpausing before replay**: buffered replay and live deltas wrote concurrently, scrambling text into code blocks. Fix: stay `paused` during replay; unpause only after the buffer is drained.
- **Two concurrent continues**: the second replay unpaused before the first finished, recreating the interleave. Fix: the `resuming` flag makes concurrent resumes wait; unpause only when no new pause happened during replay (`pauseSeq` unchanged) and the buffer is empty.
- **A stop during replay was overridden**: the replay's final “unpause” wiped out the just-clicked stop, making stop ineffective. Fix: check `pauseSeq` **before pushing every fragment**; on a new stop, interrupt the replay immediately and keep the remainder buffered for the next continue.

### 4. Avoiding Duplicate Rows: Never Substring-Match the Draft

**Symptom**: after the reply was already persisted, clicking "continue" created a second reply.

**Why it's hard**: we used to check whether the latest assistant content "contains" the streamed draft. Two fatal flaws:

1. Persisted content is normalized (`normalizeMarkdownFences` / `dedupeConsecutiveLines` / trim) while the draft is the raw stream — they do not match;
2. `ListMessages` returns DESC (newest first), but the old code iterated from the tail and returned on the first assistant found — effectively comparing against the **oldest** message in the window.

**Fix**: `latestUserTurnHasAssistantReply` — only check whether an assistant row exists after the latest user message, with **no text comparison**; combined with a per-user `agentRunLock` that serializes the resume decision and persistence, so even a double continue persists exactly one row.

### 5. Async Follow-Up Suggestions (Eliminating the End Dead-Wait)

**Symptom**: after the reply text finished streaming, the final message took a long time to appear ("it finally generated after a long wait").

**Why it's hard**: a **synchronous** second LLM call generated suggestion chips (up to 15s) and blocked persistence of the final message.

**Fix**: persist and push the reply immediately (empty suggestions), then in a background goroutine:

1. `UpdateMessageSuggestions(messageID, sugg)` updates the row;
2. Push a new WS event `agent_suggestions {message_id, suggestions}`;
3. The frontend updates the existing message's suggestions by `message_id`; chips appear later.

Measured: the gap from stream end to persisted row dropped from 5–15s to ~100ms.

### 6. Markdown Fence Normalization

**Symptom**: at stop/continue seams the model often emits extra ````go`, duplicate open fences, or repeated lines, breaking frontend markdown rendering.

**Fix**: normalize before persistence:

- `normalizeMarkdownFences`: unify fences to three backticks, drop seam artifacts (language-tagged fence while already inside a block), append a closing fence if unclosed;
- `dedupeConsecutiveLines`: remove exact consecutive duplicate lines;
- `plainTextPreview`: strip markdown for conversation-list previews.

⚠️ When asserting stream order, normalize both the client-side delta concatenation and the persisted text with the **same** normalization; otherwise fence differences cause false failures.

### 7. Frontend State Machine: In-Place Regenerate

Consecutive assistant rows are merged into a “version bubble” (`versions` + `‹ n / n ›` switcher).

Regenerate flow: `_agentRegenerating=true` → clear the old content in place → the new stream types into the same bubble.

Two pitfalls:

- **Clearing `_agentRegenerating` on stop**: the renderer assumed the in-place rewrite ended, the old version flashed back, and a duplicate `agent-draft` bubble appeared. Fix: keep the in-place flag through stop/continue until the final `dm_message` arrives.
- **Not clearing the old version's tool data**: the previous turn's `search_videos` cards/status leaked into the regenerating bubble. Fix: when rewriting in place, also clear `toolActivities` / `toolResultData` / `suggestions`.

### 8. Scrolling & UX

- Scrolling to the bottom on every delta hijacked the user's manual scroll. Fix: `onChatScroll` maintains `_userScrolledUp`; auto-scroll only runs when the view is near the bottom.
- Clicking regenerate jumps to the user's question row (explicit requirement); ~700ms after the jump animation, auto-follow resumes. Manual up-scroll always wins.
- Keeping the in-place rewrite during stop/continue prevents scroll jumps to a new bubble.

### 9. Never Drop WS Control Frames

If `sendWsControl` returns early while the WS is reconnecting, stop/continue frames are silently lost — the backend streams to completion (“stop doesn't work, it all generates at once”).

Fix: queue control frames in `_pendingWsControls` and flush them in `onopen`.

### 10. End-to-End Verification

These bugs cannot be covered by unit/API tests; they require a **real browser + real model**:

- **Browser click flow** (puppeteer + Chromium): send → stop → continue → regenerate; assert DB row count, no duplicate `agent-draft` row in the DOM, old content never flashes back, and scroll position.
- **Raw WS race**: send two `agent_continue` frames back-to-back and assert exactly one assistant row.
- **Stream-order assertion**: concatenate all deltas received by the client and compare with the persisted text under the same normalization (include frames received before the polling loop; we once mis-reported by skipping the head).
- **Stop responsiveness**: at most 2–3 in-flight fragments after stop (network latency); the stream must not keep emitting whole chunks.

### Protocol Additions

| Event | Direction | Description |
|-------|-----------|-------------|
| `agent_delta` | server → client | Streaming text fragment `{content}` |
| `agent_suggestions` | server → client | Async follow-up chips `{message_id, suggestions}` |
| `agent_cancel` | client → server | Pause (buffer; background LLM keeps running) |
| `agent_continue` | client → server | Replay buffer and resume live stream |
| `agent_regenerate` | client → server | Regenerate the turn (new version) |

### Core Invariants

- Each user message (except regenerate) ends with exactly one assistant row; regenerate adds one version per click.
- Streaming order == persisted text order; pause/continue must never reorder.
- Stop latency ≈ one in-flight fragment; a stop during replay takes effect immediately.
- Control frames must always be delivered (reconnect flush), never silently dropped.

### Related Code

- `internal/aigateway/gateway.go` — per-call deltas, streaming tool orchestration
- `internal/service/agent/agent.go` — pause/resume state machine, markdown normalization
- `internal/handler/agent_direct_message.go` — resume/regenerate, async suggestions
- `cakecake-vue/cakecake-web/src/components/cakecake/MbDmChatPanel.vue` — frontend state machine, scrolling, control-frame queue
