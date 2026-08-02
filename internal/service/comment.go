package service

import (
	"cakecake/internal/model/comment"
	"context"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"cakecake/internal/pkg/sensitive"
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

func NewCommentService(db *gorm.DB, rdb *redis.Client, log *zap.Logger, sens *sensitive.Filter, notifSvc *NotificationService, users UserProvider, videos VideoProvider, articles ArticleProvider, dynamics DynamicProvider) *CommentService {
	return &CommentService{db: db, rdb: rdb, log: log, sens: sens, notifSvc: notifSvc, users: users, videos: videos, articles: articles, dynamics: dynamics}
}

type PostCommentReq struct {
	Content  string
	ParentID uint64
}

type CommentItem struct {
	ID, UserID                                    uint64
	Username, AvatarURL                           string
	ParentID                                      uint64
	Level, UserLevel                              int
	Content                                       string
	LikeCount                                     uint64
	CreatedAt                                     string
	LikedByMe, DislikedByMe, Pinned, IsByUploader bool
	IPLocation                                    string
}

type CommentListResult struct {
	Items           []CommentItem
	CommentsCurated bool
	CommentsClosed  bool
}

func (s *CommentService) ListComments(ctx context.Context, videoID, viewerID uint64) (*CommentListResult, error) {
	v, err := s.videos.GetPublishedVideo(ctx, videoID)
	if err != nil {
		return nil, ErrNotFound
	}
	return s.buildCommentList(ctx, videoID, viewerID, v.UserID, v.CommentsCurated, v.CommentsClosed,
		func(ctx context.Context, targetID uint64, curated bool) ([]commentListRow, error) {
			q := s.db.WithContext(ctx).Where("video_id = ?", targetID)
			if curated {
				q = q.Where("approved = ?", true)
			}
			var list []comment.Comment
			if err := q.Order("id ASC").Find(&list).Error; err != nil {
				return nil, err
			}
			return toCommentRows(list), nil
		},
		func(ctx context.Context, viewerID uint64, ids []uint64) (map[uint64]bool, map[uint64]bool, error) {
			liked, err := s.loadCommentLikesByIDs(ctx, viewerID, ids)
			if err != nil {
				return nil, nil, err
			}
			disliked, err := s.loadCommentDislikesByIDs(ctx, viewerID, ids)
			if err != nil {
				return nil, nil, err
			}
			return liked, disliked, nil
		})
}

func (s *CommentService) PostComment(ctx context.Context, userID, videoID uint64, req PostCommentReq, ipLocation string) (*comment.Comment, error) {
	content := req.Content
	if err := s.validateCommentContent(content); err != nil {
		return nil, err
	}
	v, err := s.videos.GetPublishedVideo(ctx, videoID)
	if err != nil {
		return nil, ErrNotFound
	}
	if v.CommentsClosed {
		return nil, ErrCommentsClosed
	}

	parentID, level, err := s.resolveCommentParent(ctx, req.ParentID, videoID, func(ctx context.Context, parentID uint64) (int, bool, error) {
		var parent comment.Comment
		if err := s.db.WithContext(ctx).First(&parent, parentID).Error; err != nil {
			return 0, false, err
		}
		if parent.VideoID != videoID {
			return 0, false, nil
		}
		return parent.Level + 1, true, nil
	})
	if err != nil {
		return nil, err
	}

	cm := comment.Comment{
		UserID: userID, VideoID: videoID, ParentID: parentID, Content: content,
		Level: level, Approved: !v.CommentsCurated, IpLocation: ipLocation,
	}
	if err := s.db.WithContext(ctx).Create(&cm).Error; err != nil {
		return nil, ErrInternalError
	}
	_ = s.videos.IncrCommentCount(ctx, videoID, 1)
	if s.notifSvc != nil {
		if req.ParentID == 0 {
			s.notifSvc.NotifyVideoComment(ctx, v.UserID, userID, cm)
		} else {
			s.notifSvc.NotifyCommentReply(ctx, videoID, userID, &cm, parentID)
		}
	}
	return &cm, nil
}

func (s *CommentService) DeleteComment(ctx context.Context, userID, commentID uint64, isUploader bool) error {
	var cm comment.Comment
	if err := s.db.WithContext(ctx).First(&cm, commentID).Error; err != nil {
		return ErrNotFound
	}
	if !isUploader && cm.UserID != userID {
		return ErrForbidden
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		descIDs := s.collectCommentDescendants(tx, func(tx *gorm.DB, parentID uint64) ([]uint64, error) {
			var ids []uint64
			err := tx.Model(&comment.Comment{}).Where("parent_id = ?", parentID).Pluck("id", &ids).Error
			return ids, err
		}, commentID)
		allIDs := append([]uint64{commentID}, descIDs...)
		_ = tx.Where("comment_id IN ?", allIDs).Delete(&comment.CommentLike{}).Error
		_ = tx.Where("comment_id IN ?", allIDs).Delete(&comment.CommentDislike{}).Error
		res := tx.Where("id IN ?", allIDs).Delete(&comment.Comment{})
		if res.Error != nil || res.RowsAffected == 0 {
			return ErrNotFound
		}
		s.videos.IncrCommentCount(ctx, cm.VideoID, -int(res.RowsAffected))
		return nil
	})
}

func (s *CommentService) PinComment(ctx context.Context, videoID, commentID uint64) (bool, error) {
	if _, err := s.videos.GetPublishedVideo(ctx, videoID); err != nil {
		return false, ErrNotFound
	}
	var cm comment.Comment
	if err := s.db.WithContext(ctx).First(&cm, commentID).Error; err != nil {
		return false, ErrNotFound
	}
	if cm.VideoID != videoID {
		return false, ErrParamError
	}
	newPinned := !cm.Pinned
	if newPinned {
		_ = s.db.WithContext(ctx).Model(&comment.Comment{}).Where("video_id = ? AND pinned = ?", videoID, true).Update("pinned", false).Error
	}
	if err := s.db.WithContext(ctx).Model(&cm).Update("pinned", newPinned).Error; err != nil {
		return false, ErrInternalError
	}
	return newPinned, nil
}

func (s *CommentService) ToggleCommentLike(ctx context.Context, userID, commentID uint64) (bool, int, error) {
	var cm comment.Comment
	if err := s.db.WithContext(ctx).First(&cm, commentID).Error; err != nil {
		return false, 0, ErrNotFound
	}
	_ = s.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).Delete(&comment.CommentDislike{}).Error
	var existing comment.CommentLike
	if err := s.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).First(&existing).Error; err == nil {
		_ = s.db.WithContext(ctx).Delete(&existing).Error
		_ = s.db.WithContext(ctx).Model(&cm).UpdateColumn("like_count", gorm.Expr("GREATEST(like_count - 1, 0)")).Error
		var u comment.Comment
		_ = s.db.WithContext(ctx).First(&u, commentID)
		return false, int(u.LikeCount), nil
	}
	if err := s.db.WithContext(ctx).Create(&comment.CommentLike{UserID: userID, CommentID: commentID}).Error; err != nil {
		return false, 0, ErrInternalError
	}
	_ = s.db.WithContext(ctx).Model(&cm).UpdateColumn("like_count", gorm.Expr("like_count + 1")).Error
	if s.notifSvc != nil {
		s.notifSvc.NotifyCommentLike(ctx, cm, userID)
	}
	var u comment.Comment
	_ = s.db.WithContext(ctx).First(&u, commentID)
	return true, int(u.LikeCount), nil
}

func (s *CommentService) ToggleCommentDislike(ctx context.Context, userID, commentID uint64) (bool, error) {
	var cm comment.Comment
	if err := s.db.WithContext(ctx).First(&cm, commentID).Error; err != nil {
		return false, ErrNotFound
	}
	_ = s.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).Delete(&comment.CommentLike{}).Error
	var existing comment.CommentDislike
	if err := s.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).First(&existing).Error; err == nil {
		_ = s.db.WithContext(ctx).Delete(&existing).Error
		return false, nil
	}
	if err := s.db.WithContext(ctx).Create(&comment.CommentDislike{UserID: userID, CommentID: commentID}).Error; err != nil {
		return false, ErrInternalError
	}
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

func (s *CommentService) ListArticleComments(ctx context.Context, articleID, viewerID uint64) (*CommentListResult, error) {
	a, err := s.articles.GetPublishedArticle(ctx, articleID)
	if err != nil {
		return nil, ErrNotFound
	}
	return s.buildCommentList(ctx, articleID, viewerID, a.UserID, a.CommentsCurated, a.CommentsClosed,
		func(ctx context.Context, targetID uint64, curated bool) ([]commentListRow, error) {
			q := s.db.WithContext(ctx).Where("article_id = ?", targetID)
			if curated {
				q = q.Where("approved = ?", true)
			}
			var list []comment.ArticleComment
			if err := q.Order("id ASC").Find(&list).Error; err != nil {
				return nil, err
			}
			return toArticleCommentRows(list), nil
		},
		func(ctx context.Context, viewerID uint64, ids []uint64) (map[uint64]bool, map[uint64]bool, error) {
			return s.loadArticleReactionsByIDs(ctx, viewerID, ids)
		})
}

func (s *CommentService) PostArticleComment(ctx context.Context, userID, articleID uint64, req PostCommentReq, ipLocation string) (*comment.ArticleComment, error) {
	_ = s.articles.IncrCommentCount(ctx, articleID, 1)
	content := req.Content
	if err := s.validateCommentContent(content); err != nil {
		return nil, err
	}
	a, err := s.articles.GetPublishedArticle(ctx, articleID)
	if err != nil {
		return nil, ErrNotFound
	}
	if a.CommentsClosed {
		return nil, ErrCommentsClosed
	}

	parentID, level, err := s.resolveCommentParent(ctx, req.ParentID, articleID, func(ctx context.Context, parentID uint64) (int, bool, error) {
		var parent comment.ArticleComment
		if err := s.db.WithContext(ctx).First(&parent, parentID).Error; err != nil {
			return 0, false, err
		}
		if parent.ArticleID != articleID {
			return 0, false, nil
		}
		return parent.Level + 1, true, nil
	})
	if err != nil {
		return nil, err
	}
	cm := comment.ArticleComment{
		UserID: userID, ArticleID: articleID, ParentID: parentID, Content: content,
		Level: level, Approved: !a.CommentsCurated, IpLocation: ipLocation,
	}
	if err := s.db.WithContext(ctx).Create(&cm).Error; err != nil {
		return nil, ErrInternalError
	}
	if s.notifSvc != nil {
		if req.ParentID == 0 {
			s.notifSvc.NotifyArticleComment(ctx, a.UserID, userID, cm)
		} else {
			s.notifSvc.NotifyArticleCommentReply(ctx, articleID, userID, &cm, parentID)
		}
	}
	return &cm, nil
}

func (s *CommentService) DeleteArticleComment(ctx context.Context, userID, commentID uint64, isAuthor bool) error {
	var cm comment.ArticleComment
	if err := s.db.WithContext(ctx).First(&cm, commentID).Error; err != nil {
		return ErrNotFound
	}
	if !isAuthor && cm.UserID != userID {
		return ErrForbidden
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		descIDs := s.collectCommentDescendants(tx, func(tx *gorm.DB, parentID uint64) ([]uint64, error) {
			var ids []uint64
			err := tx.Model(&comment.ArticleComment{}).Where("parent_id = ?", parentID).Pluck("id", &ids).Error
			return ids, err
		}, commentID)
		allIDs := append([]uint64{commentID}, descIDs...)
		_ = tx.Where("comment_id IN ?", allIDs).Delete(&comment.ArticleCommentLike{}).Error
		_ = tx.Where("comment_id IN ?", allIDs).Delete(&comment.ArticleCommentDislike{}).Error
		res := tx.Where("id IN ?", allIDs).Delete(&comment.ArticleComment{})
		if res.Error != nil || res.RowsAffected == 0 {
			return ErrNotFound
		}
		s.articles.IncrCommentCount(ctx, cm.ArticleID, -int(res.RowsAffected))
		return nil
	})
}

func (s *CommentService) PinArticleComment(ctx context.Context, articleID, commentID uint64) (bool, error) {
	if _, err := s.articles.GetPublishedArticle(ctx, articleID); err != nil {
		return false, ErrNotFound
	}
	var cm comment.ArticleComment
	if err := s.db.WithContext(ctx).First(&cm, commentID).Error; err != nil {
		return false, ErrNotFound
	}
	if cm.ArticleID != articleID {
		return false, ErrParamError
	}
	newPinned := !cm.Pinned
	if newPinned {
		_ = s.db.WithContext(ctx).Model(&comment.ArticleComment{}).Where("article_id = ? AND pinned = ?", articleID, true).Update("pinned", false).Error
	}
	_ = s.db.WithContext(ctx).Model(&cm).Update("pinned", newPinned).Error
	return newPinned, nil
}

func (s *CommentService) ToggleArticleCommentLike(ctx context.Context, userID, commentID uint64) (bool, int, error) {
	var cm comment.ArticleComment
	if err := s.db.WithContext(ctx).First(&cm, commentID).Error; err != nil {
		return false, 0, ErrNotFound
	}
	_ = s.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).Delete(&comment.ArticleCommentDislike{}).Error
	var existing comment.ArticleCommentLike
	if err := s.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).First(&existing).Error; err == nil {
		_ = s.db.WithContext(ctx).Delete(&existing).Error
		_ = s.db.WithContext(ctx).Model(&cm).UpdateColumn("like_count", gorm.Expr("GREATEST(like_count - 1, 0)")).Error
		var u comment.ArticleComment
		_ = s.db.WithContext(ctx).First(&u, commentID)
		return false, int(u.LikeCount), nil
	}
	if err := s.db.WithContext(ctx).Create(&comment.ArticleCommentLike{UserID: userID, CommentID: commentID}).Error; err != nil {
		return false, 0, ErrInternalError
	}
	_ = s.db.WithContext(ctx).Model(&cm).UpdateColumn("like_count", gorm.Expr("like_count + 1")).Error
	var u comment.ArticleComment
	_ = s.db.WithContext(ctx).First(&u, commentID)
	return true, int(u.LikeCount), nil
}

func (s *CommentService) ToggleArticleCommentDislike(ctx context.Context, userID, commentID uint64) (bool, error) {
	var cm comment.ArticleComment
	if err := s.db.WithContext(ctx).First(&cm, commentID).Error; err != nil {
		return false, ErrNotFound
	}
	_ = s.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).Delete(&comment.ArticleCommentLike{}).Error
	var existing comment.ArticleCommentDislike
	if err := s.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).First(&existing).Error; err == nil {
		_ = s.db.WithContext(ctx).Delete(&existing).Error
		return false, nil
	}
	if err := s.db.WithContext(ctx).Create(&comment.ArticleCommentDislike{UserID: userID, CommentID: commentID}).Error; err != nil {
		return false, ErrInternalError
	}
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

func (s *CommentService) ListDynamicComments(ctx context.Context, dynamicID, viewerID uint64) (*CommentListResult, error) {
	d, err := s.dynamics.GetPublishedDynamic(ctx, dynamicID)
	if err != nil {
		return nil, ErrNotFound
	}
	return s.buildCommentList(ctx, dynamicID, viewerID, d.UserID, d.CommentsCurated, d.CommentsClosed,
		func(ctx context.Context, targetID uint64, curated bool) ([]commentListRow, error) {
			q := s.db.WithContext(ctx).Where("dynamic_id = ?", targetID)
			if curated {
				q = q.Where("approved = ?", true)
			}
			var list []comment.DynamicComment
			if err := q.Order("id ASC").Find(&list).Error; err != nil {
				return nil, err
			}
			return toDynamicCommentRows(list), nil
		},
		nil)
}

func (s *CommentService) PostDynamicComment(ctx context.Context, userID, dynamicID uint64, req PostCommentReq, ipLocation string) (*comment.DynamicComment, error) {
	_ = s.dynamics.IncrCommentCount(ctx, dynamicID, 1)
	content := req.Content
	if err := s.validateCommentContent(content); err != nil {
		return nil, err
	}
	d, err := s.dynamics.GetPublishedDynamic(ctx, dynamicID)
	if err != nil {
		return nil, ErrNotFound
	}
	if d.CommentsClosed {
		return nil, ErrCommentsClosed
	}

	parentID, level, err := s.resolveCommentParent(ctx, req.ParentID, dynamicID, func(ctx context.Context, parentID uint64) (int, bool, error) {
		var parent comment.DynamicComment
		if err := s.db.WithContext(ctx).First(&parent, parentID).Error; err != nil {
			return 0, false, err
		}
		if parent.DynamicID != dynamicID {
			return 0, false, nil
		}
		return parent.Level + 1, true, nil
	})
	if err != nil {
		return nil, err
	}
	cm := comment.DynamicComment{
		UserID: userID, DynamicID: dynamicID, ParentID: parentID, Content: content,
		Level: level, Approved: !d.CommentsCurated, IpLocation: ipLocation,
	}
	if err := s.db.WithContext(ctx).Create(&cm).Error; err != nil {
		return nil, ErrInternalError
	}
	return &cm, nil
}

func (s *CommentService) DeleteDynamicComment(ctx context.Context, userID, commentID uint64, isUploader bool) error {
	var cm comment.DynamicComment
	if err := s.db.WithContext(ctx).First(&cm, commentID).Error; err != nil {
		return ErrNotFound
	}
	if !isUploader && cm.UserID != userID {
		return ErrForbidden
	}
	_ = s.db.WithContext(ctx).Where("comment_id = ?", commentID).Delete(&comment.DynamicCommentLike{}).Error
	_ = s.db.WithContext(ctx).Where("comment_id = ?", commentID).Delete(&comment.DynamicCommentDislike{}).Error
	return s.db.WithContext(ctx).Delete(&cm).Error
}

func (s *CommentService) ToggleDynamicCommentReaction(ctx context.Context, userID, commentID uint64, like bool) (bool, int, error) {
	var cm comment.DynamicComment
	if err := s.db.WithContext(ctx).First(&cm, commentID).Error; err != nil {
		return false, 0, ErrNotFound
	}
	if like {
		_ = s.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).Delete(&comment.DynamicCommentDislike{}).Error
		var existing comment.DynamicCommentLike
		if err := s.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).First(&existing).Error; err == nil {
			_ = s.db.WithContext(ctx).Delete(&existing).Error
			_ = s.db.WithContext(ctx).Model(&cm).UpdateColumn("like_count", gorm.Expr("GREATEST(like_count - 1, 0)")).Error
			var u comment.DynamicComment
			_ = s.db.WithContext(ctx).First(&u, commentID)
			return false, int(u.LikeCount), nil
		}
		if err := s.db.WithContext(ctx).Create(&comment.DynamicCommentLike{UserID: userID, CommentID: commentID}).Error; err != nil {
			return false, 0, ErrInternalError
		}
		_ = s.db.WithContext(ctx).Model(&cm).UpdateColumn("like_count", gorm.Expr("like_count + 1")).Error
	} else {
		_ = s.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).Delete(&comment.DynamicCommentLike{}).Error
		var existing comment.DynamicCommentDislike
		if err := s.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).First(&existing).Error; err == nil {
			_ = s.db.WithContext(ctx).Delete(&existing).Error
			var u comment.DynamicComment
			_ = s.db.WithContext(ctx).First(&u, commentID)
			return false, int(u.LikeCount), nil
		}
		if err := s.db.WithContext(ctx).Create(&comment.DynamicCommentDislike{UserID: userID, CommentID: commentID}).Error; err != nil {
			return false, 0, ErrInternalError
		}
	}
	var u comment.DynamicComment
	_ = s.db.WithContext(ctx).First(&u, commentID)
	return true, int(u.LikeCount), nil
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
