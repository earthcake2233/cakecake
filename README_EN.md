<p align="center">
  <a href="README.md">
    <img src="https://img.shields.io/badge/🇨🇳中文-999999?style=flat-square" alt="中文">
  </a>
  <strong><img src="https://img.shields.io/badge/🇬🇧English-00a1d6?style=flat-square" alt="English"></strong>
</p>

# cakecake 🍰

A full-stack video-sharing community built with Go + Vue3, covering video upload, real-time danmaku, nested comments, full-text search, AI assistant, and more. Frontend brand: **cakecake** · backend module: `minibili`.

<p align="center">
  <a href="https://chengzisoft.top/#/">
    <img src="https://img.shields.io/badge/Live Demo-chengzisoft.top-00a1d6?style=flat-square" alt="Live Demo">
  </a>
  &nbsp;&nbsp;
  <a href="https://b23.tv/9VnJIWm">
    <img src="https://img.shields.io/badge/Demo Video-B站-00a1d6?style=flat-square&logo=bilibili" alt="Demo Video">
  </a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.25-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/Gin-009688?style=flat-square&logo=gin&logoColor=white" alt="Gin">
  <img src="https://img.shields.io/badge/GORM-3776AB?style=flat-square&logo=go&logoColor=white" alt="GORM">
  <img src="https://img.shields.io/badge/Vue-3.5-4FC08D?style=flat-square&logo=vuedotjs&logoColor=white" alt="Vue">
  <img src="https://img.shields.io/badge/Vite-6-646CFF?style=flat-square&logo=vite&logoColor=white" alt="Vite">
  <img src="https://img.shields.io/badge/TypeScript-3178C6?style=flat-square&logo=typescript&logoColor=white" alt="TypeScript">
  <img src="https://img.shields.io/badge/MySQL-4479A1?style=flat-square&logo=mysql&logoColor=white" alt="MySQL">
  <img src="https://img.shields.io/badge/Redis-DC382D?style=flat-square&logo=redis&logoColor=white" alt="Redis">
  <img src="https://img.shields.io/badge/RabbitMQ-FF6600?style=flat-square&logo=rabbitmq&logoColor=white" alt="RabbitMQ">
  <img src="https://img.shields.io/badge/Elasticsearch-00BFB3?style=flat-square&logo=elasticsearch&logoColor=white" alt="Elasticsearch">
  <img src="https://img.shields.io/badge/FFmpeg-007808?style=flat-square&logo=ffmpeg&logoColor=white" alt="FFmpeg">
  <img src="https://img.shields.io/badge/WebSocket-010101?style=flat-square&logo=socket.io&logoColor=white" alt="WebSocket">
  <img src="https://img.shields.io/badge/Tests-496%20passing-00a1d6?style=flat-square&logo=vitest" alt="Tests">
  <img src="https://img.shields.io/badge/Coverage-54%25-success?style=flat-square&logo=vitest" alt="Coverage">
  <img src="https://img.shields.io/badge/Go%20Tests-27%20files-00ADD8?style=flat-square&logo=go" alt="Go Tests">
</p>

**Capabilities**: JWT auth, video/article submission & review, feed & follow, private messaging (WebSocket), video upload & async transcoding (FFmpeg + RabbitMQ + OSS), real-time danmaku (bullet comments), comments & notifications, search (Elasticsearch optional), AI assistant (DeepSeek optional), admin dashboard.

---

## Screenshots

<table>
  <tr>
    <td align="center"><b>Home</b><br><img src="docs/images/homepage.png" alt="Home" width="400"/></td>
    <td align="center"><b>Video Player (with danmaku)</b><br><img src="docs/images/video-player.png" alt="Video Player" width="400"/></td>
  </tr>
  <tr>
    <td align="center"><b>Search</b><br><img src="docs/images/search.png" alt="Search" width="400"/></td>
    <td align="center"><b>Profile</b><br><img src="docs/images/profile.png" alt="Profile" width="400"/></td>
  </tr>
  <tr>
    <td align="center"><b>User Space</b><br><img src="docs/images/personal-space.png" alt="User Space" width="400"/></td>
    <td align="center"><b>Dynamics</b><br><img src="docs/images/dynamic.png" alt="Dynamics" width="400"/></td>
  </tr>
  <tr>
    <td align="center"><b>Ranking</b><br><img src="docs/images/ranking-list.png" alt="Ranking" width="400"/></td>
    <td align="center"><b>Messages</b><br><img src="docs/images/message-center.png" alt="Messages" width="400"/></td>
  </tr>
</table>

---

## Documentation

| Document | Audience | Description |
|----------|----------|-------------|
| **This file** | Full-stack / Backend | Setup, API conventions, testing |
| [cakecake-vue/bilibili-vue/README.md](./cakecake-vue/bilibili-vue/README.md) | Frontend | Installation, env vars, dev/build |
| [deploy/DEPLOY.md](./deploy/DEPLOY.md) | DevOps | Production deployment (Nginx, systemd, OSS, ES) |
| [docs/manual-video-ingest.md](./docs/manual-video-ingest.md) | DevOps | CLI video upload when web upload is disabled |
| [docs/ai-gateway.md](./docs/ai-gateway.md) | DevOps | AI assistant (DeepSeek) configuration |
| [.github/workflows/deploy.yml](./.github/workflows/deploy.yml) | DevOps | Optional GitHub Actions CI |
| [docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md) | Full-stack / Interview | System architecture & design decisions (Chinese) |
| [docs/ARCHITECTURE_EN.md](./docs/ARCHITECTURE_EN.md) | Full-stack / Interview | System architecture & design decisions |
| [SPEC.md](./SPEC.md) | Developer | Feature specs & acceptance criteria |
| [Rule.md](./Rule.md) | Developer | Engineering rules & conventions |
| [Skill.md](./Skill.md) | Developer | Standard operations guide |

---

## Repository Structure

```
Minibili/
├── cmd/mini-bili/             # Go entrypoint
├── internal/                  # handler / service / worker / ws …
├── configs/                   # sensitive_words.txt; ip2region_v4.xdb (download manually, see .gitignore)
├── deploy/                    # Nginx & systemd templates
├── go.mod                     # module minibili
└── cakecake-vue/
    └── bilibili-vue/          # Vue 3 + Vite frontend (see subdirectory README)
```

`bilibili-vue/go.mod` is isolated from the root module to prevent `go test ./...` from scanning `node_modules`.

---

## 5-Minute Local Setup

**1. Backend** (repository root)

```bash
cp .env.example .env          # Fill in JWT_SECRET, MYSQL_DSN, REDIS_*, RABBITMQ_URL, OSS_*, etc.
go mod tidy
go build -o ./bin/mini-bili ./cmd/mini-bili/
./bin/mini-bili               # Default :8080; health check: GET /api/v1/health
```

MySQL database must exist (e.g. `minibili`); tables are created by GORM **AutoMigrate** on first startup.

**2. Frontend**

```bash
cd cakecake-vue/bilibili-vue
npm install
cp .env.example .env.local    # At minimum: VITE_MINIBILI_API=true
npm run dev                   # http://localhost:8888
```

**3. Verify**

- Homepage loads, API goes through `/api/v1` (Vite proxy to `127.0.0.1:8080`)
- Login / Register: `#/minibili/login`, `#/minibili/register`
- Invalid routes or missing videos → `#/404`

Frontend details: **[bilibili-vue/README.md](./cakecake-vue/bilibili-vue/README.md)**

---

## Dependencies

| Component | Purpose |
|-----------|---------|
| **Go** 1.22+ (`go.mod` currently 1.25) | Backend |
| **Node.js** + **npm** | Frontend (npm only, don't mix with yarn) |
| **MySQL** | Persistence |
| **Redis** | Play counts, danmaku cooldown, Refresh Token |
| **RabbitMQ** | Transcoding queue (required by spec, cannot be replaced with Redis List) |
| **Elasticsearch** (optional) | Full-text search — shows "not available" if unconfigured |
| **FFmpeg / ffprobe** | Transcoding & cover thumbnail extraction |
| **Alibaba Cloud OSS** | `videos/`, `covers/`, etc. |

---

## Backend Configuration

Copy [`.env.example`](./.env.example) → `.env` and configure at minimum:

- `JWT_SECRET`, `MYSQL_DSN`
- `REDIS_*`, `RABBITMQ_URL`
- `OSS_*` (Endpoint, AccessKey, Bucket)
- `SENSITIVE_WORDS_FILE`
- `TEMP_UPLOAD_DIR`
- `ELASTICSEARCH_*` (optional; also supports OpenSearch / Bonsai)
- `VIDEO_UPLOAD_DISABLED` (optional — disables browser upload while keeping metadata submission)

### Air hot-reload (optional)

```bash
go install github.com/air-verse/air@latest
air    # Run in repository root; loads .env
```

---

## HTTP API Conventions

- Prefix: `/api/v1`
- Response: `{ "code": number, "msg": string, "data": object | null }`
- Write operations & WebSocket: `Authorization: Bearer <access_token>`

---

## Testing

### Frontend (Vitest)

```bash
cd cakecake-vue/bilibili-vue
npm run test        # 50 test files, 496 test cases
npm run test:ui     # Vitest UI dashboard
npm run coverage    # Coverage report (~54% statement coverage)
```

### Backend (Go test)

```bash
go test ./... -count=1
# Integration tests (requires MySQL/Redis, no seed data needed)
go test -tags=integration ./internal/handler/... -count=1
```

> Backend: **27 test files** covering handler / service / ws / pkg core modules.
> Integration tests use SQLite in-memory database, no external service dependency.

```bash
go test ./... -count=1
go test -tags=integration ./internal/handler/... -count=1
```

---

## Deployment

See **[deploy/DEPLOY.md](./deploy/DEPLOY.md)**. Optional **[GitHub Actions](./.github/workflows/deploy.yml)** for auto-deploy on `main` push.

---

## License

[Non-Commercial License](./LICENSE) — personal/educational use permitted, commercial use prohibited.
