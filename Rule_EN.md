<p align="center">
  <a href="Rule.md">
    <img src="https://img.shields.io/badge/🇨🇳中文-999999?style=flat-square" alt="中文">
  </a>
  <strong><img src="https://img.shields.io/badge/🇬🇧English-00a1d6?style=flat-square" alt="English"></strong>
</p>

## cakecake v1.0 Engineering Rules (Rule)

**Version**: v1.0
**Last Updated**: 2026-08-02
**Dependencies**: cakecake v1.0 SPEC

---

### About Rule

This document is the project's "engineering constitution" — it tells AI what must never be compromised. Rule is a **soft constraint**, not a hard gate; Scripts will enforce them later.

As the project evolves, Rule entries will grow. When a Rule is repeatedly violated by AI, it should be hardened into a Script for automated enforcement.

---

### 1. Database & Data Security

| ID       | Rule                               | Description |
| :------- | :--------------------------------- | :---------- |
| **R-DB-1** | **No hardcoded secrets** | Database DSN, OSS keys (Bucket `cakecake`), JWT Secret, RabbitMQ credentials, sensitive word list paths, etc. MUST be read from environment variables, falling back to config files only when the env var is missing. NEVER hardcode them in code. |
| **R-DB-2** | **No SQL concatenation** | All database queries MUST use GORM parameterized queries or safe methods such as `db.Where()`. NEVER use `db.Raw()` to concatenate user input. |
| **R-DB-3** | **Schema changes MUST go through the migration system** | All `CREATE TABLE`, `ALTER TABLE`, etc. MUST be executed via the migration system. V1–V19 are managed by GORM AutoMigrate (`RegisteredMigrations()`); V20+ are managed by goose SQL migration files (`migrations/` directory). NEVER create tables in business code or modify the production database manually. |
| **R-DB-4** | **Core query fields MUST have indexes** | The following fields MUST have explicit indexes in their tables: `play_count` (video play count, used for sorting), `created_at` (creation time, used for sorting), `user_id` (user ID, used for joins), `video_id` (video ID, used for joins). |
| **R-DB-5** | **New migrations MUST use goose SQL files** | All schema changes V20+ MUST be written as SQL files in `migrations/` (format `00002_xxx.sql`) containing both `-- +goose Up` and `-- +goose Down` sections. NEVER only append Go functions to `RegisteredMigrations()` without an SQL rollback path. |

---

### 2. API Design & Compatibility

| ID        | Rule                                        | Description |
| :-------- | :------------------------------------------ | :---------- |
| **R-API-1** | **Must follow the unified response format** | All external HTTP endpoints MUST return the following JSON format. **NEVER** use any other response structure. `code` uses business error codes (int), `msg` is obtained from the S-006 error code mapping table, `data` returns `null` when empty. Example: `c.JSON(http.StatusOK, gin.H{ "code": code, "msg": e.GetMsg(code), "data": data, })` |
| **R-API-2** | **Must follow RESTful conventions** | All external HTTP endpoint URLs and methods MUST follow RESTful style (NF-8): GET for queries, POST for creation, PUT/PATCH for updates, DELETE for deletion. URLs use plural nouns (e.g., `/videos`), not verbs. |
| **R-API-3** | **NEVER break existing endpoints** | Released v1.0 APIs MUST NOT have existing field names, types changed, or fields deleted (BC-1). Only new fields or new endpoints may be added. |
| **R-API-4** | **All write endpoints and WebSocket connections MUST be authenticated** | Any endpoint that creates/updates/deletes data (upload, danmaku, comments, likes, deletes, etc.) MUST validate the JWT Access Token at the Gin middleware layer (F1). WebSocket connections MUST also validate the JWT Access Token at handshake, following S-011. Failed validation MUST return 401 and stop further processing. |

---

### 3. Authentication & Security

| ID          | Rule                             | Description |
| :---------- | :------------------------------- | :---------- |
| **R-AUTH-1** | **Token security policy** | JWT auth MUST use a dual-token mechanism; issuance, refresh, and invalidation flows MUST strictly follow S-009. |
| **R-AUTH-2** | **Passwords MUST be bcrypt-hashed** | Any user password logic MUST use `golang.org/x/crypto/bcrypt` for hashing and verification (F1). NEVER use MD5, SHA256, or any other method to store passwords. |

---

### 4. Business Logic Constraints

| ID          | Rule                             | Description |
| :---------- | :------------------------------- | :---------- |
| **R-BIZ-1** | **Danmaku cooldown MUST be validated on both ends** | The 5-second danmaku cooldown MUST strictly follow S-007. NEVER rely on frontend blocking alone. |
| **R-BIZ-2** | **Comment deletion MUST cascade** | Deleting a comment MUST also delete all child comments at every level (F7), following S-008. NEVER leave "orphan" child comments under a deleted parent. |
| **R-BIZ-3** | **Cover images MUST be validated for format and size** | Cover upload validation MUST follow S-005. Invalid files MUST return a clear error and reject the whole request. |
| **R-BIZ-4** | **Video list sorting MUST follow the multi-level priority** | Homepage video lists MUST sort by "play count DESC → upload time DESC → danmaku count DESC" (F10). NEVER deviate from this priority. |
| **R-BIZ-5** | **Danmaku sensitive-word filtering MUST use a configurable dictionary** | Danmaku content review (F5) MUST load sensitive words from an external dictionary file whose path comes from an environment variable or config file (R-DB-1). NEVER hardcode the dictionary in code. When the dictionary is empty or the file is missing, MUST reject all danmaku sends (better to over-block than miss) and log an Error-level entry. |
| **R-BIZ-6** | **Comment deletion permissions MUST be checked** | Video owners may delete ANY comment under their videos. Regular users may only delete their own comments. The backend MUST check permissions before deletion; NEVER rely on frontend button hiding alone. Failed checks MUST return 403. |
| **R-BIZ-7** | **Danmaku color MUST be a valid hex string** | The backend MUST validate the `color` field, matching the `#[0-9A-Fa-f]{6}` regex. Invalid values return error code `40014` and the danmaku is rejected. Do not silently fill defaults or auto-correct. |
| **R-BIZ-8** | **Avatars MUST be validated for format and size** | Avatar upload (F0) MUST follow the same extension/MIME validation as Skill **S-005** (covers): only **JPEG, PNG, GIF, BMP, WEBP** allowed; single file **≤ 5 MB** (covers allow 10 MB; avatars have their own limit). Invalid files MUST return a clear business error code (`40015` / `40016`) and be rejected. |
| **R-BIZ-9** | **Global rate limiting MUST use a Redis token bucket** | Global HTTP rate limiting MUST be based on a Redis token bucket, keyed by IP. Bucket capacity (`RATE_LIMIT_BURST`) and refill rate (`RATE_LIMIT_RATE`) MUST be configurable via environment variables. Rate limiting may be disabled via `RATE_LIMIT_ENABLED=false` (default off), but hardcoded values or skipping the check are forbidden. |
| **R-BIZ-10** | **Operational parameters prefer runtime config** | Operational parameters such as Agent toggles/quotas and rate-limit thresholds MUST be read dynamically from the `system_configs` table first, falling back to environment variable defaults. New parameters MUST be registered in `internal/config/runtime.go` defaults and declared in `knownConfigKeys` in `internal/handler/admin_system_config.go`. NEVER add an operational parameter that only reads an env var without a dynamic config path. |

---

### 5. Development Workflow & Quality Bottom Line

| ID          | Rule                             | Description |
| :---------- | :------------------------------- | :---------- |
| **R-DEV-1** | **Code MUST compile after changes** | After any code modification, S-001 (build verification) MUST be run. NEVER commit code that fails to compile or panics at runtime. |
| **R-DEV-2** | **Temporary files MUST be cleaned up** | All local temp files produced during upload/transcode (raw videos, intermediate transcode output, temp covers) MUST be deleted after the task finishes (F2). NEVER let temp files occupy server disk long-term. |
| **R-DEV-3** | **Errors MUST provide clear information** | Any endpoint error MUST have a human-readable `msg` that helps the caller locate the problem. Error codes MUST be resolved through the S-006 error code mapping table. NEVER return just "error" or an empty string. |
| **R-DEV-4** | **Transcode retry MUST be bounded** | Transcode retry logic MUST strictly follow S-004. NEVER retry infinitely. |
| **R-DEV-5** | **Redis MUST configure timeouts and a pool** | Redis client initialization MUST explicitly set: dial timeout (DialTimeout), read/write timeouts (ReadTimeout / WriteTimeout), and max pool size (PoolSize). NEVER use zero-value defaults. |
| **R-DEV-6** | **Message queue MUST be RabbitMQ** | The async task message queue MUST be **RabbitMQ**. All async task delivery and consumption MUST be based on RabbitMQ. NEVER replace it with Redis List or other message queues. RabbitMQ credentials MUST be read from environment variables (R-DB-1). |
| **R-DEV-7** | **Project scripts MUST be cross-platform** | All project-level scripts/automation MUST use cross-platform approaches: build/test entry points use the Makefile (POSIX compatible); NEVER write PowerShell-specific commands (`Remove-Item`, `Select-String`, etc.) in docs/scripts. File operations should prefer Go/Python/Node.js, with `/` as the path separator. CI (GitHub Actions) uses Linux + bash; local development must keep the same commands behaving identically on Windows/WSL/macOS. |

---

### 6. Observability

| ID          | Rule                             | Description |
| :---------- | :------------------------------- | :---------- |
| **R-OBS-1** | **NEVER use fmt.Println for logging** | Log initialization MUST strictly follow S-003. All business logs MUST use the zap structured logger. `fmt.Println` is allowed only for temporary local debugging and MUST NEVER enter version control. |
| **R-OBS-2** | **Log levels MUST be differentiated** | `Info` for key business flows (e.g., "user logged in", "video transcode complete"). `Error` for exceptions and errors (e.g., "database connection failed", "OSS upload timeout", "sensitive dictionary load failed"). NEVER use the same level for everything. |

---

### 7. Compatibility & Architecture Constraints

| ID         | Rule                                 | Description |
| :--------- | :----------------------------------- | :---------- |
| **R-ARCH-1** | **Modular code, reserve microservice splitting** | Although v1.0 is a Gin monolith (NF-6), code MUST be split by feature into directories (`internal/handler/`, `internal/service/`, `internal/dao/`), and cross-service calls MUST go through interfaces rather than direct references, reserving room for a future Kratos microservice split (BC-2). |
| **R-ARCH-2** | **Video status field MUST be a closed set of defined values** | The video `status` field MUST use exactly these five values and nothing else: `processing`, `published`, `failed`, `pending_review` (reserved), `rejected` (reserved). The DB column length MUST support all five values (F2-b, BC-2). |

---

### 8. Frontend & Interaction Constraints

#### 1. Style & Structure Protection

| ID       | Rule                   | Description |
| :------- | :--------------------- | :---------- |
| **R-FE-1** | **CSS style files are frozen** | All files in `src/assets/` and `src/components/` containing Tailwind CSS class names are **read-only**. AI MUST NOT modify existing styles, but MAY combine existing Tailwind atomic classes in **new components**. |
| **R-FE-2** | **Page skeleton & routes are frozen** | Route configuration (Vue Router) and the overall page HTML hierarchy (e.g., Flex/Grid layout containers) were defined by humans; AI MUST NOT delete or refactor them. AI MAY add feature components inside **reserved container areas** (e.g., `<div id="comment-section">`). |

#### 2. Feature Enhancement & Logic Development (Open Area)

| ID       | Rule                     | Description |
| :------- | :----------------------- | :---------- |
| **R-FE-3** | **Demo buttons may be wired up** | Existing demo buttons (e.g., "send danmaku", "submit comment", "like") MAY be bound to real API calls and interaction logic. AI MUST keep the buttons' **existing styles unchanged**, only adding `@click` handlers, data binding, and state management. |
| **R-FE-4** | **New feature components allowed** | AI MAY add feature components (e.g., comment list, notification list, danmaku input) to pages such as the video player or message center, but MUST: use existing Tailwind CSS classes; place them inside reserved container areas; NOT modify the existing page skeleton. |

#### 3. Data Interaction & Logic Isolation

| ID       | Rule                                      | Description |
| :------- | :---------------------------------------- | :---------- |
| **R-FE-5** | **API calls MUST use TypeScript Interfaces** | Frontend API calls MUST be wrapped with `axios` or `fetch`. MUST define TypeScript Interfaces (Type Guards) from the backend JSON structure; NEVER use the `any` type. |
| **R-FE-6** | **Loading and Error states MUST be handled** | Any async request (upload, login, list fetching) MUST show `loading` feedback (e.g., button color change, spinner) and `error` hints (e.g., Toast) in the UI. NEVER call an API without user feedback. |
| **R-FE-7** | **Request base URL MUST come from env vars** | The backend BaseURL MUST be injected via environment variables (e.g., `.env.production`); NEVER hardcode `localhost:8080` or a specific IP. |

---

### 9. Documentation Sync

| ID         | Rule                                               | Description |
| :--------- | :------------------------------------------------- | :---------- |
| **R-DOC-1**  | **All paired CN/EN .md files MUST be synced** | For every Chinese `.md` file with an `_EN.md` counterpart (Rule, Skill, SPEC, DEPLOY, docs/*, cakecake-vue/**/README, etc.), modifying either side MUST update the other so content, tables, code blocks, and links stay consistent, differing only in language. Run `python scripts/check_en_sync.py --check-sync` before committing (heading-structure and code-block level). |
| **R-DOC-2**  | **Code changes MUST update docs — checked before every commit** | After modifying code and before `git commit`, MUST walk all Markdown docs (README*.md, SPEC.md, docs/*.md, deploy/*.md, etc.) and confirm the docs still match the code. **NEVER commit code first and docs later** — docs and code changes MUST land in the same commit. |
| **R-DOC-2a** | **Commit message MUST list doc-check results** | When a commit involves feature/API/config/dependency changes, the commit body MUST include `Docs: <list of checked md files>`. Omission counts as a violation. |
| **R-DOC-3**  | **Every .md needs an EN version with fully synced structure** | Every major .md file (root, docs/, deploy/, cakecake-vue/, etc.) MUST have a corresponding `_EN.md`. Modifying either side MUST sync the other so section structure, diagrams (Mermaid), tables, and code blocks are identical, differing only in language. No section may exist on one side and be missing on the other. |
| **R-DOC-4**  | **Git commit messages MUST be in English** | `git commit -m` messages MUST be pure English following conventional commits (e.g., `feat:`, `fix:`, `docs:`, `refactor:`, `chore:`). Chinese is forbidden in commit messages. |
| **R-DOC-5**  | **New env vars MUST be added to .env.example** | Introducing a new environment variable requires adding its comment and default value to `.env.example` in the same commit. NEVER add code that reads an env var without updating the template. |
| **R-DOC-6** | **Chinese encoding check** | After writing Chinese into code files and before committing, MUST scan the project with Python for leftover BOM, U+FFFD, or truncated Chinese. Use Python `pathlib` with explicit `encoding="utf-8"` for reading/writing Chinese files to avoid platform encoding differences. |
| **R-DOC-7** | **Modifying .md MUST update header metadata and validate references** | After modifying any `.md` (including Rule.md, Skill.md, README*.md, SPEC.md, docs/*.md): 1. update `**版本**`/`**最后更新**`/`**依赖文档**` dates and references; 2. check that referenced docs exist, paths are correct, and sections correspond; 3. when adding/removing/renaming a Rule/Skill/doc, update all cross-references in other docs. All three are required; note `Docs-checked:` in the commit message. |
| **R-DOC-8** | **Pitfalls/technical issues MUST be recorded** | After solving a technical problem, MUST add an entry to README_REFLECT.md's "采坑记录": problem symptom → root cause → solution. Applicable to framework traps (Gin middleware behavior, GORM query limits), language misuse (Go interface nil checks, PowerShell backtick escaping), architecture mistakes, performance tuning, cross-platform issues, etc. |
| **R-DOC-9** | **Python scripts with Chinese MUST run from files, not inline -c** | In PowerShell, NEVER execute `python -c "..."` with Chinese/UTF-8 multibyte code. Use one of: 1. `scripts/safe_write.py --base64 <b64> --output <path>`; 2. write the .py via .NET `[System.IO.File]::WriteAllBytes` then run it; 3. write to a temp file then run `python <file>`. Violating this is the #1 cause of garbled Chinese. |
| **R-DOC-10** | **Run table validation after Markdown table edits** | After editing any Markdown table, run `python scripts/validate_md_tables.py` before committing: no `---` breaking table continuity, no row/column mismatches, no missing `|`. Pass criterion: `OK: all tables look clean`. |
| **R-DOC-11** | **Architecture diagrams: Mermaid for flows, Text for structure** | Flow/topology/data-flow diagrams MUST use ````mermaid` code blocks (`flowchart`, `sequenceDiagram`, etc.). Directory trees (e.g., `docs/ARCHITECTURE.md`) MAY use plain text tree diagrams (`├──`/`└──`). **Forbidden** to label plain text diagrams as ` ```text ` for architecture. |
| **R-DOC-14** | **Doc "Last Updated" date is a hard gate** | When modifying any `.md` with a `**最后更新**`/`**Last Updated**` header, the field MUST be updated to the current date. `scripts/check_doc_dates.py` (wired into `check_pre_commit.py`) checks staged modified `.md` files and blocks the commit if the date is not today. |

---

### 10. Testing Standards

| ID       | Rule                                     | Description |
| :------- | :--------------------------------------- | :---------- |
| **R-TEST-1** | **New business code MUST come with tests** | When adding/modifying business logic under internal/handler/, internal/service/, internal/ws/, internal/pkg/, MUST provide corresponding unit tests in the same commit. Code depending only on external services (ES, RabbitMQ, OSS, third-party APIs) may be exempt, but the exemption reason MUST be noted in the commit message. |
| **R-TEST-2** | **Tests MUST run standalone without external services** | Unit tests MUST use SQLite in-memory (github.com/glebarez/sqlite) + miniredis (github.com/alicebob/miniredis/v2) to simulate dependencies. NEVER connect to real MySQL/Redis/RabbitMQ in tests. |
| **R-TEST-5** | Integration tests MUST inject real Provider implementations | Services depending on Provider interfaces (`CommentService`, `FavoriteService`, `EngagementService`, etc.) MUST inject real implementations in integration tests (e.g., `service.NewVideoProvider(db)`). NEVER pass `nil` — it causes runtime nil pointer panics that `go vet` cannot catch at compile time. |

---

### 11. Frontend Testing Standards

| ID         | Rule                                               | Description |
| :--------- | :------------------------------------------------- | :---------- |
| **R-FE-TEST-1** | **New .vue / .js / .ts files MUST come with tests** | When adding/modifying modules under `src/api/`, `src/router/`, `src/utils/`, `src/components/`, `src/pages/`, MUST provide a matching Vitest test file (`.spec.ts`) placed in the corresponding `src/__tests__/` subdirectory, aligned with source structure. Pure type definitions and constant files are exempt. |
| **R-FE-TEST-2** | **Tests MUST mock external dependencies with `vi.mock`** | Frontend tests MUST mock HTTP requests (mock `@/utils/http`), Vuex Store, routes, etc. NEVER make real network requests in unit tests. `import.meta.env` MUST be set via `vi.stubEnv` in `beforeAll`; beware module caching — different env configs need separate test files. |
| **R-FE-TEST-3** | **Coverage target ≥ 70%** | Overall frontend statement coverage target ≥ 70%. New code MUST maintain or raise current coverage. Current: Statements `73.02%`, Branches `68.8%`, Functions `68.01%`, Lines `72.66%`. |
| **R-FE-TEST-4** | **README CN/EN MUST sync test data** | When updating test statistics in `README.md` or `README_EN.md` (badges, coverage), the other file MUST be updated too. Never hardcode test file counts / case counts / coverage numbers in docs (see R-DOC-19); use relative descriptions or auto-updating badges. |

---

### 12. Post-Update Cleanup Standards

| ID | Rule | Description |
| :--- | :--- | :--- |
| **R-CLEAN-1** | **Clean temp & debug files** | After each code update/test writing, run `make clean` (calls `clean.py`) to auto-delete temp files such as `_*.py`, `_fix*.py`, `_gen*.py`, `_debug*`, `fix_*.py`, `make_*.py`, `write_*.py`, `test_a*.py`. Also delete coverage temp artifacts `cov_out`, `coverage_total`, `covprofile`, `coverage.out`. |
| **R-CLEAN-2** | **Update .gitignore** | After cleanup, check whether `.gitignore` needs new artifact patterns (new coverage formats, script extensions, build cache dirs). |
| **R-CLEAN-3** | **Security check before commit** | Before `git add -A`, check changed files for hardcoded secrets/tokens (`CODECOV_TOKEN=`, `password=`, `secret=`), `.env` files, personal credentials, >10MB files, and binaries. Exclude them and inform the user when found. |
| **R-CLEAN-4** | **Commit after user confirmation** | `git add -A` → `git status --short` showing the change summary (lines added/removed, file count, key changes) → **wait for user confirmation** → `git commit` + `git push`. Commit messages follow conventional commits (`test:`/`feat:`/`fix:`/`chore:`). |
| **R-CLEAN-5** | **NEVER commit sensitive content** | NEVER commit tokens, keys, passwords, `.env` files, IDE config dirs (`.idea/`/`.vscode/`/`.cursor/`), or `node_modules/`. If sensitive files were staged by mistake, immediately `git rm --cached` and update `.gitignore`. |

---

### 13. AI Gateway / Function Calling

| ID | Rule | Description |
| :--- | :--- | :--- |
| **R-DOC-11** | **After modifying AI Gateway, run go vet + related unit tests** | After any change under `internal/aigateway/` or `internal/aigateway/toolkit/`, MUST run `go vet ./internal/aigateway/...` and `go test ./internal/aigateway/...` to avoid breaking tool-calling loop logic. |
| **R-DOC-12** | **New Tools MUST sync definition + implementation** | Adding a platform tool requires updating `toolkit/tools.go`'s `definition()` AND `toolkit/platform.go`'s `Execute()` switch branch in the same change, so definition and implementation never diverge. |
| **R-DOC-13** | **Tool input and output MUST pass sensitive-word filtering** | `PlatformExecutor.Execute()` MUST check args with the sensitive filter before the switch, and every tool implementation MUST filter its result before returning. |

---

### 14. Local Verification

| ID | Rule | Description |
| :--- | :--- | :--- |
| **R-VERIFY-1** | **MUST pass local tests before push** | Any code change (frontend/backend/docs) MUST pass local verification before pushing to GitHub: backend runs `go build ./...` + `go vet ./...` + related `go test`; frontend runs `npm run build`. Failed verification forbids push. |
| **R-VERIFY-2** | **Bypassing verification is a violation** | If local test environment issues prevent compilation/testing, fix the environment first. NEVER skip verification citing "environment issues" and push directly. |
| **R-VERIFY-3** | **Run the full test suite before PR** | Before submitting a PR, run `go test ./... -count=1` to ensure all tests pass. NEVER skip citing "only changed one line." |
| **R-VERIFY-4** | **The author MUST manually verify before push** | After each code change, the project author (earthcake) MUST personally log in and verify functionality works correctly and is bug-free before allowing push. NEVER push without author verification. |
| **R-VERIFY-5** | **Author MUST review changes before commit (single confirmation gate)** | Before any `git commit`, MUST show the author (earthcake) `git status --short` (file list) and `git diff` (key content) and get explicit confirmation. When committing, stage ONLY the files changed in this task; NEVER use `git add -A` or `git add .` (which could sweep unconfirmed changes or accidental deletions into the commit). Before committing, re-check that the staged snapshot matches what was shown. The single confirmation gate is at commit time; separate add confirmation is not required. NEVER commit without showing and confirming. |

---

### 15. Documentation Standards

| ID | Rule | Description |
| :--- | :--- | :--- |
| **R-DOC-14** | **README_REFLECT "Current Status" and "Future Outlook" MUST be at the end** | In README_REFLECT.md, dated entries are in reverse chronological order (newest first), but the "Current Status" and "Future Outlook" summary sections MUST be at the very end, never interleaved between dated entries. |
| **R-DOC-15** | **gitignore MUST exclude paired EN files** | If a Chinese `.md` is excluded in `.gitignore` (e.g., `Minibili.md`, `README_REFLECT.md`, `docs/benchmark.md`), its `_EN.md` counterpart MUST also be added. Wildcards like `incident-*.md` already cover both. Violating this exposes private documents publicly. |
| **R-DOC-16** | **New .md files MUST create the EN version and register the check** | Creating any new Chinese `.md` MUST create the corresponding `_EN.md` in the same commit and register it in `check_en_sync.py`'s whitelist. Single-language files are forbidden. |
| **R-DOC-17** | **Pre-commit CN/EN content sync is a hard gate** | Every commit MUST run `python scripts/check_en_sync.py --check-sync` (wired into `check_pre_commit.py` and the local pre-commit hook): it verifies heading structure (title sequence) and code-block consistency across all CN/EN pairs. On failure, the commit is blocked; committing with drift requires explicit human confirmation (interactive `y`, or `--yes`). Violating this counts as failing the gate. |
| **R-DOC-18** | **Markdown relative links MUST resolve** | Relative links (`](path)`) in `.md` docs MUST point to existing files. After renaming/moving files, all referencing links MUST be updated. Before committing, MUST run `python scripts/check_md_links.py` (wired into `check_pre_commit.py` and `make doc-check`); broken links block the commit. |
| **R-DOC-19** | **Docs MUST NOT hardcode test statistics** | `.md` docs MUST NOT hardcode test file counts, case counts, or coverage percentages (they go stale as code evolves — e.g., "27 test files" was actually 130). Use relative descriptions (e.g., "full frontend test suite", "backend unit + integration tests") or auto-updating CI / Codecov badges. Before committing, check added/modified docs and replace hardcoded numbers. |

---

### 16. Encoding Standards

| ID | Rule | Description |
| :--- | :--- | :--- |
| **R-ENCODE-1** | **No UTF-8 BOM in project files** | BOM (Byte Order Mark) causes Go compilation failures and Markdown rendering issues. MUST use `python scripts/check_bom.py` to detect all `.go`, `.md`, `.yaml`, `.json`, `.py` files. The script supports `--fix` and `--path`. File writes MUST use `scripts/safe_write.py` (auto-strips BOM); NEVER use `Set-Content` directly. |
| **R-ENCODE-2** | **Pre-commit BOM check is a hard gate** | `python scripts/check_pre_commit.py` runs `check_bom.py` on every commit. BOM detection blocks the commit; MUST run `python scripts/check_bom.py --fix` and re-commit. Skipping for any reason is forbidden. |
| **R-ENCODE-3** | **File writes MUST use safe_write.py** | On Windows, PowerShell `Set-Content -Encoding UTF8` writes a UTF-8 BOM that breaks Go compilation. Any script/tool writing `.go`, `.md`, `.yaml`, `.json`, `.py` files MUST use `python scripts/safe_write.py --text "..." --output <path>`, which strips BOM. Direct `Set-Content`/`Out-File` text writes are forbidden. |
| **R-FMT-1** | **gofmt is a hard gate before commit** | All Go code MUST pass gofmt (`gofmt -l internal cmd` empty). `scripts/check_pre_commit.py` and GitHub CI both check; unformatted files block commits. Run `gofmt -w` after adding/modifying Go files. |

---

### 17. Build & Cache Cleanup

| ID | Rule | Description |
| :--- | :--- | :--- |
| **R-BUILD-1** | **Cross-compilation MUST go through the Makefile** | NEVER manually run `$env:GOOS=linux` + `go build` line by line in PowerShell. MUST use `make build-linux`, which exports GOCACHE/GOTMPDIR to the D: drive to avoid filling up C:. |
| **R-BUILD-2** | **Clean the Go cache after every build** | The `build-linux` target recipe ends with `python scripts/clean_go_cache.py`, auto-deleting all `go-*/gc-*/gm-*` dirs under the C: Temp and `AppData\Local\go-build`. Other build targets MUST also call this script. |
| **R-BUILD-3** | **No compile cache on the C: drive** | GOCACHE and GOTMPDIR MUST point to the D: drive (currently `D:\cakecake\.gocache` and `D:\cakecake\.gotmp`), both in `.gitignore`. If temporarily redirected locally, run `python scripts/clean_go_cache.py --local-only` afterwards. |

---

### 18. Shutdown & Graceful Exit

| ID | Rule | Description |
| :--- | :--- | :--- |
| **R-SHUTDOWN-1** | **Graceful shutdown with timeout is REQUIRED** | All background goroutines (transcode consumer, playcount flush, danmaku relay, HTTP Server, etc.) MUST be tracked via sync.WaitGroup. On SIGTERM, execute in order: ① cancel context to stop accepting new tasks → ② `http.Server.Shutdown()` drain HTTP connections → ③ `wg.Wait()` for background tasks with `SHUTDOWN_TIMEOUT` (default 30s) → ④ force exit on timeout. NEVER just `time.Sleep` and exit. |
| **R-SHUTDOWN-2** | **PlayCount MUST do a final flush on exit** | The PlayCount flush goroutine MUST execute a final `Flush()` in its defer to avoid losing Redis increments. |
| **R-SHUTDOWN-3** | **Resource close order MUST be correct** | defer-registered `Close` calls execute in reverse order, ensuring MQ Channel → MQ Connection → ES close order matches their dependency chain. |

---

### 19. Code Organization

| ID | Rule | Description |
| :--- | :--- | :--- |
| **R-ORG-1** | **Script files MUST live in `scripts/`** | All `.py`, `.sh`, and other helper scripts MUST reside under the project root `scripts/`. They MUST NOT be scattered inside `internal/` or other business directories. handler/service/model packages may only contain `.go` source and `_test.go` test files. |
| **R-ORG-2** | **Business code directories MUST NOT contain non-Go files** | Core packages such as `internal/handler/`, `internal/service/`, `internal/model/` MUST NOT contain Python scripts, temp files, or data files. Data files go in `configs/`; temporary files go in `tmp/`. |

---

### 20. Dependency Injection

| ID | Rule | Description |
| :--- | :--- | :--- |
| **R-DI-1** | **Dependencies MUST be injected via constructor** | All Service dependencies (DB, Redis, Logger, other Services, Provider interfaces, etc.) MUST be passed in and assigned inside the `NewXxxService()` constructor in a single call. Two-phase injection (`SetXxx()` / `SetProviders()`) is FORBIDDEN. This catches missing dependencies at compile time instead of runtime nil pointer panics. |
| **R-DI-2** | **Constructor parameter order MUST be consistent** | All Service constructors follow a unified order: `db *gorm.DB, rdb *redis.Client, log *zap.Logger` (the "three essentials") first, then other Services, Provider interfaces, and external clients in dependency order. |
| **R-DI-3** | **Circular dependencies are FORBIDDEN** | Services MUST NOT form circular dependencies (A depends on B and B depends on A). If mutual calls are genuinely needed, extract shared logic into an independent third Service or use Provider interfaces to break the cycle. |

---

### 21. Comment Standards

| ID | Rule | Description |
| :--- | :--- | :--- |
| **R-COMMENT-1** | **Code comments MUST be in English** | All package comments, function comments, struct field comments, and inline comments in `.go` source MUST be written in English (including punctuation). Chinese is allowed ONLY in: ① user-facing error message strings (`errcode.GetMsg()` return values), ② Swagger `@description` annotations, ③ operator-facing hints in log output. |
| **R-COMMENT-2** | **Exported functions MUST have English doc comments** | All exported functions, types, and constants MUST have an English comment in `// Name describes ...` format explaining purpose, parameters, and return values. Comments start with the object's name and end with a period. |
| **R-COMMENT-3** | **Model field comments MUST explain business semantics** | Every field in GORM model structs MUST have an inline comment (`// comment`) in English explaining its business meaning, constraints, and defaults. Boolean flag fields (e.g., `CommentsClosed`) MUST describe the behavioral change when true. |
