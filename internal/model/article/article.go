package article

import "time"

const (
	StatusPublished     = "published"
	StatusDraft         = "draft"
	StatusProcessing    = "processing"
	StatusPendingReview = "pending_review"
	StatusPassed        = "passed"
	StatusFailed        = "failed"
	StatusRejected      = "rejected"
)

type Article struct {
	ID           uint64 `gorm:"primaryKey"`
	UserID       uint64 `gorm:"index:idx_article_user;not null"`
	Title        string `gorm:"size:80;not null"`
	CoverURL     string `gorm:"size:1024"`
	BodyMD       string `gorm:"type:longtext;not null"`
	Status       string `gorm:"size:32;index:idx_article_status;not null;default:draft"`
	TagsJSON     string `gorm:"type:text"`
	ViewCount    uint64 `gorm:"default:0"`
	CommentCount uint64 `gorm:"default:0"`
	// CommentsClosed: when the author closes comments, new comments are blocked; list returns empty for visitors.
	CommentsClosed bool `gorm:"not null;default:0"`
	// CommentsCurated: when curation is enabled, new comments require author approval before being visible to all.
	CommentsCurated   bool   `gorm:"not null;default:0"`
	CoinCount         uint64 `gorm:"default:0"`
	FavCount          uint64 `gorm:"default:0"`
	ForwardCount      uint64 `gorm:"default:0"`
	FailReason        string `gorm:"size:2000"`
	PublishedAt       *time.Time
	ReviewedAt        *time.Time
	ReviewedByAdminID *uint64   `gorm:"index"`
	CreatedAt         time.Time `gorm:"index:idx_article_created"`
	UpdatedAt         time.Time
}
type ArticleFavorite struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64 `gorm:"uniqueIndex:idx_article_fav_user_article;not null"`
	ArticleID uint64 `gorm:"uniqueIndex:idx_article_fav_user_article;not null"`
	CreatedAt time.Time
}
type ArticleCoin struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64 `gorm:"uniqueIndex:idx_article_coin_user_article;not null"`
	ArticleID uint64 `gorm:"uniqueIndex:idx_article_coin_user_article;not null"`
	Amount    int    `gorm:"not null;default:1"`
	CreatedAt time.Time
}
