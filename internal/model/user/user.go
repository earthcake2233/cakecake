package user

import (
	"fmt"
	"time"
)

type User struct {
	ID           uint64 `gorm:"primaryKey"`
	Username     string `gorm:"size:64;uniqueIndex;not null"`
	PasswordHash string `gorm:"size:128;not null"`
	AvatarURL    string `gorm:"size:1024"`
	// CakeID is the public immutable id shown as "username" in the personal center (cake_XXXXXXXXXXX).
	CakeID   string `gorm:"size:36;index"` // immutable public id; FormatCakeID(id), filled after insert
	Nickname string `gorm:"size:64"`
	Sign     string `gorm:"size:500"`
	// SpaceAnnouncement is the personal-space sidebar bulletin (≤150 UTF-8 runes, validated in handler).
	SpaceAnnouncement string `gorm:"size:600"`
	Gender            string `gorm:"size:16"` // male | female | secret
	Birthday          string `gorm:"size:10"` // YYYY-MM-DD, may be empty
	// Space privacy toggles (personal space settings).
	PrivacyPublicFavorites   bool      `gorm:"not null;default:0"`
	PrivacyPublicRecentCoins bool      `gorm:"not null;default:0"`
	PrivacyPublicFollowing   bool      `gorm:"not null;default:0"`
	PrivacyPublicFans        bool      `gorm:"not null;default:0"`
	PrivacyPublicBirthday    bool      `gorm:"not null;default:1"`
	CreatedAt                time.Time `gorm:"index"`
	UpdatedAt                time.Time
	// FirstPublishedAt is set once when the user's first video reaches published (transcode OK);
	// retained if that video is later deleted so the "became a creator" day count still has an anchor.
	FirstPublishedAt *time.Time `gorm:"index"`
	// DeletionRequestedAt is set when the user submits account cancellation (cooling-off period starts).
	DeletionRequestedAt *time.Time `gorm:"index"`
	// DeletionEffectiveAt is when the account becomes permanently anonymized (7–30 days after request).
	DeletionEffectiveAt *time.Time `gorm:"index"`
	// AnonymizedAt is set after finalization; public comments/danmaku still reference this user_id.
	AnonymizedAt *time.Time `gorm:"index"`
	// Experience is total user EXP for account level (Lv1–Lv6 thresholds in userlevel package).
	Experience uint64 `gorm:"not null;default:0"`
	// CoinBalanceTenths is the user's coin balance in 0.1-coin units (230 = 23.0 coins).
	CoinBalanceTenths int64 `gorm:"not null;default:230"`
	// ViewHistoryPaused stops recording new watch-history entries when true.
	ViewHistoryPaused bool `gorm:"not null;default:0"`
}
type UserFollow struct {
	ID         uint64 `gorm:"primaryKey"`
	FollowerID uint64 `gorm:"uniqueIndex:idx_user_follow_pair,priority:1;index:idx_user_follow_follower;not null"`
	FolloweeID uint64 `gorm:"uniqueIndex:idx_user_follow_pair,priority:2;index:idx_user_follow_followee;not null"`
	CreatedAt  time.Time
}
type UserBlock struct {
	ID        uint64 `gorm:"primaryKey"`
	BlockerID uint64 `gorm:"uniqueIndex:idx_user_block_pair,priority:1;index;not null"`
	BlockedID uint64 `gorm:"uniqueIndex:idx_user_block_pair,priority:2;index;not null"`
	CreatedAt time.Time
}
type CoinLedger struct {
	ID          uint64    `gorm:"primaryKey"`
	UserID      uint64    `gorm:"index:idx_coin_ledger_user_created,priority:1;not null"`
	DeltaTenths int64     `gorm:"not null"`
	ReasonType  string    `gorm:"size:32;not null;index"`
	VideoID     uint64    `gorm:"index;not null;default:0"`
	CreatedAt   time.Time `gorm:"index:idx_coin_ledger_user_created,priority:2"`
}
type UserFollowGroup struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64 `gorm:"index:idx_follow_group_user;not null"`
	Name      string `gorm:"size:20;not null"`
	CreatedAt time.Time
}
type UserFollowGroupMember struct {
	ID         uint64 `gorm:"primaryKey"`
	GroupID    uint64 `gorm:"uniqueIndex:idx_follow_group_member,priority:1;index;not null"`
	FolloweeID uint64 `gorm:"uniqueIndex:idx_follow_group_member,priority:2;index;not null"`
	CreatedAt  time.Time
}

func IsUserAnonymized(u *User) bool {
	return u != nil && u.AnonymizedAt != nil
}
func DisplayUsername(u *User) string {
	if u == nil {
		return ""
	}
	if IsUserAnonymized(u) {
		return "\u5df2\u6ce8\u9500\u7528\u6237"
	}
	return u.Username
}

func FormatCakeID(id uint64) string {
	return fmt.Sprintf("cake_%011d", id)
}
