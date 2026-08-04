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

func NewCommentService(db *gorm.DB, rdb *redis.Client, log *zap.Logger, sens *sensitive.Filter, notifSvc *notification.NotificationService, users service.UserProvider, videos vsvc.VideoProvider, articles service.ArticleProvider, dynamics service.DynamicProvider) *CommentService {
	return &CommentService{comments: NewCommentProvider(db), rdb: rdb, log: log, sens: sens, notifSvc: notifSvc, users: users, videos: videos, articles: articles, dynamics: dynamics}
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
			s.videos.IncrCommentCount(ctx, targetID, delta)
		},
	})
}

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

func (s *CommentService) ToggleCommentLike(ctx context.Context, userID, commentID uint64) (bool, int, error) {
	return s.toggleCommentLikeGeneric(ctx, userID, commentID, s.videoReactionAdapter())
}

func (s *CommentService) ToggleCommentDislike(ctx context.Context, userID, commentID uint64) (bool, error) {
	return s.toggleCommentDislikeGeneric(ctx, userID, commentID, s.videoReactionAdapter())
}

func (s *CommentService) ApproveComment(ctx context.Context, commentID uint64) error {
	return s.comments.ApproveComment(ctx, CommentVideo, commentID)
}

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
			s.articles.IncrCommentCount(ctx, targetID, delta)
		},
	})
}

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

func (s *CommentService) ToggleArticleCommentLike(ctx context.Context, userID, commentID uint64) (bool, int, error) {
	return s.toggleCommentLikeGeneric(ctx, userID, commentID, s.articleReactionAdapter())
}

func (s *CommentService) ToggleArticleCommentDislike(ctx context.Context, userID, commentID uint64) (bool, error) {
	return s.toggleCommentDislikeGeneric(ctx, userID, commentID, s.articleReactionAdapter())
}

func (s *CommentService) ApproveArticleComment(ctx context.Context, commentID uint64) error {
	return s.comments.ApproveComment(ctx, CommentArticle, commentID)
}

func (s *CommentService) IgnoreArticleComment(ctx context.Context, commentID uint64) error {
	return s.comments.IgnoreComment(ctx, CommentArticle, commentID)
}

// GetArticleComment fetches a single article comment by ID.
func (s *CommentService) GetArticleComment(ctx context.Context, commentID uint64) (*comment.ArticleComment, error) {
	return s.comments.GetArticleComment(ctx, commentID)
}

// ─── Dynamic Comments ───

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
			s.dynamics.IncrCommentCount(ctx, targetID, delta)
		},
	})
}

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

func (s *CommentService) ApproveDynComment(ctx context.Context, commentID uint64) error {
	return s.comments.ApproveComment(ctx, CommentDynamic, commentID)
}

func (s *CommentService) IgnoreDynComment(ctx context.Context, commentID uint64) error {
	return s.comments.IgnoreComment(ctx, CommentDynamic, commentID)
}

// ─── Internal helpers ───
