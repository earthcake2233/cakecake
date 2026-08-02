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
	return s.deleteCommentGeneric(ctx, userID, commentID, isUploader, commentDeleteAdapter{
		fetch: func(ctx context.Context, id uint64) (uint64, uint64, error) {
			var cm comment.Comment
			if err := s.db.WithContext(ctx).First(&cm, id).Error; err != nil {
				return 0, 0, err
			}
			return cm.UserID, cm.VideoID, nil
		},
		cascade: true,
		descendants: func(tx *gorm.DB, root uint64) []uint64 {
			return s.collectCommentDescendants(tx, func(tx *gorm.DB, parentID uint64) ([]uint64, error) {
				var ids []uint64
				err := tx.Model(&comment.Comment{}).Where("parent_id = ?", parentID).Pluck("id", &ids).Error
				return ids, err
			}, root)
		},
		deleteLikes: func(tx *gorm.DB, ids []uint64) error {
			return tx.Where("comment_id IN ?", ids).Delete(&comment.CommentLike{}).Error
		},
		deleteDislikes: func(tx *gorm.DB, ids []uint64) error {
			return tx.Where("comment_id IN ?", ids).Delete(&comment.CommentDislike{}).Error
		},
		deleteRows: func(tx *gorm.DB, ids []uint64) (int64, error) {
			res := tx.Where("id IN ?", ids).Delete(&comment.Comment{})
			return res.RowsAffected, res.Error
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
			var cm comment.Comment
			if err := s.db.WithContext(ctx).First(&cm, id).Error; err != nil {
				return 0, false, err
			}
			return cm.VideoID, cm.Pinned, nil
		},
		unpinOthers: func(ctx context.Context, targetID uint64) error {
			return s.db.WithContext(ctx).Model(&comment.Comment{}).
				Where("video_id = ? AND pinned = ?", targetID, true).Update("pinned", false).Error
		},
		update: func(ctx context.Context, id uint64, pinned bool) error {
			return s.db.WithContext(ctx).Model(&comment.Comment{}).Where("id = ?", id).Update("pinned", pinned).Error
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
	return s.approveCommentGeneric(ctx, &comment.Comment{}, commentID)
}

func (s *CommentService) IgnoreCuratedComment(ctx context.Context, commentID uint64) error {
	return s.ignoreCommentGeneric(ctx, &comment.Comment{}, commentID)
}

// GetCommentByID returns a comment by its ID.
func (s *CommentService) GetCommentByID(ctx context.Context, commentID uint64) (*comment.Comment, error) {
	return getCommentGeneric[comment.Comment](s, ctx, commentID)
}

// GetDynamicCommentByID returns a dynamic comment by its ID.
func (s *CommentService) GetDynamicCommentByID(ctx context.Context, commentID uint64) (*comment.DynamicComment, error) {
	return getCommentGeneric[comment.DynamicComment](s, ctx, commentID)
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
	return s.deleteCommentGeneric(ctx, userID, commentID, isAuthor, commentDeleteAdapter{
		fetch: func(ctx context.Context, id uint64) (uint64, uint64, error) {
			var cm comment.ArticleComment
			if err := s.db.WithContext(ctx).First(&cm, id).Error; err != nil {
				return 0, 0, err
			}
			return cm.UserID, cm.ArticleID, nil
		},
		cascade: true,
		descendants: func(tx *gorm.DB, root uint64) []uint64 {
			return s.collectCommentDescendants(tx, func(tx *gorm.DB, parentID uint64) ([]uint64, error) {
				var ids []uint64
				err := tx.Model(&comment.ArticleComment{}).Where("parent_id = ?", parentID).Pluck("id", &ids).Error
				return ids, err
			}, root)
		},
		deleteLikes: func(tx *gorm.DB, ids []uint64) error {
			return tx.Where("comment_id IN ?", ids).Delete(&comment.ArticleCommentLike{}).Error
		},
		deleteDislikes: func(tx *gorm.DB, ids []uint64) error {
			return tx.Where("comment_id IN ?", ids).Delete(&comment.ArticleCommentDislike{}).Error
		},
		deleteRows: func(tx *gorm.DB, ids []uint64) (int64, error) {
			res := tx.Where("id IN ?", ids).Delete(&comment.ArticleComment{})
			return res.RowsAffected, res.Error
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
			var cm comment.ArticleComment
			if err := s.db.WithContext(ctx).First(&cm, id).Error; err != nil {
				return 0, false, err
			}
			return cm.ArticleID, cm.Pinned, nil
		},
		unpinOthers: func(ctx context.Context, targetID uint64) error {
			return s.db.WithContext(ctx).Model(&comment.ArticleComment{}).
				Where("article_id = ? AND pinned = ?", targetID, true).Update("pinned", false).Error
		},
		update: func(ctx context.Context, id uint64, pinned bool) error {
			return s.db.WithContext(ctx).Model(&comment.ArticleComment{}).Where("id = ?", id).Update("pinned", pinned).Error
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
	return s.approveCommentGeneric(ctx, &comment.ArticleComment{}, commentID)
}

func (s *CommentService) IgnoreArticleComment(ctx context.Context, commentID uint64) error {
	return s.ignoreCommentGeneric(ctx, &comment.ArticleComment{}, commentID)
}

// GetArticleComment fetches a single article comment by ID.
func (s *CommentService) GetArticleComment(ctx context.Context, commentID uint64) (*comment.ArticleComment, error) {
	return getCommentGeneric[comment.ArticleComment](s, ctx, commentID)
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
	return s.deleteCommentGeneric(ctx, userID, commentID, isUploader, commentDeleteAdapter{
		fetch: func(ctx context.Context, id uint64) (uint64, uint64, error) {
			var cm comment.DynamicComment
			if err := s.db.WithContext(ctx).First(&cm, id).Error; err != nil {
				return 0, 0, err
			}
			return cm.UserID, cm.DynamicID, nil
		},
		cascade: true,
		descendants: func(tx *gorm.DB, root uint64) []uint64 {
			return s.collectCommentDescendants(tx, func(tx *gorm.DB, parentID uint64) ([]uint64, error) {
				var ids []uint64
				err := tx.Model(&comment.DynamicComment{}).Where("parent_id = ?", parentID).Pluck("id", &ids).Error
				return ids, err
			}, root)
		},
		deleteLikes: func(tx *gorm.DB, ids []uint64) error {
			return tx.Where("comment_id IN ?", ids).Delete(&comment.DynamicCommentLike{}).Error
		},
		deleteDislikes: func(tx *gorm.DB, ids []uint64) error {
			return tx.Where("comment_id IN ?", ids).Delete(&comment.DynamicCommentDislike{}).Error
		},
		deleteRows: func(tx *gorm.DB, ids []uint64) (int64, error) {
			res := tx.Where("id IN ?", ids).Delete(&comment.DynamicComment{})
			return res.RowsAffected, res.Error
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
	return s.approveCommentGeneric(ctx, &comment.DynamicComment{}, commentID)
}

func (s *CommentService) IgnoreDynComment(ctx context.Context, commentID uint64) error {
	return s.ignoreCommentGeneric(ctx, &comment.DynamicComment{}, commentID)
}

// ─── Internal helpers ───
