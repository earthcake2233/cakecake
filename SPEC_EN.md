<p align="center">
  <a href="SPEC.md">
    <img src="https://img.shields.io/badge/🇨🇳中文-999999?style=flat-square" alt="中文">
  </a>
  <strong><img src="https://img.shields.io/badge/🇬🇧English-00a1d6?style=flat-square" alt="English"></strong>
</p>

  </a>
  </a>
</p>

  </a>
</p>

# Mini-Bili v1.0 Design Specification (SPEC)

**Version**: v1.0
**Last Updated**: 2026-05-12
**Nature**: Deterministic requirements specification

---

### 1. Version Goals & Scope

#### 1.1 Core Goals
This version MUST deliver the following three core capabilities:
1. **User Authentication & Video Management**: Users can register, log in, **maintain personal info** (username, avatar, password) and **"My Videos" list**, upload any common video format, system transcodes server-side to H.264 MP4 and uploads to OSS, manage video status and listing, support custom cover upload.
2. **Real-time Danmaku System**: Users can send danmaku while watching videos, and see other users' danmaku in real-time (delay <= 200ms).
3. **Multi-level Comment System**: Users can post and reply to comments below videos, supporting up to 2-level nesting. Like operations trigger notifications displayed in a standalone Message Center page.

#### 1.2 Explicit Exclusions (Non-v1.0 Goals)
- No video recommendation algorithm.
- No live streaming.
- No social features (private messages, feeds, follows).
- No video search functionality (videos accessible only via list browsing or direct ID entry).
- No payment, membership, or commercial features.
- No CDN distribution ? all services on a single server.
- No admin management system.
- No content moderation functionality.

---

### 2. Functional Requirements

#### F0: User Profile Management (Auth Extension)

| Requirement | Specification |
| :---------- | :------------ |
| Get current user info | `GET /api/v1/users/me`, requires valid Access Token. Response `data`: `user_id` (int), `username`, `avatar_url` (string, empty if not set), `created_at` (`YYYY-MM-DD HH:MM:SS`). |
| Modify username | `PUT /api/v1/users/me`, Body JSON: `{ "username": "..." }`. Same rules as registration (3-20 chars, supports Chinese, letters, digits, underscores). Returns `40006` (`CodeUsernameExists`) if duplicate. |
| Modify avatar | `POST /api/v1/users/me/avatar`, `multipart/form-data`, field name **`avatar`**. Format: JPEG/PNG/GIF/BMP/WEBP only; single file <= 5 MB (Rule R-BIZ-8). Uploads to OSS `avatars/{user_id}.{ext}` (NF-7). Returns `50000` if OSS not configured. |
| Modify password | `PUT /api/v1/users/me/password`, Body JSON: `{ "old_password", "new_password" }`. New password >= 8 chars; stored with bcrypt (R-AUTH-2). Returns HTTP 403 with `40301` (`CodePasswordMismatch`, distinct from generic 40300). |
| My videos list | `GET /api/v1/users/me/videos` (F2-b): returns all statuses including `failed` with `fail_reason`. |

---

#### F1: User Authentication

| Requirement | Specification |
| :---------- | :------------ |
| Register | Username + password. Unique username check; returns error on duplicate. |
| Login | Username + password. Returns JWT Token. |
| Password storage | bcrypt hash only; never plaintext. |
| Authorization | All endpoints modifying user data require valid token. |

---

#### F2: Video Management

| Requirement | Specification |
| :---------- | :------------ |
| Upload | `POST /api/v1/videos`, `multipart/form-data`. Fields: `title` (required), `description` (optional), `cover` (optional image), `zone` (required). Server transcodes async. |
| Play | `GET /api/v1/videos/:id` returns video metadata. Video file served from OSS. |
| Delete | `DELETE /api/v1/videos/:id`. Only owner can delete. |
| List (homepage) | `GET /api/v1/videos?page=&page_size=&zone=&sort=`. |
| Cover upload | Custom cover via `POST /api/v1/videos/:id/cover`. |

---

#### F3: Danmaku System

| Requirement | Specification |
| :---------- | :------------ |
| Send danmaku | `POST /api/v1/videos/:id/danmaku`. Body: text, color, timestamp. |
| Real-time receive | WebSocket subscription to video room. Relay via Redis Pub/Sub for multi-instance. |
| History | `GET /api/v1/videos/:id/danmaku?current_time=X`. Returns danmaku in time window [X-10s, X+2s]. |

---

#### F4: Comment System

| Requirement | Specification |
| :---------- | :------------ |
| Post comment | `POST /api/v1/videos/:id/comments`. Max 2-level nesting. |
| List comments | `GET /api/v1/videos/:id/comments?page=&page_size=&sort=`. |
| Delete comment | `DELETE /api/v1/comments/:id`. UP? can delete any; users only their own. Cascade deletes children. |
| Like comment | `POST /api/v1/comments/:id/like`. Toggle. Sends notification. |

---

#### F5: Notifications & Message Center

| Requirement | Specification |
| :---------- | :------------ |
| Message Center | `GET /api/v1/notifications?type=`. Standalone page `/messages` with left nav: Replies, @Mentions, Likes Received, System Notices, My Messages. |
| Unread count | Badge per category. WebSocket push for real-time updates. |

---

#### F6: Search

| Requirement | Specification |
| :---------- | :------------ |
| Search videos | `GET /api/v1/search?keyword=&type=&sort=&page=&page_size=`. Elasticsearch primary, MySQL LIKE fallback. |
| Search cache | Redis query cache (30s TTL, SHA256 key -> JSON value). |

---

### 3. Acceptance Criteria

| ID | Description | Spec |
| :--- | :---------- | :--- |
| **AC-1** | Registration duplicate | Duplicate username -> clear error, not 500. |
| **AC-2** | Login success | Correct credentials -> JWT, useable for subsequent requests. |
| **AC-3** | Upload video | Valid file -> transcode starts. User sees status flow: uploading -> transcoding -> published. |
| **AC-4** | Danmaku real-time | Send in room A, appear in room A within 200ms (local dev). |
| **AC-5** | Comment nesting | Max 2 levels. Reply to level-2 shows as level-2. |
| **AC-6** | Comment cascade delete | Delete parent -> all children deleted. |
| **AC-7** | UP? permissions | Can delete any comment on own videos. |
| **AC-8** | Like notification | Like triggers notification to comment author. |
| **AC-9** | Notification categories | 5 nav items, each filters correctly, unread badges accurate. |
| **AC-10** | Password change | Old password correct -> change succeeds. Wrong -> 403. |
| **AC-11** | Avatar upload | Valid image -> OSS URL returned, displayed on frontend. Invalid format/size -> clear error. |
| **AC-12** | Video zones | Upload selects zone; homepage filters by zone. |
| **AC-13** | Cover upload | Upload video with cover -> custom cover displayed. No cover -> auto-generate frame 1 as default. |
| **AC-14** | Cover format validation | Invalid cover -> entire request rejected with clear error. |
| **AC-15** | Comment delete permissions | UP? can delete any; users only own. |
| **AC-16** | Comment delete cascade | Delete parent with children -> all children sync deleted. |

---

### 4. Compatibility Requirements

| ID | Type | Spec |
| :--- | :--- | :-- |
| **BC-1** | Backward | First formal version, no legacy burden. Once interfaces and models defined, future versions must not make breaking changes without review. |
| **BC-2** | Forward | Code architecture must support smooth future split into Kratos microservices. v1.0 monolith must not introduce coupling that blocks later decomposition. Video table must reserve `status` field values `pending_review` and `rejected` for future moderation. |

---

### 5. Non-Functional Requirements

| ID | Type | Spec |
| :--- | :--- | :-- |
| NF-1 | Concurrency | Danmaku system must support 100 simultaneous users in same room without crash/deadlock. |
| NF-2 | Latency | Danmaku end-to-end delay <= 200ms in local dev environment. |
| NF-3 | Storage | MySQL: users, video metadata, status, comments, likes, notifications. Redis: danmaku real-time relay, video play count hot data. |
| NF-4 | Message Queue | Video transcode tasks must use message queue for async dispatch; no synchronous FFmpeg in upload requests. |
| NF-5 | Frontend | Vue 3 + Vite SPA, no SSR. |
| NF-6 | Backend | Go language. Standard project layout. Code architecture must support future Kratos microservice split. |
| NF-7 | File Storage | All files in Alibaba Cloud OSS single Bucket `mini-bili`, organized by prefix: `videos/{video_id}.mp4`, `covers/{video_id}.{ext}`, `avatars/{user_id}.{ext}`. Raw uploads and temp files only on server during processing, cleaned immediately after. |
| NF-8 | API Style | All external HTTP endpoints follow RESTful conventions. |
| NF-9 | Message Center | Standalone first-class page (`/messages`) with left nav bar containing exactly five categories: Replies, @Mentions, Likes Received, System Notices, My Messages. Each displays unread count badge. |

---

### 6. Technology Stack

| Layer | Technology |
| :---- | :--------- |
| Backend Language | Go 1.25+ |
| Web Framework | Gin |
| ORM | GORM |
| Database | MySQL 8.0 |
| Cache | Redis 7.0 |
| Message Queue | RabbitMQ |
| Search Engine | Elasticsearch 8.x (Tencent Cloud Serverless or Bonsai) |
| Object Storage | Alibaba Cloud OSS |
| Transcoding | FFmpeg |
| Frontend | Vue 3 + Vite + TypeScript |
| UI Framework | Custom (based on Bilibili design) |
| Auth | JWT (Access + Refresh tokens) |

---

### 7. API Endpoint Summary

| Method | Path | Description |
| :----- | :--- | :---------- |
| POST | `/api/v1/auth/register` | User registration |
| POST | `/api/v1/auth/login` | User login |
| POST | `/api/v1/auth/refresh` | Refresh access token |
| GET | `/api/v1/users/me` | Get current user profile |
| PUT | `/api/v1/users/me` | Update username |
| POST | `/api/v1/users/me/avatar` | Upload avatar |
| PUT | `/api/v1/users/me/password` | Change password |
| GET | `/api/v1/users/me/videos` | My videos list |
| POST | `/api/v1/videos` | Upload video |
| GET | `/api/v1/videos/:id` | Get video detail |
| DELETE | `/api/v1/videos/:id` | Delete video |
| GET | `/api/v1/videos` | List videos (homepage) |
| POST | `/api/v1/videos/:id/cover` | Upload custom cover |
| POST | `/api/v1/videos/:id/danmaku` | Send danmaku |
| GET | `/api/v1/videos/:id/danmaku` | Get danmaku history |
| WS | `/api/v1/ws?video_id=` | WebSocket for danmaku |
| POST | `/api/v1/videos/:id/comments` | Post comment |
| GET | `/api/v1/videos/:id/comments` | List comments |
| DELETE | `/api/v1/comments/:id` | Delete comment |
| POST | `/api/v1/comments/:id/like` | Toggle comment like |
| GET | `/api/v1/notifications` | List notifications |
| GET | `/api/v1/search` | Search videos |
| GET | `/api/v1/health` | Health check |
