package comment

import (
	"cakecake/internal/model/comment"
	"cakecake/internal/service"
	"cakecake/internal/service/notification"
	vsvc "cakecake/internal/service/video"
	"context"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"cakecake/internal/pkg/sensitive"
)

// CommentService handles comment domain logic across videos, articles, and dynamics.
type CommentService struct {
	comments CommentProvider
	rdb      *redis.Client
	log      *zap.Logger
	sens     *sensitive.Filter
	notifSvc *notification.NotificationService

	// Domain providers (Phase 1: *gorm.DB impl; Phase 2+: gRPC clients)
	users    service.UserProvider
	videos   vsvc.VideoProvider
	articles service.ArticleProvider
	dynamics service.DynamicProvider
}

// NewCommentService creates a CommentService with the given storage, cache,
// content filter, notification callback, and cross-domain providers.
func NewCommentService(db *gorm.DB, rdb *redis.Client, log *zap.Logger, sens *sensitive.Filter, notifSvc *notification.NotificationService, users service.UserProvider, videos vsvc.VideoProvider, articles service.ArticleProvider, dynamics service.DynamicProvider) *CommentService {
	return &CommentService{comments: NewCommentProvider(db), rdb: rdb, log: log, sens: sens, notifSvc: notifSvc, users: users, videos: videos, articles: articles, dynamics: dynamics}
}

// PostCommentReq carries the fields for creating a comment.
type PostCommentReq struct {
	Content  string
	ParentID uint64
}

// CommentItem is the unified comment DTO returned to clients.
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

// CommentListResult is a comment list with media curation state.
type CommentListResult struct {
	Items           []CommentItem
	CommentsCurated bool
	CommentsClosed  bool
}

// ListComments lists comments on a video with viewer interaction flags.
func (s *CommentService) ListComments(ctx context.Context, videoID, viewerID uint64) (*CommentListResult, error) {
	v, err := s.videos.GetPublishedVideo(ctx, videoID)
	if err != nil {
		return nil, service.ErrNotFound
	}
	return s.buildCommentList(ctx, videoID, viewerID, v.UserID, v.CommentsCurated, v.CommentsClosed,
		func(ctx context.Context, targetID uint64, curated bool) ([]commentListRow, error) {
			return s.comments.ListComments(ctx, CommentVideo, targetID, curated)
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

// PostComment creates a video comment (or reply) with validation, curation,
// count increments, and notifications.
func (s *CommentService) PostComment(ctx context.Context, userID, videoID uint64, req PostCommentReq, ipLocation string) (*comment.Comment, error) {
	content := req.Content
	if err := s.validateCommentContent(content); err != nil {
		return nil, err
	}
	v, err := s.videos.GetPublishedVideo(ctx, videoID)
	if err != nil {
		return nil, service.ErrNotFound
	}
	if v.CommentsClosed {
		return nil, service.ErrCommentsClosed
	}

	parentID, level, err := s.resolveCommentParent(ctx, req.ParentID, videoID, func(ctx context.Context, parentID uint64) (int, bool, error) {
		targetID, level, err := s.comments.GetCommentParent(ctx, CommentVideo, parentID)
		if err != nil {
			return 0, false, err
		}
		if targetID != videoID {
			return 0, false, nil
		}
		return level + 1, true, nil
	})
	if err != nil {
		return nil, err
	}

	cm := comment.Comment{
		UserID: userID, VideoID: videoID, ParentID: parentID, Content: content,
		Level: level, Approved: !v.CommentsCurated, IpLocation: ipLocation,
	}
	if err := s.comments.CreateVideoComment(ctx, &cm); err != nil {
		return nil, service.ErrInternalError
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

// DeleteComment deletes a video comment (cascading replies) with ownership checks.
func (s *CommentService) DeleteComment(ctx context.Context, userID, commentID uint64, isUploader bool) error {
	return s.deleteCommentGeneric(ctx, userID, commentID, isUploader, commentDeleteAdapter{
		fetch: func(ctx context.Context, id uint64) (uint64, uint64, error) {
			return s.comments.GetCommentForDelete(ctx, CommentVideo, id)
		},
		cascade: true,
		descendants: func(tx *gorm.DB, root uint64) []uint64 {
			return s.comments.CollectCommentDescendants(tx, CommentVideo, root)
		},
		deleteLikes: func(tx *gorm.DB, ids []uint64) error {
			return s.comments.DeleteCommentLikesTx(tx, CommentVideo, ids)
		},
		deleteDislikes: func(tx *gorm.DB, ids []uint64) error {
			return s.comments.DeleteCommentDislikesTx(tx, CommentVideo, ids)
		},
		deleteRows: func(tx *gorm.DB, ids []uint64) (int64, error) {
			return s.comments.DeleteCommentRowsTx(tx, CommentVideo, ids)
		},
		incrCount: func(ctx context.Context, targetID uint64, delta int) {
			if err := s.videos.IncrCommentCount(ctx, targetID, delta); err != nil && s.log != nil {
				s.log.Warn("incr video comment count failed", zap.Uint64("video_id", targetID), zap.Int("delta", delta), zap.Error(err))
			}
		},
	})
}

// PinComment pins or unpins a video comment (returning the new state).
func (s *CommentService) PinComment(ctx context.Context, videoID, commentID uint64) (bool, error) {
	return s.pinCommentGeneric(ctx, videoID, commentID, commentPinAdapter{
		checkTarget: func(ctx context.Context, targetID uint64) error {
			_, err := s.videos.GetPublishedVideo(ctx, targetID)
			return err
		},
		fetch: func(ctx context.Context, id uint64) (uint64, bool, error) {
			return s.comments.GetCommentPin(ctx, CommentVideo, id)
		},
		unpinOthers: func(ctx context.Context, targetID uint64) error {
			return s.comments.UnpinComments(ctx, CommentVideo, targetID)
		},
		update: func(ctx context.Context, id uint64, pinned bool) error {
			return s.comments.UpdateCommentPinned(ctx, CommentVideo, id, pinned)
		},
	})
}

// ToggleCommentLike toggles a video comment like (returning the new state and like count).
func (s *CommentService) ToggleCommentLike(ctx context.Context, userID, commentID uint64) (bool, int, error) {
	return s.toggleCommentLikeGeneric(ctx, userID, commentID, s.videoReactionAdapter())
}

// ToggleCommentDislike toggles a video comment dislike (returning the new state).
func (s *CommentService) ToggleCommentDislike(ctx context.Context, userID, commentID uint64) (bool, error) {
	return s.toggleCommentDislikeGeneric(ctx, userID, commentID, s.videoReactionAdapter())
}

// ApproveComment approves a video comment for public display.
func (s *CommentService) ApproveComment(ctx context.Context, commentID uint64) error {
	return s.comments.ApproveComment(ctx, CommentVideo, commentID)
}

// IgnoreCuratedComment marks a video comment as curated-ignored.
func (s *CommentService) IgnoreCuratedComment(ctx context.Context, commentID uint64) error {
	return s.comments.IgnoreComment(ctx, CommentVideo, commentID)
}

// GetCommentByID returns a comment by its ID.
func (s *CommentService) GetCommentByID(ctx context.Context, commentID uint64) (*comment.Comment, error) {
	return s.comments.GetVideoComment(ctx, commentID)
}

// GetDynamicCommentByID returns a dynamic comment by its ID.
func (s *CommentService) GetDynamicCommentByID(ctx context.Context, commentID uint64) (*comment.DynamicComment, error) {
	return s.comments.GetDynamicComment(ctx, commentID)
}

// ─── Article Comments ───

// ListArticleComments lists comments on an article with viewer interaction flags.
func (s *CommentService) ListArticleComments(ctx context.Context, articleID, viewerID uint64) (*CommentListResult, error) {
	a, err := s.articles.GetPublishedArticle(ctx, articleID)
	if err != nil {
		return nil, service.ErrNotFound
	}
	return s.buildCommentList(ctx, articleID, viewerID, a.UserID, a.CommentsCurated, a.CommentsClosed,
		func(ctx context.Context, targetID uint64, curated bool) ([]commentListRow, error) {
			return s.comments.ListComments(ctx, CommentArticle, targetID, curated)
		},
		func(ctx context.Context, viewerID uint64, ids []uint64) (map[uint64]bool, map[uint64]bool, error) {
			return s.loadArticleReactionsByIDs(ctx, viewerID, ids)
		})
}

// PostArticleComment creates an article comment (or reply) with validation and notifications.
func (s *CommentService) PostArticleComment(ctx context.Context, userID, articleID uint64, req PostCommentReq, ipLocation string) (*comment.ArticleComment, error) {
	_ = s.articles.IncrCommentCount(ctx, articleID, 1)
	content := req.Content
	if err := s.validateCommentContent(content); err != nil {
		return nil, err
	}
	a, err := s.articles.GetPublishedArticle(ctx, articleID)
	if err != nil {
		return nil, service.ErrNotFound
	}
	if a.CommentsClosed {
		return nil, service.ErrCommentsClosed
	}

	parentID, level, err := s.resolveCommentParent(ctx, req.ParentID, articleID, func(ctx context.Context, parentID uint64) (int, bool, error) {
		targetID, level, err := s.comments.GetCommentParent(ctx, CommentArticle, parentID)
		if err != nil {
			return 0, false, err
		}
		if targetID != articleID {
			return 0, false, nil
		}
		return level + 1, true, nil
	})
	if err != nil {
		return nil, err
	}
	cm := comment.ArticleComment{
		UserID: userID, ArticleID: articleID, ParentID: parentID, Content: content,
		Level: level, Approved: !a.CommentsCurated, IpLocation: ipLocation,
	}
	if err := s.comments.CreateArticleComment(ctx, &cm); err != nil {
		return nil, service.ErrInternalError
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

// DeleteArticleComment deletes an article comment (cascading replies) with ownership checks.
func (s *CommentService) DeleteArticleComment(ctx context.Context, userID, commentID uint64, isAuthor bool) error {
	return s.deleteCommentGeneric(ctx, userID, commentID, isAuthor, commentDeleteAdapter{
		fetch: func(ctx context.Context, id uint64) (uint64, uint64, error) {
			return s.comments.GetCommentForDelete(ctx, CommentArticle, id)
		},
		cascade: true,
		descendants: func(tx *gorm.DB, root uint64) []uint64 {
			return s.comments.CollectCommentDescendants(tx, CommentArticle, root)
		},
		deleteLikes: func(tx *gorm.DB, ids []uint64) error {
			return s.comments.DeleteCommentLikesTx(tx, CommentArticle, ids)
		},
		deleteDislikes: func(tx *gorm.DB, ids []uint64) error {
			return s.comments.DeleteCommentDislikesTx(tx, CommentArticle, ids)
		},
		deleteRows: func(tx *gorm.DB, ids []uint64) (int64, error) {
			return s.comments.DeleteCommentRowsTx(tx, CommentArticle, ids)
		},
		incrCount: func(ctx context.Context, targetID uint64, delta int) {
			if err := s.articles.IncrCommentCount(ctx, targetID, delta); err != nil && s.log != nil {
				s.log.Warn("incr article comment count failed", zap.Uint64("article_id", targetID), zap.Int("delta", delta), zap.Error(err))
			}
		},
	})
}

// PinArticleComment pins or unpins an article comment (returning the new state).
func (s *CommentService) PinArticleComment(ctx context.Context, articleID, commentID uint64) (bool, error) {
	return s.pinCommentGeneric(ctx, articleID, commentID, commentPinAdapter{
		checkTarget: func(ctx context.Context, targetID uint64) error {
			_, err := s.articles.GetPublishedArticle(ctx, targetID)
			return err
		},
		fetch: func(ctx context.Context, id uint64) (uint64, bool, error) {
			return s.comments.GetCommentPin(ctx, CommentArticle, id)
		},
		unpinOthers: func(ctx context.Context, targetID uint64) error {
			return s.comments.UnpinComments(ctx, CommentArticle, targetID)
		},
		update: func(ctx context.Context, id uint64, pinned bool) error {
			return s.comments.UpdateCommentPinned(ctx, CommentArticle, id, pinned)
		},
	})
}

// ToggleArticleCommentLike toggles an article comment like (returning the new state and like count).
func (s *CommentService) ToggleArticleCommentLike(ctx context.Context, userID, commentID uint64) (bool, int, error) {
	return s.toggleCommentLikeGeneric(ctx, userID, commentID, s.articleReactionAdapter())
}

// ToggleArticleCommentDislike toggles an article comment dislike (returning the new state).
func (s *CommentService) ToggleArticleCommentDislike(ctx context.Context, userID, commentID uint64) (bool, error) {
	return s.toggleCommentDislikeGeneric(ctx, userID, commentID, s.articleReactionAdapter())
}

// ApproveArticleComment approves an article comment for public display.
func (s *CommentService) ApproveArticleComment(ctx context.Context, commentID uint64) error {
	return s.comments.ApproveComment(ctx, CommentArticle, commentID)
}

// IgnoreArticleComment marks an article comment as curated-ignored.
func (s *CommentService) IgnoreArticleComment(ctx context.Context, commentID uint64) error {
	return s.comments.IgnoreComment(ctx, CommentArticle, commentID)
}

// GetArticleComment fetches a single article comment by ID.
func (s *CommentService) GetArticleComment(ctx context.Context, commentID uint64) (*comment.ArticleComment, error) {
	return s.comments.GetArticleComment(ctx, commentID)
}

// ─── Dynamic Comments ───

// ListDynamicComments lists comments on a dynamic with viewer interaction flags.
func (s *CommentService) ListDynamicComments(ctx context.Context, dynamicID, viewerID uint64) (*CommentListResult, error) {
	d, err := s.dynamics.GetPublishedDynamic(ctx, dynamicID)
	if err != nil {
		return nil, service.ErrNotFound
	}
	return s.buildCommentList(ctx, dynamicID, viewerID, d.UserID, d.CommentsCurated, d.CommentsClosed,
		func(ctx context.Context, targetID uint64, curated bool) ([]commentListRow, error) {
			return s.comments.ListComments(ctx, CommentDynamic, targetID, curated)
		},
		nil)
}

// PostDynamicComment creates a dynamic comment (or reply) with validation and notifications.
func (s *CommentService) PostDynamicComment(ctx context.Context, userID, dynamicID uint64, req PostCommentReq, ipLocation string) (*comment.DynamicComment, error) {
	_ = s.dynamics.IncrCommentCount(ctx, dynamicID, 1)
	content := req.Content
	if err := s.validateCommentContent(content); err != nil {
		return nil, err
	}
	d, err := s.dynamics.GetPublishedDynamic(ctx, dynamicID)
	if err != nil {
		return nil, service.ErrNotFound
	}
	if d.CommentsClosed {
		return nil, service.ErrCommentsClosed
	}

	parentID, level, err := s.resolveCommentParent(ctx, req.ParentID, dynamicID, func(ctx context.Context, parentID uint64) (int, bool, error) {
		targetID, level, err := s.comments.GetCommentParent(ctx, CommentDynamic, parentID)
		if err != nil {
			return 0, false, err
		}
		if targetID != dynamicID {
			return 0, false, nil
		}
		return level + 1, true, nil
	})
	if err != nil {
		return nil, err
	}
	cm := comment.DynamicComment{
		UserID: userID, DynamicID: dynamicID, ParentID: parentID, Content: content,
		Level: level, Approved: !d.CommentsCurated, IpLocation: ipLocation,
	}
	if err := s.comments.CreateDynamicComment(ctx, &cm); err != nil {
		return nil, service.ErrInternalError
	}
	return &cm, nil
}

// DeleteDynamicComment deletes a dynamic comment (cascading replies) with ownership checks.
func (s *CommentService) DeleteDynamicComment(ctx context.Context, userID, commentID uint64, isUploader bool) error {
	return s.deleteCommentGeneric(ctx, userID, commentID, isUploader, commentDeleteAdapter{
		fetch: func(ctx context.Context, id uint64) (uint64, uint64, error) {
			return s.comments.GetCommentForDelete(ctx, CommentDynamic, id)
		},
		cascade: true,
		descendants: func(tx *gorm.DB, root uint64) []uint64 {
			return s.comments.CollectCommentDescendants(tx, CommentDynamic, root)
		},
		deleteLikes: func(tx *gorm.DB, ids []uint64) error {
			return s.comments.DeleteCommentLikesTx(tx, CommentDynamic, ids)
		},
		deleteDislikes: func(tx *gorm.DB, ids []uint64) error {
			return s.comments.DeleteCommentDislikesTx(tx, CommentDynamic, ids)
		},
		deleteRows: func(tx *gorm.DB, ids []uint64) (int64, error) {
			return s.comments.DeleteCommentRowsTx(tx, CommentDynamic, ids)
		},
		incrCount: func(ctx context.Context, targetID uint64, delta int) {
			if err := s.dynamics.IncrCommentCount(ctx, targetID, delta); err != nil && s.log != nil {
				s.log.Warn("incr dynamic comment count failed", zap.Uint64("dynamic_id", targetID), zap.Int("delta", delta), zap.Error(err))
			}
		},
	})
}

// ToggleDynamicCommentReaction toggles a dynamic comment like/dislike (returning the new state and like count).
func (s *CommentService) ToggleDynamicCommentReaction(ctx context.Context, userID, commentID uint64, like bool) (bool, int, error) {
	ad := s.dynamicReactionAdapter()
	if like {
		return s.toggleCommentLikeGeneric(ctx, userID, commentID, ad)
	}
	b, err := s.toggleCommentDislikeGeneric(ctx, userID, commentID, ad)
	if err != nil {
		return false, 0, err
	}
	return b, int(ad.count(ctx, commentID)), nil
}

// ApproveDynComment approves a dynamic comment for public display.
func (s *CommentService) ApproveDynComment(ctx context.Context, commentID uint64) error {
	return s.comments.ApproveComment(ctx, CommentDynamic, commentID)
}

// IgnoreDynComment marks a dynamic comment as curated-ignored.
func (s *CommentService) IgnoreDynComment(ctx context.Context, commentID uint64) error {
	return s.comments.IgnoreComment(ctx, CommentDynamic, commentID)
}

// ─── Internal helpers ───
