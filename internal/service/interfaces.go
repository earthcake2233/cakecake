package service

import (
	"context"
	"time"
)

// ─── Domain interfaces for Phase 1 microservice preparation ───

// UserProvider is the user domain boundary.
// In Phase 1 (monolith), implemented by *gorm.DB directly.
// In Phase 2+, replaced by gRPC client.
type UserProvider interface {
	GetUser(ctx context.Context, id uint64) (UserInfo, error)
	GetUsersByIDs(ctx context.Context, ids []uint64) (map[uint64]UserInfo, error)
	BatchCurrentLevels(ctx context.Context, ids []uint64) (map[uint64]int, error)
}

// UserInfo is the cross-domain user data.
type UserInfo struct {
	ID        uint64
	Username  string
	Nickname  string
	AvatarURL string
	Level     int
}

// VideoProvider is the video domain boundary.
type VideoProvider interface {
	GetPublishedVideo(ctx context.Context, id uint64) (*VideoInfo, error)
	GetVideoAuthor(ctx context.Context, id uint64) (uint64, error)
}

// VideoInfo is the cross-domain video data.
type VideoInfo struct {
	ID              uint64
	UserID          uint64
	Title           string
	Status          string
	CommentsClosed  bool
	CommentsCurated bool
	DanmakuClosed   bool
	CreatedAt       time.Time
}

// ArticleProvider is the article domain boundary.
type ArticleProvider interface {
	GetPublishedArticle(ctx context.Context, id uint64) (*ArticleInfo, error)
	GetArticleAuthor(ctx context.Context, id uint64) (uint64, error)
}

// ArticleInfo is the cross-domain article data.
type ArticleInfo struct {
	ID              uint64
	UserID          uint64
	Title           string
	Status          string
	CommentsClosed  bool
	CommentsCurated bool
	CreatedAt       time.Time
}

// DynamicProvider is the dynamic domain boundary.
type DynamicProvider interface {
	GetPublishedDynamic(ctx context.Context, id uint64) (*DynamicInfo, error)
	GetDynamicAuthor(ctx context.Context, id uint64) (uint64, error)
}

// DynamicInfo is the cross-domain dynamic data.
type DynamicInfo struct {
	ID              uint64
	UserID          uint64
	Status          string
	CommentsClosed  bool
	CommentsCurated bool
	CreatedAt       time.Time
}