<p align="center">
  <a href="SPEC.md">
    <img src="https://img.shields.io/badge/🇨🇳中文-999999?style=flat-square" alt="中文">
  </a>
  <strong><img src="https://img.shields.io/badge/🇬🇧English-00a1d6?style=flat-square" alt="English"></strong>
</p>

## Mini-Bili v1.0 Design Specification (SPEC)

**Version**: v1.0

**Last Updated**: 2026-08-01

**Nature**: Deterministic requirements specification

**Revision History**

| Version | Date       | Description                                                                                                                                |
| :------ | :--------- | :----------------------------------------------------------------------------------------------------------------------------------------- |
| v1.0    | 2026-05-12 | Original design scope: user auth, video management, danmaku, comments                                                                      |
| —       | 2026-07    | Incremental delivery beyond v1.0: feeds, follows, direct messages (WebSocket), full-text search (ES), AI assistant (DeepSeek), admin panel |

---

### 1. Version Goals & Scope

#### 1.1 Core Goals

This version MUST deliver the following three core capabilities:

1. **User Authentication & Video Management**: Users can register, log in, **maintain personal info (username, avatar, password) and the "My Videos" list**, upload any common video format, have the system transcode asynchronously server-side to H.264 MP4 and upload to OSS, manage video status and listing, and support custom cover upload.
2. **Real-time Danmaku System**: Users can send danmaku while watching videos and see other users' danmaku in real time (delay ≤ 200ms).
3. **Multi-level Comment System**: Users can post and reply to comments below videos, supporting up to **3-level nesting**. Likes trigger notifications, aggregated and displayed in a standalone Message Center page.

#### 1.2 Explicit Exclusions (Non-v1.0 Goals)

The following are OUT of scope for this version:

- No video recommendation algorithm.
- No live streaming.
- No social features (private messages, feeds, follows).
- No video search (videos accessible only via list browsing or direct ID entry).
- No payment, membership, or commercial features.
- No CDN distribution — all services on a single server.
- No admin management system.
- No content moderation functionality.

---

### 2. Functional Requirements

#### F0: User Profile Management (Auth Extension, supporting "User Auth & Video Management")

| Requirement           | Specification                                                                                                                                                                                                                                                                                   |
| :-------------------- | :---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Get current user info | `GET /api/v1/users/me`, requires a valid Access Token. Response `data`: `user_id` (uint), `username`, `avatar_url` (string, empty if not set), `created_at` (`YYYY-MM-DD HH:MM:SS`).                                                                                                            |
| Modify username       | `PUT /api/v1/users/me`, Body JSON: `{ "username": "..." }`. Username rules same as registration (3–32 chars, supports Chinese, letters, digits, underscores). If it collides with **another user**, return business code **`40006`** (`CodeUsernameExists`). Success returns `user_id`, `username`. |
| Modify avatar         | `POST /api/v1/users/me/avatar`, `multipart/form-data`, field name **`avatar`**. Formats: **JPEG, PNG, GIF, BMP, WEBP** only; single file **≤ 5 MB** (Rule **R-BIZ-8**; validation flow same as Skill **S-005** cover extension/size logic, size cap 5MB). Uploads to OSS Bucket `mini-bili`, key **`avatars/{user_id}.{ext}`** (see NF-7). Success returns `data.avatar_url`. Returns `50000` if OSS is not configured. |
| Modify password       | `PUT /api/v1/users/me/password`, Body JSON: `{ "old_password", "new_password" }`. New password **≥ 8** chars; stored with **bcrypt** (R-AUTH-2). Wrong old password returns HTTP **403** with business code **`40301`** (`CodePasswordMismatch`, distinct from the generic `40300`).                |
| My videos list        | `GET /api/v1/users/me/videos` (F2-b): returns all of the user's videos in every status, including `failed` with **`fail_reason`**, supporting the failure display in the personal center (AC-3).                                                                                                   |

---

#### F1: User Authentication

| Requirement      | Specification                                                                                                                                                                                                                                                                                         |
| :--------------- | :---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Register         | User must provide username and password. The system enforces username uniqueness and returns an error on duplicates.                                                                                                                                                                                  |
| Login            | Login with username and password. On success the server returns a JWT Token used as the credential for subsequent authenticated endpoints.                                                                                                                                                             |
| Password storage | Passwords MUST be hashed with bcrypt before storage; plaintext is forbidden.                                                                                                                                                                                                                          |
| Authorization    | All endpoints that modify user data (**get/update own info, avatar, password**, video upload, cover upload, danmaku, comments, likes, comment deletion) MUST carry a valid JWT Token in the `Authorization: Bearer <token>` header, enforced by the middleware layer. Failed validation returns 401. |

---

#### F2: Video Upload

| Requirement        | Specification                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| :----------------- | :--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Prerequisites      | User MUST be logged in.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| Request format     | Frontend uploads with `multipart/form-data`. Required fields: - `file`: video file - `title`: video title (1-80 chars) - `description`: video description (0-2000 chars). Optional: - `cover`: custom cover image. Formats: **JPEG, PNG, GIF, BMP, WEBP** only; size **≤ 10 MB**.                                                                                                                                                                                                                                                                                          |
| File format        | Video format is NOT restricted; all formats are accepted.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| File size          | Single video file MUST be ≤ 500 MB, otherwise return 400.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| Video duration     | Single video duration MUST be ≤ 30 minutes, otherwise return 400.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| Cover validation   | If the user submits a `cover` file with invalid format or size, the endpoint MUST return a clear error (e.g., "cover format unsupported" or "cover exceeds 10MB") and **reject the entire upload request**, requiring the user to fix and resubmit.                                                                                                                                                                                                                                                                                                                    |
| Post-upload flow   | After a successful upload, the raw video is stored in a local temp directory on the server. If a compliant `cover` file was submitted, it is also stored locally. The system immediately creates a video record with status `processing`, storing duration and uploader info. It then enqueues the video ID, local raw file path, and cover path (if any) to the async transcode queue. The upload endpoint returns 201 right after enqueueing, without waiting for transcode.                                                                                            |
| Transcode & upload | The async consumer processes the task in order: 1. Read the raw video from the local path, transcode with FFmpeg to an H.264 MP4, output to a local temp dir. 2. Cover handling: - If a custom cover was submitted, validate format/size and use it directly. - If no cover was submitted, extract the video's 1-second frame as a JPEG default cover. 3. Upload the transcoded MP4 to Aliyun OSS Bucket `mini-bili` at `videos/{video_id}.mp4`, record the OSS URL. 4. Upload the cover to OSS Bucket `mini-bili` at `covers/{video_id}.{ext}` (ext = actual format suffix), record the OSS URL. 5. Update the video record: status → `published`, write the transcoded URL and cover URL. 6. On success or failure, delete ALL local temp files for this task. On transcode failure, status → `failed`, record the failure reason, and only step 6 cleanup runs. |

---

#### F2-b: Video Status Management

| Requirement    | Specification                                                                                                                                                                                                                                                                  |
| :------------- | :----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Status set     | The video record's status field defines exactly five values. `pending_review` and `rejected` are reserved and not used in v1.0. - `processing`: processing - `published`: published - `failed`: transcode failed - `pending_review`: pending review (reserved, unused in v1.0) - `rejected`: review rejected (reserved, unused in v1.0) |
| Visibility     | Video lists and detail pages show ONLY `published` videos to external regular users. The uploader can see all of their own videos in the personal center, including the failure reason field.                                                                                  |
| Failure        | On transcode failure, the failure reason MUST be stored on the video record. When the uploader views it, the frontend shows "Video processing failed: {reason}".                                                                                                                 |

---

#### F3: Video Info Display & Cover

| Requirement     | Specification                                                                                                                                                                                                                                                                                     |
| :-------------- | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Displayed info  | The video playback page shows: video title, description, play count, danmaku count, comment count, duration, uploader, created time (format: `YYYY-MM-DD HH:MM:SS`).                                                                                                                              |
| Play count      | Play-count increments are stored in Redis (using `INCR`) and flushed to MySQL every 10 seconds. The page reads the displayed count from Redis.                                                                                                                                                    |
| Default cover   | During transcode, if no custom cover was submitted, the system extracts the video's 1-second frame as the default cover and uploads it to OSS Bucket `mini-bili` at `covers/{video_id}.jpg`.                                                                                                     |
| Custom cover    | Users can submit a custom cover with the upload (F2 `cover` field). After publishing, a cover-update endpoint can replace it. Requirements: - Formats: JPEG, PNG, GIF, BMP, WEBP only - Size ≤ 10 MB - Uploaded to OSS Bucket `mini-bili` at `covers/{video_id}.{ext}`, replacing the old cover URL |

---

#### F4: Video Playback

| Requirement  | Specification                                                                                                     |
| :----------- | :---------------------------------------------------------------------------------------------------------------- |
| Play source  | The frontend player plays the transcoded H.264 MP4 stored in Aliyun OSS Bucket `mini-bili` under `videos/`, via its OSS access URL. |
| Player       | The frontend uses the native HTML5 `<video>` player, no extra decoding libraries.                                 |
| Controls     | MUST support: play/pause, seeking, volume, mute, fullscreen.                                                      |

---

#### F5: Danmaku Sending

| Requirement | Specification                                                                                                                                                                                                                                                                                                                                                                                           |
| :---------- | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Send format | Users can send danmaku at any point during playback. The request MUST include: - `content`: danmaku text (1-100 chars) - `color`: color, hex codes supported (e.g., `#FF6B6B`), format MUST be `#` followed by 6 hex chars (`0-9A-Fa-f`), case-insensitive. Optional presets: `#FFFFFF`, `#FF0000`, `#00FF00`, `#0000FF`, `#FFFF00`. - `type`: position (only `scroll`, `top`, `bottom`) - `video_time`: position on the video timeline (float, seconds) |
| Cooldown    | After a successful send, the **frontend send button MUST immediately gray out (disabled) and show a 5-second countdown**, re-enabling when it reaches zero. **The backend also enforces the cooldown**; a send attempt during the cooldown returns "sending too frequently, please try again later".                                                                                                      |
| Review      | Danmaku content MUST pass sensitive-word filtering. If any dictionary word matches, the backend returns an error and refuses to store or broadcast.                                                                                                                                                                                                                                                       |
| Storage     | Successfully sent danmaku MUST be persisted to MySQL.                                                                                                                                                                                                                                                                                                                                                   |

---

#### F6: Real-time Danmaku Display

| Requirement    | Specification                                                                                                                                                                                                  |
| :------------- | :-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Protocol       | The frontend and backend danmaku service MUST use a WebSocket long connection. HTTP short polling is FORBIDDEN.                                                                                                |
| Latency        | From the sender's click to the receiver's screen render, total delay MUST be ≤ 200ms in the local dev environment.                                                                                             |
| History        | When a user opens a video, the backend pushes the most recent 200 historical danmaku of that video over the WebSocket connection in one batch. The frontend renders them at their `video_time` positions.       |
| Rendering      | The frontend MUST render danmaku with Canvas or WebGL. Multiple danmaku on the same track MUST have spacing; overlapping is forbidden.                                                                            |

---

#### F7: Comment Posting & Deletion

| Requirement  | Specification                                                                                     |
| :----------- | :------------------------------------------------------------------------------------------------ |
| Prerequisites | User MUST be logged in.                                                                            |
| Top-level    | Users can post comments directly under a video; content is plain text (1-1000 chars).              |
| Editing      | FORBIDDEN. Once posted, a comment cannot be edited.                                                |
| Delete perms | **Video owners (UP主)** can delete ANY comment under their videos. **Regular users** can only delete their own comments. |
| Cascade      | Deleting a comment MUST also delete all child comments at every level. All data of the deleted comment is removed from the DB and no longer shown. |

---

#### F8: Comment Replies

| Requirement      | Specification                                                                                                                                                                                            |
| :--------------- | :-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Nesting levels   | Comments support up to **3 levels**: - replying directly to the video: level 1 - replying to a level-1 comment: level 2 - replying to a level-2 comment: level 3                                          |
| Overflow        | When a user replies to a level-3 comment, the new comment is also treated as level 3, with its parent pointing to the replied level-3 comment. The frontend does NOT block this.                         |
| Edit & delete    | Same as F7: editing forbidden; UP主 can delete any reply, regular users only their own; deletion cascades to all child replies.                                                                           |

---

#### F9: Comment Likes & Notifications

| Requirement    | Specification                                                                                                                                                                                                                                                        |
| :------------- | :-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Interaction    | The same user can like the same comment only once; clicking again unlikes it.                                                                                                                                                                                        |
| Notification   | When user A likes user B's comment, the system checks for an existing unread aggregated like notification: - if one exists (same comment ID), append user A to the liker list, update the total count, do NOT create a new notification. - otherwise, create a new aggregated notification. |
| Format         | Display format: - 1 liker: "UserA liked your comment" - 2 likers: "UserA, UserB liked your comment" - 3+: "UserA, UserB, UserC and X others liked your comment" (show first 3 usernames, X = total). Each notification MUST include a preview of the first 15 chars of the liked comment. |
| Category       | Like notifications belong to the "Received Likes" category.                                                                                                                                                                                                          |
| Read logic     | When the user enters the Message Center, notifications are unread by default. Clicking a notification's detail marks it as read.                                                                                                                                     |

---

#### F10: Video List

| Requirement  | Specification                                                                         |
| :----------- | :------------------------------------------------------------------------------------- |
| Location     | Homepage (`/` path).                                                                   |
| Data source  | Only videos with status `published`.                                                    |
| Sorting      | Priority: 1. play count DESC 2. upload time DESC 3. danmaku count DESC                  |
| Card content | Each card shows: **cover, title, play count, danmaku count, duration, UP主, created time**. |
| Pagination   | MUST use cursor pagination. The endpoint returns the next cursor; the frontend uses it to determine whether there is a next page. |

---

### 3. Acceptance Criteria

| ID        | Acceptance Item          | Passing Standard                                                                                                                                                                                                                                                                                                              |
| :-------- | :----------------------- | :----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **AC-1**  | Register/login loop      | Open register page → enter username and password → register succeeds. Login with those credentials → JWT Token obtained. Access an authenticated endpoint with the token → returns data, not 401.                                                                                                                             |
| **AC-2**  | Upload & transcode loop  | Upload a 100MB MOV video while logged in → endpoint returns 201, video appears in the list with status "processing". After transcode, refresh → status "published", H.264 video plays.                                                                                                                                          |
| **AC-3**  | Upload failure handling  | Upload a deliberately corrupt/untranscodable video → status becomes "transcode failed". The uploader opens the personal center → sees the video with the specific failure reason.                                                                                                                                               |
| **AC-4**  | List sorting & cards     | Homepage list sorts: play count DESC, then upload time DESC, then danmaku count DESC. Videos with equal play count: newer first; equal upload time: more danmaku first. Each card correctly shows cover, title, play count, danmaku count, duration, UP主, created time.                                                       |
| **AC-5**  | Real-time danmaku        | In the same video room, 100 users online sending danmaku simultaneously. From any sender's click to another user's screen render, total delay ≤ 200ms.                                                                                                                                                                          |
| **AC-6**  | Danmaku history loading  | Opening any video pushes the most recent 200 historical danmaku over the WebSocket. The frontend renders them at their `video_time` positions.                                                                                                                                                                                |
| **AC-7**  | Danmaku sensitive words  | Sending danmaku containing a dictionary word → backend returns a clear error, not broadcast, not stored.                                                                                                                                                                                                                       |
| **AC-8**  | Danmaku cooldown         | After a successful send, the frontend button grays out with a 5-second countdown. Any send attempt during the countdown is rejected by all means. Button restores when the countdown ends.                                                                                                                                      |
| **AC-9**  | Multi-level comments     | Post a level-1 comment → succeeds. Reply to it → level-2. Reply to that → level-3. Reply to that level-3 → the new comment is still level-3, its parent pointing to the replied level-3 comment.                                                                                                                               |
| **AC-10** | Comment like/unlike      | Like a comment → count +1. The same user clicks again → unliked, count -1.                                                                                                                                                                                                                                                     |
| **AC-11** | Aggregated like notify   | 3 different users like the same comment of user B. User B opens the Message Center → "Received Likes" shows **1** notification: "UserA, UserB, UserC and 3 others liked your comment" (first 3 shown), with a 15-char preview of the comment.                                                                                  |
| **AC-12** | Message Center structure | Message Center is a standalone page at `/messages`. The left nav contains exactly five items: **Replies, @Mentions, Received Likes, System Notices, My Messages**. Each shows an unread count badge.                                                                                                                            |
| **AC-13** | Cover management         | Upload with a custom cover → after transcode, card and player show the custom cover. Upload without cover → system auto-generates the 1-second frame as default. Replace via the cover endpoint after publishing → new cover shown.                                                                                             |
| **AC-14** | Cover format validation  | Upload with an invalid cover (format not JPEG/PNG/GIF/BMP/WEBP or > 10MB) → clear error, entire upload rejected.                                                                                                                                                                                                               |
| **AC-15** | Comment delete perms     | UP主 can delete any comment under their videos. Regular users can delete their own but not others'.                                                                                                                                                                                                                             |
| **AC-16** | Comment delete cascade   | Deleting a comment with children → all children deleted in sync, no longer shown.                                                                                                                                                                                                                                               |

---

### 4. Compatibility Requirements

| ID       | Type     | Spec                                                                                                                                                                                                                                                                      |
| :------- | :------- | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **BC-1** | Backward | First formal version, no legacy burden. Once interfaces and models are defined, future versions MUST NOT make breaking changes to existing fields without review.                                                                                                           |
| **BC-2** | Forward  | The code architecture MUST support a smooth future split into Kratos microservices. The v1.0 monolith MUST NOT introduce coupling that blocks later decomposition. The video table MUST reserve `status` values `pending_review` and `rejected` for future moderation.      |

---

### 5. Non-Functional Requirements

| ID   | Type           | Spec                                                                                                                                                                                                                                                                 |
| :--- | :------------- | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| NF-1 | Concurrency    | The danmaku system MUST support 100 simultaneous users in the same video room sending danmaku, without crashes or deadlocks.                                                                                                                                        |
| NF-2 | Latency        | Danmaku end-to-end delay MUST be ≤ 200ms in the local dev environment.                                                                                                                                                                                               |
| NF-3 | Storage        | MySQL: users, video metadata, video status, comments, likes, notifications. Redis: danmaku real-time relay, video play-count hot data.                                                                                                                              |
| NF-4 | Message Queue  | Video transcode tasks MUST be delivered/consumed via a message queue; synchronous FFmpeg in upload requests is FORBIDDEN.                                                                                                                                            |
| NF-5 | Frontend       | MUST be a pure SPA built with Vue 3 + Vite; no SSR.                                                                                                                                                                                                                  |
| NF-6 | Backend        | MUST be Go. All code MUST follow the Go standard project layout. The architecture MUST support a future smooth split into Kratos microservices, though v1.0 does not mandate the split.                                                                              |
| NF-7 | File Storage   | All files in a **single** Aliyun OSS Bucket `mini-bili`, separated by prefix: - transcoded videos: `videos/{video_id}.mp4` - covers (default & custom): `covers/{video_id}.{ext}` - **avatars**: `avatars/{user_id}.{ext}` Raw videos and temp covers exist locally ONLY during upload/transcode, cleaned immediately after, never uploaded to OSS. |
| NF-8 | API Style      | All external HTTP endpoints follow RESTful API design conventions.                                                                                                                                                                                                   |
| NF-9 | Message Center | The Message Center MUST be a standalone first-level page (route `/messages`) with a left nav containing exactly five items: Replies, @Mentions, Received Likes, System Notices, My Messages. Clicking a category filters its notification list. The nav shows unread badges per category; the bell icon in the top nav provides the entry. |
