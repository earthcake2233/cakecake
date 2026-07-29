package handler

import (
	"encoding/json"
	"time"

	"minibili/internal/model"
	"minibili/internal/pkg/useravatar"
)

// replyInboxTarget holds the resolved target for a notification.
type replyInboxTarget struct {
	VideoID   uint64
	CommentID uint64
	ArticleID uint64
}

// likeNotifPayloadIsArticle returns true if the notification's payload indicates an article comment like.
func (a *API) likeNotifPayloadIsArticle(n *model.Notification) bool {
	if n.PayloadJSON == "" {
		return false
	}
	var p struct {
		LikeSubject string `json:"like_subject"`
	}
	if err := json.Unmarshal([]byte(n.PayloadJSON), &p); err != nil {
		return false
	}
	return p.LikeSubject == "article_comment"
}

// consolidateDuplicateLikeAggregations removes duplicate like_aggregation notifications,
// keeping only the most recent one for each (recipient_id, related_id) pair.
func (a *API) consolidateDuplicateLikeAggregations(userID uint64) {
	var rows []model.Notification
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
		Delete(&model.Notification{})
}

// likeAggTotalFromDB returns the total likes count for a given related entity.
func (a *API) likeAggTotalFromDB(relatedID uint64, isArticle bool) int {
	if isArticle {
		var total int64
		a.DB.Model(&model.ArticleFavorite{}).Where("article_id = ?", relatedID).Count(&total)
		return int(total)
	}
	var total int64
	a.DB.Model(&model.VideoLike{}).Where("video_id = ? ", relatedID).Count(&total)
	return int(total)
}

// likeAggTopLikerNames returns the top N liker names for a given related entity.
func (a *API) likeAggTopLikerNames(relatedID uint64, isArticle bool, limit int) []string {
	var userIDs []uint64
	if isArticle {
		a.DB.Model(&model.ArticleFavorite{}).Where("article_id = ?", relatedID).
			Order("id ASC").Limit(limit).Pluck("user_id", &userIDs)
	} else {
		a.DB.Model(&model.VideoLike{}).Where("video_id = ? ", relatedID, 1).
			Order("id ASC").Limit(limit).Pluck("user_id", &userIDs)
	}
	var names []string
	for _, uid := range userIDs {
		var u model.User
		if err := a.DB.First(&u, uid).Error; err == nil {
			names = append(names, u.Username)
		}
	}
	return names
}

// formatNotification formats a notification for API response.
func (a *API) formatNotification(n model.Notification) map[string]interface{} {
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
		if a.likeNotifPayloadIsArticle(&n) {
			out["like_target"] = "文章"
		} else {
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
		var u model.User
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
		Delete(&model.CommentDislike{})
	return res.RowsAffected > 0, res.Error
}

// clearCommentLike removes a like record for the given user and comment.
func (a *API) clearCommentLike(userID, commentID uint64, cm *model.Comment) (bool, error) {
	res := a.DB.Where("user_id = ? AND comment_id = ?", userID, commentID).
		Delete(&model.CommentLike{})
	return res.RowsAffected > 0, res.Error
}

// resolveReplyInboxTarget resolves the target (video/comment/article) from a notification.
func (a *API) resolveReplyInboxTarget(n *model.Notification) (replyInboxTarget, bool) {
	switch n.Type {
	case "reply_received":
		var cm model.Comment
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