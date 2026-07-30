package handler

import (
	"minibili/internal/model/article"
	"minibili/internal/model/comment"
	"minibili/internal/model/notification"
	"minibili/internal/model/user"
	"minibili/internal/model/video"
	"encoding/json"
	"time"


	"minibili/internal/pkg/useravatar"
)

// replyInboxTarget holds the resolved target for a notification.
type replyInboxTarget struct {
	VideoID   uint64
	CommentID uint64
	ArticleID uint64
}

// likeNotifPayloadSubject returns the like_subject field from a notification payload.
func (a *API) likeNotifPayloadSubject(n *notification.Notification) string {
	if n.PayloadJSON == "" {
		return ""
	}
	var p struct {
		LikeSubject string `json:"like_subject"`
	}
	if err := json.Unmarshal([]byte(n.PayloadJSON), &p); err != nil {
		return ""
	}
	return p.LikeSubject
}

// consolidateDuplicateLikeAggregations removes duplicate like_aggregation notifications,
// keeping only the most recent one for each (recipient_id, related_id) pair.
func (a *API) consolidateDuplicateLikeAggregations(userID uint64) {
	var rows []notification.Notification
	a.DB.Where("recipient_id = ? AND type = ?", userID, "like_aggregation").
		Order("created_at DESC").Find(&rows)
	seen := make(map[uint64]bool)
	for _, n := range rows {
		if seen[n.RelatedID] {
			a.DB.Delete(&n)
		} else {
			seen[n.RelatedID] = true
		}
	}
}

// consolidateLikeAggregationNotifs removes like_aggregation notifications older than maxAgeDays.
func (a *API) consolidateLikeAggregationNotifs(userID uint64, maxAgeDays int) {
	cutoff := time.Now().AddDate(0, 0, -maxAgeDays)
	a.DB.Where("recipient_id = ? AND type = ? AND created_at < ?", userID, "like_aggregation", cutoff).
		Delete(&notification.Notification{})
}

// likeAggTotalFromDB returns the total likes count for a given related entity.
func (a *API) likeAggTotalFromDB(relatedID uint64, isArticle bool) int {
	if isArticle {
		var total int64
		a.DB.Model(&article.ArticleFavorite{}).Where("article_id = ?", relatedID).Count(&total)
		return int(total)
	}
	var total int64
	a.DB.Model(&video.VideoLike{}).Where("video_id = ?", relatedID).Count(&total)
	return int(total)
}

// likeAggTopLikerNames returns the top N liker names for a given related entity.
func (a *API) likeAggTopLikerNames(relatedID uint64, isArticle bool, limit int) []string {
	var userIDs []uint64
	if isArticle {
		a.DB.Model(&article.ArticleFavorite{}).Where("article_id = ?", relatedID).
			Order("id ASC").Limit(limit).Pluck("user_id", &userIDs)
	} else {
		a.DB.Model(&video.VideoLike{}).Where("video_id = ?", relatedID).
			Order("id ASC").Limit(limit).Pluck("user_id", &userIDs)
	}
	var names []string
	for _, uid := range userIDs {
		var u user.User
		if err := a.DB.First(&u, uid).Error; err == nil {
			names = append(names, u.Username)
		}
	}
	return names
}

// formatNotification formats a notification for API response.
func (a *API) formatNotification(n notification.Notification) map[string]interface{} {
	out := map[string]interface{}{
		"id":               n.ID,
		"type":             n.Type,
		"related_id":       n.RelatedID,
		"comment_preview":  n.CommentPreview,
		"sender_names":     n.SenderNamesJSON,
		"total_likes":      n.TotalLikes,
		"is_read":          n.IsRead,
		"payload":          n.PayloadJSON,
		"created_at":       n.CreatedAt.Format(time.RFC3339),
	}
	if n.Type == "like_aggregation" {
		subject := a.likeNotifPayloadSubject(&n)
		switch subject {
		case "article_comment":
			out["like_target"] = "文章"
		case "danmaku":
			out["like_target"] = "弹幕"
		default:
			out["like_target"] = "评论"
		}
	}
	return out
}

// likeNotifTopSenders resolves sender names to avatar URLs and user IDs.
func (a *API) likeNotifTopSenders(senderNames []string, limit int) ([]string, []uint64) {
	var urls []string
	var ids []uint64
	for _, name := range senderNames {
		if limit > 0 && len(urls) >= limit {
			break
		}
		var u user.User
		if err := a.DB.Where("username = ?", name).First(&u).Error; err == nil {
			urls = append(urls, useravatar.PublicURL(&u))
			ids = append(ids, u.ID)
		}
	}
	return urls, ids
}

// clearCommentDislike removes a dislike record for the given user and comment.
func (a *API) clearCommentDislike(userID, commentID uint64) (bool, error) {
	res := a.DB.Where("user_id = ? AND comment_id = ?", userID, commentID).
		Delete(&comment.CommentDislike{})
	return res.RowsAffected > 0, res.Error
}

// clearCommentLike removes a like record for the given user and comment.
func (a *API) clearCommentLike(userID, commentID uint64, cm *comment.Comment) (bool, error) {
	res := a.DB.Where("user_id = ? AND comment_id = ?", userID, commentID).
		Delete(&comment.CommentLike{})
	return res.RowsAffected > 0, res.Error
}

// resolveReplyInboxTarget resolves the target (video/comment/article) from a notification.
func (a *API) resolveReplyInboxTarget(n *notification.Notification) (replyInboxTarget, bool) {
	switch n.Type {
	case "reply_received":
		var cm comment.Comment
		if err := a.DB.First(&cm, n.RelatedID).Error; err != nil {
			return replyInboxTarget{}, false
		}
		return replyInboxTarget{CommentID: cm.ID, VideoID: cm.VideoID}, true
	case "like_aggregation":
		var p struct {
			VideoID   uint64 `json:"video_id"`
			CommentID uint64 `json:"comment_id"`
			ArticleID uint64 `json:"article_id"`
		}
		if err := json.Unmarshal([]byte(n.PayloadJSON), &p); err != nil {
			return replyInboxTarget{}, false
		}
		return replyInboxTarget{VideoID: p.VideoID, CommentID: p.CommentID, ArticleID: p.ArticleID}, true
	default:
		return replyInboxTarget{}, false
	}
}


// mergeUniqueDisplayNames deduplicates and returns unique display names in original order.
func mergeUniqueDisplayNames(names []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, n := range names {
		if !seen[n] { seen[n] = true; out = append(out, n) }
	}
	return out
}

// isReplyInboxType returns true if the notification type is reply-related.
func isReplyInboxType(notifType string) bool {
	switch notifType {
	case "reply_received", "article_reply_received", "dynamic_reply_received":
		return true
	default:
		return false
	}
}

// notifUint64 converts a value to uint64 for notification processing.
func notifUint64(v interface{}) uint64 {
	switch val := v.(type) {
	case float64:
		return uint64(val)
	case uint64:
		return val
	default:
		return 0
	}
}