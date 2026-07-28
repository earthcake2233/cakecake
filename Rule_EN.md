<p align="center">
  <a href="Rule.md">
    <img src="https://img.shields.io/badge/🇨🇳中文-999999?style=flat-square" alt="中文">
  </a>
  <strong><img src="https://img.shields.io/badge/🇬🇧English-00a1d6?style=flat-square" alt="English"></strong>
</p>

  </a>
  </a>
</p>

  </a>
</p>

# Mini-Bili v1.0 Engineering Rules (Rule)

**Version**: v1.0
**Last Updated**: 2026-07-26
**Dependencies**: Mini-Bili v1.0 SPEC

---

### About Rule

This document is the project's "engineering constitution" ? it tells AI what must never be compromised. Rule is a **soft constraint**, not a hard gate ? Scripts will enforce them later.

As the project evolves, Rule entries will grow. When a Rule is repeatedly violated by AI, it should be hardened into a Script for automated enforcement.

---

### 1. Database & Data Security

| ID       | Rule                               | Description |
| :------- | :--------------------------------- | :---------- |
| **R-DB-1** | **No hardcoded secrets** | Database DSN, OSS keys (Bucket `mini-bili`), JWT Secret, RabbitMQ credentials, sensitive word lists, etc. MUST be read from environment variables. NEVER hardcode. |
| **R-DB-2** | **No SQL concatenation** | All DB queries MUST use GORM parameterized queries or `db.Where()` safe methods. NEVER use `db.Raw()` with user input concatenation. |
| **R-DB-3** | **Schema changes MUST go through migration system** | All CREATE TABLE, ALTER TABLE etc. MUST execute via migration system. V1-V19 managed by GORM AutoMigrate (`RegisteredMigrations()`), V20+ managed by goose SQL migration files (`migrations/` directory). NEVER create tables or manually modify schemas in production. |
| **R-DB-4** | **Core query fields MUST have indexes** | The following fields MUST have explicit indexes: `play_count` (sorting), `created_at` (sorting), `user_id` (joins), `video_id` (joins). |
| **R-DB-5** | **New migrations MUST use goose SQL files** | All schema changes V20+ MUST be written in `migrations/` directory as SQL files (format `00002_xxx.sql`) containing both `-- +goose Up` and `-- +goose Down` sections. NEVER add Go functions alone to `RegisteredMigrations()` without SQL rollback paths. |

---

### 2. API Design & Compatibility

| ID        | Rule                                        | Description |
| :-------- | :------------------------------------------ | :---------- |
| **R-API-1** | **Must follow unified response format** | All external HTTP endpoints MUST return the following JSON format. **NEVER** use any other response structure. `code` uses business error codes (int), `msg` is mapped via S-006 error code table, `data` returns `null` when empty. Template: `c.JSON(http.StatusOK, gin.H{ "code": code, "msg": e.GetMsg(code), "data": data, })` |
| **R-API-2** | **Must follow RESTful conventions** | All external HTTP endpoint URLs and methods MUST follow RESTful style (NF-8): GET for queries, POST for creation, PUT/PATCH for updates, DELETE for deletion. Use plural nouns for resource names (e.g., `/api/v1/videos`). |

---

### 3. Authentication & Security

| ID          | Rule                             | Description |
| :---------- | :------------------------------- | :---------- |
| **R-AUTH-1** | **Passwords MUST be bcrypt-hashed** | All user passwords MUST be hashed with bcrypt before storage. NEVER store plaintext. |
| **R-AUTH-2** | **JWT Secret from env, minimum 32 chars** | JWT secret MUST be read from `JWT_SECRET` env var with a minimum length of 32 characters. |
| **R-AUTH-3** | **All sensitive endpoints require valid token** | Operations involving user data modification (personal info, videos, comments, likes, etc.) MUST validate the Access Token. Return 401 on invalid/missing tokens. |

---

### 4. Business Logic Constraints

| ID          | Rule                             | Description |
| :---------- | :------------------------------- | :---------- |
| **R-BIZ-1** | **Danmaku color validation** | Danmaku color MUST be a valid hex color string (e.g., `#FFFFFF`). |
| **R-BIZ-2** | **Video upload size limit** | Single video file must be ? 500MB. Server-side enforcement required. |
| **R-BIZ-3** | **Comment nesting depth limit** | Maximum 2-level nesting for comments. |
| **R-BIZ-4** | **Username uniqueness enforced** | Username uniqueness validated at registration and modification. Duplicate returns business code `40006`. |
| **R-BIZ-5** | **Password minimum length** | Password must be ? 8 characters. |
| **R-BIZ-6** | **Username validation** | Username: 3-20 characters, supports Chinese, letters, digits, underscores. |
| **R-BIZ-7** | **Danmaku color validation** | See R-BIZ-1. |
| **R-BIZ-8** | **Avatar validation** | Avatar format: JPEG/PNG/GIF/BMP/WEBP only, ? 5MB. Validation logic same as S-005 cover extension logic, but with 5MB size limit. |

---

### 5. Observability

| ID          | Rule                             | Description |
| :---------- | :------------------------------- | :---------- |
| **R-OBS-1** | **No fmt.Println for logging** | NEVER use `fmt.Println` for logging. MUST use zap structured logger. |
| **R-OBS-2** | **Log levels must be differentiated** | Use appropriate levels: Debug for dev, Info for normal ops, Warn for recoverable issues, Error for failures requiring attention. |

---

### 6. Development Workflow

| ID          | Rule                             | Description |
| :---------- | :------------------------------- | :---------- |
| **R-DEV-1** | **Code must compile after changes** | After any code modification, `go build ./...` must succeed. |
| **R-DEV-2** | **Commit messages in English, conventional commits** | Commit messages MUST be in English and follow conventional commits format: `type(scope): description`. Types: feat, fix, docs, refactor, perf, test, chore. |

---

### 7. File Organization

| ID          | Rule                             | Description |
| :---------- | :------------------------------- | :---------- |
| **R-FILE-1** | **Follow standard Go project layout** | Code MUST follow Go standard project layout. `cmd/` for entry points, `internal/` for private packages, handler ? service ? data dependency direction. |
| **R-FILE-2** | **Config files in configs/, not in code** | Configuration files like `sensitive_words.txt`, `ip2region_v4.xdb` etc. belong in `configs/` directory, never inside `internal/`. |

---

### 8. AI Gateway Rules

| ID          | Rule                             | Description |
| :---------- | :------------------------------- | :---------- |
| **R-AI-1** | **Sensitive word filtering before and after tool calls** | All tool input args and output results MUST pass sensitive word filtering (using `sensitive.Filter`). |
| **R-AI-2** | **Rate limiting per user per day** | AI agent calls MUST have daily per-user quota via Redis counters (`AGENT_DAILY_QUOTA`). |
| **R-AI-3** | **Max tool call rounds enforced** | Max 5 rounds of tool calls per user message turn, with graceful degradation on overflow. |
| **R-AI-4** | **Each tool has independent RuntimeConfig toggle** | Admin can enable/disable individual tools via `tool_{name}_enabled` config keys. |

---

### 9. Documentation Rules

| ID          | Rule                             | Description |
| :---------- | :------------------------------- | :---------- |
| **R-DOC-1** | **All paired .md files must be synced** | When modifying any Chinese `.md` file that has a `_EN.md` counterpart (including Rule, Skill, SPEC, DEPLOY, docs/*, cakecake-vue/**/README, etc.), the paired English file MUST be updated in sync. Content, tables, code blocks, and links must be consistent, differing only in language. Run `python scripts/check_en_sync.py` before committing. | After any feature change, README.md must be updated to reflect current state. |
| **R-DOC-2** | **SPEC.md is the source of truth** | All functional requirements MUST trace back to SPEC.md. SPEC.md defines the contract. |
| **R-DOC-3** | **All .md files need EN version and full structural sync** | Every major .md file (root, docs/, deploy/, cakecake-vue/, etc.) must have a corresponding `_EN.md` English version. Modifications to either must sync the other, ensuring section structure, diagrams (Mermaid), tables, and code blocks are identical, differing only in language. |
| **R-DOC-4** | **Language switcher badges required** | All paired .md files must have shield.io badge language switchers at the top, matching main README style. |
| **R-DOC-5** | **Mermaid diagrams for architecture** | Architecture documentation must use Mermaid diagrams for visual clarity. |
| **R-DOC-6** | **Architecture decisions recorded in README_REFLECT** | Significant design decisions and trade-offs must be documented in README_REFLECT.md. |
| **R-DOC-7** | **No Chinese in commit messages** | Commit messages, branch names, PR titles must be in English only. |
| **R-DOC-8** | **Table formatting must be consistent** | Markdown tables must use consistent column alignment and header separators. Run `python scripts/validate_md_tables.py` after changes. |
| **R-DOC-9** | **No inline Chinese in Python -c** | Python one-liners must not contain Chinese characters. Use file-based scripts instead. |
| **R-DOC-10** | **Run table validation after markdown changes** | After modifying any Markdown table, run `python scripts/validate_md_tables.py`. |
| **R-DOC-11** | **Architecture diagrams must use Mermaid** | Architecture/sequence/flow diagrams in docs must use Mermaid. ASCII art is acceptable for directory trees only. |
| **R-DOC-12** | **New tools must sync definition + implementation** | When adding platform tools, MUST simultaneously update `toolkit/tools.go` (definition) and `toolkit/platform.go` (`Execute()` switch). |
| **R-DOC-13** | **Tool input/output must pass sensitive filter** | `PlatformExecutor.Execute()` must filter args before switch; each tool implementation must filter results before return. |
| **R-DOC-14** | **README_REFLECT "Current Status" and "Future Outlook" at the end** | In README_REFLECT.md, all dated entries are in reverse chronological order. The "Current Status" and "Future Outlook" summary sections MUST be at the very end of the document, never interleaved between dated entries. |
| **R-DOC-15** | **gitignore must exclude paired EN files** | If a Chinese `.md` is excluded in `.gitignore` (e.g., `Minibili.md`, `README_REFLECT.md`, `docs/benchmark.md`), its `_EN.md` counterpart MUST also be added to `.gitignore`. Wildcard rules like `incident-*.md` already cover both. Violating this rule will expose private documents publicly. |
| **R-DOC-16** | **New .md files must create EN version simultaneously** | When creating any new Chinese `.md` file, a corresponding `_EN.md` English version MUST be created in the same commit and registered in `check_en_sync.py`'s whitelist. Single-language files are forbidden. |

---

### 10. Encoding Rules

| ID          | Rule                             | Description |
| :---------- | :------------------------------- | :---------- |
| **R-ENCODE-1** | **No UTF-8 BOM in project files** | BOM (Byte Order Mark) causes Go compilation failures and Markdown rendering issues. Use `python scripts/check_bom.py` to detect all `.go`, `.md`, `.yaml`, `.json`, `.py` files. Script supports `--fix` for auto-repair and `--path` for path filtering. Use `python scripts/safe_write.py` for file writes (auto-strips BOM). NEVER use `Set-Content` for text file writes. |
| **R-ENCODE-2** | **Run BOM check before commit** | Run `python scripts/check_bom.py` scanning `internal/`, `cmd/`, `deploy/`, `scripts/`, `docs/` text files (`.go`, `.md`, `.yaml`, `.json`, `.py`) and root `.md` files. Check both file-level BOM and embedded `\ufeff` in content. Fix with `--fix` before committing. |

---

### 11. Build & Cache Cleanup

| ID          | Rule                             | Description |
| :---------- | :------------------------------- | :---------- |
| **R-BUILD-1** | **Cross-compilation MUST use Makefile** | NEVER manually set `$env:GOOS=linux` + `go build` in PowerShell. MUST use `make build-linux` which ensures GOCACHE/GOTMPDIR are redirected to D: drive. |
| **R-BUILD-2** | **Clean Go cache after each build** | The `build-linux` target recipe includes `python scripts/clean_go_cache.py` at the end, auto-deleting all `go-*/gc-*/gm-*` dirs under C:\\Temp and `AppData\Local\go-build`. New build targets must also call this script. |
| **R-BUILD-3** | **No compile cache on C: drive** | GOCACHE and GOTMPDIR MUST point to D: drive (currently `D:\Minibili\.gocache` and `D:\Minibili\.gotmp`), both in `.gitignore`. If locally redirected temporarily, run `python scripts/clean_go_cache.py --local-only` after. |

---

### 12. Local Verification

| ID          | Rule                             | Description |
| :---------- | :------------------------------- | :---------- |
| **R-VERIFY-1** | **Must pass local tests before push** | Any code change (frontend/backend/docs) MUST pass local verification before pushing to GitHub: backend runs `go build ./...` + `go vet ./...` + related `go test`, frontend runs `npm run build`. Verification failure prohibits push. |
| **R-VERIFY-2** | **Skipping verification is a violation** | If local test environment issues prevent compilation/testing, fix the environment first. NEVER skip verification citing "environment issues" and push directly. |
| **R-VERIFY-3** | **Run full test suite before PR** | Before submitting a PR, run `go test ./... -count=1` to ensure all tests pass. NEVER skip citing "only changed one line." |
| **R-VERIFY-4** | **Author (earthcake) must manually verify before push** | After each code change, the project author (earthcake) must personally log in and verify functionality works correctly and is bug-free before allowing push. NEVER push without author verification. |
