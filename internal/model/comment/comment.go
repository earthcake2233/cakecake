package comment
import "time"
type Comment struct {
	ID        uint64 `gorm:"primaryKey"`
	VideoID   uint64 `gorm:"index:idx_comment_video;not null"`
	UserID    uint64 `gorm:"index;not null"`
	ParentID  uint64 `gorm:"index;default:0"`
	Level     int    `gorm:"not null"`
	Content   string `gorm:"size:2000;not null"`
	LikeCount  uint64 `gorm:"default:0"`
	Pinned     bool   `gorm:"index;default:0"`
	// Approved：评论精选模式下，false 表示待 UP 精选；非精选模式创建时设为 true。
	Approved   bool `gorm:"not null;default:0;index"`
	// CuratedIgnored：精选模式下 UP 忽略（不公开），仍保持 approved=false。
	CuratedIgnored bool `gorm:"not null;default:0;index"`
	IpLocation string `gorm:"size:32;not null;default:''"`
	CreatedAt  time.Time
}
type CommentLike struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64 `gorm:"uniqueIndex:idx_like_user_comment;not null"`
	CommentID uint64 `gorm:"uniqueIndex:idx_like_user_comment;not null"`
	CreatedAt time.Time
}
type CommentDislike struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64 `gorm:"uniqueIndex:idx_dislike_user_comment;not null"`
	CommentID uint64 `gorm:"uniqueIndex:idx_dislike_user_comment;not null"`
	CreatedAt time.Time
}
type ArticleComment struct {
	ID         uint64 `gorm:"primaryKey"`
	ArticleID  uint64 `gorm:"index:idx_article_comment_article;not null"`
	UserID     uint64 `gorm:"index;not null"`
	ParentID   uint64 `gorm:"index;default:0"`
	Level      int    `gorm:"not null"`
	Content    string `gorm:"size:2000;not null"`
	LikeCount  uint64 `gorm:"default:0"`
	Pinned     bool   `gorm:"index;default:0"`
	// Approved：评论精选模式下，false 表示待作者精选；非精选模式创建时设为 true。
	Approved       bool `gorm:"not null;default:0;index"`
	CuratedIgnored bool `gorm:"not null;default:0;index"`
	IpLocation string `gorm:"size:32;not null;default:''"`
	CreatedAt  time.Time
}
type ArticleCommentLike struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64 `gorm:"uniqueIndex:idx_article_cmt_like_user_cmt;not null"`
	CommentID uint64 `gorm:"uniqueIndex:idx_article_cmt_like_user_cmt;not null"`
	CreatedAt time.Time
}
type ArticleCommentDislike struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64 `gorm:"uniqueIndex:idx_article_cmt_dislike_user_cmt;not null"`
	CommentID uint64 `gorm:"uniqueIndex:idx_article_cmt_dislike_user_cmt;not null"`
	CreatedAt time.Time
}
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
type DynamicCommentLike struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64 `gorm:"uniqueIndex:idx_dyn_cmt_like_user_cmt;not null"`
	CommentID uint64 `gorm:"uniqueIndex:idx_dyn_cmt_like_user_cmt;not null"`
	CreatedAt time.Time
}
type DynamicCommentDislike struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64 `gorm:"uniqueIndex:idx_dyn_cmt_dislike_user_cmt;not null"`
	CommentID uint64 `gorm:"uniqueIndex:idx_dyn_cmt_dislike_user_cmt;not null"`
	CreatedAt time.Time
}
type UserDynamicLike struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64 `gorm:"uniqueIndex:idx_dyn_like_user_dyn;not null"`
	DynamicID uint64 `gorm:"uniqueIndex:idx_dyn_like_user_dyn;not null"`
	CreatedAt time.Time
}
