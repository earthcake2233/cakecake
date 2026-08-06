<p align="center">
  <a href="docs/manual-video-ingest.md">
    <img src="https://img.shields.io/badge/🇨🇳中文-999999?style=flat-square" alt="中文">
  </a>
  <strong><img src="https://img.shields.io/badge/🇬🇧English-00a1d6?style=flat-square" alt="English"></strong>
</p>

  </a>
  </a>
</p>

  </a>
</p>

# Local Manual Video Publishing (When Server Cannot Transcode)

When production has `VITE_VIDEO_UPLOAD_DISABLED=true`, the user-facing Creator Center preserves the UI but shows "Cloud upload temporarily disabled." Admins can transcode on their **Windows machine**, upload to OSS, then write the draft record to cloud server **MySQL**.

---

## Workflow Overview

```
1. Local FFmpeg transcode MP4 + screenshot cover
2. Upload to Alibaba Cloud OSS (videos/{id}.mp4, covers/{id}.jpg)
3. MySQL INSERT/UPDATE videos table
4. (Optional) Admin panel review and publish, or directly status=published
```

---

## 1. Prepare Files Locally

Transcode example (adjust paths):

```bash
ffmpeg -i original.mp4 -c:v libx264 -preset medium -crf 23 -c:a aac -movflags +faststart out.mp4
ffmpeg -i out.mp4 -ss 00:00:01 -vframes 1 cover.jpg
```

Upload to OSS (requires [ossutil](https://help.aliyun.com/document_detail/120075.html) or console):

```bash
ossutil cp out.mp4 oss://your-bucket/videos/42.mp4
ossutil cp cover.jpg oss://your-bucket/covers/42.jpg
```

Public URL example (must match `.env` `OSS_PUBLIC_URL_PREFIX`):

```
https://your-bucket.oss-cn-beijing.aliyuncs.com/videos/42.mp4
https://your-bucket.oss-cn-beijing.aliyuncs.com/covers/42.jpg
```

`42` is the **videos table primary key id**. If no record yet, INSERT first to get auto-increment id, then upload OSS with that id. Or pre-reserve an id and upload first.

---

## 2. Find user_id

```sql
SELECT id, username FROM users WHERE username = 'your_username';
```

Note the `id`, e.g., `3`.

---

## 3. Insert Video Record

```sql
INSERT INTO videos (
  user_id, title, description, duration_sec,
  status, video_url, cover_url, zone,
  play_count, danmaku_count, comment_count, like_count, fav_count, coin_count,
  comments_closed, comments_curated, danmaku_closed,
  tags_json, created_at, updated_at
) VALUES (
  3,
  'Video Title',
  'Optional description',
  125.5,
  'published',
  'https://your-bucket.oss-cn-beijing.aliyuncs.com/videos/42.mp4',
  'https://your-bucket.oss-cn-beijing.aliyuncs.com/covers/42.jpg',
  'Animation',
  0, 0, 0, 0, 0, 0,
  0, 0, 0,
  '[]',
  NOW(), NOW()
);
```

- **`status`**:
  - `published` -- directly live (homepage/space visible)
  - `pending_review` -- requires admin `#/admin` review before publish
- **`zone`**: zone name, e.g., `Animation`, `Life-Daily` (see frontend zone constants)
- After insert, access: `https://your-domain/#/video/BV{id}` (`BV` + auto-increment `id`)

Check new id:

```sql
SELECT LAST_INSERT_ID();
```

If OSS path used a placeholder id, must match `LAST_INSERT_ID()`. Simpler approach: **INSERT first to get id, then upload OSS and UPDATE url**.

---

## 4. Recommended: INSERT First, Then UPDATE URL

```sql
INSERT INTO videos (user_id, title, description, status, zone, created_at, updated_at)
VALUES (3, 'Title', 'Description', 'pending_review', 'Animation', NOW(), NOW());

SELECT LAST_INSERT_ID();   -- assume 42

UPDATE videos SET
  video_url = 'https://your-bucket.oss-cn-beijing.aliyuncs.com/videos/42.mp4',
  cover_url = 'https://your-bucket.oss-cn-beijing.aliyuncs.com/covers/42.jpg',
  duration_sec = 125.5,
  status = 'published',
  updated_at = NOW()
WHERE id = 42;
```

---

## 5. Search Indexing (Optional)

If Elasticsearch is configured, restarting the backend after publishing will gradually index new videos. Or admin review approval in admin panel triggers indexing. Can ignore if ES not configured.

---

## 6. Re-enable Web Upload

Modify frontend `.env.production` locally:

```env
VITE_VIDEO_UPLOAD_DISABLED=false
```

Then `npm run build` and upload `dist/`.

Local dev `.env.local` can keep unset or `false` for normal upload endpoint debugging.

For Docker Compose (published images + dev override rebuild):

```bash
# set VITE_VIDEO_UPLOAD_DISABLED=false in .env, then
docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d --build web backend
```

Note: the published Docker Hub images have the frontend flag baked to `true` — a local frontend rebuild is required for this to take effect; full steps live in [deploy/DEPLOY_EN.md](../deploy/DEPLOY_EN.md).

---

## 7. Docker Compose Environments

The compose demo mode also disables upload by default; manual ingest is identical (transcode locally → OSS → insert into DB) — only where you run the SQL differs:

- The database runs in the `cakecake-mysql` container; make sure the stack is up: `docker compose ps`
- Open the MySQL shell and run the same SQL as sections 2 / 3 / 4:

```bash
docker compose exec mysql mysql -uroot -p cakecake
# Password: MYSQL_ROOT_PASSWORD from .env (default cakecake_dev); database: MYSQL_DATABASE (default cakecake)
```

- Finding user_id and the INSERT statements are exactly the same as sections 2 / 3 / 4; `user_id` can be one of the demo accounts (e.g., `暗猫の祝福`)
- After inserting, open `http://localhost:8888/#/video/BV{id}`; with ES enabled in compose, `docker compose restart backend` triggers indexing

---

## Related Files

| File | Description |
|------|-------------|
| `cakecake-vue/cakecake-web/.env.production` | Production upload toggle |
| `cakecake-vue/cakecake-web/src/utils/videoUploadPolicy.js` | Frontend toggle logic |
| `/opt/minibili/.env` | Backend OSS config (manual OSS upload uses same Bucket) |
