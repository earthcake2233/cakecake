package service

import (
	"minibili/internal/model/comment"
	"context"
	"unicode/utf8"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"minibili/internal/pkg/sensitive"
)

type CommentService struct {
	db       *gorm.DB
	rdb      *redis.Client
	log      *zap.Logger
	sens     *sensitive.Filter
	notifSvc *NotificationService

	// Domain providers (Phase 1: *gorm.DB impl; Phase 2+: gRPC clients)
	users    UserProvider
	videos   VideoProvider
	articles ArticleProvider
	dynamics DynamicProvider
}

func NewCommentService(db *gorm.DB, rdb *redis.Client, log *zap.Logger, sens *sensitive.Filter) *CommentService {
	return &CommentService{db: db, rdb: rdb, log: log, sens: sens}
}

func (s *CommentService) SetNotificationService(ns *NotificationService) { s.notifSvc = ns }

// SetProviders injects domain providers (must be called before use).
func (s *CommentService) SetProviders(users UserProvider, videos VideoProvider, articles ArticleProvider, dynamics DynamicProvider) {
	s.users = users; s.videos = videos; s.articles = articles; s.dynamics = dynamics
}

type PostCommentReq struct {
	Content  string
	ParentID uint64
}

type CommentItem struct {
	ID, UserID           uint64
	Username, AvatarURL  string
	ParentID             uint64
	Level, UserLevel     int
	Content              string
	LikeCount            uint64
	CreatedAt            string
	LikedByMe, DislikedByMe, Pinned, IsByUploader bool
	IPLocation           string
}

type ArticleCommentItem struct {
	ID, UserID           uint64
	Username, AvatarURL  string
	ParentID             uint64
	Level, UserLevel     int
	Content              string
	LikeCount            uint64
	CreatedAt            string
	LikedByMe, DislikedByMe, Pinned, IsByAuthor bool
	IPLocation           string
}

type DynamicCommentItem struct {
	ID, UserID           uint64
	Username, AvatarURL  string
	ParentID             uint64
	Level, UserLevel     int
	Content              string
	LikeCount            uint64
	CreatedAt            string
	LikedByMe, DislikedByMe, Pinned, IsByUploader bool
	IPLocation           string
}

type CommentListResult struct {
	Items           []CommentItem
	CommentsCurated bool
	CommentsClosed  bool
}

type ArticleCommentListResult struct {
	Items           []ArticleCommentItem
	CommentsCurated bool
	CommentsClosed  bool
}

type DynamicCommentListResult struct {
	Items           []DynamicCommentItem
	CommentsCurated bool
	CommentsClosed  bool
}

func (s *CommentService) ListComments(ctx context.Context, videoID, viewerID uint64) (*CommentListResult, error) {
	v, err := s.videos.GetPublishedVideo(ctx, videoID)
	if err != nil { return nil, ErrNotFound }
	r := &CommentListResult{CommentsCurated: v.CommentsCurated, CommentsClosed: v.CommentsClosed}
	if v.CommentsClosed { r.Items = []CommentItem{}; return r, nil }

	q := s.db.WithContext(ctx).Where("video_id = ?", videoID)
	if v.CommentsCurated { q = q.Where("approved = ?", true) }
	var list []comment.Comment
	if err := q.Order("id ASC").Find(&list).Error; err != nil { return nil, ErrInternalError }
	if len(list) == 0 { r.Items = []CommentItem{}; return r, nil }

	users, levels := s.loadUsersWithLevels(ctx, list)
	liked := s.loadCommentLikes(ctx, viewerID, list)
	disliked := s.loadCommentDislikes(ctx, viewerID, list)

	out := make([]CommentItem, 0, len(list))
	for _, cm := range list {
		ulv := levels[cm.UserID]; if ulv < 1 { ulv = 1 }
		u := users[cm.UserID]
		out = append(out, CommentItem{
			ID: cm.ID, UserID: cm.UserID, Username: u.Nickname, AvatarURL: u.AvatarURL,
			ParentID: cm.ParentID, Level: cm.Level, UserLevel: ulv, Content: cm.Content,
			LikeCount: cm.LikeCount, CreatedAt: cm.CreatedAt.Format("2006-01-02 15:04:05"),
			LikedByMe: liked[cm.ID], DislikedByMe: disliked[cm.ID], Pinned: cm.Pinned,
			IsByUploader: cm.UserID == v.UserID, IPLocation: cm.IpLocation,
		})
	}
	r.Items = out; return r, nil
}

func (s *CommentService) PostComment(ctx context.Context, userID, videoID uint64, req PostCommentReq, ipLocation string) (*comment.Comment, error) {
	content := req.Content
	if n := utf8.RuneCountInString(content); n < 1 || n > 1000 { return nil, ErrParamError }
	if s.sens != nil {
		if err := s.sens.Check(content); err != nil {
			if _, ok := err.(sensitive.ErrBlocked); ok { return nil, ErrCommentSensitive }
			return nil, ErrInternalError
		}
	}
	v, err := s.videos.GetPublishedVideo(ctx, videoID)
	if err != nil { return nil, ErrNotFound }
	if v.CommentsClosed { return nil, ErrNotFound }

	var parentID uint64
	level := 1
	if req.ParentID != 0 {
		parentID = req.ParentID
		var parent comment.Comment
		if err := s.db.WithContext(ctx).First(&parent, req.ParentID).Error; err != nil { return nil, ErrNotFound }
		if parent.VideoID != videoID { return nil, ErrParamError }
		level = parent.Level + 1; if level > 3 { level = 3 }
	}

	cm := comment.Comment{
		UserID: userID, VideoID: videoID, ParentID: parentID, Content: content,
		Level: level, Approved: !v.CommentsCurated, IpLocation: ipLocation,
	}
	if err := s.db.WithContext(ctx).Create(&cm).Error; err != nil { return nil, ErrInternalError }
	_ = s.videos.IncrCommentCount(ctx, videoID, 1)
	if s.notifSvc != nil {
		if req.ParentID == 0 { s.notifSvc.NotifyVideoComment(ctx, v.UserID, userID, cm)
		} else { s.notifSvc.NotifyCommentReply(ctx, videoID, userID, &cm, parentID) }
	}
	return &cm, nil
}

func (s *CommentService) DeleteComment(ctx context.Context, userID, commentID uint64, isUploader bool) error {
	var cm comment.Comment
	if err := s.db.WithContext(ctx).First(&cm, commentID).Error; err != nil { return ErrNotFound }
	if !isUploader && cm.UserID != userID { return ErrForbidden }
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		descIDs := s.collectDescendantIDs(tx, commentID)
		allIDs := append([]uint64{commentID}, descIDs...)
		_ = tx.Where("comment_id IN ?", allIDs).Delete(&comment.CommentLike{}).Error
		_ = tx.Where("comment_id IN ?", allIDs).Delete(&comment.CommentDislike{}).Error
		res := tx.Where("id IN ?", allIDs).Delete(&comment.Comment{})
		if res.Error != nil || res.RowsAffected == 0 { return ErrNotFound }
		s.videos.IncrCommentCount(ctx, cm.VideoID, -int(res.RowsAffected))
		return nil
	})
}

func (s *CommentService) PinComment(ctx context.Context, videoID, commentID uint64) (bool, error) {
	if _, err := s.videos.GetPublishedVideo(ctx, videoID); err != nil { return false, ErrNotFound }
	var cm comment.Comment
	if err := s.db.WithContext(ctx).First(&cm, commentID).Error; err != nil { return false, ErrNotFound }
	if cm.VideoID != videoID { return false, ErrParamError }
	newPinned := !cm.Pinned
	if newPinned {
		_ = s.db.WithContext(ctx).Model(&comment.Comment{}).Where("video_id = ? AND pinned = ?", videoID, true).Update("pinned", false).Error
	}
	if err := s.db.WithContext(ctx).Model(&cm).Update("pinned", newPinned).Error; err != nil { return false, ErrInternalError }
	return newPinned, nil
}

func (s *CommentService) ToggleCommentLike(ctx context.Context, userID, commentID uint64) (bool, int, error) {
	var cm comment.Comment
	if err := s.db.WithContext(ctx).First(&cm, commentID).Error; err != nil { return false, 0, ErrNotFound }
	_ = s.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).Delete(&comment.CommentDislike{}).Error
	var existing comment.CommentLike
	if err := s.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).First(&existing).Error; err == nil {
		_ = s.db.WithContext(ctx).Delete(&existing).Error
		_ = s.db.WithContext(ctx).Model(&cm).UpdateColumn("like_count", gorm.Expr("GREATEST(like_count - 1, 0)")).Error
		var u comment.Comment; _ = s.db.WithContext(ctx).First(&u, commentID); return false, int(u.LikeCount), nil
	}
	if err := s.db.WithContext(ctx).Create(&comment.CommentLike{UserID: userID, CommentID: commentID}).Error; err != nil { return false, 0, ErrInternalError }
	_ = s.db.WithContext(ctx).Model(&cm).UpdateColumn("like_count", gorm.Expr("like_count + 1")).Error
	if s.notifSvc != nil { s.notifSvc.NotifyCommentLike(ctx, cm, userID) }
	var u comment.Comment; _ = s.db.WithContext(ctx).First(&u, commentID); return true, int(u.LikeCount), nil
}

func (s *CommentService) ToggleCommentDislike(ctx context.Context, userID, commentID uint64) (bool, error) {
	var cm comment.Comment
	if err := s.db.WithContext(ctx).First(&cm, commentID).Error; err != nil { return false, ErrNotFound }
	_ = s.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).Delete(&comment.CommentLike{}).Error
	var existing comment.CommentDislike
	if err := s.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).First(&existing).Error; err == nil {
		_ = s.db.WithContext(ctx).Delete(&existing).Error; return false, nil
	}
	if err := s.db.WithContext(ctx).Create(&comment.CommentDislike{UserID: userID, CommentID: commentID}).Error; err != nil { return false, ErrInternalError }
	return true, nil
}

func (s *CommentService) ApproveComment(ctx context.Context, commentID uint64) error {
	return s.db.WithContext(ctx).Model(&comment.Comment{}).Where("id = ?", commentID).Update("approved", true).Error
}

func (s *CommentService) IgnoreCuratedComment(ctx context.Context, commentID uint64) error {
	return s.db.WithContext(ctx).Model(&comment.Comment{}).Where("id = ?", commentID).Update("curated_ignored", true).Error
}
// GetCommentByID returns a comment by its ID.
func (s *CommentService) GetCommentByID(ctx context.Context, commentID uint64) (*comment.Comment, error) {
	var cm comment.Comment
	if err := s.db.WithContext(ctx).First(&cm, commentID).Error; err != nil {
		return nil, err
	}
	return &cm, nil
}

// GetDynamicCommentByID returns a dynamic comment by its ID.
func (s *CommentService) GetDynamicCommentByID(ctx context.Context, commentID uint64) (*comment.DynamicComment, error) {
	var cm comment.DynamicComment
	if err := s.db.WithContext(ctx).First(&cm, commentID).Error; err != nil {
		return nil, err
	}
	return &cm, nil
}

// ─── Article Comments ───

func (s *CommentService) ListArticleComments(ctx context.Context, articleID, viewerID uint64) (*ArticleCommentListResult, error) {
	a, err := s.articles.GetPublishedArticle(ctx, articleID)
	if err != nil { return nil, ErrNotFound }
	r := &ArticleCommentListResult{CommentsCurated: a.CommentsCurated, CommentsClosed: a.CommentsClosed}
	if a.CommentsClosed { r.Items = []ArticleCommentItem{}; return r, nil }

	q := s.db.WithContext(ctx).Where("article_id = ?", articleID)
	if a.CommentsCurated { q = q.Where("approved = ?", true) }
	var list []comment.ArticleComment
	if err := q.Order("id ASC").Find(&list).Error; err != nil { return nil, ErrInternalError }
	if len(list) == 0 { r.Items = []ArticleCommentItem{}; return r, nil }

	users, levels := s.loadArticleUsers(ctx, list)
	liked, disliked := s.loadArticleReactions(ctx, viewerID, list)
	out := make([]ArticleCommentItem, 0, len(list))
	for _, cm := range list {
		ulv := levels[cm.UserID]; if ulv < 1 { ulv = 1 }
		u := users[cm.UserID]
		out = append(out, ArticleCommentItem{
			ID: cm.ID, UserID: cm.UserID, Username: u.Nickname, AvatarURL: u.AvatarURL,
			ParentID: cm.ParentID, Level: cm.Level, UserLevel: ulv, Content: cm.Content,
			LikeCount: cm.LikeCount, CreatedAt: cm.CreatedAt.Format("2006-01-02 15:04:05"),
			LikedByMe: liked[cm.ID], DislikedByMe: disliked[cm.ID], Pinned: cm.Pinned,
			IsByAuthor: cm.UserID == a.UserID, IPLocation: cm.IpLocation,
		})
	}
	r.Items = out; return r, nil
}

func (s *CommentService) PostArticleComment(ctx context.Context, userID, articleID uint64, req PostCommentReq, ipLocation string) (*comment.ArticleComment, error) {
	_ = s.articles.IncrCommentCount(ctx, articleID, 1)
	content := req.Content
	if n := utf8.RuneCountInString(content); n < 1 || n > 1000 { return nil, ErrParamError }
	if s.sens != nil {
		if err := s.sens.Check(content); err != nil {
			if _, ok := err.(sensitive.ErrBlocked); ok { return nil, ErrCommentSensitive }
			return nil, ErrInternalError
		}
	}
	a, err := s.articles.GetPublishedArticle(ctx, articleID)
	if err != nil { return nil, ErrNotFound }
	if a.CommentsClosed { return nil, ErrNotFound }

	var parentID uint64
	level := 1
	if req.ParentID != 0 {
		parentID = req.ParentID
		var parent comment.ArticleComment
		if err := s.db.WithContext(ctx).First(&parent, req.ParentID).Error; err != nil { return nil, ErrNotFound }
		if parent.ArticleID != articleID { return nil, ErrParamError }
		level = parent.Level + 1; if level > 3 { level = 3 }
	}
	cm := comment.ArticleComment{
		UserID: userID, ArticleID: articleID, ParentID: parentID, Content: content,
		Level: level, Approved: !a.CommentsCurated, IpLocation: ipLocation,
	}
	if err := s.db.WithContext(ctx).Create(&cm).Error; err != nil { return nil, ErrInternalError }
	if s.notifSvc != nil {
		if req.ParentID == 0 { s.notifSvc.NotifyArticleComment(ctx, a.UserID, userID, cm)
		} else { s.notifSvc.NotifyArticleCommentReply(ctx, articleID, userID, &cm, parentID) }
	}
	return &cm, nil
}

func (s *CommentService) DeleteArticleComment(ctx context.Context, userID, commentID uint64, isAuthor bool) error {
	var cm comment.ArticleComment
	if err := s.db.WithContext(ctx).First(&cm, commentID).Error; err != nil { return ErrNotFound }
	if !isAuthor && cm.UserID != userID { return ErrForbidden }
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		descIDs := s.collectArticleDescendantIDs(tx, commentID)
		allIDs := append([]uint64{commentID}, descIDs...)
		_ = tx.Where("comment_id IN ?", allIDs).Delete(&comment.ArticleCommentLike{}).Error
		_ = tx.Where("comment_id IN ?", allIDs).Delete(&comment.ArticleCommentDislike{}).Error
		res := tx.Where("id IN ?", allIDs).Delete(&comment.ArticleComment{})
		if res.Error != nil || res.RowsAffected == 0 { return ErrNotFound }
		s.articles.IncrCommentCount(ctx, cm.ArticleID, -int(res.RowsAffected))
		return nil
	})
}

func (s *CommentService) PinArticleComment(ctx context.Context, articleID, commentID uint64) (bool, error) {
	if _, err := s.articles.GetPublishedArticle(ctx, articleID); err != nil { return false, ErrNotFound }
	var cm comment.ArticleComment
	if err := s.db.WithContext(ctx).First(&cm, commentID).Error; err != nil { return false, ErrNotFound }
	if cm.ArticleID != articleID { return false, ErrParamError }
	newPinned := !cm.Pinned
	if newPinned {
		_ = s.db.WithContext(ctx).Model(&comment.ArticleComment{}).Where("article_id = ? AND pinned = ?", articleID, true).Update("pinned", false).Error
	}
	_ = s.db.WithContext(ctx).Model(&cm).Update("pinned", newPinned).Error; return newPinned, nil
}

func (s *CommentService) ToggleArticleCommentLike(ctx context.Context, userID, commentID uint64) (bool, int, error) {
	var cm comment.ArticleComment
	if err := s.db.WithContext(ctx).First(&cm, commentID).Error; err != nil { return false, 0, ErrNotFound }
	_ = s.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).Delete(&comment.ArticleCommentDislike{}).Error
	var existing comment.ArticleCommentLike
	if err := s.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).First(&existing).Error; err == nil {
		_ = s.db.WithContext(ctx).Delete(&existing).Error
		_ = s.db.WithContext(ctx).Model(&cm).UpdateColumn("like_count", gorm.Expr("GREATEST(like_count - 1, 0)")).Error
		var u comment.ArticleComment; _ = s.db.WithContext(ctx).First(&u, commentID); return false, int(u.LikeCount), nil
	}
	if err := s.db.WithContext(ctx).Create(&comment.ArticleCommentLike{UserID: userID, CommentID: commentID}).Error; err != nil { return false, 0, ErrInternalError }
	_ = s.db.WithContext(ctx).Model(&cm).UpdateColumn("like_count", gorm.Expr("like_count + 1")).Error
	var u comment.ArticleComment; _ = s.db.WithContext(ctx).First(&u, commentID); return true, int(u.LikeCount), nil
}

func (s *CommentService) ToggleArticleCommentDislike(ctx context.Context, userID, commentID uint64) (bool, error) {
	var cm comment.ArticleComment
	if err := s.db.WithContext(ctx).First(&cm, commentID).Error; err != nil { return false, ErrNotFound }
	_ = s.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).Delete(&comment.ArticleCommentLike{}).Error
	var existing comment.ArticleCommentDislike
	if err := s.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).First(&existing).Error; err == nil {
		_ = s.db.WithContext(ctx).Delete(&existing).Error; return false, nil
	}
	if err := s.db.WithContext(ctx).Create(&comment.ArticleCommentDislike{UserID: userID, CommentID: commentID}).Error; err != nil { return false, ErrInternalError }
	return true, nil
}

func (s *CommentService) ApproveArticleComment(ctx context.Context, commentID uint64) error {
	return s.db.WithContext(ctx).Model(&comment.ArticleComment{}).Where("id = ?", commentID).Update("approved", true).Error
}

func (s *CommentService) IgnoreArticleComment(ctx context.Context, commentID uint64) error {
	_ = s.db.WithContext(ctx).Where("comment_id = ?", commentID).Delete(&comment.ArticleCommentLike{}).Error
	_ = s.db.WithContext(ctx).Where("comment_id = ?", commentID).Delete(&comment.ArticleCommentDislike{}).Error
	return s.db.WithContext(ctx).Delete(&comment.ArticleComment{}, commentID).Error
}

// GetArticleComment fetches a single article comment by ID.
func (s *CommentService) GetArticleComment(ctx context.Context, commentID uint64) (*comment.ArticleComment, error) {
	var cm comment.ArticleComment
	if err := s.db.WithContext(ctx).First(&cm, commentID).Error; err != nil {
		return nil, err
	}
	return &cm, nil
}

// ─── Dynamic Comments ───

func (s *CommentService) ListDynamicComments(ctx context.Context, dynamicID, viewerID uint64) (*DynamicCommentListResult, error) {
	d, err := s.dynamics.GetPublishedDynamic(ctx, dynamicID)
	if err != nil { return nil, ErrNotFound }
	r := &DynamicCommentListResult{CommentsCurated: d.CommentsCurated, CommentsClosed: d.CommentsClosed}
	if d.CommentsClosed { r.Items = []DynamicCommentItem{}; return r, nil }

	q := s.db.WithContext(ctx).Where("dynamic_id = ?", dynamicID)
	if d.CommentsCurated { q = q.Where("approved = ?", true) }
	var list []comment.DynamicComment
	if err := q.Order("id ASC").Find(&list).Error; err != nil { return nil, ErrInternalError }
	if len(list) == 0 { r.Items = []DynamicCommentItem{}; return r, nil }

	users, levels := s.loadDynamicUsers(ctx, list)
	out := make([]DynamicCommentItem, 0, len(list))
	for _, cm := range list {
		ulv := levels[cm.UserID]; if ulv < 1 { ulv = 1 }
		u := users[cm.UserID]
		out = append(out, DynamicCommentItem{
			ID: cm.ID, UserID: cm.UserID, Username: u.Nickname, AvatarURL: u.AvatarURL,
			ParentID: cm.ParentID, Level: cm.Level, UserLevel: ulv, Content: cm.Content,
			LikeCount: cm.LikeCount, CreatedAt: cm.CreatedAt.Format("2006-01-02 15:04:05"),
			IPLocation: cm.IpLocation, IsByUploader: cm.UserID == d.UserID,
		})
	}
	r.Items = out; return r, nil
}

func (s *CommentService) PostDynamicComment(ctx context.Context, userID, dynamicID uint64, req PostCommentReq, ipLocation string) (*comment.DynamicComment, error) {
	_ = s.dynamics.IncrCommentCount(ctx, dynamicID, 1)
	content := req.Content
	if n := utf8.RuneCountInString(content); n < 1 || n > 1000 { return nil, ErrParamError }
	if s.sens != nil {
		if err := s.sens.Check(content); err != nil {
			if _, ok := err.(sensitive.ErrBlocked); ok { return nil, ErrCommentSensitive }
			return nil, ErrInternalError
		}
	}
	d, err := s.dynamics.GetPublishedDynamic(ctx, dynamicID)
	if err != nil { return nil, ErrNotFound }
	if d.CommentsClosed { return nil, ErrCommentsClosed }

	var parentID uint64
	level := 1
	if req.ParentID != 0 {
		parentID = req.ParentID
		var parent comment.DynamicComment
		if err := s.db.WithContext(ctx).First(&parent, req.ParentID).Error; err != nil { return nil, ErrNotFound }
		if parent.DynamicID != dynamicID { return nil, ErrParamError }
		level = parent.Level + 1; if level > 3 { level = 3 }
	}
	cm := comment.DynamicComment{
		UserID: userID, DynamicID: dynamicID, ParentID: parentID, Content: content,
		Level: level, Approved: !d.CommentsCurated, IpLocation: ipLocation,
	}
	if err := s.db.WithContext(ctx).Create(&cm).Error; err != nil { return nil, ErrInternalError }
	return &cm, nil
}

func (s *CommentService) DeleteDynamicComment(ctx context.Context, userID, commentID uint64, isUploader bool) error {
	var cm comment.DynamicComment
	if err := s.db.WithContext(ctx).First(&cm, commentID).Error; err != nil { return ErrNotFound }
	if !isUploader && cm.UserID != userID { return ErrForbidden }
	_ = s.db.WithContext(ctx).Where("comment_id = ?", commentID).Delete(&comment.DynamicCommentLike{}).Error
	_ = s.db.WithContext(ctx).Where("comment_id = ?", commentID).Delete(&comment.DynamicCommentDislike{}).Error
	return s.db.WithContext(ctx).Delete(&cm).Error
}

func (s *CommentService) ToggleDynamicCommentReaction(ctx context.Context, userID, commentID uint64, like bool) (bool, int, error) {
	var cm comment.DynamicComment
	if err := s.db.WithContext(ctx).First(&cm, commentID).Error; err != nil { return false, 0, ErrNotFound }
	if like {
		_ = s.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).Delete(&comment.DynamicCommentDislike{}).Error
		var existing comment.DynamicCommentLike
		if err := s.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).First(&existing).Error; err == nil {
			_ = s.db.WithContext(ctx).Delete(&existing).Error
			_ = s.db.WithContext(ctx).Model(&cm).UpdateColumn("like_count", gorm.Expr("GREATEST(like_count - 1, 0)")).Error
			var u comment.DynamicComment; _ = s.db.WithContext(ctx).First(&u, commentID); return false, int(u.LikeCount), nil
		}
		if err := s.db.WithContext(ctx).Create(&comment.DynamicCommentLike{UserID: userID, CommentID: commentID}).Error; err != nil { return false, 0, ErrInternalError }
		_ = s.db.WithContext(ctx).Model(&cm).UpdateColumn("like_count", gorm.Expr("like_count + 1")).Error
	} else {
		_ = s.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).Delete(&comment.DynamicCommentLike{}).Error
		var existing comment.DynamicCommentDislike
		if err := s.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).First(&existing).Error; err == nil {
			_ = s.db.WithContext(ctx).Delete(&existing).Error
			var u comment.DynamicComment; _ = s.db.WithContext(ctx).First(&u, commentID); return false, int(u.LikeCount), nil
		}
		if err := s.db.WithContext(ctx).Create(&comment.DynamicCommentDislike{UserID: userID, CommentID: commentID}).Error; err != nil { return false, 0, ErrInternalError }
	}
	var u comment.DynamicComment; _ = s.db.WithContext(ctx).First(&u, commentID); return true, int(u.LikeCount), nil
}

func (s *CommentService) ApproveDynComment(ctx context.Context, commentID uint64) error {
	return s.db.WithContext(ctx).Model(&comment.DynamicComment{}).Where("id = ?", commentID).Update("approved", true).Error
}

func (s *CommentService) IgnoreDynComment(ctx context.Context, commentID uint64) error {
	_ = s.db.WithContext(ctx).Where("comment_id = ?", commentID).Delete(&comment.DynamicCommentLike{}).Error
	_ = s.db.WithContext(ctx).Where("comment_id = ?", commentID).Delete(&comment.DynamicCommentDislike{}).Error
	return s.db.WithContext(ctx).Delete(&comment.DynamicComment{}, commentID).Error
}
// ─── Internal helpers ───

func (s *CommentService) loadUsersWithLevels(ctx context.Context, comments []comment.Comment) (map[uint64]UserInfo, map[uint64]int) {
	if len(comments) == 0 { return nil, nil }
	uids := uniqueUint64(extractCommentUIDs(comments))
	users, _ := s.users.GetUsersByIDs(ctx, uids)
	levels, _ := s.users.BatchCurrentLevels(ctx, uids)
	return users, levels
}

func (s *CommentService) loadArticleUsers(ctx context.Context, comments []comment.ArticleComment) (map[uint64]UserInfo, map[uint64]int) {
	if len(comments) == 0 { return nil, nil }
	uids := uniqueUint64(extractArticleUIDs(comments))
	users, _ := s.users.GetUsersByIDs(ctx, uids)
	levels, _ := s.users.BatchCurrentLevels(ctx, uids)
	return users, levels
}

func (s *CommentService) loadDynamicUsers(ctx context.Context, comments []comment.DynamicComment) (map[uint64]UserInfo, map[uint64]int) {
	if len(comments) == 0 { return nil, nil }
	uids := uniqueUint64(extractDynamicUIDs(comments))
	users, _ := s.users.GetUsersByIDs(ctx, uids)
	levels, _ := s.users.BatchCurrentLevels(ctx, uids)
	return users, levels
}

func extractCommentUIDs(comments []comment.Comment) []uint64 {
	ids := make([]uint64, len(comments))
	for i, cm := range comments { ids[i] = cm.UserID }
	return ids
}

func extractArticleUIDs(comments []comment.ArticleComment) []uint64 {
	ids := make([]uint64, len(comments))
	for i, cm := range comments { ids[i] = cm.UserID }
	return ids
}

func extractDynamicUIDs(comments []comment.DynamicComment) []uint64 {
	ids := make([]uint64, len(comments))
	for i, cm := range comments { ids[i] = cm.UserID }
	return ids
}

func uniqueUint64(ids []uint64) []uint64 {
	seen := make(map[uint64]struct{}, len(ids))
	out := make([]uint64, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; !ok { seen[id] = struct{}{}; out = append(out, id) }
	}
	return out
}

func (s *CommentService) loadCommentLikes(ctx context.Context, viewerID uint64, comments []comment.Comment) map[uint64]bool {
	result := make(map[uint64]bool)
	if viewerID == 0 || len(comments) == 0 { return result }
	ids := make([]uint64, len(comments))
	for i := range comments { ids[i] = comments[i].ID }
	var likes []comment.CommentLike
	_ = s.db.WithContext(ctx).Where("user_id = ? AND comment_id IN ?", viewerID, ids).Find(&likes).Error
	for _, lk := range likes { result[lk.CommentID] = true }
	return result
}

func (s *CommentService) loadCommentDislikes(ctx context.Context, viewerID uint64, comments []comment.Comment) map[uint64]bool {
	result := make(map[uint64]bool)
	if viewerID == 0 || len(comments) == 0 { return result }
	ids := make([]uint64, len(comments))
	for i := range comments { ids[i] = comments[i].ID }
	var dislikes []comment.CommentDislike
	_ = s.db.WithContext(ctx).Where("user_id = ? AND comment_id IN ?", viewerID, ids).Find(&dislikes).Error
	for _, dk := range dislikes { result[dk.CommentID] = true }
	return result
}

func (s *CommentService) loadArticleReactions(ctx context.Context, viewerID uint64, comments []comment.ArticleComment) (map[uint64]bool, map[uint64]bool) {
	liked := make(map[uint64]bool)
	disliked := make(map[uint64]bool)
	if viewerID == 0 || len(comments) == 0 { return liked, disliked }
	ids := make([]uint64, len(comments))
	for i := range comments { ids[i] = comments[i].ID }
	var likes []comment.ArticleCommentLike
	_ = s.db.WithContext(ctx).Where("user_id = ? AND comment_id IN ?", viewerID, ids).Find(&likes).Error
	for _, lk := range likes { liked[lk.CommentID] = true }
	var dis []comment.ArticleCommentDislike
	_ = s.db.WithContext(ctx).Where("user_id = ? AND comment_id IN ?", viewerID, ids).Find(&dis).Error
	for _, dk := range dis { disliked[dk.CommentID] = true }
	return liked, disliked
}

func (s *CommentService) collectDescendantIDs(tx *gorm.DB, root uint64) []uint64 {
	var ids []uint64
	tx.Model(&comment.Comment{}).Where("parent_id = ?", root).Pluck("id", &ids)
	for _, id := range ids { ids = append(ids, s.collectDescendantIDs(tx, id)...) }
	return ids
}

func (s *CommentService) collectArticleDescendantIDs(tx *gorm.DB, root uint64) []uint64 {
	var ids []uint64
	tx.Model(&comment.ArticleComment{}).Where("parent_id = ?", root).Pluck("id", &ids)
	for _, id := range ids { ids = append(ids, s.collectArticleDescendantIDs(tx, id)...) }
	return ids
}