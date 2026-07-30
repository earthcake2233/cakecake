package danmaku
import "time"
type Danmaku struct {
	ID        uint64  `gorm:"primaryKey"`
	VideoID   uint64  `gorm:"index:idx_danmaku_video;not null"`
	UserID    uint64  `gorm:"index;not null"`
	Content   string  `gorm:"size:400;not null"`
	Color     string  `gorm:"size:16;not null"`
	Type      string  `gorm:"size:16;not null"`
	// FontSize: sm | md | lg (danmaku font size, default md)
	FontSize  string  `gorm:"size:8;not null;default:md"`
	VideoTime float64 `gorm:"column:video_time;not null"`
	LikeCount uint64  `gorm:"default:0"`
	CreatedAt time.Time
}
func (Danmaku) TableName() string { return "danmakus" }
type DanmakuLike struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64 `gorm:"uniqueIndex:idx_danmaku_like_user_dm;not null"`
	DanmakuID uint64 `gorm:"uniqueIndex:idx_danmaku_like_user_dm;not null"`
}
func (DanmakuLike) TableName() string { return "danmaku_likes" }
