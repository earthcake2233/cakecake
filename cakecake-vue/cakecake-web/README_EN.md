<p align="center">
  <a href="README.md">
    <img src="https://img.shields.io/badge/🇨🇳中文-999999?style=flat-square" alt="中文">
  </a>
  <strong><img src="https://img.shields.io/badge/🇬🇧English-00a1d6?style=flat-square" alt="English"></strong>
</p>

# cakecake Frontend (cakecake-web)

A Bilibili-style SPA (user-facing + admin panel) powered by Vue 3 + Vite, connecting to the Go API (`/api/v1`) in the repository root. npm package name: `cakecake-web` (`package.json`).

| Mode | Description |
|------|-------------|
| **Development (recommended)** | `VITE_MINIBILI_API=true` — login, video, search, DMs all hit the backend |
| **UI-only demo** | `VITE_MINIBILI_API=false` — some pages use `src/mock/localApi.js` as placeholder |

See root **[README.md](../../README.md)** for backend setup and environment.

---

## Quick Start

```bash
npm install
cp .env.example .env.local    # or .env (both gitignored, do not commit)
npm run dev      # http://localhost:8888
```

Minimal `.env.local` / `.env` example (pointing to the Go backend):

```env
VITE_MINIBILI_API=true
VITE_USE_REMOTE_API=false
# VITE_REMOTE_API_BASE=   # leave empty → Vite proxies to http://127.0.0.1:8080
```

Production build:

```bash
cp .env.production.example .env.production
npm run build    # outputs dist/, deploy via deploy/DEPLOY.md
```

---

## Environment Variables

| Variable | Description |
|----------|-------------|
| `VITE_MINIBILI_API` | `true` enables the cakecake Go backend integration |
| `VITE_USE_REMOTE_API` | Legacy remote mock domain; keep `false` in cakecake mode |
| `VITE_REMOTE_API_BASE` | API base URL; leave empty in dev to use Vite proxy |
| `VITE_VIDEO_UPLOAD_DISABLED` | `true` disables actual video file upload in Creator Center (metadata is still saved); see `.env.production.example` |

For MySQL / Redis / OSS / ES config see root `.env.example`.  
When upload is disabled, follow **[docs/manual-video-ingest.md](../../docs/manual-video-ingest.md)**.

---

## Key Routes (Hash mode)

| Path | Description |
|------|-------------|
| `#/` | Home |
| `#/video/BV{id}` | Video player (invalid BV or missing → `#/404`) |
| `#/search/all?keyword=…` | Search (requires backend ES) |
| `#/cakecake/login` | Login |
| `#/cakecake/register` | Register |
| `#/cakecake/account` | Account center |
| `#/cakecake/up/:userId` | User space |
| `#/upload` | Creator Center |
| `#/admin` | Admin panel |
| `#/404` | 404 (unknown paths redirect here) |

---

## Directory Overview

```
src/
├── api/              # API wrappers (cakecake.ts, index.js)
├── pages/            # Pages (home, video, cakecake, upload, admin …)
├── components/       # Shared components
├── store/            # Vuex
├── mock/localApi.js  # API fallback / non-core tab demo
└── constants/        # Site title, partition, etc.
```

---

## Scripts & Collaboration

| Command / File | Description |
|----------------|-------------|
| `npm run lint` | ESLint |
| `npm run check:encoding` | Detect Chinese garbled text (`????`) |
| [scripts/README.md](./scripts/README.md) | Maintenance script docs |
| `PersonalSpace.vue` | Large file — **never** edit Chinese text with non-UTF-8 tools. Use `src/i18n/*.zh-CN.ts` for translations, run `npm run check:encoding` before commit |

---

## Tests

```bash
npm run test        # 50 test files, 496 cases (Vitest + jsdom)
npm run test:watch  # Watch mode
npm run test:ui     # Vitest UI dashboard
npm run coverage    # Coverage report
```

Covers `utils/`, `store/`, `constants/`, and select `components/` and `api/` modules.

---

## Dependencies & Lockfiles

Use **npm** (`package-lock.json`). If you do not use Yarn, remove `yarn.lock` to avoid conflicts.

---

## Deployment Notes

- Use the root CI and **`deploy/nginx-minibili.conf`**. Do not use the legacy Docker / Travis configs.
- In production, the Nginx reverse proxy handles `/api`. Do **not** set `VITE_REMOTE_API_BASE` on the frontend.

Full steps: **[deploy/DEPLOY.md](../../deploy/DEPLOY.md)**.
