package dynamic
import "time"
type UserDynamic struct {
	ID         uint64 `gorm:"primaryKey"`
	UserID     uint64 `gorm:"index:idx_dyn_user_created;not null"`
	Title      string `gorm:"size:20;not null;default:''"`
	Content    string `gorm:"size:233;not null;default:''"`
	ImagesJSON    string `gorm:"type:text;not null"`
	LikeCount     uint64 `gorm:"not null;default:0"`
	CommentCount  uint64 `gorm:"not null;default:0"`
	// CommentsClosed: when the author closes comments, new comments are blocked; list returns empty for visitors.
	CommentsClosed bool `gorm:"not null;default:0"`
	// CommentsCurated: when curation is enabled, new comments require author approval before being visible to all.
	CommentsCurated bool `gorm:"not null;default:0"`
	CreatedAt     time.Time `gorm:"index:idx_dyn_user_created"`
}
