package notification

import (
	"cakecake/internal/model/comment"
	"cakecake/internal/model/notification"
	"cakecake/internal/model/user"
	"context"

	"gorm.io/gorm"
)

// NotificationStore is the notification-domain storage boundary.
// Phase 1: *gorm.DB impl. Phase 2+: replaced by gRPC client / per-domain store.
type NotificationStore interface {
	CreateNotification(ctx context.Context, n *notification.Notification) error
	GetCommentByID(ctx context.Context, id uint64) (*comment.Comment, error)
	GetArticleCommentByID(ctx context.Context, id uint64) (*comment.ArticleComment, error)
	GetUserByID(ctx context.Context, id uint64) (*user.User, error)
	CountLikeNotifMute(ctx context.Context, recipientID, commentID uint64) (int64, error)
	FindLikeAggregation(ctx context.Context, recipientID uint64, relatedKey string) (*notification.Notification, error)
	UpdateNotification(ctx context.Context, id uint64, fields map[string]interface{}) error
	UnreadSummary(ctx context.Context, userID uint64) map[string]int64
	ListNotifications(ctx context.Context, userID uint64, cat string, page, pageSize int) ([]notification.Notification, int64, error)
	MarkNotificationsRead(ctx context.Context, userID uint64, ids []uint64) error
	MarkCategoryRead(ctx context.Context, userID uint64, cat string) error
	DeleteNotification(ctx context.Context, userID, notifID uint64) error
	GetNotification(ctx context.Context, notifID, userID uint64) (*notification.Notification, error)
	MuteCommentForRecipient(ctx context.Context, recipientID, commentID uint64) error
	ListCommentLikers(ctx context.Context, commentID uint64) ([]comment.CommentLike, error)
	GetUsersByIDsRaw(ctx context.Context, ids []uint64) ([]user.User, error)
}

// NotificationStoreImpl implements NotificationStore using *gorm.DB (Phase 1 monolith).
type NotificationStoreImpl struct {
	db *gorm.DB
}

func NewNotificationStore(db *gorm.DB) *NotificationStoreImpl {
	return &NotificationStoreImpl{db: db}
}

func (p *NotificationStoreImpl) CreateNotification(ctx context.Context, n *notification.Notification) error {
	return p.db.WithContext(ctx).Create(n).Error
}

func (p *NotificationStoreImpl) GetCommentByID(ctx context.Context, id uint64) (*comment.Comment, error) {
	var cm comment.Comment
	if err := p.db.WithContext(ctx).First(&cm, id).Error; err != nil {
		return nil, err
	}
	return &cm, nil
}

func (p *NotificationStoreImpl) GetArticleCommentByID(ctx context.Context, id uint64) (*comment.ArticleComment, error) {
	var cm comment.ArticleComment
	if err := p.db.WithContext(ctx).First(&cm, id).Error; err != nil {
		return nil, err
	}
	return &cm, nil
}

func (p *NotificationStoreImpl) GetUserByID(ctx context.Context, id uint64) (*user.User, error) {
	var u user.User
	if err := p.db.WithContext(ctx).First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (p *NotificationStoreImpl) CountLikeNotifMute(ctx context.Context, recipientID, commentID uint64) (int64, error) {
	var muteCount int64
	err := p.db.WithContext(ctx).Model(&notification.LikeNotifMute{}).
		Where("recipient_id = ? AND comment_id = ?", recipientID, commentID).Count(&muteCount).Error
	return muteCount, err
}

func (p *NotificationStoreImpl) FindLikeAggregation(ctx context.Context, recipientID uint64, relatedKey string) (*notification.Notification, error) {
	var existing notification.Notification
	err := p.db.WithContext(ctx).Where("recipient_id = ? AND type = ? AND payload_json LIKE ?",
		recipientID, "like_aggregation", relatedKey+"%").First(&existing).Error
	if err != nil {
		return nil, err
	}
	return &existing, nil
}

func (p *NotificationStoreImpl) UpdateNotification(ctx context.Context, id uint64, fields map[string]interface{}) error {
	return p.db.WithContext(ctx).Model(&notification.Notification{}).Where("id = ?", id).Updates(fields).Error
}

func (p *NotificationStoreImpl) UnreadSummary(ctx context.Context, userID uint64) map[string]int64 {
	r := map[string]int64{"reply": 0, "at": 0, "like": 0, "system": 0, "dm": 0}
	if userID == 0 {
		return r
	}
	var rows []struct {
		Type  string
		Count int64
	}
	p.db.WithContext(ctx).Model(&notification.Notification{}).
		Select("type, COUNT(*) as count").
		Where("recipient_id = ? AND is_read = ?", userID, false).
		Group("type").Find(&rows)
	for _, row := range rows {
		switch row.Type {
		case "reply", "reply_like":
			r["reply"] += row.Count
		case "at":
			r["at"] += row.Count
		case "like", "like_aggregation":
			r["like"] += row.Count
		case "system":
			r["system"] += row.Count
		case "dm":
			r["dm"] += row.Count
		}
	}
	return r
}

func (p *NotificationStoreImpl) ListNotifications(ctx context.Context, userID uint64, cat string, page, pageSize int) ([]notification.Notification, int64, error) {
	q := p.db.WithContext(ctx).Where("recipient_id = ?", userID)
	switch cat {
	case "reply":
		q = q.Where("type IN ?", []string{"reply", "reply_like"})
	case "at":
		q = q.Where("type = ?", "at")
	case "like":
		q = q.Where("type IN ?", []string{"like", "like_aggregation"})
	case "system":
		q = q.Where("type = ?", "system")
	case "dm":
		q = q.Where("type = ?", "dm")
	}
	var total int64
	q.Model(&notification.Notification{}).Count(&total)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}
	var list []notification.Notification
	q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list)
	return list, total, nil
}

func (p *NotificationStoreImpl) MarkNotificationsRead(ctx context.Context, userID uint64, ids []uint64) error {
	return p.db.WithContext(ctx).Model(&notification.Notification{}).
		Where("id IN ? AND recipient_id = ?", ids, userID).Update("is_read", true).Error
}

func (p *NotificationStoreImpl) MarkCategoryRead(ctx context.Context, userID uint64, cat string) error {
	q := p.db.WithContext(ctx).Model(&notification.Notification{}).Where("recipient_id = ?", userID)
	switch cat {
	case "reply":
		q = q.Where("type IN ?", []string{"reply", "reply_like"})
	case "at":
		q = q.Where("type = ?", "at")
	case "like":
		q = q.Where("type IN ?", []string{"like", "like_aggregation"})
	case "system":
		q = q.Where("type = ?", "system")
	case "dm":
		q = q.Where("type = ?", "dm")
	}
	return q.Update("is_read", true).Error
}

func (p *NotificationStoreImpl) DeleteNotification(ctx context.Context, userID, notifID uint64) error {
	return p.db.WithContext(ctx).Where("id = ? AND recipient_id = ?", notifID, userID).Delete(&notification.Notification{}).Error
}

func (p *NotificationStoreImpl) GetNotification(ctx context.Context, notifID, userID uint64) (*notification.Notification, error) {
	var n notification.Notification
	if err := p.db.WithContext(ctx).Where("id = ? AND recipient_id = ?", notifID, userID).First(&n).Error; err != nil {
		return nil, err
	}
	return &n, nil
}

func (p *NotificationStoreImpl) MuteCommentForRecipient(ctx context.Context, recipientID, commentID uint64) error {
	mute := notification.LikeNotifMute{RecipientID: recipientID, CommentID: commentID}
	return p.db.WithContext(ctx).Where("recipient_id = ? AND comment_id = ?", recipientID, commentID).
		FirstOrCreate(&mute).Error
}

func (p *NotificationStoreImpl) ListCommentLikers(ctx context.Context, commentID uint64) ([]comment.CommentLike, error) {
	var commentLikes []comment.CommentLike
	if err := p.db.WithContext(ctx).Where("comment_id = ?", commentID).Find(&commentLikes).Error; err != nil {
		return nil, err
	}
	return commentLikes, nil
}

func (p *NotificationStoreImpl) GetUsersByIDsRaw(ctx context.Context, ids []uint64) ([]user.User, error) {
	var dbUsers []user.User
	if err := p.db.WithContext(ctx).Where("id IN ?", ids).Find(&dbUsers).Error; err != nil {
		return nil, err
	}
	return dbUsers, nil
}
