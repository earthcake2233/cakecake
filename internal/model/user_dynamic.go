package model

import "time"

// UserDynamic is a user-published image/text feed post (动态图文).
type UserDynamic struct {
	ID         uint64 `gorm:"primaryKey"`
	UserID     uint64 `gorm:"index:idx_dyn_user_created;not null"`
	Title      string `gorm:"size:20;not null;default:''"`
	Content    string `gorm:"size:233;not null;default:''"`
	ImagesJSON    string `gorm:"type:text;not null"`
	LikeCount     uint64 `gorm:"not null;default:0"`
	CommentCount  uint64 `gorm:"not null;default:0"`
	// CommentsClosed：作者关闭评论区后禁止新发评论；列表对访客返回空。
	CommentsClosed bool `gorm:"not null;default:0"`
	// CommentsCurated：开启评论精选后，新评论需作者确认才对所有人可见。
	CommentsCurated bool `gorm:"not null;default:0"`
	CreatedAt     time.Time `gorm:"index:idx_dyn_user_created"`
}
