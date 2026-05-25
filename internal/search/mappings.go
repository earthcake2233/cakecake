package search

const videoIndexMapping = `{
  "mappings": {
    "properties": {
      "id": { "type": "long" },
      "user_id": { "type": "long" },
      "title": { "type": "text" },
      "description": { "type": "text" },
      "tags": { "type": "text" },
      "uploader": { "type": "text", "fields": { "keyword": { "type": "keyword" } } },
      "cover_url": { "type": "keyword", "index": false },
      "play_count": { "type": "long" },
      "danmaku_count": { "type": "long" },
      "duration_sec": { "type": "double" },
      "fav_count": { "type": "long" },
      "zone": { "type": "keyword" },
      "zone_parent": { "type": "keyword" },
      "created_at": { "type": "date" },
      "status": { "type": "keyword" }
    }
  }
}`

const articleIndexMapping = `{
  "mappings": {
    "properties": {
      "id": { "type": "long" },
      "user_id": { "type": "long" },
      "title": { "type": "text" },
      "body": { "type": "text" },
      "author": { "type": "text", "fields": { "keyword": { "type": "keyword" } } },
      "cover_url": { "type": "keyword", "index": false },
      "view_count": { "type": "long" },
      "comment_count": { "type": "long" },
      "fav_count": { "type": "long" },
      "author_avatar": { "type": "keyword", "index": false },
      "tags": { "type": "text" },
      "category": { "type": "keyword" },
      "excerpt": { "type": "text" },
      "published_at": { "type": "date" },
      "status": { "type": "keyword" }
    }
  }
}`

const userIndexMapping = `{
  "mappings": {
    "properties": {
      "id": { "type": "long" },
      "nickname": { "type": "text" },
      "username": { "type": "text" },
      "cake_id": { "type": "keyword" },
      "sign": { "type": "text" },
      "avatar_url": { "type": "keyword", "index": false },
      "status": { "type": "keyword" }
    }
  }
}`
