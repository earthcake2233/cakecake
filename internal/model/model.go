package model

import "time"

// User is a registered account.
type User struct {
	ID           uint64 `gorm:"primaryKey"`
	Username     string `gorm:"size:64;uniqueIndex;not null"`
	PasswordHash string `gorm:"size:128;not null"`
	AvatarURL    string `gorm:"size:1024"`
	// CakeID is the public immutable id shown as「用户名」in the personal center (cake_XXXXXXXXXXX).
	CakeID   string `gorm:"size:36;index"` // immutable public id; FormatCakeID(id), filled after insert
	Nickname string `gorm:"size:64"`
	Sign     string `gorm:"size:500"`
	// SpaceAnnouncement is the personal-space sidebar bulletin (≤150 UTF-8 runes, validated in handler).
	SpaceAnnouncement string    `gorm:"size:600"`
	Gender            string    `gorm:"size:16"` // male | female | secret
	Birthday          string `gorm:"size:10"` // YYYY-MM-DD, may be empty
	// Space privacy toggles (personal space settings).
	PrivacyPublicFavorites    bool `gorm:"not null;default:0"`
	PrivacyPublicRecentCoins  bool `gorm:"not null;default:0"`
	PrivacyPublicFollowing    bool `gorm:"not null;default:0"`
	PrivacyPublicFans         bool `gorm:"not null;default:0"`
	PrivacyPublicBirthday     bool `gorm:"not null;default:1"`
	CreatedAt                 time.Time `gorm:"index"`
	UpdatedAt         time.Time
	// FirstPublishedAt is set once when the user's first video reaches published (transcode OK);
	// retained if that video is later deleted so「成为 UP 主」day count still has an anchor.
	FirstPublishedAt *time.Time `gorm:"index"`
	// DeletionRequestedAt is set when the user submits account cancellation (冷静期开始).
	DeletionRequestedAt *time.Time `gorm:"index"`
	// DeletionEffectiveAt is when the account becomes permanently anonymized (7–30 days after request).
	DeletionEffectiveAt *time.Time `gorm:"index"`
	// AnonymizedAt is set after finalization; public comments/danmaku still reference this user_id.
	AnonymizedAt *time.Time `gorm:"index"`
	// Experience is total user EXP for account level (Lv1–Lv6 thresholds in userlevel package).
	Experience uint64 `gorm:"not null;default:0"`
	// CoinBalanceTenths is the user's 硬币 balance in 0.1-coin units (230 = 23.0 coins).
	CoinBalanceTenths int64 `gorm:"not null;default:230"`
	// ViewHistoryPaused stops recording new watch-history entries when true.
	ViewHistoryPaused bool `gorm:"not null;default:0"`
}

// Video stores metadata and OSS URLs after transcoding.
type Video struct {
	ID           uint64  `gorm:"primaryKey"`
	UserID       uint64  `gorm:"index:idx_video_user;not null"`
	Title        string  `gorm:"size:80;not null"`
	Description  string  `gorm:"size:2000"`
	DurationSec  float64 `gorm:"column:duration_sec"`
	Status       string  `gorm:"size:32;index:idx_video_status"`
	FailReason   string  `gorm:"size:2000"`
	VideoURL     string  `gorm:"size:1024"`
	CoverURL     string  `gorm:"size:1024"`
	PlayCount    uint64  `gorm:"default:0;index:idx_video_play"`
	DanmakuCount uint64  `gorm:"default:0"`
	CommentCount uint64  `gorm:"default:0"`
	LikeCount    uint64  `gorm:"default:0"`
	FavCount     uint64  `gorm:"default:0"`
	CoinCount    uint64  `gorm:"default:0"`
	// CommentsClosed：UP 关闭评论区后禁止新发评论；列表对访客返回空。
	CommentsClosed bool `gorm:"not null;default:0"`
	// CommentsCurated：开启评论精选后，新评论需 UP 确认才对所有人可见。
	CommentsCurated bool `gorm:"not null;default:0"`
	// DanmakuClosed：UP 关闭弹幕后禁止新发弹幕。
	DanmakuClosed bool `gorm:"not null;default:0"`
	// TagsJSON is a JSON array of strings, e.g. ["录屏","教程"]；空串表示无标签。
	TagsJSON string `gorm:"type:text"`
	// Zone is the publish partition, e.g. "动画" or "生活-日常".
	Zone string `gorm:"size:64"`
	// DraftRawPath / DraftCoverPath：status=draft 时本地暂存路径，投稿转码前使用。
	DraftRawPath   string    `gorm:"size:1024"`
	DraftCoverPath string    `gorm:"size:1024"`
	ReviewedAt     *time.Time
	ReviewedByAdminID *uint64 `gorm:"index"`
	CreatedAt      time.Time `gorm:"index:idx_video_created"`
	UpdatedAt      time.Time
}

// Danmaku is a persisted bullet comment.
type Danmaku struct {
	ID        uint64  `gorm:"primaryKey"`
	VideoID   uint64  `gorm:"index:idx_danmaku_video;not null"`
	UserID    uint64  `gorm:"index;not null"`
	Content   string  `gorm:"size:400;not null"`
	Color     string  `gorm:"size:16;not null"`
	Type      string  `gorm:"size:16;not null"`
	// FontSize: sm | md | lg（弹幕字号，默认 md）
	FontSize  string  `gorm:"size:8;not null;default:md"`
	VideoTime float64 `gorm:"column:video_time;not null"`
	LikeCount uint64  `gorm:"default:0"`
	CreatedAt time.Time
}

func (Danmaku) TableName() string { return "danmakus" }

// DanmakuLike records a user's like on a danmaku.
type DanmakuLike struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64 `gorm:"uniqueIndex:idx_danmaku_like_user_dm;not null"`
	DanmakuID uint64 `gorm:"uniqueIndex:idx_danmaku_like_user_dm;not null"`
}

func (DanmakuLike) TableName() string { return "danmaku_likes" }

// Comment is a threaded comment under a video (max depth 3 per SPEC).
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

// CommentLike records a user's like on a comment.
type CommentLike struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64 `gorm:"uniqueIndex:idx_like_user_comment;not null"`
	CommentID uint64 `gorm:"uniqueIndex:idx_like_user_comment;not null"`
	CreatedAt time.Time
}

// CommentDislike records a user's dislike on a comment (no public count).
type CommentDislike struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64 `gorm:"uniqueIndex:idx_dislike_user_comment;not null"`
	CommentID uint64 `gorm:"uniqueIndex:idx_dislike_user_comment;not null"`
	CreatedAt time.Time
}

// VideoLike records a user's like on a published video (e.g. 动态点赞).
type VideoLike struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64 `gorm:"uniqueIndex:idx_video_like_user_video;not null"`
	VideoID   uint64 `gorm:"uniqueIndex:idx_video_like_user_video;not null"`
	CreatedAt time.Time
}

// FavoriteFolder groups a user's favorited videos (收藏夹).
type FavoriteFolder struct {
	ID          uint64 `gorm:"primaryKey"`
	UserID      uint64 `gorm:"index:idx_fav_folder_user;not null"`
	Title       string `gorm:"size:20;not null"`
	Description string `gorm:"size:200;not null;default:''"`
	CoverURL    string `gorm:"size:1024;not null;default:''"`
	IsPublic    bool   `gorm:"not null;default:1"`
	IsDefault   bool   `gorm:"not null;default:0"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// VideoFavorite records a user's favorite (收藏) in one folder (same video may appear in multiple folders).
type VideoFavorite struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64 `gorm:"uniqueIndex:idx_video_fav_user_video_folder,priority:1;not null"`
	VideoID   uint64 `gorm:"uniqueIndex:idx_video_fav_user_video_folder,priority:2;not null"`
	FolderID  uint64 `gorm:"uniqueIndex:idx_video_fav_user_video_folder,priority:3;index:idx_video_fav_folder;not null;default:0"`
	CreatedAt time.Time
}

// VideoCoin records a user's coin (投币) on a published video (one per user per video).
type VideoCoin struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64 `gorm:"uniqueIndex:idx_video_coin_user_video;not null"`
	VideoID   uint64 `gorm:"uniqueIndex:idx_video_coin_user_video;not null"`
	Amount    int    `gorm:"not null;default:1"` // 1 or 2
	CreatedAt time.Time
}

// WatchLater is the user's watch-later (稍后再看) queue entry.
type WatchLater struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64 `gorm:"uniqueIndex:idx_watch_later_user_video;not null"`
	VideoID   uint64 `gorm:"uniqueIndex:idx_watch_later_user_video;not null"`
	Watched   bool   `gorm:"not null;default:0"`
	CreatedAt time.Time `gorm:"index"`
}

// UserFollow records follower -> followee (关注关系).
type UserFollow struct {
	ID         uint64 `gorm:"primaryKey"`
	FollowerID uint64 `gorm:"uniqueIndex:idx_user_follow_pair,priority:1;index:idx_user_follow_follower;not null"`
	FolloweeID uint64 `gorm:"uniqueIndex:idx_user_follow_pair,priority:2;index:idx_user_follow_followee;not null"`
	CreatedAt  time.Time
}

// UserBlock records blocker -> blocked (黑名单：被拉黑用户不得与拉黑者互动).
type UserBlock struct {
	ID        uint64 `gorm:"primaryKey"`
	BlockerID uint64 `gorm:"uniqueIndex:idx_user_block_pair,priority:1;index;not null"`
	BlockedID uint64 `gorm:"uniqueIndex:idx_user_block_pair,priority:2;index;not null"`
	CreatedAt time.Time
}

func (UserBlock) TableName() string { return "user_blocks" }

// UserFollowGroup is a custom following list group (关注分组).
type UserFollowGroup struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64 `gorm:"index:idx_follow_group_user;not null"`
	Name      string `gorm:"size:20;not null"`
	CreatedAt time.Time
}

// UserFollowGroupMember links a followee into a custom group.
type UserFollowGroupMember struct {
	ID         uint64 `gorm:"primaryKey"`
	GroupID    uint64 `gorm:"uniqueIndex:idx_follow_group_member,priority:1;index;not null"`
	FolloweeID uint64 `gorm:"uniqueIndex:idx_follow_group_member,priority:2;index;not null"`
	CreatedAt  time.Time
}

// Notification is an inbox item (like aggregation, etc.).
type Notification struct {
	ID              uint64 `gorm:"primaryKey"`
	RecipientID     uint64 `gorm:"index:idx_notif_recipient;not null"`
	Type            string `gorm:"size:48;index;not null"`
	RelatedID       uint64 `gorm:"index"`
	SenderNamesJSON string `gorm:"type:text"`
	TotalLikes      int    `gorm:"default:0"`
	CommentPreview  string `gorm:"size:32"`
	// PayloadJSON holds type-specific fields (e.g. reply_received: sender, reply body, video_id).
	PayloadJSON string `gorm:"type:text"`
	IsRead      bool   `gorm:"index"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// LikeNotifMute marks「不再通知」for like aggregation on a specific comment (recipient + comment_id).
type LikeNotifMute struct {
	RecipientID uint64 `gorm:"uniqueIndex:idx_like_notif_mute_pair;not null"`
	CommentID   uint64 `gorm:"uniqueIndex:idx_like_notif_mute_pair;not null"`
	CreatedAt   time.Time
}

func (LikeNotifMute) TableName() string { return "like_notif_mutes" }

// IsUserAnonymized returns true once the account has been finalized as deactivated.
func IsUserAnonymized(u *User) bool {
	return u != nil && u.AnonymizedAt != nil
}

// DisplayUsername returns the public display name; anonymized accounts show 已注销用户.
func DisplayUsername(u *User) string {
	if u == nil {
		return ""
	}
	if IsUserAnonymized(u) {
		return "已注销用户"
	}
	return u.Username
}

// Article is a published column (专栏) with Markdown body.
type Article struct {
	ID            uint64 `gorm:"primaryKey"`
	UserID        uint64 `gorm:"index:idx_article_user;not null"`
	Title         string `gorm:"size:80;not null"`
	CoverURL      string `gorm:"size:1024"`
	BodyMD        string `gorm:"type:longtext;not null"`
	Status        string `gorm:"size:32;index:idx_article_status;not null;default:draft"`
	TagsJSON      string `gorm:"type:text"`
	ViewCount     uint64 `gorm:"default:0"`
	CommentCount  uint64 `gorm:"default:0"`
	// CommentsClosed：作者关闭评论区后禁止新发评论；列表对访客返回空。
	CommentsClosed bool `gorm:"not null;default:0"`
	// CommentsCurated：开启评论精选后，新评论需作者确认才对所有人可见。
	CommentsCurated bool `gorm:"not null;default:0"`
	CoinCount     uint64 `gorm:"default:0"`
	FavCount      uint64 `gorm:"default:0"`
	ForwardCount  uint64 `gorm:"default:0"`
	FailReason    string `gorm:"size:2000"`
	PublishedAt   *time.Time
	ReviewedAt         *time.Time
	ReviewedByAdminID  *uint64 `gorm:"index"`
	CreatedAt     time.Time `gorm:"index:idx_article_created"`
	UpdatedAt     time.Time
}

// ArticleFavorite records a user's favorite on an article (图文收藏夹).
type ArticleFavorite struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64 `gorm:"uniqueIndex:idx_article_fav_user_article;not null"`
	ArticleID uint64 `gorm:"uniqueIndex:idx_article_fav_user_article;not null"`
	CreatedAt time.Time
}

// ArticleCoin records coins tipped on an article (one row per user per article).
type ArticleCoin struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64 `gorm:"uniqueIndex:idx_article_coin_user_article;not null"`
	ArticleID uint64 `gorm:"uniqueIndex:idx_article_coin_user_article;not null"`
	Amount    int    `gorm:"not null;default:1"`
	CreatedAt time.Time
}

// ArticleComment is a threaded comment under an article (max depth 3).
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
