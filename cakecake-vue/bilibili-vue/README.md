# cakecake 前端（cakecake-web）

仿 B 站界面的 **cakecake** 用户端 + 运营后台，对接仓库根目录 Go API（`/api/v1`）。npm 包名：`cakecake-web`（`package.json`）。

| 场景 | 说明 |
|------|------|
| **日常开发（推荐）** | `VITE_MINIBILI_API=true`，登录 / 视频 / 搜索 / 私信等均走后端 |
| **纯 UI 演示** | `VITE_MINIBILI_API=false`，部分页面用 `src/mock/localApi.js` 占位 |

后端启动与环境见仓库根 **[README.md](../../README.md)**。

---

## 快速开始

```bash
npm install
cp .env.example .env.local    # 或 .env（二者均被 gitignore，勿提交）
npm run dev      # http://localhost:8888
```

`.env.local` / `.env` 最小示例（对接本仓库 Go 后端）：

```env
VITE_MINIBILI_API=true
VITE_USE_REMOTE_API=false
# VITE_REMOTE_API_BASE=   # 留空 → Vite 代理到 http://127.0.0.1:8080
```

生产构建：

```bash
cp .env.production.example .env.production
npm run build    # 产出 dist/，部署见 deploy/DEPLOY.md
```

---

## 环境变量

| 变量 | 说明 |
|------|------|
| `VITE_MINIBILI_API` | `true` 时对接 Go 后端（cakecake 模式） |
| `VITE_USE_REMOTE_API` | 旧版远程 mock 域名，cakecake 模式保持 `false` |
| `VITE_REMOTE_API_BASE` | API 根地址；开发留空走 `vite.config.js` 代理 |
| `VITE_VIDEO_UPLOAD_DISABLED` | `true` 时创作中心禁止实际上传视频文件（仍可保存元数据）；生产见 `.env.production.example` |

后端 MySQL / Redis / OSS / ES 等见根目录 `.env.example`。关闭上传时的运维流程见 **[docs/manual-video-ingest.md](../../docs/manual-video-ingest.md)**。

---

## 常用路由（Hash 模式）

| 路径 | 说明 |
|------|------|
| `#/` | 首页 |
| `#/video/BV{id}` | 播放页（无效 BV 或不存在稿件 → `#/404`） |
| `#/search/all?keyword=…` | 搜索（需后端 ES） |
| `#/minibili/login` | 登录 |
| `#/minibili/register` | 注册 |
| `#/minibili/account` | 个人中心 |
| `#/minibili/up/:userId` | 个人空间 |
| `#/upload` | 创作中心 |
| `#/admin` | 运营后台 |
| `#/404` | 404 页（未知路径会自动重定向到此） |

---

## 目录速览

```
src/
├── api/              # 接口封装（minibili.ts、index.js）
├── pages/            # 页面（home、video、minibili、upload、admin …）
├── components/       # 公共组件
├── store/            # Vuex
├── mock/localApi.js  # API 失败占位 / 非核心 Tab 演示
└── constants/        # 站点标题、分区等常量
```

---

## 脚本与协作

| 命令 / 文件 | 说明 |
|-------------|------|
| `npm run lint` | ESLint |
| `npm run check:encoding` | 检查中文乱码（`????`） |
| [scripts/README.md](./scripts/README.md) | 维护脚本说明 |
| [AGENTS.md](./AGENTS.md) | 协作者须知（如 PersonalSpace 编码） |

---

## 依赖与锁文件

请使用 **npm**（`package-lock.json`）。若不用 Yarn，可删除 `yarn.lock`，避免与 npm 锁文件混用。

---

## 部署注意

- 使用根目录 CI 与 **`deploy/nginx-minibili.conf`**，勿用仓库内旧 Docker / Travis 配置。
- 生产 Nginx 反代 `/api` 时，前端 **不要** 设置 `VITE_REMOTE_API_BASE`。

完整步骤：**[deploy/DEPLOY.md](../../deploy/DEPLOY.md)**。
