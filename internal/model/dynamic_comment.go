package model

import "time"

// DynamicComment is a comment on a user image/text dynamic.
type DynamicComment struct {
	ID         uint64 `gorm:"primaryKey"`
	DynamicID  uint64 `gorm:"index:idx_dyn_cmt_dynamic;not null"`
	UserID     uint64 `gorm:"index;not null"`
	ParentID   uint64 `gorm:"index;not null;default:0"`
	Level      int    `gorm:"not null;default:1"`
	Content    string `gorm:"size:1000;not null"`
	LikeCount  uint64 `gorm:"default:0"`
	Pinned     bool   `gorm:"index;default:0"`
	// Approved：评论精选模式下，false 表示待作者精选；非精选模式创建时设为 true。
	Approved       bool `gorm:"not null;default:0;index"`
	CuratedIgnored bool `gorm:"not null;default:0;index"`
	IpLocation string `gorm:"size:32;not null;default:''"`
	CreatedAt  time.Time
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

// UserDynamicLike records a user's like on a feed dynamic.
type UserDynamicLike struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64 `gorm:"uniqueIndex:idx_dyn_like_user_dyn;not null"`
	DynamicID uint64 `gorm:"uniqueIndex:idx_dyn_like_user_dyn;not null"`
	CreatedAt time.Time
}
