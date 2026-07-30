package service

import (
	"minibili/internal/model/comment"
	"minibili/internal/model/notification"
	"minibili/internal/model/user"
	"fmt"
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

)

type NotificationService struct {
	db  *gorm.DB
	rdb *redis.Client
	log *zap.Logger

	// providers for cross-domain data
	users UserProvider
}

func NewNotificationService(db *gorm.DB, rdb *redis.Client, log *zap.Logger, users UserProvider) *NotificationService {
	return &NotificationService{db: db, rdb: rdb, log: log, users: users}
}


func truncateStr(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max { return s }
	return string(runes[:max])
}

func (ns *NotificationService) NotifyVideoComment(ctx context.Context, uploaderID, commenterID uint64, cm comment.Comment) {
	if uploaderID == 0 || commenterID == uploaderID { return }
	payload, _ := json.Marshal(map[string]interface{}{"video_id": cm.VideoID, "sender_id": commenterID})
	ns.db.WithContext(ctx).Create(&notification.Notification{
		RecipientID: uploaderID, Type: "reply", RelatedID: cm.ID,
		CommentPreview: truncateStr(cm.Content, 15), PayloadJSON: string(payload),
	})
}

func (ns *NotificationService) NotifyCommentReply(ctx context.Context, videoID, replierID uint64, reply *comment.Comment, parentID uint64) {
	var parent comment.Comment
	if err := ns.db.WithContext(ctx).First(&parent, parentID).Error; err != nil { return }
	if parent.UserID == replierID { return }
	payload, _ := json.Marshal(map[string]interface{}{"video_id": videoID, "sender_id": replierID})
	ns.db.WithContext(ctx).Create(&notification.Notification{
		RecipientID: parent.UserID, Type: "reply", RelatedID: reply.ID,
		CommentPreview: truncateStr(reply.Content, 15), PayloadJSON: string(payload),
	})
}

func (ns *NotificationService) NotifyCommentLike(ctx context.Context, cm comment.Comment, likerID uint64) {
	if cm.UserID == 0 || cm.UserID == likerID { return }
	var likerName string
	if ns.users != nil {
		u, err := ns.users.GetUser(ctx, likerID)
		if err != nil { return }
		likerName = u.Username
		if u.Nickname != "" { likerName = u.Nickname }
	} else {
		var liker user.User
		if err := ns.db.WithContext(ctx).First(&liker, likerID).Error; err != nil { return }
		likerName = liker.Username
		if liker.Nickname != "" { likerName = liker.Nickname }
	}

	var muteCount int64
	ns.db.WithContext(ctx).Model(&notification.LikeNotifMute{}).Where("recipient_id = ? AND comment_id = ?", cm.UserID, cm.ID).Count(&muteCount)
	if muteCount > 0 { return }

	// Upsert like aggregation
	var existing notification.Notification
	// Use PayloadJSON to store related comment ID for lookups
	relatedKey := "like_comment:" + itoa(cm.ID)
	err := ns.db.WithContext(ctx).Where("recipient_id = ? AND type = ? AND payload_json LIKE ?",
		cm.UserID, "like_aggregation", relatedKey+"%").First(&existing).Error
	if err == nil {
		var names []string
		if existing.SenderNamesJSON != "" {
			_ = json.Unmarshal([]byte(existing.SenderNamesJSON), &names)
		}
		names = append(names, likerName)
		if len(names) > 10 { names = names[:10] }
		b, _ := json.Marshal(names)
		ns.db.WithContext(ctx).Model(&existing).Updates(map[string]interface{}{
			"comment_preview": truncateStr(cm.Content, 15),
			"sender_names_json": string(b),
			"total_likes": gorm.Expr("total_likes + 1"),
		})
	} else {
		names, _ := json.Marshal([]string{likerName})
		ns.db.WithContext(ctx).Create(&notification.Notification{
			RecipientID: cm.UserID, Type: "like_aggregation", RelatedID: cm.ID,
			CommentPreview: truncateStr(cm.Content, 15),
			SenderNamesJSON: string(names),
			TotalLikes: 1,
			PayloadJSON: relatedKey,
		})
	}
}

func (ns *NotificationService) NotifyArticleComment(ctx context.Context, authorID, commenterID uint64, cm comment.ArticleComment) {
	if authorID == 0 || commenterID == authorID { return }
	payload, _ := json.Marshal(map[string]interface{}{"article_id": cm.ArticleID, "sender_id": commenterID})
	ns.db.WithContext(ctx).Create(&notification.Notification{
		RecipientID: authorID, Type: "reply", RelatedID: cm.ID,
		CommentPreview: truncateStr(cm.Content, 15), PayloadJSON: string(payload),
	})
}

func (ns *NotificationService) NotifyArticleCommentReply(ctx context.Context, articleID, replierID uint64, reply *comment.ArticleComment, parentID uint64) {
	var parent comment.ArticleComment
	if err := ns.db.WithContext(ctx).First(&parent, parentID).Error; err != nil { return }
	if parent.UserID == replierID { return }
	payload, _ := json.Marshal(map[string]interface{}{"article_id": articleID, "sender_id": replierID})
	ns.db.WithContext(ctx).Create(&notification.Notification{
		RecipientID: parent.UserID, Type: "reply", RelatedID: reply.ID,
		CommentPreview: truncateStr(reply.Content, 15), PayloadJSON: string(payload),
	})
}

// UnreadSummary returns unread counts per category.
func (ns *NotificationService) UnreadSummary(ctx context.Context, userID uint64) map[string]int64 {
	r := map[string]int64{"reply": 0, "at": 0, "like": 0, "system": 0, "dm": 0}
	if userID == 0 { return r }
	var rows []struct {
		Type  string
		Count int64
	}
	ns.db.WithContext(ctx).Model(&notification.Notification{}).
		Select("type, COUNT(*) as count").
		Where("recipient_id = ? AND is_read = ?", userID, false).
		Group("type").Find(&rows)
	for _, row := range rows {
		switch row.Type {
		case "reply", "reply_like": r["reply"] += row.Count
		case "at": r["at"] += row.Count
		case "like", "like_aggregation": r["like"] += row.Count
		case "system": r["system"] += row.Count
		case "dm": r["dm"] += row.Count
		}
	}
	return r
}

func (ns *NotificationService) ListNotifications(ctx context.Context, userID uint64, cat string, page, pageSize int) ([]notification.Notification, int64, error) {
	q := ns.db.WithContext(ctx).Where("recipient_id = ?", userID)
	switch cat {
	case "reply": q = q.Where("type IN ?", []string{"reply", "reply_like"})
	case "at": q = q.Where("type = ?", "at")
	case "like": q = q.Where("type IN ?", []string{"like", "like_aggregation"})
	case "system": q = q.Where("type = ?", "system")
	case "dm": q = q.Where("type = ?", "dm")
	}
	var total int64
	q.Model(&notification.Notification{}).Count(&total)
	if page < 1 { page = 1 }
	if pageSize < 1 || pageSize > 50 { pageSize = 20 }
	var list []notification.Notification
	q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list)
	return list, total, nil
}

func (ns *NotificationService) MarkNotificationsRead(ctx context.Context, userID uint64, ids []uint64) error {
	return ns.db.WithContext(ctx).Model(&notification.Notification{}).
		Where("id IN ? AND recipient_id = ?", ids, userID).Update("is_read", true).Error
}

func (ns *NotificationService) MarkCategoryRead(ctx context.Context, userID uint64, cat string) error {
	q := ns.db.WithContext(ctx).Model(&notification.Notification{}).Where("recipient_id = ?", userID)
	switch cat {
	case "reply": q = q.Where("type IN ?", []string{"reply", "reply_like"})
	case "at": q = q.Where("type = ?", "at")
	case "like": q = q.Where("type IN ?", []string{"like", "like_aggregation"})
	case "system": q = q.Where("type = ?", "system")
	case "dm": q = q.Where("type = ?", "dm")
	}
	return q.Update("is_read", true).Error
}

func (ns *NotificationService) DeleteNotification(ctx context.Context, userID, notifID uint64) error {
	return ns.db.WithContext(ctx).Where("id = ? AND recipient_id = ?", notifID, userID).Delete(&notification.Notification{}).Error
}


func (ns *NotificationService) GetNotification(ctx context.Context, notifID, userID uint64) (*notification.Notification, error) {
	var n notification.Notification
	if err := ns.db.WithContext(ctx).Where("id = ? AND recipient_id = ?", notifID, userID).First(&n).Error; err != nil {
		return nil, err
	}
	return &n, nil
}

func (ns *NotificationService) MuteLikeNotification(ctx context.Context, userID, notifID uint64) error {
	n, err := ns.GetNotification(ctx, notifID, userID)
	if err != nil { return err }
	if n.Type != "like_aggregation" || n.RelatedID == 0 { return ErrParamError }
	mute := notification.LikeNotifMute{RecipientID: userID, CommentID: n.RelatedID}
	return ns.db.WithContext(ctx).Where("recipient_id = ? AND comment_id = ?", userID, n.RelatedID).
		FirstOrCreate(&mute).Error
}

// ListNotificationLikers returns the users who liked a comment referenced by a notification.
func (ns *NotificationService) ListNotificationLikers(ctx context.Context, userID, notifID uint64) ([]UserInfo, error) {
	n, err := ns.GetNotification(ctx, notifID, userID)
	if err != nil { return nil, err }
	var commentLikes []comment.CommentLike
	ns.db.WithContext(ctx).Where("comment_id = ?", n.RelatedID).Find(&commentLikes)
	uids := make([]uint64, 0, len(commentLikes))
	for _, l := range commentLikes { uids = append(uids, l.UserID) }
	if len(uids) == 0 { return nil, nil }
	if ns.users != nil {
		result, err := ns.users.GetUsersByIDs(ctx, uids)
		if err != nil { return nil, err }
		out := make([]UserInfo, 0, len(result))
		for _, u := range result { out = append(out, u) }
		return out, nil
	}
	// Fallback: direct DB query (legacy path)
	var dbUsers []user.User
	if err := ns.db.WithContext(ctx).Where("id IN ?", uids).Find(&dbUsers).Error; err != nil { return nil, err }
	out := make([]UserInfo, len(dbUsers))
	for i, u := range dbUsers { out[i] = toUserInfo(&u) }
	return out, nil
}

func itoa(n uint64) string { return fmt.Sprintf("%d", n) }