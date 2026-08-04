package service

import (
	"cakecake/internal/model/user"
	"context"
	"time"

	"gorm.io/gorm"
)

// Domain interfaces for Phase 1 microservice preparation

// UserProvider is the user domain boundary.
// In Phase 1 (monolith), implemented by *gorm.DB directly.
// In Phase 2+, replaced by gRPC client.
type UserProvider interface {
	// DecrementCoins subtracts coins from user balance.
	DecrementCoins(ctx context.Context, userID uint64, amount int) error
	GetUser(ctx context.Context, id uint64) (UserInfo, error)
	GetUsersByIDs(ctx context.Context, ids []uint64) (map[uint64]UserInfo, error)
	BatchCurrentLevels(ctx context.Context, ids []uint64) (map[uint64]int, error)

	// User-domain storage boundary (Phase 1: *gorm.DB impl; Phase 2+: gRPC client).
	// GetUserByID returns the raw user model for a single id.
	GetUserByID(ctx context.Context, id uint64) (*user.User, error)
	// GetUserByUsername looks up a user by exact username.
	GetUserByUsername(ctx context.Context, name string) (*user.User, error)
	// BatchGetUsersByIDs returns a map of id -> raw user model.
	BatchGetUsersByIDs(ctx context.Context, ids []uint64) (map[uint64]*user.User, error)
	// UsernameTaken reports whether a username is used by another user.
	UsernameTaken(ctx context.Context, name string, excludeID uint64) bool
	// UpdateUsername sets a user's username.
	UpdateUsername(ctx context.Context, id uint64, name string) error
	// UpdateUserFields applies profile field updates.
	UpdateUserFields(ctx context.Context, id uint64, fields map[string]interface{}) error
	// UpdatePasswordHash sets a user's password hash.
	UpdatePasswordHash(ctx context.Context, id uint64, hash string) error
	// UpdateAnnouncement sets the space announcement.
	UpdateAnnouncement(ctx context.Context, id uint64, announcement string) error
	// UpdateAvatar sets the avatar object key.
	UpdateAvatar(ctx context.Context, id uint64, objectKey string) error
	// SetDeletion schedules or revokes account deletion (nil times revoke).
	SetDeletion(ctx context.Context, id uint64, requestedAt, effectiveAt *time.Time) error
	// GetPasswordHash returns the stored password hash.
	GetPasswordHash(ctx context.Context, id uint64) (string, error)
	// EnsureCakeID assigns a cake id if missing.
	EnsureCakeID(ctx context.Context, u *user.User) error
	// ListCoinLedger returns paginated coin ledger rows.
	ListCoinLedger(ctx context.Context, userID uint64, since time.Time, limit, offset int) (total int64, rows []user.CoinLedger, err error)
	// CreateUserWithCakeID creates a user and assigns a cake id atomically.
	CreateUserWithCakeID(ctx context.Context, u *user.User) error
	// MarkLogin records a daily login for a user.
	MarkLogin(ctx context.Context, userID uint64) error
	// WithTx runs fn inside a database transaction (Phase 1 monolith seam).
	WithTx(ctx context.Context, fn func(tx *gorm.DB) error) error
}

// UserInfo is the cross-domain user data.
type UserInfo struct {
	CoinBalanceTenths int64
	ID                uint64
	Username          string
	Nickname          string
	AvatarURL         string
	Level             int
	AnonymizedAt      *time.Time
}

// ArticleProvider is the article domain boundary.
type ArticleProvider interface {
	GetPublishedArticle(ctx context.Context, id uint64) (*ArticleInfo, error)
	GetArticleAuthor(ctx context.Context, id uint64) (uint64, error)
	IncrCommentCount(ctx context.Context, id uint64, delta int) error
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
	IncrCommentCount(ctx context.Context, id uint64, delta int) error
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
