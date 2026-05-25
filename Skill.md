
## Mini-Bili v1.0 技能手册（Skill）

**版本**：v1.0
**最后更新**：2026-05-11
**依赖文档**：Mini-Bili v1.0 SPEC、Mini-Bili v1.0 Rule

### 关于 Skill 的说明

本文档是项目的"标准操作手册"，告诉 AI 某些固定动作具体应该怎么执行。Skill 的存在是为了避免 AI 临场发挥、每次用不同的方式做同一件事。

Rule 说"这件事必须做"，Skill 说"这件事这样做"。

---

### S-001：编译验证

**对应 Rule**：R-DEV-1（改完代码必须可运行）

**触发条件**：每次代码修改完成后，必须执行本 Skill。

**执行步骤**：

1. 在项目根目录下依次执行：
   ```go
   go mod tidy
   go build -o ./bin/mini-bili ./cmd/
   ```
   `go mod tidy` 必须在 `go build` 之前执行，确保 `go.mod` 和 `go.sum` 与当前代码中的 import 一致。
2. 检查编译输出：
   - 若 `go build` 退出码为 0 且无任何 error 级别 stderr 输出 → 编译通过。
   - 若退出码非 0 或有 error 级别输出 → 编译失败。
3. 编译失败时：
   - 读取完整的编译错误信息。
   - 定位第一个错误（而非最后一个），修复它。
   - 修复后从步骤 1 开始重新执行。
   - 若同一错误修复 3 次仍未通过，停止并向人报告具体错误信息和已尝试的修复步骤。
4. 编译通过后，确认 `./bin/mini-bili` 文件已生成且可执行。

**禁止行为**：
- 严禁跳过 `go mod tidy` 直接执行 `go build`。
- 严禁跳过编译直接声称"代码没问题"。
- 严禁使用 `go run` 代替 `go build` 作为编译验证。
- 严禁在编译失败时修改无关代码来"碰运气"。

---

### S-002：数据库迁移

**对应 Rule**：R-DB-3（数据库结构变更必须通过迁移脚本）、R-DB-4（核心字段必须建索引）

**触发条件**：任何涉及新增表、修改表结构、新增索引的操作，必须执行本 Skill。

**执行步骤**：

1. 确认 GORM AutoMigrate 已正确配置在 `internal/data/` 目录下的数据层初始化代码中。
2. 在对应的模型结构体（Model）中定义或修改字段及标签（tag）。**定义 `play_count` 字段时，必须显式设置 `default:0` 标签**（如 `gorm:"default:0"`），防止新视频因 NULL 值排序问题意外出现在首页。
3. 将新增或变更的模型注册到 AutoMigrate 的迁移列表中：
   ```go
   db.AutoMigrate(
       &User{},
       &Video{},
       &Danmaku{},
       &Comment{},
       &Notification{},
       // 在此处追加新增模型
   )
   ```
4. 启动应用，检查日志确认 AutoMigrate 执行成功：
   - 成功标志：日志中出现 GORM 的 `AutoMigrate` 完成信息，无 error 日志。
   - 失败标志：日志中出现数据库错误（如权限不足、字段类型冲突）。
5. 迁移失败时：
   - 读取完整错误日志。
   - 回滚模型定义至迁移前状态。
   - 分析失败原因并修正模型定义后重试。

**数据安全底线**：
- 严禁在生产环境使用 `db.Migrator().DropTable` 或 `db.Exec("DROP TABLE")` 等破坏性语句。
- 所有结构变更必须是**增量式**的（仅允许 `Add Column`、`Modify Column` 扩大长度/允许为空、`Create Index`）。
- 严禁直接修改已有字段的数据类型导致数据丢失或截断（如 `VARCHAR(50)` → `VARCHAR(20)`）。
- 若确实需要收缩字段长度、删除字段或重命名字段，必须先向人征得明确同意。

**禁止行为**：
- 严禁直接在数据库中手动执行 `CREATE TABLE` 或 `ALTER TABLE`。
- 严禁在业务代码（如 handler/service 层）中调用建表语句。
- 严禁跳过 AutoMigrate 直接假设表结构已存在。

---

### S-003：日志初始化

**对应 Rule**：R-OBS-1（严禁使用 fmt.Println 打印日志）、R-OBS-2（日志必须区分级别）

**触发条件**：项目初始化，或任何需要替换/重新配置日志模块时，必须执行本 Skill。

**执行步骤**：

1. 在 `go.mod` 中引入 `go.uber.org/zap` 依赖：
   ```go
   go get -u go.uber.org/zap
   ```
2. 在 `internal/logger/` 目录下创建日志初始化文件。
3. 使用以下标准配置初始化 zap：
   ```go
   import "go.uber.org/zap"
   import "go.uber.org/zap/zapcore"
   
   func InitLogger() *zap.Logger {
       config := zap.NewProductionConfig()
       config.EncoderConfig.TimeKey = "timestamp"
       config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
       logger, _ := config.Build()
       return logger
   }
   ```
4. 日志注入方式（**以下两步必须全部执行，不可只做其一**）：
   - **Gin 中间件注入**：在 Gin 路由初始化时，通过自定义中间件将 `*zap.Logger` 实例写入 `*gin.Context`：
     ```go
     func LoggerMiddleware(logger *zap.Logger) gin.HandlerFunc {
         return func(c *gin.Context) {
             c.Set("logger", logger)
             c.Next()
         }
     }
     ```
   - **全局变量兜底**：在 `internal/logger/` 包中暴露一个包级变量 `L`，供非 HTTP 上下文场景（如 RabbitMQ 消费者、定时任务）使用：
     ```go
     var L *zap.Logger
     
     func Init() {
         L = InitLogger()
     }
     ```
5. 业务日志调用方式：
   - HTTP handler 中：`c.MustGet("logger").(*zap.Logger).Info("用户登录成功", zap.String("username", username))`
   - 非 HTTP 场景中：`logger.L.Error("数据库连接失败", zap.Error(err))`
6. 确认项目中不存在任何 `fmt.Println` 或 `fmt.Printf`（除 `main.go` 中启动前的临时调试，提交前必须删除）。

**禁止行为**：
- 严禁在生产代码中使用 `fmt.Println` 或 `fmt.Printf` 输出日志。
- 严禁所有日志使用同一级别（如全部用 Info 输出错误信息）。
- 严禁直接使用 `zap.NewExample()` 或 `zap.NewDevelopment()` 作为生产环境配置（必须用 `NewProductionConfig`）。
- 严禁只做全局变量注入而跳过 Gin 中间件注入，或反之。

---

### S-004：转码重试

**对应 Rule**：R-DEV-4（转码失败重试次数限制）

**触发条件**：视频转码任务执行失败时，必须执行本 Skill。

**执行步骤**：

1. 捕获 FFmpeg 转码失败的错误信息和退出码。
2. **错误分类（必须在重试前执行）**：
   - 读取 FFmpeg stderr 输出。
   - 若包含以下特征，判定为**永久性错误**，**立即标记 `failed`，不进行重试**：
     - `Invalid data found when processing input`（源文件损坏）
     - `Unsupported codec`（编码不支持）
     - `No such file or directory`（源文件丢失）
     - `Permission denied`（权限不足）
   - 永久性错误需将完整的 FFmpeg 错误输出写入失败原因字段。
3. 若不属于永久性错误，检查当前重试次数：
   - 从任务队列的消息属性中读取当前重试计数（初始为 0）。
4. 若重试次数 < 3：
   - 重试计数器 +1。
   - 等待 `30秒 × 当前重试次数` 后重新投递任务（即第1次等30秒，第2次等60秒，第3次等90秒）。
   - 将更新后的重试计数写入任务消息，通过 RabbitMQ 重新入队。
5. 若重试次数 = 3（即已重试3次且最后一次也失败）：
   - 更新视频状态为 `failed`，同时写入失败原因字段（F2-b）。
   - **关键**：确保该记录对上传者可见，但**不对公共列表（首页 `/` 路径）可见**。必须严格遵守 SPEC F2-b 的可见性规则：`published` 状态的视频才对全站可见，`failed` 状态的视频仅上传者本人可见。首页视频列表数据源仅限 `published` 视频（F10），不得混入 `failed` 视频。
   - 不再重新入队。

**禁止行为**：
- 严禁对永久性错误进行重试。
- 严禁无限循环重试。
- 严禁在重试时不更新重试计数。
- 严禁在标记 `failed` 时使用模糊原因（如"转码失败"），必须包含 FFmpeg 的具体错误输出。
- 严禁将 `failed` 状态的视频出现在首页公共视频列表中。

---

### S-005：封面校验

**对应 Rule**：R-BIZ-3（封面图必须校验格式和大小）

**触发条件**：用户上传视频附带 `cover` 文件时（F2），或通过封面修改接口上传新封面时（F3），必须执行本 Skill。

**执行步骤**：

1. 读取上传文件的文件名后缀，提取扩展名（转为小写）。
2. 校验扩展名是否属于允许集合：
   - 允许：`.jpg`、`.jpeg`、`.png`、`.gif`、`.bmp`、`.webp`
   - 若扩展名不在此集合内，立即返回错误，错误码使用 S-006 中注册的 `40002`。
3. 校验文件大小：
   - 读取 `multipart.FileHeader.Size` 获取文件大小（字节）。
   - 若文件大小 > 10MB（即 > 10485760 字节），立即返回错误，错误码使用 S-006 中注册的 `40003`。
4. 格式和大小均通过后，继续后续处理流程（保存至本地临时目录，进入转码队列）。

**禁止行为**：
- 严禁仅在前端校验，禁止不经过后端校验直接存储或使用文件。
- 严禁格式错误和大小错误使用相同的错误码（必须分别使用 40002 和 40003）。
- 严禁不符合要求时仅打印日志而继续处理。

**扩展（F0 用户头像）**：扩展名集合与步骤 1–2 **相同**；单文件大小上限为 **5 MB**，错误码 **40015 / 40016**（Rule **R-BIZ-8**）。实现使用 `internal/pkg/coverval.ValidateAvatarHeader`。

---

### S-006：错误码注册

**对应 Rule**：R-API-1（统一响应格式）、R-DEV-3（错误必须提供明确信息）

**触发条件**：项目中需要新增任何业务错误码时，必须执行本 Skill。

**错误码分配表**（当前已注册）：

| 错误码  | 常量名                 | 消息模板                                         |
| :------ | :--------------------- | :----------------------------------------------- |
| `0`     | `CodeSuccess`          | `"ok"`                                           |
| `40001` | `CodeParamError`       | `"参数错误"`                                     |
| `40002` | `CodeCoverFormat`      | `"封面格式不支持，仅支持 JPEG/PNG/GIF/BMP/WEBP"` |
| `40003` | `CodeCoverSize`        | `"封面大小超过 10MB，请压缩后重新上传"`          |
| `40004` | `CodeDanmakuCooldown`  | `"发送过于频繁，请稍后再试"`                     |
| `40005` | `CodeDanmakuSensitive` | `"弹幕内容包含违规信息"`                         |
| `40006` | `CodeUsernameExists`   | `"用户名已存在"`                                 |
| `40007` | `CodeMultipartParseError` | `"multipart 请求解析失败，请检查网络或稍后重试"` |
| `40008` | `CodeUploadMissingFile` | `"未收到视频文件，请重新选择文件后再提交"`      |
| `40009` | `CodeVideoProbeFailed` | `"无法解析视频：请确认文件为有效视频；服务器 PATH 中需有 ffprobe，或在环境变量 FFPROBE_PATH 中填写其绝对路径"` |
| `40010` | `CodeVideoDurationExceeded` | `"视频时长超过 30 分钟上限"`                |
| `40011` | `CodeVideoFileTooLarge` | `"视频文件超过 500 MB 上限"`                    |
| `40012` | `CodeTitleInvalid`     | `"标题须为 1–80 个字"`                           |
| `40013` | `CodeIntroTooLong`     | `"简介不能超过 2000 个字"`                       |
| `40014` | `CodeInvalidColor`     | `"弹幕颜色格式无效，请输入有效的十六进制色号（如 #FF0000）"` |
| `40015` | `CodeAvatarFormat`     | `"头像格式不支持，仅支持 JPEG/PNG/GIF/BMP/WEBP"` |
| `40016` | `CodeAvatarSize`       | `"头像大小超过 5MB，请压缩后重新上传"` |
| `40100` | `CodeUnauthorized`     | `"未登录或 Token 已过期"`                        |
| `40101` | `CodeInvalidLogin`     | `"用户名或密码错误"`                             |
| `40300` | `CodeForbidden`        | `"无权限执行此操作"`                             |
| `40301` | `CodePasswordMismatch` | `"原密码错误"`                                   |
| `40400` | `CodeNotFound`         | `"资源不存在"`                                   |
| `50000` | `CodeInternalError`    | `"服务器内部错误"`                               |

**新增错误码的注册步骤**：

1. 在 `internal/errcode/` 目录下的错误码定义文件中，按上述表结构新增常量。
2. 错误码编号规则：
   - `40001-40099`：参数校验类错误
   - `40100-40199`：认证/授权类错误
   - `40300-40399`：权限类错误
   - `40400-40499`：资源不存在类错误
   - `50000-50099`：服务器内部错误
3. 新增后必须同步更新本文档的错误码分配表。

**禁止行为**：
- 严禁不同业务场景复用同一错误码。
- 严禁在代码中直接写死错误消息字符串（必须通过错误码映射表获取）。
- 严禁新增错误码不更新本表。

---

### S-007：弹幕冷却校验

**对应 Rule**：R-BIZ-1（弹幕冷却必须双端校验）

**触发条件**：用户发送弹幕请求时，必须执行本 Skill。

**执行步骤**：

1. 从 JWT Token 中提取当前用户 ID。
2. 从请求参数中提取目标视频 ID。
3. 构造 Redis 键：`danmaku:cooldown:{user_id}:{video_id}`。
4. 查询该键是否存在：
   - 若存在（未过期），返回错误 `40004`（发送过于频繁），拒绝本次请求。
   - 若不存在，继续第 5 步。
5. 在 Redis 中设置该键，过期时间（TTL）为 **5 秒**。
6. 正常处理弹幕发送逻辑（S-014 颜色格式校验 → 敏感词过滤 → 入库、广播）。

**注意**：
- 步骤 4 和步骤 5 必须使用 Redis 的原子操作（`SETNX` 或 `SET NX EX`），防止并发场景下的竞态条件。
- 前端按钮变灰和倒计时是**用户体验辅助**，后端校验是**唯一安全门禁**。仅在步骤 5 成功后才能认为冷却校验通过。

**禁止行为**：
- 严禁仅依赖前端冷却逻辑。
- 严禁使用非原子操作导致并发绕过冷却。

---

### S-008：评论删除级联

**对应 Rule**：R-BIZ-2（评论删除必须级联删除）、R-BIZ-6（评论删除权限必须校验）

**触发条件**：用户请求删除评论时，必须执行本 Skill。

**执行步骤**：

1. 从 JWT Token 中提取当前用户 ID。
2. 根据评论 ID 查询数据库，获取该评论的 `user_id`（评论发布者）和所属视频的 `uploader_id`（UP主）。
3. **权限校验**（先校验权限，再执行删除）：
   - 若当前用户 ID = 评论发布者 ID → 有权限。
   - 若当前用户 ID = 所属视频的 UP主 ID → 有权限。
   - 否则 → 返回错误 `40300`（无权限），拒绝请求。
4. **查询所有子评论**：在数据库中查询 `parent_id` = 当前评论 ID 的所有评论。若子评论还有子评论，递归查询直到无更多后代。
5. **在事务中执行删除**：
   - 开启数据库事务。
   - 将所有待删除的评论 ID（父评论 + 所有后代）批量 DELETE。
   - 提交事务。
   - 若事务失败，回滚并返回错误 `50000`。
6. **删除完成后**：
   - 事务提交成功后，服务端通过该视频的 WebSocket 房间广播删除事件，消息格式为：
     ```json
     {"type": "comment_deleted", "comment_id": "<被删除的评论ID>"}
     ```
   - 前端收到该事件后，**直接从 DOM 中移除**对应的评论节点及其所有子评论节点，不显示任何占位文本。

**禁止行为**：
- 严禁只删除父评论而不删除子评论。
- 严禁在事务外执行删除操作。
- 严禁跳过权限校验直接删除。
- 严禁使用逻辑删除（软删除）代替物理删除（SPEC F7 要求"所有数据从数据库中移除"）。

---

### S-009：Token 颁发刷新失效

**对应 Rule**：R-AUTH-1（Token 安全策略）

**触发条件**：用户登录成功时颁发 Token，或用户使用 Refresh Token 刷新时，必须执行本 Skill。

**执行步骤**：

**A. 登录时颁发**：

1. 用户登录成功后，生成唯一 Token ID（UUID）。
2. 生成 **Access Token**：
   - Payload 包含：`user_id`、`token_id`、`type: "access"`
   - 过期时间：当前时间 + **2 小时**
   - 使用 HS256 签名，密钥从环境变量 `JWT_SECRET` 读取。
3. 生成 **Refresh Token**：
   - Payload 包含：`user_id`、`token_id`、`type: "refresh"`
   - 过期时间：当前时间 + **3 天**
   - 使用 HS256 签名，密钥从环境变量 `JWT_SECRET` 读取。
4. 将 Access Token 和 Refresh Token 一并返回给客户端。

**B. 刷新时**：

1. 接收客户端传来的 Refresh Token。
2. 校验 Refresh Token 的签名和过期时间：
   - 若签名无效或已过期 → 返回 `40100`，要求重新登录。
3. 从 Refresh Token 中提取 `user_id` 和 `token_id`。
4. 在 Redis 中检查该 Refresh Token 是否已被标记为失效：
   - Redis 键：`refresh_token:invalid:{token_id}`
   - 若存在 → 该 Refresh Token 已被使用过，返回 `40100`（可能被盗用），并要求重新登录。
5. 将当前 Refresh Token 的 `token_id` 标记为失效：
   - Redis 键：`refresh_token:invalid:{token_id}`，TTL 设为 **3 天**（与原 Refresh Token 有效期一致）。
6. 生成新的 `token_id`，按"登录时颁发"的流程生成新的 Access Token 和 Refresh Token，返回给客户端。

**禁止行为**：

- 严禁 Refresh Token 用于业务 API 访问（只能调用刷新接口）。
- 严禁刷新成功后不标记旧 Refresh Token 为失效。
- 严禁 Access Token 有效期超过 2 小时。
- 严禁 Refresh Token 有效期超过 3 天。
- 严禁在 Access Token 中存储敏感信息（如密码）。

---

### S-010：点赞通知聚合

**对应 Rule**：SPEC F9（评论点赞与通知）

**触发条件**：用户对评论点赞时，必须执行本 Skill。

**执行步骤**：

1. 点赞操作成功后（点赞数已更新），获取被点赞评论的 `comment_id` 和评论发布者 `comment_owner_id`。
2. 若点赞者 = 评论发布者（自己赞自己），跳过通知，直接结束。
3. 在通知表中查询是否存在满足以下条件的未读通知：
   - `recipient_id` = `comment_owner_id`
   - `type` = `"like_aggregation"`
   - `related_id` = `comment_id`
   - `is_read` = `false`
4. 若存在：
   - 将当前点赞者的用户名追加到该通知的 `sender_names` 字段（JSON 数组）。
   - 更新 `total_likes` = `total_likes + 1`。
   - 更新 `updated_at` 为当前时间。
   - **不创建新通知**。
5. 若不存在：
   - 创建新通知记录：
     - `recipient_id` = `comment_owner_id`
     - `type` = `"like_aggregation"`
     - `related_id` = `comment_id`
     - `sender_names` = `["点赞者用户名"]`
     - `total_likes` = 1
     - `comment_preview` = 被点赞评论的**前 15 个字符**
     - `is_read` = `false`
6. 通知展示时（前端或 API 返回），按以下规则格式化：
   - `total_likes` = 1 → "用户A 赞了你的评论"
   - `total_likes` = 2 → "用户A、用户B 赞了你的评论"
   - `total_likes` ≥ 3 → "用户A、用户B、用户C 等X人赞了你的评论"（展示前 3 个用户名，X = total_likes）

**禁止行为**：
- 严禁每个点赞都创建独立通知（必须聚合）。
- 严禁通知中不附带评论预览。
- 严禁对同一个评论存在多条未读的点赞聚合通知。

---

### S-011：WebSocket 鉴权

**对应 Rule**：R-API-4（WebSocket 连接必须鉴权）

**触发条件**：客户端发起 WebSocket 连接请求时，必须执行本 Skill。

**执行步骤**：

1. 客户端在 WebSocket 连接请求中，必须通过在连接 URL 的查询参数中携带 Access Token：
   ```
   ws://host/ws/danmaku?token=<access_token>
   ```
2. 服务端在 HTTP Upgrade 请求的查询参数中提取 `token`。
3. 校验 Access Token：
   - 校验签名（HMAC-SHA256，密钥为 `JWT_SECRET`）。
   - 校验过期时间（当前时间 < `exp`）。
   - 校验 Token 类型（`type` 必须为 `"access"`，严禁接受 Refresh Token）。
4. 校验失败：
   - WebSocket 握手完成后**立即发送一条错误消息**：`{"type": "auth_failed", "msg": "Token 无效或已过期"}`
   - 发送后**立即关闭 WebSocket 连接**。
5. 校验成功：
   - 将 `user_id` 存入该 WebSocket 连接的上下文中。
   - 正常处理后续弹幕收发逻辑。

**禁止行为**：
- 严禁未校验 Token 就允许 WebSocket 连接持续保持。
- 严禁接受 Refresh Token 作为 WebSocket 鉴权凭证。
- 严禁鉴权失败时仅打印日志而不关闭连接。

---

**S-013：技术决策记录（ADR）规范**

**目的**：防止未来的维护者（包括未来的你和 AI）忘记当初做技术选型的上下文，防止“为了重构而重构”。

**执行时机**：
每当你完成一个完整的业务模块（如“用户认证模块”、“视频上传模块”），在提交代码前，必须执行此步骤。

**操作流程**：
请在根目录下的 `Mini-Bili.md`（项目说明文档）中，追加一段格式如下的记录：

### 模块名称：[例如：双 Token 鉴权机制]

**1. 遇到的问题 **

- [简述当时的技术痛点，例如：如何防止 Refresh Token 被盗用？]

**2. 解决方案 **
- [描述最终采用的方案，例如：引入 Redis 黑名单机制，设置 TTL 与 Token 有效期一致。]

**3. 决策依据（为什么选这个）**
- **方案 A（JWT 黑名单）**：[优点：高性能] / [缺点：需要维护状态]
- **方案 B（数据库轮询）**：[优点：强一致性] / [缺点：性能差，不符合 NF-3]
- **结论**：选择了方案 A，因为本项目是高并发场景，性能优先。

**4. 关联约束**
- 此决策影响 **Rule R-SEC-2**（安全红线）。
- 此逻辑实现在 **Skill S-009**（鉴权流程）。

---

### S-014：弹幕颜色校验

**对应 Rule**：R-BIZ-7（弹幕颜色必须校验十六进制格式）

**触发条件**：用户发送弹幕请求时，在 S-007（弹幕冷却校验）之后、敏感词过滤之前，必须执行本 Skill。

**执行步骤**：

1. 从请求参数中提取 `color` 字段。
2. 若 `color` 为空或未提供，使用默认值 `#FFFFFF`（白色），跳过后续校验。
3. 使用正则表达式 `^#[0-9A-Fa-f]{6}$` 校验 `color` 格式：

```go
import "regexp"

var colorRegex = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
if !colorRegex.MatchString(color) {
    // 返回错误码 40014
}
```

4. 格式校验通过后，将 `color` 存储（统一转为大写）。

**禁止行为**：

- 严禁跳过颜色格式校验。
- 严禁对非法颜色进行默认填充或静默修正（必须明确拒绝并返回错误）。
- 严禁在数据库中存储未经校验的颜色值。

**前端配合**：

- 前端应在弹幕发送框提供颜色选择器（`<input type="color">`），让用户可视化选择任意颜色。
- 前端也应对用户手动输入的色号进行格式预校验，提升用户体验。
