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
| AgentEnabled | AI 助手总开关 |
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

#### Codecov / CI

| 问题 | 原因 | 解决 |
|------|------|------|
| 手动 curl 下载 CLI + 传参上传后 Codecov 无记录 | --git-service github 对 upload-coverage 是无效参数 | 改用 codecov-action@v5.5.5 处理三步流程 |
| 上传后 upload 状态卡在 started 不处理 | Codecov 免费计划异步处理队列有约 2 小时延迟 | 等队列跑完或升级付费计划 |
| codecov-action@v5 浮动标签导致行为变化 | @v5 是新版本，内部从 bash uploader 换成了 CLI | 锁定 @v5.5.5 具体版本 |
| CODECOV_TOKEN 可选但 create-commit 仍需认证 | CLI v11 的 create-commit 无 --ci-passed 参数 | 直接传 token 并通过 fail_ci_if_error: true 暴露错误 |

#### Windows 开发环境

| 问题 | 原因 | 解决 |
|------|------|------|
| Go 测试文件中 struct tag 反引号丢失 | PowerShell 用反引号(`)做转义字符，通过 @"..."@ heredoc 或管道传字符串时反引号被消耗 | 用 Python 脚本写入文件，或使用 .NET 的 WriteAllText 直接写入字节数组 |
| Go build cache 报 Access is denied | Windows 文件锁或权限问题导致 cache 文件损坏 | Remove-Item C:\Users\15072\AppData\Local\go-build -Recurse -Force 删除缓存目录后重建 |
| go test -coverprofile=file 未生成 profile | test 失败退出码非 0 时不写 profile 文件 | 用 cmd /c "go test ... 2>&1" 绕开 PowerShell 的退出码处理 |
| Set-Content 写入 UTF-8 文件带 BOM | PowerShell 默认对 UTF-8 添加 BOM，Go 编译器报错 | 用 [System.IO.File]::WriteAllText(path, content, [System.Text.UTF8Encoding]::new(\$false)) 写入无 BOM 的 UTF-8 |
| CRLF/LF 行尾导致 go vet 报错或 Git 警告 | Windows 上 Git 自动转换 CRLF，Go 工具链期望 LF | 设置 .gitattributes 或提交前 git add 时 Git 自动转换 |
| inline Python 脚本在 PowerShell 中引号/反斜杠被转义 | PowerShell 的双引号字符串内 \" 和 \\ 有特殊含义 | 复杂脚本写到 .py 文件后用 python file.py 执行，避免 inline |

#### Go 测试

| 问题 | 原因 | 解决 |
|------|------|------|
| handler 测试中 Redis 未设置导致 nil pointer | admin handler 的 AdminRefresh 等函数直接调用 a.Redis.Exists()，测试未提供 Redis | 在 newTestAPI 中引入 miniredis，即使 handler 不直接使用 Redis 也要初始化 |
| gatewayReady() 在 nil receiver 上调用时 panic | 方法体在 s != nil 检查前访问了 s.RC 字段 | 将 nil 检查移到方法体最前面，或在测试中避免直接调用 nil 对象的方法 |
| AgentEnabled 未设置导致 gatewayReady() 总是返回 false | config.C{} 的 AgentEnabled 默认为 false，测试中未显式设为 true | 测试所有与 gatewayReady() 相关的调用时，必须同时设置 AgentEnabled: true |
| 覆盖率统计包含零覆盖框架文件拉低整体 | go test ./... 会统计所有包，包括仅含 migration/seed 的数据层 | 用 go test -coverprofile 配合 -coverpkg 聚焦业务包，或单独统计各包 |

#### Git / 文档

| 问题 | 原因 | 解决 |
|------|------|------|
| Rule.md 章节编号错乱 | 编辑时误把章节内容粘贴到文件头部 | 提交前 Select-String 验证章节标题顺序，确保连续 |
| .gitignore 中 cov_out 重复 3 次 | 多次修改 .gitignore 追加同类规则时未去重 | 清理 .gitignore，按功能分组，同类规则只保留一条 |
| Makefile 中文乱码 | 文件保存为非 UTF-8 编码或 BOM 问题 | 用 Python pathlib.Path.write_text(encoding='utf-8') 写入确保 UTF-8 无 BOM |
## 2026-07-23：Markdown 写入格式规范化 + 中文编码脚本化

### 动机

反复出现两个问题：1）PowerShell 下 inline Python 含中文必乱码（`python -c "..."` 走系统代码页截断 UTF-8）；2）AI 程序化编辑 Markdown 表格时新增行掉出表格（插到 `---` 节分隔符之后）。REFLECT 记录不足以防止重犯，需要可执行方案。

### 实现

- **scripts/safe_write.py** — 通过 base64 参数安全写入 UTF-8 文件，完全绕过 PowerShell 编码问题
- **scripts/validate_md_tables.py** — 扫描全部 `.md` 文件的表格，检查：表格连续性（无 `---` 打断）、列数匹配、pipe 符号完整性
- **R-DOC-9** — 禁止 `python -c "..."` 含中文，必须用文件执行
- **R-DOC-10** — 改完 Markdown 表格后必须运行 `validate_md_tables.py` 校验

### 采坑记录

#### AI 行为 / 工具链

| 问题 | 原因 | 解决 |
|------|------|------|
| Markdown 表格新增行掉出表格（如 R-DOC-9 被 --- 隔开） | 程序化插入行时定位在 `---` 节分隔符之后而非之前 | 插入表格行前先找到该节的 `---` 结束标记，插入到 ***前面***；插入后立即重读验证整表连续性。另新增 R-DOC-10 + validate_md_tables.py 硬检查 |

### 当前状态

- 仓库 total: 15.76% 覆盖率，193 文件（含零覆盖的框架/配置类文件）
- 前端: ~73% 语句覆盖，77 测试文件
- 后端: ~71% 综合覆盖，27 测试文件
- 新 commit 上传后需等待 Codecov 后台处理


## 2026-07-23：AI Gateway 接入 Function Calling / Tool Use

### 动机

让 AI 助手从纯文本聊天升级为**能用工具的 Agent**。用户提问后，AI 可根据需要调用平台内置工具（搜索视频、查详情、看评论等），再综合结果回答问题。

### 架构决策

| 决策 | 选型 | 理由 |
|------|------|------|
| Tool 定义位置 | `internal/aigateway/toolkit/` 独立子包 | 解耦，后续 MCP Server 可复用同一套 Executor |
| 执行模式 | 多轮工具链，最多 5 轮 | 模型可连续调多个工具，兼顾灵活性与安全性 |
| History 策略 | 全量存（含 tool_call/tool_result 中间消息） | 保证上下文完整，避免模型在后续对话中重复调用 |
| 前端展示 | 聊天气泡间内嵌 Trace 行 | 用户可感知工具调用过程，无 emoji，现代感 |
| WebSocket 协议 | `tool_call_start` / `tool_call_end` 新类型 | 与最终回复解耦，前端独立渲染 |
| trace_id | UUID 前 8 位，贯穿日志 + 前端 | 面试亮点：同一 trace_id 既写 Zap 日志又推前端 |

### 采坑记录

| 问题 | 原因 | 解决 |
|------|------|------|
| PowerShell 写 Go 文件时反引号转义 | Go struct tag 用反引号，PowerShell 也用反引号转义 | 用 `[char]0x60` 变量存反引号，配合字符串拼接写入 |
| DeepSeek function calling 格式差异 | 初期不确定 DeepSeek 是否完全兼容 OpenAI tool format | 已验证：DeepSeek 使用与 OpenAI 完全相同的 `tools`/`tool_calls` 格式 |
| search 返回类型不匹配 | `search.Client.SearchAll` 返回 `AllResult`，字段名与预期不同 | `res.Result.Video` 获得 `[]VideoHit`，`v.Aid` 为视频 ID |
| `ChatHub` 无 `BroadcastJSON` 方法 | 错误调用了不存在的方法 | 改用 `PushJSON(userID, v)` 推送给指定用户 |

## 2026-07-23：Tool Result 前端卡片展示 + WebSocket 并发写入崩溃排查

### 动机

工具调用结果只以纯文本摘要传到前端，用户看不到视频封面、评论头像、弹幕时间点等结构化信息。需要：
1. **后端**：工具返回时附带结构化字段（封面、作者、用户信息）
2. **前端**：工具调用气泡（持久展示）+ 结构化结果卡片（AI 回复内联）
3. **修复运行时崩溃**：AI 对话中偶发 `panic: concurrent write to websocket connection`

### 实现

**后端**（`internal/aigateway/toolkit/platform.go`）：

- `searchVideos` / `getTrending`：增加 `cover_url`、`uploader_name`
- `getVideoComments`：增加 `user_name`、`user_avatar`
- `getVideoDanmaku`：增加 `user_name`
- `getVideoDetail`：格式统一为 `{"items": [...]}`

**后端**（`internal/aigateway/gateway.go` + `internal/service/agent.go`）：

- 新增 `OnToolResultData` 回调，工具返回结果含 `items` 字段时触发
- 发送 `tool_result_data` WS 事件

**前端**（`MbDmChatPanel.vue`）：

- 新增 `_pendingToolActs` / `_pendingResultData` 数据缓冲
- `onChatWsPayload` 处理 `tool_call_start` / `tool_call_end` / `tool_result_data` 事件
- AI 回复消息携带 `_toolActivities` + `_toolResultData` 字段
- 在 AI 气泡上方渲染工具调用活动行（↻ / ✓ + 耗时）
- 在 AI 气泡下方渲染结构化结果卡片（视频封面列表、评论/弹幕列表）

**WebSocket 并发写入修复**（`internal/ws/chathub.go`）：

- 给每个 `*websocket.Conn` 绑定 `*sync.Mutex`（存 `sync.Map`）
- `PushRaw` 写入前 Lock，写入后 Unlock，彻底杜绝并发写入 panic

### 采坑记录

| 问题 | 原因 | 解决 |
|------|------|------|
| WebSocket 并发写入 panic: `concurrent write to websocket connection` | `executeToolCalls` 用 goroutine 并发执行多个工具，各 goroutine 的 `OnToolCallStart` / `OnToolCallEnd` / `OnToolResultData` 回调都调用 `PushJSON` → `PushRaw` → `c.WriteMessage`，gorilla/websocket 禁止并发写同一个连接 | 给每个 `*websocket.Conn` 绑定 `sync.Mutex`，`PushRaw` 写入前 Lock 后 Unlock |
| `m` 在模板中不在作用域内 | Vue 3 中 `v-for="m in grp.messages"` 作用于 `<div>` 及其子元素，工具活动/结果卡片是该 `<div>` 的同级元素，`m` 不暴露给同级 | 用内层 `<template v-for="m in grp.messages" :key="m.id">` 包裹消息气泡 + 工具块 |
| Vue SFC 解析错误 `Unexpected token, expected ","` | `data()` 中 `_pendingResultData: {}` 漏了尾逗号 | 补逗号 |
| PowerShell 不支持 heredoc | 多次尝试 `python << 'EOF'` 失败，PowerShell 无此语法 | 改由 Node.js MCP 写 Python/JS 脚本到文件再执行，或用 `python -c` + base64 |
| `lines.splice` 插入偏移计算失误 | 插入 34 行后忘记原元素位置已偏移，把工具块插入到 `</template>` 之后而非之前 | 调试时逐段打印数组状态验证 |
| `onChatWsPayload` 方法被错误嵌入 `ws.onmessage` | 搜索 `onChatWsPayload(data)` 找到了 `this.onChatWsPayload(data);` 调用行而非函数定义行，把定义体替换进了调用位置 | 加 `!lines[i].includes("this.")` 排除调用行，准确定位函数定义 |
| CRLF vs LF 行尾不一致 | `git show` 输出是 LF，而项目用 CRLF，多次替换后混用导致编译错误 | 最终统一用 `lines.join("\r\n")` 输出 |
| `};,` 语法错误 | 方法体内 `ws.onmessage = ev => { ... };` 末尾加了分号后又加逗号，JS 对象方法间不能用分号 | 保持原文的 `};}`, 然后在方法间用 `},` |

### 与 Minibili.md 的对照

- **「AI 助手不仅仅是聊」**：本次实现了完整的工具执行链路 + 前端感知展示，面试时可从 WS 协议设计 → 并发写入锁 → 前端卡片渲染完整展开
- **「用工具的核心痛点」**：并发写入 panic 是真实线上问题，面试中聊 `gorilla/websocket` 的并发模型是个加分点

## 二次修复：WS 重连 + 工具链持久化 + 实时展示（2026-07-23 续）

工具卡片上线后用户反馈三个问题：

1. **搜索偶尔卡死**：一直显示"AI 正在输入..."，刷新才显示回复
2. **工具调用不可见**：直到 AI 回复到达前用户不知道在发生什么
3. **刷新丢失工具链**：刷新页面后消息在，但工具调用气泡和结果卡片全部消失

### 根因分析

| 问题 | 根因 | 修复 |
|------|------|------|
| "AI 正在输入..." 永久卡死 | `connectChatWs()` 只设了 `ws.onmessage`，`onclose` / `onerror` 全都没设。后端重启或网络抖动后 WS 永久断开，所有 `tool_call_*` / `dm_message` 事件全部丢失，前端 `chatAwaitingAgent` 一直 true（120s 超时后才清） | 添加 `ws.onclose` / `onerror` 处理器，指数退避重连 1s→2s→4s→...→30s；发送前检查连接状态，断连时自动触发重连 |
| 工具调用对用户不可见 | `_pendingToolActs` 只在下一条 `dm_message` WS 事件到达时才附着到消息对象上。如果 WS 先断连后重连，`dm_message` 和 `tool_call_*` 事件可能在两条不同的 WS 连接上到达，导致工具链永久丢失 | 新增独立实时缓冲区 `_liveToolActs`，`tool_call_start` 立即推入，在 AI 打字提示上方独立区域展示（黄色背景卡片），用户即刻可见 |
| 刷新后工具链丢失 | `_toolActivities` / `_toolResultData` 只存在 Vue 组件内存中，REST API `ListDmMessages` 不返回这些字段，HTTP 加载的消息对象上没有工具数据 | `dm_message` 到达时将工具链写入 `sessionStorage`（key: `mb_tool_acts_{convId}_{msgId}`）；`chatMessageGroups` 计算时优先从 `sessionStorage` 读取，回退到 `raw._toolActivities` |

### 更深的教训

1. **WS 通信模式比 HTTP 脆弱得多**，HTTP 失败可以简单重试，WS 断开后所有状态（pending tool calls、等待中的回复）全部丢失。前端的首要防御不是优雅地处理 WS 消息，而是**确保 WS 不断**。
2. **工具调用是一个「有状态过程」**，不能把全部信息放在一条最终消息上。应该让每个工具调用事件自包含、可独立展示、可独立恢复。
3. **持久化不只在服务端**。前端 `sessionStorage` 是零成本的前置缓存，能解决绝大多数"刷新丢状态"的问题。不应该每个字段都等后端加接口。

### 后续建议

- 服务端在 HTTP `ListDmMessages` 响应中附加工具数据，彻底消除对 `sessionStorage` 的依赖
- 前端 WS 连接状态可视化（连接中/已断开/重连中），让用户感知到网络状态
- 考虑 `WebSocket heartbeat` 机制，更快发现断连

## 三次修复：工具状态错误显示 ↻ 转圈（2026-07-23 续二）

修复 WS 重连后用户反馈新问题：消息下方的工具活动列表里有些工具持线显示 ↻（运行中），但结果卡片已经渲染出来了。

### 根因

`tool_call_end` 和 `dm_message` 在竞争同一把 WebSocket 写锁。后端 `executeToolCalls` 用 goroutine 并发执行工具，每个 goroutine 调用 `OnToolCallEnd` → `PushJSON` → `PushRaw`（加锁写 WebSocket）。而 `dm_message` 在 `CompleteUserTurnWithTools` 返回后在 handler goroutine 中同样调用 `PushJSON` → `PushRaw`（竞争同一把锁）。

由于 Go goroutine 调度不确定性，`dm_message` 可能在部分工具 goroutine 的 `tool_call_end` 获取到锁之前就拿到了锁并写入 WS，导致前端收到的事件顺序为：

```
tool_call_end A → dm_message → tool_call_end B → tool_call_end C
```

此时前端 `_pendingToolActs` 中 B 和 C 还是 "running" 状态就被附着到了消息上。

### 修复

在 `dm_message` 到达时，遍历 `_pendingToolActs` 将所有仍为 "running" 的项标记为 "done"。这是安全的——`dm_message` 到达意味着 LLM 已经拿到了所有工具的结果，所有工具事实上已经完成。

```javascript
this._pendingToolActs.forEach(t => { if (t.status === "running") t.status = "done"; });
```

### 教训

1. **WebSocket 不是事务性通道**。即使后端在函数返回前发了所有事件，goroutine 间的锁竞争会导致接收端顺序与发送端顺序不同。
2. **状态机的前端状态需要容错**。`dm_message` 是「本轮处理已结束」的强信号，应当以此为准修正所有中间状态。
3. **更好的方案**：后端用单线程的事件队列序列化所有 WS 事件，彻底消除乱序可能。


## 2026-07-24 — 弹幕结构化结果头像修复

### 问题
前端弹幕（danmaku）结构化结果卡片中，用户头像一直显示为破损图片（broken image icon），而评论（comment）卡片头像正常显示。

### 根因分析
1. **后端 `getVideoDanmaku` 未填充 `user_avatar`**：虽然 item struct 已定义 `UserAvatar` 字段，但在构建 items 时未从数据库查询并赋值，导致 JSON 响应始终返回 `"user_avatar":""`。
2. **前端缺少图片加载失败兜底**：`defaultFace` SVG data URL 作为 `||` 回退理论上应生效，但缺少 `@error` 事件兜底，部分浏览器或特定环境下 data URL SVG 渲染失败后无后备方案。

### 对比 `getVideoComments`
评论接口正确实现了头像查询：
- 使用 `userMap := make(map[uint64]*model.User)` 存储完整 User 对象
- `userAvatar = u.AvatarURL` 赋值
- 构建 items 时传入 `UserAvatar: userAvatar`

### 修复
1. **后端** (`internal/aigateway/toolkit/platform.go`):
   - 新增 `avatarMap := make(map[uint64]string)` 并行存储用户头像 URL
   - 在用户查询循环中填充 `avatarMap[u.ID] = u.AvatarURL`
   - 构建 danmaku items 时传入 `UserAvatar: userAvatar`
2. **前端** (`MbDmChatPanel.vue`):
   - 为头像 `<img>` 添加 `@error="onAvatarError"` 事件处理
   - `onAvatarError` 方法在图片加载失败时将 `src` 替换为 `defaultFace` SVG data URL

### 教训
- 「相同的 template 分支、评论正常弹幕异常」应优先排查 **数据源** 而非渲染层
- Go 结构体中定义了字段但不赋值，JSON 序列化后为空字符串 `""`，在前端 `||` 表达式中空字符串为 falsy 所以理论上回退应生效——但图片 loading 失败仍需要 `@error` 作为最终防线
- 多层兜底策略：真实 URL → `||` 默认值 → `@error` 兜底


## 2026-07-24 - 弹幕点击跳转时序修复（Vue 组件挂载时序 bug）

### 问题

点击弹幕结构化结果卡片跳转到视频播放页后，视频不会跳转到弹幕指定的播放时间点。但先点击侧栏推荐视频切换到其他视频，再回退回来，跳转却正常。

### 根因分析

这是 **Vue 组件生命周期与异步数据加载之间的时序 bug**，涉及三个环节的断裂：

**环节一：VideoPlayerBox 首次挂载时 seekTo watcher 不触发**

Vue 3 的 watch 默认不带 immediate: true，只在响应式数据变化时触发。组件首次挂载时 seekTo prop 已经有值，但对 watcher 来说没有从旧值变成新值的过程，所以 watcher 函数体完全不执行。_pendingSeek 保持初始值 0。

**环节二：videoSrc 异步加载后才变化**

video.vue 的 aidParam watcher 调用 syncMinibiliDetail()（async），该函数发起 HTTP 请求获取视频详情。在此期间 media-src prop 为空字符串，VideoPlayerBox 使用 fallback demoSrc。等到 API 响应返回后才设置真实 media-src，videoSrc watcher 触发，但此时 _pendingSeek 为 0（环节一未设置），seek 条件判断失败。

**环节三：aidParam watcher 不响应仅 query 变化**

当在相同视频 ID 下通过 ?t=seconds 导航（从侧栏回退回来），aidParam 没有变化（aid 参数相同），其 watcher 不触发。没有独立的 watcher 监听 route.query.t，_seekTime 不会更新。

### 修复（3 处变更）

| 文件 | 变更 |
|------|------|
| video.vue | 新增 "$route.query.t" watcher，当仅 query 变化时更新 _seekTime |
| VideoPlayerBox.vue mounted() | 添加 if (this.seekTo > 0) this._pendingSeek = this.seekTo（首次挂载时赋值） |
| VideoPlayerBox.vue seekTo watcher | 仅设 _pendingSeek + 视频已加载时直接 currentTime；不再调用 _doSeek |
| VideoPlayerBox.vue videoSrc watcher | 成功 seek 后 _pendingSeek = 0 清理（防残留） |
| VideoPlayerBox.vue minibiliVideoId watcher | 添加 _pendingSeek = 0（视频 ID 切换时清理） |

### 教训

1. **Vue watch 默认不立即执行**。需要首次挂载时处理的 prop 必须在 mounted() 中额外赋值，或使用 immediate: true（注意 immediate watcher 在 beforeMount 阶段触发，此时 refs 不可用）。
2. **异步数据流转的时序断点**。一个 prop 的变化可能依赖另一个 prop 的异步加载结果。必须确保在异步完成前中间状态正确持有待消费的值（_pendingSeek）。
3. **Vue Router query 参数变化不触发 route param watcher**。aidParam computed 只依赖 route.params.aid，不依赖 route.query。需要独立的 watcher。
4. **测试建议**：手动测试至少覆盖三种场景——（1）从其他页面首次导航到视频页（mount 场景）；（2）同一视频页内 query 变化；（3）不同视频间切换后回退。
## 当前状态

- 仓库 total: 15.76% 覆盖率，193 文件（含零覆盖的框架/配置类文件）
- 前端: ~73% 语句覆盖，77 测试文件
- 后端: ~71% 综合覆盖，27 测试文件
- 新 commit 上传后需等待 Codecov 后台处理



## 后续展望

- **Redis Stream / 消费组**：若弹幕量级继续增长，可引入消费组做削峰填谷，或将 WebSocket 网关拆为独立进程。
- **Kratos 微服务拆分**：v1.0 的领域分层（`handler/` → `service/` → `dao/`）已预留拆分空间，后续可按需拆出用户/视频/弹幕/评论服务。
- **E2E 自动化测试**：当前 E2E 为手动，可引入 Playwright 做端到端回归验证。
- **前端测试补充**：当前测试集中在后端，Vue 组件层（弹幕渲染、发布表单）尚缺自动化测试覆盖。
---

