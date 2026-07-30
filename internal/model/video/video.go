package video
import "time"
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
	// CommentsClosed: when the uploader closes comments, new comments are blocked; list returns empty for visitors.
	CommentsClosed bool `gorm:"not null;default:0"`
	// CommentsCurated: when curation is enabled, new comments require uploader approval before being visible to all.
	CommentsCurated bool `gorm:"not null;default:0"`
	// DanmakuClosed: when the uploader closes danmaku, new danmaku are blocked.
	DanmakuClosed bool `gorm:"not null;default:0"`
	// TagsJSON is a JSON array of strings, e.g. ["tutorial","guide"]; empty string means no tags.
	TagsJSON string `gorm:"type:text"`
	// Zone is the publish partition, e.g. "anime" or "lifestyle-daily".
	Zone string `gorm:"size:64"`
	// DraftRawPath / DraftCoverPath: local staging paths when status=draft, used before submission transcoding.
	DraftRawPath   string    `gorm:"size:1024"`
	DraftCoverPath string    `gorm:"size:1024"`
	ReviewedAt     *time.Time
	ReviewedByAdminID *uint64 `gorm:"index"`
	CreatedAt      time.Time `gorm:"index:idx_video_created"`
	UpdatedAt      time.Time
}
type VideoLike struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64 `gorm:"uniqueIndex:idx_video_like_user_video;not null"`
	VideoID   uint64 `gorm:"uniqueIndex:idx_video_like_user_video;not null"`
	CreatedAt time.Time
}
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
type VideoFavorite struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64 `gorm:"uniqueIndex:idx_video_fav_user_video_folder,priority:1;not null"`
	VideoID   uint64 `gorm:"uniqueIndex:idx_video_fav_user_video_folder,priority:2;not null"`
	FolderID  uint64 `gorm:"uniqueIndex:idx_video_fav_user_video_folder,priority:3;index:idx_video_fav_folder;not null;default:0"`
	CreatedAt time.Time
}
type VideoCoin struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64 `gorm:"uniqueIndex:idx_video_coin_user_video;not null"`
	VideoID   uint64 `gorm:"uniqueIndex:idx_video_coin_user_video;not null"`
	Amount    int    `gorm:"not null;default:1"` // 1 or 2
	CreatedAt time.Time
}
type WatchLater struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64 `gorm:"uniqueIndex:idx_watch_later_user_video;not null"`
	VideoID   uint64 `gorm:"uniqueIndex:idx_watch_later_user_video;not null"`
	Watched   bool   `gorm:"not null;default:0"`
	CreatedAt time.Time `gorm:"index"`
}
