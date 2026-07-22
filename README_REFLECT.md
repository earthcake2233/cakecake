# README_REFLECT — 架构与反思记录

本文档记录 **Mini-Bili 与 SPEC 对齐过程中的架构取舍**，并与仓库内 [`Minibili.md`](Minibili.md)（面试向项目建议）中的方向做对照，便于日后复盘与简历讲述。

---

## 2026-05-12：弹幕实时下行走 Redis Pub/Sub（对齐 SPEC NF-3）

### SPEC 原文（摘录）

**NF-3 存储划分**：MySQL 存持久数据；**Redis：弹幕实时通道中转**；另有播放量热数据等。

此前实现为 **单进程内 `Hub` 内存广播**：`POST /danmaku` 与「正在看」更新直接 `Hub.BroadcastJSON`，与 NF-3「经 Redis 中转」的字面表述不一致；多副本部署时也无法跨进程 fan-out。

### 当前实现（概要）

| 组件 | 职责 |
|------|------|
| **Redis 频道** `minibili:danmaku:fanout`（常量见 `internal/data/redis.go` 的 `ChannelDanmakuFanout`） | 承载 **房间级下行消息** 的跨进程中转。 |
| **`service.DanmakuRelay`**（`internal/service/danmaku_relay.go`） | `Publish`：将 `{ "video_id", "body" }` 信封写入 Redis；`RunSubscriber`：订阅该频道，解析后对本地 **`ws.Hub` 调用 `BroadcastRaw`**，把 `body` 原样写入各 WebSocket。 |
| **`ws.Hub`** | 仍只负责 **本机** WebSocket 连接注册与写扇出；新增 `BroadcastRaw` 避免中继路径下二次 JSON 序列化。 |

**走中继的负载**（与弹幕长连接同一前端通道）：

- 新弹幕广播：`PostDanmaku` 在入库后 `DanmakuRelay.Publish`，不再直接 `Hub.BroadcastJSON`（`DanmakuRelay == nil` 时回退到 Hub，便于极简测试桩）。
- 「正在看」人数广播：`pushWatchingCount` 同样经 `Publish`。

**仍仅走本地 Hub 的消息**（本轮未纳入 NF-3「弹幕通道」范围）：

- 评论删除等对 **同一视频房间** 的 `comment_deleted` 推送仍用 `Hub.BroadcastJSON`。若未来多副本也要同步该事件，可复用同一 `DanmakuRelay` 信封机制。

### 运维与扩展含义

- 多实例 API **必须共用同一 Redis**，且 **每个进程** 启动时 `go relay.RunSubscriber(ctx)`（已在 `cmd/mini-bili/main.go` 接入）。
- 单机单副本时行为与改造前一致：本机 Publish → 本机 Subscriber → 本机 Hub，多一跳 Redis，用于与 SPEC 及后续水平扩展对齐。

### 与 `Minibili.md` 的对照（为何记在这里）

`Minibili.md` 中强调：

1. **「分布式弹幕系统」作为核心亮点** —— 用 WebSocket + 后端协同讲清「连接与广播」还不够；**经 Redis 的中转层** 是多副本/与面试官讨论「多机 fan-out」时的自然落点。
2. **「文档化你的思考」** —— 策略 3 建议把架构与选型写进 README；本文件即承接该习惯，避免只改代码不留痕。

后续若引入 **Redis Stream / 消费组** 做削峰或与独立网关进程拆分，可在此文件追加一节「演进记录」。

---

## 2026-05-12：F0 用户个人信息管理（SPEC / Rule / Skill 同步）

- **SPEC**：新增 **F0**（置于 F1 前），约定 `GET/PUT /api/v1/users/me`、`PUT .../password`、`POST .../avatar`（表单字段 `avatar`，OSS `avatars/{user_id}.{ext}`），「我的视频」沿用 **`GET /api/v1/users/me/videos`**；**NF-7** 增补头像路径；核心目标 1.1 补充「维护个人信息」。
- **Rule**：新增 **R-BIZ-8**（头像校验同 S-005 扩展名逻辑，大小 ≤5MB），与既有 **R-BIZ-7**（弹幕颜色）区分。
- **实现**：`User.avatar_url`；`coverval.ValidateAvatarHeader`；`handler/user_me.go`；错误码 **40015 / 40016 / 40301**；`TestGetMe_SQLite`。
- **前端**：`minibili.ts` 增加 `mbGetMe`、`mbPutMeUsername`、`mbPutMePassword`、`mbPostMeAvatar`。

---

## 2026-05-27：JWT 主动刷新与 Refresh Token 生命周期延长

### 背景

此前 JWT Access Token 过期后需用户手动重新登录，体验割裂。面试官视角也关注「Token 安全与用户体验的平衡」。

### 决策

- **Refresh Token 生命周期从 7 天延长至 30 天**，减少用户频繁登录的摩擦。
- **实现主动刷新**：前端在 Axios 拦截器中检测 Access Token 即将过期，自动调用刷新接口换取新 Token，对用户无感。
- 双 Token 机制不变（Access 短期 + Refresh 长期），Refresh Token 仍在 Redis 中管理轮转。

### 与 `Minibili.md` 的对照

`Minibili.md` 未显式提及 Token 策略，但面试沟通中「认证安全」是常见追问方向。双 Token + 主动刷新是标准的 SPA 认证实践，可作为独立话题展开。

---

## 2026-05-27：搜索系统接入 Elasticsearch（可选组件）

### 决策

- 引入 **Elasticsearch** 作为视频全文搜索引擎，覆盖标题、描述字段。
- 架构上定位为 **可选依赖**：未配置 ES 时搜索页优雅降级（返回空或走简单 DB like 查询），降低入门门槛。
- 独立搜索模块 `internal/search/`，封装 ES 客户端与查询构建逻辑。

### 与 SPEC 的对应

SPEC v1.0 明确将「视频搜索」排除在 v1 目标外，但实际项目中搜索是用户高频路径。将该能力做成可选组件，既满足了功能完整性，又不影响 SPEC 的版本承诺。

---

## 2026-06-26：自定义非商业许可协议

### 决策

- 新增 `LICENSE` 文件，采用自定义 **Non-Commercial License**，声明本项目仅限非商业用途。
- 与 README 截图、在线 Demo 地址等共同构成项目的完整对外呈现。

---

## 2026-06-27：README 国际化与在线体验入口

### 变更

- 新增 **`README_EN.md`**（英文版），与 `README.md` 中文版通过顶栏语言切换器联动。
- 加入 **在线 Demo 链接**（`chengzisoft.top`）和 **B 站演示视频链接**，提供立即可体验的入口。
- 技术栈徽标统一美化，从纯文本变为 shields.io 徽章行。

### 文档同步规则

此次变更触发了对文档同步的规范化思考，后续演进为 **R-DOC-1**（中英 README 同步），详见下方 2026-07-20 的 Rule 体系建设。

---

## 2026-07-20：ARCHITECTURE 文档体系与 Mermaid 架构图

### 背景

截至七月中旬，项目超过 5000 行 Go + 完整 Vue 前端，但缺少一份面向面试/技术评审的系统架构文档。此前架构信息分散在 README、`Minibili.md`（面试建议）、`SPEC.md`（需求规格）中，缺乏统一的「设计总览」。

### 新增文档

| 文件 | 内容 |
|------|------|
| `docs/ARCHITECTURE.md` | 中文系统架构文档：系统总览图、模块深挖（弹幕/评论/转码/搜索/AI 网关）、关键设计决策表、端到端数据流、测试策略。 |
| `docs/ARCHITECTURE_EN.md` | 英文对应版本，与中文保持章节、图表、表格完全同步（受 R-DOC-3 约束）。 |

### 架构图从 ASCII 升级为 Mermaid

- 此前设计文档中手绘 ASCII 图（`+--+--+` 风格）不易维护、难以嵌入 Markdown 渲染。
- 全部替换为 **Mermaid** 声明式图表：系统总览 graph、弹幕实时流 sequence diagram、评论模块 erDiagram。
- 数轮迭代修正：移除 subgraph 样式冲突、修正 Nginx 角色（静态文件服务 + API 反向代理）、移除「热搜时间窗口」等未实现功能。

### 关键决策表（首次系统化整理）

| 决策 | 理由 |
|------|------|
| v1 用单体而非微服务 | 单人开发，快速迭代。按领域分层为后续拆 Kratos 微服务预留空间 |
| Redis Pub/Sub 做弹幕广播中继 | 解耦广播与 HTTP handler，多副本订阅同一频道即可水平扩展 |
| 转码用 RabbitMQ 而非 Redis List | 消息持久化 + 消费确认 + 死信队列——视频处理不可接受数据丢失 |
| ES 可选而非强制依赖 | 降低上手门槛，未配置时优雅降级 |

### 与 `Minibili.md` 的对照

`Minibili.md` 策略 2 强调「注入架构味道」，ARCHITECTURE 文档正是该策略的具体落地——不再只是「写了什么代码」，而是「为什么这么设计」。面试时可据此展开模块深挖。

---

## 2026-07-20：单元测试基础设施搭建

### 现状

此前测试覆盖有限，仅 `internal/handler/` 有少量 SQLite 内存库测试。核心模块（弹幕 Hub、JWT、Handler 业务逻辑）缺少自动化验证。

### 新增测试（571 行）

| 测试文件 | 行数 | 覆盖内容 |
|---------|------|---------|
| `internal/ws/hub_test.go` | 147 | Hub 的客户端注册/注销、BroadcastJSON/BroadcastRaw 扇出 |
| `internal/pkg/jwttoken/jwt_test.go` | 160 | Token 签发、校验、Refresh 流程 |
| `internal/handler/danmaku_test.go` | 167 | 弹幕发送、敏感词拦截、5s 冷却校验 |
| `internal/handler/comment_test.go` | 97 | 评论级联删除（AC-16） |

### 测试策略文档化

在 `ARCHITECTURE.md` 中新增「测试策略」章节，按层级划分：

- `internal/pkg/*`：表驱动单元测试
- `internal/handler/*`：SQLite 内存库 + miniredis 模拟依赖
- `internal/handler/*`（integration 标签）：黑盒测试（连真实服务）
- E2E：手动验证

### 后续规范化

此次测试基础设施的搭建，为 **R-TEST-1 / R-TEST-2**（见 2026-07-21）提供了落地基础。

---

## 2026-07-20 ~ 2026-07-21：工程规范体系建设（R-DOC / R-TEST）

### 背景

项目发展到中后期，代码量增大、多人协作（AI + 人）频率提高，需要显式规则确保质量底线。

### R-DOC 文档规则

| 编号 | 规则 |
|------|------|
| **R-DOC-1** | 中英 README 必须同步更新 |
| **R-DOC-2** | 代码变更必须同步更新文档，每次提交前强制检查 |
| **R-DOC-2a** | commit message 中必须列出已检查的文档列表 |
| **R-DOC-3** | ARCHITECTURE 中英文必须完全同步 |
| **R-DOC-4** | Git 提交信息必须使用英文，遵循 conventional commits 格式 |

### R-TEST 测试规则

| 编号 | 规则 |
|------|------|
| **R-TEST-1** | 新增业务代码必须同步编写测试（handler/service/ws/pkg 层） |
| **R-TEST-2** | 测试必须可独立运行，使用 SQLite 内存库 + miniredis 模拟依赖，不连真实服务 |

### 与 `Minibili.md` 的对照

`Minibili.md` 建议「文档化你的思考」——R-DOC 系列规则将该建议制度化，确保架构决策不随开发人员更迭而丢失。R-TEST 则对应面试中常问的「你的项目如何保证质量」。

---


---

## 2026-07-21：全局限流 — Redis 令牌桶中间件

### 背景

此前仅弹幕发送有 5s SetNX 冷却（业务级别），全局限流缺失。公开接口（视频列表、搜索、用户空间）在高并发或爬虫扫描时无保护，存在被打垮的风险。

### 决策

- **算法**：Redis 令牌桶（桶容量 50，每秒填充 20 个），通过 Lua 脚本保证原子性。
- **粒度**：按 IP 维度（c.ClientIP()）。
- **实现**：internal/middleware/ratelimit.go，RateLimiter 结构体封装 Redis 客户端与配置。
- **WebSocket**：检查 Upgrade: websocket 请求头跳过限流，不依赖硬编码路径。
- **配置**：RATE_LIMIT_ENABLED=false（默认关闭，不破坏现有部署）、RATE_LIMIT_RATE=20、RATE_LIMIT_BURST=50。
- **响应**：HTTP 429 + Retry-After: 1 + X-RateLimit-Remaining header + 业务错误码 42900。

### 测试覆盖

| 测试 | 内容 |
|------|------|
| TestRateLimit_Allowed | 正常请求通过，返回 200 并携带剩余令牌 header |
| TestRateLimit_Blocked | 耗尽令牌后返回 429 和 Retry-After |
| TestRateLimit_Refill | 等待一秒后令牌 refill，请求恢复正常 |
| TestRateLimit_SkipsWebSocket | WebSocket 握手请求不受限流影响 |

测试使用 miniredis 模拟 Redis（R-TEST-2），不依赖外部服务。

### 与 Minibili.md 的对照

Minibili.md 中强调「注入架构味道」——全局限流是面试高频话题，回答结构可自然展开：
1. 为什么需要（公开接口保护、暴力破解防御）
2. 为什么用令牌桶（支持突发 + 稳态限速）
3. 为什么用 Redis Lua（原子性、多副本共享）
4. 大流量下怎么优化（Lua 脚本预加载、EVALSHA 缓存）



## 2026-07-21：运行时配置系统

### 动机
环境变量管理运营参数（AGENT_DAILY_QUOTA、RATE_LIMIT_ENABLED 等）需要重启服务才能生效。管理员在面板上直接调整限流阈值或 AI 开关是更合理的生产实践。

### 设计
- **SystemConfig 表**（MySQL key-value）：持久化运营参数
- **RuntimeConfig 管理器**（internal/config/runtime.go）：内存缓存 + 30s 定时轮询 DB
- **分层回退**：优先读动态配置 → 回退到环境变量默认值
- **⛓ 避免循环引用**：RuntimeConfig 直接使用 gorm.DB 操作 system_configs 表，不走 data 包，避免 config → data → config 的 import cycle

### 动态化配置清单

| Key | 说明 |
|-----|------|
| agent_enabled | AI 助手总开关 |
| agent_daily_quota | 每用户每日对话上限 |
| agent_max_history | 历史消息条数限制 |
| agent_history_ttl | 历史消息过期时间 |
| agent_request_timeout | LLM 请求超时 |
| rate_limit_enabled | 全局限流开关 |
| rate_limit_rate | 令牌桶速率 |
| rate_limit_burst | 令牌桶容量 |

### 管理员 API
- GET /api/v1/admin/system-configs — 返回全部已知配置键值
- PUT /api/v1/admin/system-configs — 批量更新，立即生效
- 未知 key 被拒绝（400），防止误写

### 中间件处理
RateLimit 中间件**始终挂载**，内部检查 rate_limit_enabled 动态开关。重启不再是调参的前提。

### 管理员前端

```
src/pages/admin/SystemConfigManage.vue
```

基于 Element Plus 的卡片式配置管理页面，分「AI 助手」「全局限流」两个模块组：

| 特性 | 说明 |
|------|------|
| 模块化布局 | 每类配置独立卡片，带图标 header + 环境变量回退标签 |
| 实时变化追踪 | 修改过的行高亮黄色背景，顶栏 badge 显示变更数 |
| 底栏保存条 | 有未保存变更时自动弹出，一键批量保存或放弃 |
| 令牌桶可视化 | 开启限流后展示速率/容量进度条 + 并发估算 |
| 单字段还原 | 每个修改项旁有「还原」按钮，点一下回到原始值 |
| 零网络依赖同步 | 保存时直接本地同步 original，不依赖二次 GET |

构建后生成约 14KB JS + 5KB CSS，通过 `VITE_REMOTE_API_BASE` 环境变量对接后端 API。

### 面试切入点
- 「配置热加载怎么实现？」→ 30s 轮询 + RWMutex 保护的 map
- 「重启才能生效的痛点怎么解决？」→ 运行时配置 + 管理员 API + 前端面板
- 「如何避免循环引用？」→ 分层分割，ORM 直连而非 data 包
- 「前端配置面板怎么设计？」→ 卡片模块化、脏标记追踪、底栏批量保存

---

## 2026-07-22：Codecov 覆盖率 CI 集成

### 动机

此前测试覆盖率高（前端 ~73%、后端 ~70%+），但缺少**可视化展示**和**回归门槛**。团队在 CR 时无法直观判断新增代码的覆盖情况，历史 commit 的覆盖率变化也缺乏追踪。

### 实现

- **CI 集成**：在 `.github/workflows/ci.yml` 中接入 `codecov/codecov-action@v5.5.5`，前后端各一个 job，分别上传 lcov.info 和 Go coverage.out
- **配置管理**：`codecov.yml` 定义了前后端 flag 路径、carryforward 策略、70% 覆盖率目标
- **README 徽章**：`Vue Coverage` 和 `Go Coverage` 两个 badge，点击跳转 Codecov 详情页
- **本地 CLI**：已安装 `codecov.exe v11.3.1`，支持本地手动上传调试

### 采坑记录

| 问题 | 原因 | 解决 |
|------|------|------|
| 手动 curl 下载 CLI + 传参上传后 Codecov 无记录 | `--git-service github` 对 `upload-coverage` 是无效参数 | 改用 `codecov-action@v5.5.5` 处理三步流程 |
| 上传后 upload 状态卡在 `started` 不处理 | Codecov 免费计划异步处理队列有约 2 小时延迟 | 等队列跑完或升级付费计划 |
| `codecov-action@v5` 浮动标签导致行为变化 | `@v5` 是新版本，内部从 bash uploader 换成了 CLI | 锁定 `@v5.5.5` 具体版本 |
| CODECOV_TOKEN 可选但 create-commit 仍需认证 | CLI v11 的 `create-commit` 无 `--ci-passed` 参数 | 直接传 token 并通过 `fail_ci_if_error: true` 暴露错误 |

### 当前状态

- 仓库 total: 15.76% 覆盖率，193 文件（含零覆盖的框架/配置类文件）
- 前端: ~73% 语句覆盖，77 测试文件
- 后端: ~71% 综合覆盖，27 测试文件
- 新 commit 上传后需等待 Codecov 后台处理


## 后续展望

- **Redis Stream / 消费组**：若弹幕量级继续增长，可引入消费组做削峰填谷，或将 WebSocket 网关拆为独立进程。
- **Kratos 微服务拆分**：v1.0 的领域分层（`handler/` → `service/` → `dao/`）已预留拆分空间，后续可按需拆出用户/视频/弹幕/评论服务。
- **E2E 自动化测试**：当前 E2E 为手动，可引入 Playwright 做端到端回归验证。
- **前端测试补充**：当前测试集中在后端，Vue 组件层（弹幕渲染、发布表单）尚缺自动化测试覆盖。
