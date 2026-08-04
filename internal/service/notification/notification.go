package notification

import (
	"cakecake/internal/model/comment"
	"cakecake/internal/model/notification"
	"cakecake/internal/service"
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type NotificationService struct {
	store NotificationStore
	rdb   *redis.Client
	log   *zap.Logger

	// providers for cross-domain data
	users service.UserProvider
}

func NewNotificationService(db *gorm.DB, rdb *redis.Client, log *zap.Logger, users service.UserProvider) *NotificationService {
	return &NotificationService{store: NewNotificationStore(db), rdb: rdb, log: log, users: users}
}

func truncateStr(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}

func (ns *NotificationService) NotifyVideoComment(ctx context.Context, uploaderID, commenterID uint64, cm comment.Comment) {
	if uploaderID == 0 || commenterID == uploaderID {
		return
	}
	payload, _ := json.Marshal(map[string]interface{}{"video_id": cm.VideoID, "sender_id": commenterID})
	_ = ns.store.CreateNotification(ctx, &notification.Notification{
		RecipientID: uploaderID, Type: "reply", RelatedID: cm.ID,
		CommentPreview: truncateStr(cm.Content, 15), PayloadJSON: string(payload),
	})
}

func (ns *NotificationService) NotifyCommentReply(ctx context.Context, videoID, replierID uint64, reply *comment.Comment, parentID uint64) {
	parent, err := ns.store.GetCommentByID(ctx, parentID)
	if err != nil {
		return
	}
	if parent.UserID == replierID {
		return
	}
	payload, _ := json.Marshal(map[string]interface{}{"video_id": videoID, "sender_id": replierID})
	_ = ns.store.CreateNotification(ctx, &notification.Notification{
		RecipientID: parent.UserID, Type: "reply", RelatedID: reply.ID,
		CommentPreview: truncateStr(reply.Content, 15), PayloadJSON: string(payload),
	})
}

func (ns *NotificationService) NotifyCommentLike(ctx context.Context, cm comment.Comment, likerID uint64) {
	if cm.UserID == 0 || cm.UserID == likerID {
		return
	}
	var likerName string
	if ns.users != nil {
		u, err := ns.users.GetUser(ctx, likerID)
		if err != nil {
			return
		}
		likerName = u.Username
		if u.Nickname != "" {
			likerName = u.Nickname
		}
	} else {
		liker, err := ns.store.GetUserByID(ctx, likerID)
		if err != nil {
			return
		}
		likerName = liker.Username
		if liker.Nickname != "" {
			likerName = liker.Nickname
		}
	}

	muteCount, _ := ns.store.CountLikeNotifMute(ctx, cm.UserID, cm.ID)
	if muteCount > 0 {
		return
	}

	// Upsert like aggregation
	// Use PayloadJSON to store related comment ID for lookups
	relatedKey := "like_comment:" + itoa(cm.ID)
	existing, err := ns.store.FindLikeAggregation(ctx, cm.UserID, relatedKey)
	if err == nil {
		var names []string
		if existing.SenderNamesJSON != "" {
			_ = json.Unmarshal([]byte(existing.SenderNamesJSON), &names)
		}
		names = append(names, likerName)
		if len(names) > 10 {
			names = names[:10]
		}
		b, _ := json.Marshal(names)
		_ = ns.store.UpdateNotification(ctx, existing.ID, map[string]interface{}{
			"comment_preview":   truncateStr(cm.Content, 15),
			"sender_names_json": string(b),
			"total_likes":       gorm.Expr("total_likes + 1"),
		})
	} else {
		names, _ := json.Marshal([]string{likerName})
		_ = ns.store.CreateNotification(ctx, &notification.Notification{
			RecipientID: cm.UserID, Type: "like_aggregation", RelatedID: cm.ID,
			CommentPreview:  truncateStr(cm.Content, 15),
			SenderNamesJSON: string(names),
			TotalLikes:      1,
			PayloadJSON:     relatedKey,
		})
	}
}

func (ns *NotificationService) NotifyArticleComment(ctx context.Context, authorID, commenterID uint64, cm comment.ArticleComment) {
	if authorID == 0 || commenterID == authorID {
		return
	}
	payload, _ := json.Marshal(map[string]interface{}{"article_id": cm.ArticleID, "sender_id": commenterID})
	_ = ns.store.CreateNotification(ctx, &notification.Notification{
		RecipientID: authorID, Type: "reply", RelatedID: cm.ID,
		CommentPreview: truncateStr(cm.Content, 15), PayloadJSON: string(payload),
	})
}

func (ns *NotificationService) NotifyArticleCommentReply(ctx context.Context, articleID, replierID uint64, reply *comment.ArticleComment, parentID uint64) {
	parent, err := ns.store.GetArticleCommentByID(ctx, parentID)
	if err != nil {
		return
	}
	if parent.UserID == replierID {
		return
	}
	payload, _ := json.Marshal(map[string]interface{}{"article_id": articleID, "sender_id": replierID})
	_ = ns.store.CreateNotification(ctx, &notification.Notification{
		RecipientID: parent.UserID, Type: "reply", RelatedID: reply.ID,
		CommentPreview: truncateStr(reply.Content, 15), PayloadJSON: string(payload),
	})
}

// UnreadSummary returns unread counts per category.
func (ns *NotificationService) UnreadSummary(ctx context.Context, userID uint64) map[string]int64 {
	return ns.store.UnreadSummary(ctx, userID)
}

func (ns *NotificationService) ListNotifications(ctx context.Context, userID uint64, cat string, page, pageSize int) ([]notification.Notification, int64, error) {
	return ns.store.ListNotifications(ctx, userID, cat, page, pageSize)
}

func (ns *NotificationService) MarkNotificationsRead(ctx context.Context, userID uint64, ids []uint64) error {
	return ns.store.MarkNotificationsRead(ctx, userID, ids)
}

func (ns *NotificationService) MarkCategoryRead(ctx context.Context, userID uint64, cat string) error {
	return ns.store.MarkCategoryRead(ctx, userID, cat)
}

func (ns *NotificationService) DeleteNotification(ctx context.Context, userID, notifID uint64) error {
	return ns.store.DeleteNotification(ctx, userID, notifID)
}

func (ns *NotificationService) GetNotification(ctx context.Context, notifID, userID uint64) (*notification.Notification, error) {
	return ns.store.GetNotification(ctx, notifID, userID)
}

func (ns *NotificationService) MuteLikeNotification(ctx context.Context, userID, notifID uint64) error {
	n, err := ns.store.GetNotification(ctx, notifID, userID)
	if err != nil {
		return err
	}
	if n.Type != "like_aggregation" || n.RelatedID == 0 {
		return service.ErrParamError
	}
	return ns.store.MuteCommentForRecipient(ctx, userID, n.RelatedID)
}

// ListNotificationLikers returns the users who liked a comment referenced by a notification.
func (ns *NotificationService) ListNotificationLikers(ctx context.Context, userID, notifID uint64) ([]service.UserInfo, error) {
	n, err := ns.store.GetNotification(ctx, notifID, userID)
	if err != nil {
		return nil, err
	}
	commentLikes, err := ns.store.ListCommentLikers(ctx, n.RelatedID)
	if err != nil {
		return nil, err
	}
	uids := make([]uint64, 0, len(commentLikes))
	for _, l := range commentLikes {
		uids = append(uids, l.UserID)
	}
	if len(uids) == 0 {
		return nil, nil
	}
	if ns.users != nil {
		result, err := ns.users.GetUsersByIDs(ctx, uids)
		if err != nil {
			return nil, err
		}
		out := make([]service.UserInfo, 0, len(result))
		for _, u := range result {
			out = append(out, u)
		}
		return out, nil
	}
	// Fallback: direct DB query (legacy path)
	dbUsers, err := ns.store.GetUsersByIDsRaw(ctx, uids)
	if err != nil {
		return nil, err
	}
	out := make([]service.UserInfo, len(dbUsers))
	for i, u := range dbUsers {
		out[i] = service.ToUserInfo(&u)
	}
	return out, nil
}

func itoa(n uint64) string { return fmt.Sprintf("%d", n) }
