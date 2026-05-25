# 本地手动发布视频（服务器无法转码时）

当生产环境开启 `VITE_VIDEO_UPLOAD_DISABLED=true` 时，用户端创作中心会保留界面并提示「云端投稿暂时关闭」。管理员可在 **Windows 本机** 完成转码、上传 OSS，再在云服务器 **MySQL** 写入稿件记录。

---

## 流程概览

```text
1. 本机 FFmpeg 转码 MP4 + 截封面
2. 上传到阿里云 OSS（videos/{id}.mp4、covers/{id}.jpg）
3. MySQL 插入/更新 videos 表
4. （可选）运营后台审核发布，或直接 status=published
```

---

## 1. 本机准备文件

转码示例（PowerShell，路径按实际修改）：

```powershell
ffmpeg -i 原片.mp4 -c:v libx264 -preset medium -crf 23 -c:a aac -movflags +faststart out.mp4
ffmpeg -i out.mp4 -ss 00:00:01 -vframes 1 cover.jpg
```

上传到 OSS（需安装 [ossutil](https://help.aliyun.com/document_detail/120075.html) 或使用控制台）：

```powershell
ossutil cp out.mp4 oss://your-bucket/videos/42.mp4
ossutil cp cover.jpg oss://your-bucket/covers/42.jpg
```

公网 URL 示例（与 `.env` 中 `OSS_PUBLIC_URL_PREFIX` 一致）：

```text
https://your-bucket.oss-cn-beijing.aliyuncs.com/videos/42.mp4
https://your-bucket.oss-cn-beijing.aliyuncs.com/covers/42.jpg
```

`42` 为 **videos 表主键 id**；若尚未建记录，可先 INSERT 拿到自增 id，再按 id 上传 OSS（或先占 id 再上传）。

---

## 2. 查 user_id

```sql
SELECT id, username FROM users WHERE username = '你的用户名';
```

记下 `id`，例如 `3`。

---

## 3. 插入视频记录

```sql
INSERT INTO videos (
  user_id, title, description, duration_sec,
  status, video_url, cover_url, zone,
  play_count, danmaku_count, comment_count, like_count, fav_count, coin_count,
  comments_closed, comments_curated, danmaku_closed,
  tags_json, created_at, updated_at
) VALUES (
  3,
  '视频标题',
  '简介可选',
  125.5,
  'published',
  'https://your-bucket.oss-cn-beijing.aliyuncs.com/videos/42.mp4',
  'https://your-bucket.oss-cn-beijing.aliyuncs.com/covers/42.jpg',
  '动画',
  0, 0, 0, 0, 0, 0,
  0, 0, 0,
  '[]',
  NOW(), NOW()
);
```

- **`status`**：
  - `published` — 直接上架（首页/空间可见）
  - `pending_review` — 走运营后台 `#/admin` 审核后再发布
- **`zone`**：分区，如 `动画`、`生活-日常`（见前端分区常量）
- 插入后访问：`https://你的域名/#/video/BV{id}`（`BV` + 上表自增 `id`）

查看新 id：

```sql
SELECT LAST_INSERT_ID();
```

若 OSS 路径用了占位 id，需与 `LAST_INSERT_ID()` 一致；更简单做法是 **先 INSERT 拿 id，再上传 OSS 并 UPDATE url**。

---

## 4. 推荐：先 INSERT 再 UPDATE URL

```sql
INSERT INTO videos (user_id, title, description, status, zone, created_at, updated_at)
VALUES (3, '标题', '简介', 'pending_review', '动画', NOW(), NOW());

SELECT LAST_INSERT_ID();   -- 假设得到 42

UPDATE videos SET
  video_url = 'https://your-bucket.oss-cn-beijing.aliyuncs.com/videos/42.mp4',
  cover_url = 'https://your-bucket.oss-cn-beijing.aliyuncs.com/covers/42.jpg',
  duration_sec = 125.5,
  status = 'published',
  updated_at = NOW()
WHERE id = 42;
```

---

## 5. 搜索索引（可选）

若配置了 Elasticsearch，发布后重启后端会逐步索引；或登录运营后台审核通过时会触发索引。未配 ES 可忽略。

---

## 6. 重新开放网页上传

本机构建前改 `.env.production`：

```env
VITE_VIDEO_UPLOAD_DISABLED=false
```

再 `npm run build` 并上传 `dist/`。

本地开发 `.env.local` 不设或设为 `false`，本机仍可正常走上传接口联调。

---

## 相关文件

| 文件 | 说明 |
|------|------|
| `cakecake-vue/bilibili-vue/.env.production` | 生产是否关闭上传 |
| `cakecake-vue/bilibili-vue/src/utils/videoUploadPolicy.js` | 前端开关逻辑 |
| `/opt/minibili/.env` | 后端 OSS 等（手动上传 OSS 与本机 ossutil 用同一 Bucket） |
