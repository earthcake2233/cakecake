<p align="center">
  <a href="Skill.md">
    <img src="https://img.shields.io/badge/🇨🇳中文-999999?style=flat-square" alt="中文">
  </a>
  <strong><img src="https://img.shields.io/badge/🇬🇧English-00a1d6?style=flat-square" alt="English"></strong>
</p>

  </a>
  </a>
</p>

  </a>
</p>

# Mini-Bili v1.0 Skills Manual (Skill)

**Version**: v1.0
**Last Updated**: 2026-07-26
**Dependencies**: Mini-Bili v1.0 SPEC, Mini-Bili v1.0 Rule

### About Skill

This document is the project's "standard operating procedures" (SOP) ? it tells AI exactly how to execute fixed workflows. Skills exist to prevent AI from improvising and doing the same thing differently each time.

Rule says "this must be done"; Skill says "do it this way."

---

### S-001: Build Verification

**Corresponding Rule**: R-DEV-1 (code must compile after changes)

**Trigger**: After every code modification.

**Steps**:
1. In project root, execute sequentially:
   ```
   go mod tidy
   go build -o ./bin/mini-bili ./cmd/
   ```
   `go mod tidy` MUST run before `go build`.
2. Check: exit code 0 and no error stderr -> passed. Non-zero or errors -> failed.
3. On failure: fix FIRST error, restart from step 1. Max 3 retries for same error, then report.
4. After build passes, confirm `./bin/mini-bili` exists and is executable.

**Prohibited**: skip `go mod tidy`, skip compilation, use `go run` instead, modify unrelated code.

---

### S-002: Database Migration (Versioned)

**Corresponding Rules**: R-DB-3, R-DB-4, R-DB-5

**Trigger**: Any table create/modify/index operations.

**Migration Architecture**:
- V1-V19: Go function migrations in `RegisteredMigrations()`, tracked in `schema_versions` table.
- V20+: SQL files in `migrations/` directory, managed by goose with up/down support.
- Production (`APP_ENV=production`): goose SQL only, skips Go AutoMigrate.

**Steps**:
1. Determine type:
   - New GORM models/fields/indexes -> append to `RegisteredMigrations()`, write Go migration function.
   - V20+ changes -> create `NNNNN_desc.sql` in `migrations/` with `-- +goose Up` and `-- +goose Down`.
2. Go function signature: `func xxx(db *gorm.DB, lg *zap.Logger) error`. Must be idempotent (check `HasColumn`/`HasIndex`).
3. SQL format:
   ```sql
   -- +goose Up
   ALTER TABLE videos ADD COLUMN new_field VARCHAR(255);
   -- +goose Down
   ALTER TABLE videos DROP COLUMN new_field;
   ```
4. Verify: `go vet ./internal/data/ && go test ./internal/data/ -count=1`
5. Start app, check logs.

---

### S-003: Logger Initialization

**Corresponding Rules**: R-OBS-1, R-OBS-2

**Trigger**: Project init or logger reconfiguration.

**Steps**:
1. Import `go.uber.org/zap`.
2. Create logger init in `internal/logger/`.
3. Use standard config:
   ```go
   config := zap.NewProductionConfig()
   config.EncoderConfig.TimeKey = "timestamp"
   config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
   logger, _ := config.Build()
   ```
4. Register before router init.

**Prohibited**: `fmt.Println` for logging, mixing log levels.

---

### S-004: Sensitive Word Filtering

**Corresponding Rule**: R-AI-1

**Trigger**: Any text content needing safety check.

**Steps**:
1. Ensure `configs/sensitive_words.txt` is loaded.
2. Use `sensitive.Filter(text)`.
3. For tool calls: filter before execution (args) and after (results).
4. Return clear error codes on hits.

---

### S-005: Cover Image Validation

**Corresponding Rule**: R-BIZ-8

**Trigger**: Cover/avatar image upload.

**Steps**:
1. Validate extension: JPEG, PNG, GIF, BMP, WEBP only.
2. Validate size: <= 10MB (cover), <= 5MB (avatar).
3. Upload to OSS: `covers/{video_id}.{ext}` or `avatars/{user_id}.{ext}`.

---

### S-006: Error Code Mapping

**Corresponding Rule**: R-API-1

**Trigger**: Adding/modifying error codes.

**Steps**:
1. Define codes in `internal/e/code.go`: 40000-40999 (client), 50000-50999 (server), 40300-40399 (permission).
2. Add messages in `internal/e/msg.go`.
3. Ensure `GetMsg(code)` returns correct message.

---

### S-007: WebSocket Connection Management

**Corresponding Rules**: Performance requirements

**Trigger**: WebSocket connection management for danmaku/chat.

**Steps**:
1. Per-connection write goroutine + buffered channel (capacity 64).
2. `SetReadDeadline` + `SetPongHandler` for heartbeat.
3. Periodic Ping for keep-alive.
4. Non-blocking broadcast via channel push.
5. Batch user info via `WHERE id IN (...)` on connect.

---

### S-009: JWT Authentication Flow

**Corresponding Rules**: R-AUTH-2, R-AUTH-3

**Trigger**: Auth-related code changes.

**Steps**:
1. Generate JWT: Access (15min), Refresh (7 days).
2. Store refresh tokens in MySQL + Redis.
3. Auto-refresh on access token expiry.
4. Middleware validation on protected routes.

---

### S-010: Video Transcoding Pipeline

**Corresponding Rule**: R-BIZ-2

**Trigger**: Upload/transcoding flow changes.

**Steps**:
1. Upload -> validate (<= 500MB) -> save temp.
2. Push task to RabbitMQ.
3. Worker -> FFmpeg transcode H.264 MP4 -> generate cover.
4. Upload to OSS (`videos/{id}.mp4`, `covers/{id}.jpg`).
5. Update video status in DB.

---

### S-012: Frontend Encoding Check

**Corresponding Rule**: R-ENCODE-1

**Trigger**: Before committing frontend changes.

**Steps**:
1. Run `npm run check:encoding` in `cakecake-vue/bilibili-vue/`.
2. Check `src/pages/minibili` and `src/i18n` for garbled characters.
3. Fix before commit.

---

### S-013: Markdown Table Validation

**Corresponding Rule**: R-DOC-10

**Trigger**: After modifying Markdown tables.

**Steps**:
1. Run `python scripts/validate_md_tables.py`.
2. Fix alignment/formatting issues.
3. Ensure proper header separator rows.

---

### S-015: Fix Mermaid Pink Background (Render Errors)

**Corresponding Rule**: R-DOC-11

**Trigger**: Mermaid diagrams show pink/red background in preview.

**Common error checks**:
1. **Equals signs `=`**: Avoid in node text (e.g. `status=ready`). Fix: `status: ready`.
2. **Question marks `?`**: Avoid in diamond nodes `{}`. Use rounded rect `()` instead.
3. **Cross-subgraph links**: Nodes in one subgraph linking to another may error. Merge subgraphs instead.
4. **Arrow labels**: Use `-->|text|` pipe syntax, not `-- "text" -->`.
5. **Subgraph title brackets**: Use `Name["Title"]` format for titles with brackets/Chinese.
6. **Special chars**: Mermaid does not support ->, <- etc. in node text.

**Debugging mantra**: Pink appears, check equals + question marks first. Cross-subgraph links are dangerous. Arrow labels use pipe syntax. Special chars can't stay, simplify to find root cause.

---

### S-016: Create goose Migration Files

**Corresponding Rule**: R-DB-5

**Trigger**: Need V20+ database schema change.

**Steps**:
1. Check `migrations/` for max number. New file = max + 1 (e.g. `00002_desc.sql`).
2. Create file:
   ```sql
   -- +goose Up
   -- Forward SQL

   -- +goose Down
   -- Rollback SQL
   ```
3. Write both Up and Down SQL. Down must fully undo Up.
4. Verify: start app (development) or `goose up` manually.
5. Verify rollback: `goose down`.

**Notes**:
- One change per file.
- No data ops (INSERT/UPDATE) in SQL migrations ? use Go functions for data.
- `-- +goose Up`/`-- +goose Down` markers are mandatory.


---

### S-017: Graceful Shutdown Implementation

**Corresponding Rule**: R-SHUTDOWN-1/2/3

**Trigger**: Adding background goroutines or modifying main.go startup/shutdown.

**Steps**:

1. Declare sync.WaitGroup in main(), wg.Add(1) per goroutine, defer wg.Done().
2. Use http.Server (never gin.Run()), call srv.Shutdown(ctx) on exit.
3. Shutdown sequence: cancel contexts -> srv.Shutdown -> wg.Wait with timeout.
4. Final PlayCount Flush in goroutine defer.
5. Resource Close via defers in reverse declaration order.

**Forbidden**: time.Sleep wait, gin.Run(), exit without waiting, omit final flush.

**Verification**: go vet + go build + Ctrl+C log confirmation.
