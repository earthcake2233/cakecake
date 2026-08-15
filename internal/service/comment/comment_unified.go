package comment

import (
	"context"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"cakecake/internal/model/comment"
	"cakecake/internal/pkg/sensitive"
	"cakecake/internal/service"
)

// commentListRow is the flattened row shared by all three comment domains
// (video / article / dynamic) so list building is implemented once.
type commentListRow struct {
	ID         uint64
	UserID     uint64
	ParentID   uint64
	Level      int
	Content    string
	LikeCount  uint64
	Pinned     bool
	Approved   bool
	IPLocation string
	CreatedAt  time.Time
}

// commentReactionLoader loads per-viewer liked/disliked maps for comment IDs.
type commentReactionLoader func(ctx context.Context, viewerID uint64, ids []uint64) (map[uint64]bool, map[uint64]bool, error)

func toCommentRows(list []comment.Comment) []commentListRow {
	out := make([]commentListRow, len(list))
	for i, cm := range list {
		out[i] = commentListRow{
			ID: cm.ID, UserID: cm.UserID, ParentID: cm.ParentID, Level: cm.Level,
			Content: cm.Content, LikeCount: cm.LikeCount, Pinned: cm.Pinned,
			Approved: cm.Approved, IPLocation: cm.IpLocation, CreatedAt: cm.CreatedAt,
		}
	}
	return out
}

func toArticleCommentRows(list []comment.ArticleComment) []commentListRow {
	out := make([]commentListRow, len(list))
	for i, cm := range list {
		out[i] = commentListRow{
			ID: cm.ID, UserID: cm.UserID, ParentID: cm.ParentID, Level: cm.Level,
			Content: cm.Content, LikeCount: cm.LikeCount, Pinned: cm.Pinned,
			Approved: cm.Approved, IPLocation: cm.IpLocation, CreatedAt: cm.CreatedAt,
		}
	}
	return out
}

func toDynamicCommentRows(list []comment.DynamicComment) []commentListRow {
	out := make([]commentListRow, len(list))
	for i, cm := range list {
		out[i] = commentListRow{
			ID: cm.ID, UserID: cm.UserID, ParentID: cm.ParentID, Level: cm.Level,
			Content: cm.Content, LikeCount: cm.LikeCount, Pinned: cm.Pinned,
			Approved: cm.Approved, IPLocation: cm.IpLocation, CreatedAt: cm.CreatedAt,
		}
	}
	return out
}

// buildCommentList is the shared list implementation for all comment domains.
func (s *CommentService) buildCommentList(
	ctx context.Context,
	targetID, viewerID, ownerID uint64,
	curated, closed bool,
	q CommentListQuery,
	load func(ctx context.Context, targetID uint64, curated bool, q CommentListQuery) (*CommentPage, error),
	react commentReactionLoader,
) (*CommentListResult, error) {
	page, pageSize, sort := q.Normalized()
	r := &CommentListResult{
		CommentsCurated: curated,
		CommentsClosed:  closed,
		Page:            page,
		PageSize:        pageSize,
	}
	if closed {
		r.Items = []CommentItem{}
		return r, nil
	}
	pg, err := load(ctx, targetID, curated, CommentListQuery{Page: page, PageSize: pageSize, Sort: sort})
	if err != nil {
		return nil, service.ErrInternalError
	}
	r.Total = pg.Total
	r.TotalPages = commentTotalPages(pg.Total, pageSize)
	list := pg.Rows
	if len(list) == 0 {
		r.Items = []CommentItem{}
		return r, nil
	}

	ids := make([]uint64, 0, len(list))
	uids := make([]uint64, 0, len(list))
	seenUID := make(map[uint64]bool, len(list))
	for _, cm := range list {
		ids = append(ids, cm.ID)
		if !seenUID[cm.UserID] {
			seenUID[cm.UserID] = true
			uids = append(uids, cm.UserID)
		}
	}
	users, _ := s.users.GetUsersByIDs(ctx, uids)
	levels, _ := s.users.BatchCurrentLevels(ctx, uids)

	var liked, disliked map[uint64]bool
	if react != nil {
		liked, disliked, err = react(ctx, viewerID, ids)
		if err != nil {
			return nil, service.ErrInternalError
		}
	}

	out := make([]CommentItem, 0, len(list))
	for _, cm := range list {
		ulv := levels[cm.UserID]
		if ulv < 1 {
			ulv = 1
		}
		u := users[cm.UserID]
		out = append(out, CommentItem{
			ID: cm.ID, UserID: cm.UserID, Username: u.Nickname, AvatarURL: u.AvatarURL,
			ParentID: cm.ParentID, Level: cm.Level, UserLevel: ulv, Content: cm.Content,
			LikeCount: cm.LikeCount, CreatedAt: cm.CreatedAt.Format("2006-01-02 15:04:05"),
			LikedByMe: liked[cm.ID], DislikedByMe: disliked[cm.ID], Pinned: cm.Pinned,
			IsByUploader: cm.UserID == ownerID, IPLocation: cm.IPLocation,
		})
	}
	r.Items = out
	return r, nil
}

// commentTotalPages returns the page count for a total, at least 1.
func commentTotalPages(total int64, pageSize int) int {
	if total <= 0 {
		return 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	return int((total + int64(pageSize) - 1) / int64(pageSize))
}

func (s *CommentService) loadCommentLikesByIDs(ctx context.Context, viewerID uint64, ids []uint64) (map[uint64]bool, error) {
	return s.comments.LoadCommentLikes(ctx, CommentVideo, viewerID, ids)
}

func (s *CommentService) loadCommentDislikesByIDs(ctx context.Context, viewerID uint64, ids []uint64) (map[uint64]bool, error) {
	return s.comments.LoadCommentDislikes(ctx, CommentVideo, viewerID, ids)
}

func (s *CommentService) loadArticleReactionsByIDs(ctx context.Context, viewerID uint64, ids []uint64) (map[uint64]bool, map[uint64]bool, error) {
	liked, err := s.comments.LoadCommentLikes(ctx, CommentArticle, viewerID, ids)
	if err != nil {
		return nil, nil, err
	}
	disliked, err := s.comments.LoadCommentDislikes(ctx, CommentArticle, viewerID, ids)
	if err != nil {
		return nil, nil, err
	}
	return liked, disliked, nil
}

// validateCommentContent is the shared content validation for all comment domains.
func (s *CommentService) validateCommentContent(content string) error {
	if n := utf8.RuneCountInString(content); n < 1 || n > 1000 {
		return service.ErrParamError
	}
	if s.sens != nil {
		if err := s.sens.Check(content); err != nil {
			if _, ok := err.(sensitive.ErrBlocked); ok {
				return service.ErrCommentSensitive
			}
			return service.ErrInternalError
		}
	}
	return nil
}

// commentParentCheck verifies a parent comment belongs to the target and returns its level.
type commentParentCheck func(ctx context.Context, parentID uint64) (level int, ok bool, err error)

// resolveCommentParent validates and resolves a reply parent, capping nesting at 3.
func (s *CommentService) resolveCommentParent(ctx context.Context, parentID, targetID uint64, check commentParentCheck) (uint64, int, error) {
	if parentID == 0 {
		return 0, 1, nil
	}
	level, ok, err := check(ctx, parentID)
	if err != nil {
		return 0, 0, service.ErrNotFound
	}
	if !ok {
		return 0, 0, service.ErrParamError
	}
	if level > 3 {
		level = 3
	}
	return parentID, level, nil
}

// commentDeleteAdapter maps a comment domain's tables to the shared delete logic.
type commentDeleteAdapter struct {
	fetch          func(ctx context.Context, id uint64) (ownerID, targetID uint64, err error)
	cascade        bool
	descendants    func(tx *gorm.DB, root uint64) []uint64
	deleteLikes    func(tx *gorm.DB, ids []uint64) error
	deleteDislikes func(tx *gorm.DB, ids []uint64) error
	deleteRows     func(tx *gorm.DB, ids []uint64) (int64, error)
	incrCount      func(ctx context.Context, targetID uint64, delta int)
}

// deleteCommentGeneric is the shared delete implementation for all comment domains.
func (s *CommentService) deleteCommentGeneric(ctx context.Context, userID, commentID uint64, isOwner bool, ad commentDeleteAdapter) error {
	ownerID, targetID, err := ad.fetch(ctx, commentID)
	if err != nil {
		return service.ErrNotFound
	}
	if !isOwner && ownerID != userID {
		return service.ErrForbidden
	}
	if !ad.cascade {
		return s.comments.WithTx(ctx, func(tx *gorm.DB) error {
			_ = ad.deleteLikes(tx, []uint64{commentID})
			_ = ad.deleteDislikes(tx, []uint64{commentID})
			_, err := ad.deleteRows(tx, []uint64{commentID})
			return err
		})
	}
	return s.comments.WithTx(ctx, func(tx *gorm.DB) error {
		descIDs := ad.descendants(tx, commentID)
		allIDs := append([]uint64{commentID}, descIDs...)
		_ = ad.deleteLikes(tx, allIDs)
		_ = ad.deleteDislikes(tx, allIDs)
		affected, err := ad.deleteRows(tx, allIDs)
		if err != nil || affected == 0 {
			return service.ErrNotFound
		}
		if ad.incrCount != nil {
			ad.incrCount(ctx, targetID, -int(affected))
		}
		return nil
	})
}

// commentPinAdapter maps a comment domain's tables to the shared pin logic.
type commentPinAdapter struct {
	checkTarget func(ctx context.Context, targetID uint64) error
	fetch       func(ctx context.Context, id uint64) (targetID uint64, pinned bool, err error)
	unpinOthers func(ctx context.Context, targetID uint64) error
	update      func(ctx context.Context, id uint64, pinned bool) error
}

// pinCommentGeneric is the shared pin toggle for all comment domains.
func (s *CommentService) pinCommentGeneric(ctx context.Context, targetID, commentID uint64, ad commentPinAdapter) (bool, error) {
	if err := ad.checkTarget(ctx, targetID); err != nil {
		return false, service.ErrNotFound
	}
	tid, pinned, err := ad.fetch(ctx, commentID)
	if err != nil {
		return false, service.ErrNotFound
	}
	if tid != targetID {
		return false, service.ErrParamError
	}
	newPinned := !pinned
	if newPinned {
		_ = ad.unpinOthers(ctx, targetID)
	}
	if err := ad.update(ctx, commentID, newPinned); err != nil {
		return false, service.ErrInternalError
	}
	return newPinned, nil
}

// commentReactionAdapter maps a comment domain's like/dislike tables to the shared toggle logic.
type commentReactionAdapter struct {
	fetch         func(ctx context.Context, id uint64) (uint64, error)
	clearDislike  func(ctx context.Context, userID, commentID uint64) error
	clearLike     func(ctx context.Context, userID, commentID uint64) error
	hasLike       func(ctx context.Context, userID, commentID uint64) (bool, error)
	hasDislike    func(ctx context.Context, userID, commentID uint64) (bool, error)
	createLike    func(ctx context.Context, userID, commentID uint64) error
	deleteLike    func(ctx context.Context, userID, commentID uint64) error
	createDislike func(ctx context.Context, userID, commentID uint64) error
	deleteDislike func(ctx context.Context, userID, commentID uint64) error
	updateCount   func(ctx context.Context, commentID uint64, delta int) error
	count         func(ctx context.Context, commentID uint64) uint64
	notifyLike    func(ctx context.Context, commentID, userID uint64)
}

// toggleCommentLikeGeneric is the shared like-toggle implementation.
func (s *CommentService) toggleCommentLikeGeneric(ctx context.Context, userID, commentID uint64, ad commentReactionAdapter) (bool, int, error) {
	if _, err := ad.fetch(ctx, commentID); err != nil {
		return false, 0, service.ErrNotFound
	}
	_ = ad.clearDislike(ctx, userID, commentID)
	liked, err := ad.hasLike(ctx, userID, commentID)
	if err == nil && liked {
		_ = ad.deleteLike(ctx, userID, commentID)
		_ = ad.updateCount(ctx, commentID, -1)
		return false, int(ad.count(ctx, commentID)), nil
	}
	if err := ad.createLike(ctx, userID, commentID); err != nil {
		return false, 0, service.ErrInternalError
	}
	_ = ad.updateCount(ctx, commentID, 1)
	if ad.notifyLike != nil {
		ad.notifyLike(ctx, commentID, userID)
	}
	return true, int(ad.count(ctx, commentID)), nil
}

// toggleCommentDislikeGeneric is the shared dislike-toggle implementation.
func (s *CommentService) toggleCommentDislikeGeneric(ctx context.Context, userID, commentID uint64, ad commentReactionAdapter) (bool, error) {
	if _, err := ad.fetch(ctx, commentID); err != nil {
		return false, service.ErrNotFound
	}
	_ = ad.clearLike(ctx, userID, commentID)
	disliked, err := ad.hasDislike(ctx, userID, commentID)
	if err == nil && disliked {
		_ = ad.deleteDislike(ctx, userID, commentID)
		return false, nil
	}
	if err := ad.createDislike(ctx, userID, commentID); err != nil {
		return false, service.ErrInternalError
	}
	return true, nil
}

// reactionAdapter builds the per-kind like/dislike adapter for the shared toggle logic.
func (s *CommentService) reactionAdapter(kind CommentKind, withNotify bool) commentReactionAdapter {
	return commentReactionAdapter{
		fetch: func(ctx context.Context, id uint64) (uint64, error) {
			return s.comments.GetCommentLikeCount(ctx, kind, id)
		},
		clearDislike: func(ctx context.Context, userID, commentID uint64) error {
			return s.comments.ClearCommentDislike(ctx, kind, userID, commentID)
		},
		clearLike: func(ctx context.Context, userID, commentID uint64) error {
			return s.comments.ClearCommentLike(ctx, kind, userID, commentID)
		},
		hasLike: func(ctx context.Context, userID, commentID uint64) (bool, error) {
			return s.comments.HasCommentLike(ctx, kind, userID, commentID)
		},
		hasDislike: func(ctx context.Context, userID, commentID uint64) (bool, error) {
			return s.comments.HasCommentDislike(ctx, kind, userID, commentID)
		},
		createLike: func(ctx context.Context, userID, commentID uint64) error {
			return s.comments.CreateCommentLike(ctx, kind, userID, commentID)
		},
		deleteLike: func(ctx context.Context, userID, commentID uint64) error {
			return s.comments.DeleteCommentLike(ctx, kind, userID, commentID)
		},
		createDislike: func(ctx context.Context, userID, commentID uint64) error {
			return s.comments.CreateCommentDislike(ctx, kind, userID, commentID)
		},
		deleteDislike: func(ctx context.Context, userID, commentID uint64) error {
			return s.comments.DeleteCommentDislike(ctx, kind, userID, commentID)
		},
		updateCount: func(ctx context.Context, commentID uint64, delta int) error {
			return s.comments.IncrCommentLikeCount(ctx, kind, commentID, delta)
		},
		count: func(ctx context.Context, commentID uint64) uint64 {
			return s.comments.CountCommentLikes(ctx, kind, commentID)
		},
		notifyLike: func(ctx context.Context, commentID, userID uint64) {
			if !withNotify || s.notifSvc == nil {
				return
			}
			cm, err := s.comments.GetVideoComment(ctx, commentID)
			if err == nil {
				s.notifSvc.NotifyCommentLike(ctx, *cm, userID)
			}
		},
	}
}

// videoReactionAdapter maps comment.CommentLike/CommentDislike to the shared toggle logic.
func (s *CommentService) videoReactionAdapter() commentReactionAdapter {
	return s.reactionAdapter(CommentVideo, true)
}

// articleReactionAdapter maps comment.ArticleCommentLike/Dislike to the shared toggle logic.
func (s *CommentService) articleReactionAdapter() commentReactionAdapter {
	return s.reactionAdapter(CommentArticle, false)
}

// dynamicReactionAdapter maps comment.DynamicCommentLike/Dislike to the shared toggle logic.
func (s *CommentService) dynamicReactionAdapter() commentReactionAdapter {
	return s.reactionAdapter(CommentDynamic, false)
}
