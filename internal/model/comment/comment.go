package comment

import "time"

// Comment is a video comment row.
type Comment struct {
	ID        uint64 `gorm:"primaryKey"`
	VideoID   uint64 `gorm:"index:idx_comment_video;not null"`
	UserID    uint64 `gorm:"index;not null"`
	ParentID  uint64 `gorm:"index;default:0"`
	Level     int    `gorm:"not null"`
	Content   string `gorm:"size:2000;not null"`
	LikeCount uint64 `gorm:"default:0"`
	Pinned    bool   `gorm:"index;default:0"`
	// Approved: in curated mode, false means pending curation; in non-curated mode, set to true on creation.
	Approved bool `gorm:"not null;default:0;index"`
	// CuratedIgnored: in curated mode, means the uploader ignored it (not public); approved stays false.
	CuratedIgnored bool   `gorm:"not null;default:0;index"`
	IpLocation     string `gorm:"size:32;not null;default:''"`
	CreatedAt      time.Time
}

// CommentLike records a user's like on a video comment.
type CommentLike struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64 `gorm:"uniqueIndex:idx_like_user_comment;not null"`
	CommentID uint64 `gorm:"uniqueIndex:idx_like_user_comment;not null"`
	CreatedAt time.Time
}

// CommentDislike records a user's dislike on a video comment.
type CommentDislike struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64 `gorm:"uniqueIndex:idx_dislike_user_comment;not null"`
	CommentID uint64 `gorm:"uniqueIndex:idx_dislike_user_comment;not null"`
	CreatedAt time.Time
}

// ArticleComment is an article comment row.
type ArticleComment struct {
	ID        uint64 `gorm:"primaryKey"`
	ArticleID uint64 `gorm:"index:idx_article_comment_article;not null"`
	UserID    uint64 `gorm:"index;not null"`
	ParentID  uint64 `gorm:"index;default:0"`
	Level     int    `gorm:"not null"`
	Content   string `gorm:"size:2000;not null"`
	LikeCount uint64 `gorm:"default:0"`
	Pinned    bool   `gorm:"index;default:0"`
	// Approved: in curated mode, false means pending author curation; in non-curated mode, set to true on creation.
	Approved       bool   `gorm:"not null;default:0;index"`
	CuratedIgnored bool   `gorm:"not null;default:0;index"`
	IpLocation     string `gorm:"size:32;not null;default:''"`
	CreatedAt      time.Time
}

// ArticleCommentLike records a user's like on an article comment.
type ArticleCommentLike struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64 `gorm:"uniqueIndex:idx_article_cmt_like_user_cmt;not null"`
	CommentID uint64 `gorm:"uniqueIndex:idx_article_cmt_like_user_cmt;not null"`
	CreatedAt time.Time
}

// ArticleCommentDislike records a user's dislike on an article comment.
type ArticleCommentDislike struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64 `gorm:"uniqueIndex:idx_article_cmt_dislike_user_cmt;not null"`
	CommentID uint64 `gorm:"uniqueIndex:idx_article_cmt_dislike_user_cmt;not null"`
	CreatedAt time.Time
}

// DynamicComment is a dynamic comment row.
type DynamicComment struct {
	ID        uint64 `gorm:"primaryKey"`
	DynamicID uint64 `gorm:"index:idx_dyn_cmt_dynamic;not null"`
	UserID    uint64 `gorm:"index;not null"`
	ParentID  uint64 `gorm:"index;not null;default:0"`
	Level     int    `gorm:"not null;default:1"`
	Content   string `gorm:"size:1000;not null"`
	LikeCount uint64 `gorm:"default:0"`
	Pinned    bool   `gorm:"index;default:0"`
	// Approved: in curated mode, false means pending author curation; in non-curated mode, set to true on creation.
	Approved       bool   `gorm:"not null;default:0;index"`
	CuratedIgnored bool   `gorm:"not null;default:0;index"`
	IpLocation     string `gorm:"size:32;not null;default:''"`
	CreatedAt      time.Time
}

// DynamicCommentLike records a user's like on a dynamic comment.
type DynamicCommentLike struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64 `gorm:"uniqueIndex:idx_dyn_cmt_like_user_cmt;not null"`
	CommentID uint64 `gorm:"uniqueIndex:idx_dyn_cmt_like_user_cmt;not null"`
	CreatedAt time.Time
}

// DynamicCommentDislike records a user's dislike on a dynamic comment.
type DynamicCommentDislike struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64 `gorm:"uniqueIndex:idx_dyn_cmt_dislike_user_cmt;not null"`
	CommentID uint64 `gorm:"uniqueIndex:idx_dyn_cmt_dislike_user_cmt;not null"`
	CreatedAt time.Time
}

// UserDynamicLike records a user's like on a dynamic.
type UserDynamicLike struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64 `gorm:"uniqueIndex:idx_dyn_like_user_dyn;not null"`
	DynamicID uint64 `gorm:"uniqueIndex:idx_dyn_like_user_dyn;not null"`
	CreatedAt time.Time
}
