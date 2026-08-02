package service

import (
	"context"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"cakecake/internal/model/comment"
	"cakecake/internal/pkg/sensitive"
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
	load func(ctx context.Context, targetID uint64, curated bool) ([]commentListRow, error),
	react commentReactionLoader,
) (*CommentListResult, error) {
	r := &CommentListResult{CommentsCurated: curated, CommentsClosed: closed}
	if closed {
		r.Items = []CommentItem{}
		return r, nil
	}
	list, err := load(ctx, targetID, curated)
	if err != nil {
		return nil, ErrInternalError
	}
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
			return nil, ErrInternalError
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

// loadUsersForCommentIDs loads users and levels for a set of user IDs.
func (s *CommentService) loadUsersForCommentIDs(ctx context.Context, uids []uint64) (map[uint64]UserInfo, map[uint64]int) {
	if len(uids) == 0 {
		return nil, nil
	}
	users, _ := s.users.GetUsersByIDs(ctx, uids)
	levels, _ := s.users.BatchCurrentLevels(ctx, uids)
	return users, levels
}

func (s *CommentService) loadCommentLikesByIDs(ctx context.Context, viewerID uint64, ids []uint64) (map[uint64]bool, error) {
	result := make(map[uint64]bool)
	if viewerID == 0 || len(ids) == 0 {
		return result, nil
	}
	var likes []comment.CommentLike
	if err := s.db.WithContext(ctx).Where("user_id = ? AND comment_id IN ?", viewerID, ids).Find(&likes).Error; err != nil {
		return nil, err
	}
	for _, lk := range likes {
		result[lk.CommentID] = true
	}
	return result, nil
}

func (s *CommentService) loadCommentDislikesByIDs(ctx context.Context, viewerID uint64, ids []uint64) (map[uint64]bool, error) {
	result := make(map[uint64]bool)
	if viewerID == 0 || len(ids) == 0 {
		return result, nil
	}
	var dislikes []comment.CommentDislike
	if err := s.db.WithContext(ctx).Where("user_id = ? AND comment_id IN ?", viewerID, ids).Find(&dislikes).Error; err != nil {
		return nil, err
	}
	for _, dk := range dislikes {
		result[dk.CommentID] = true
	}
	return result, nil
}

func (s *CommentService) loadArticleReactionsByIDs(ctx context.Context, viewerID uint64, ids []uint64) (map[uint64]bool, map[uint64]bool, error) {
	liked := make(map[uint64]bool)
	disliked := make(map[uint64]bool)
	if viewerID == 0 || len(ids) == 0 {
		return liked, disliked, nil
	}
	var likes []comment.ArticleCommentLike
	if err := s.db.WithContext(ctx).Where("user_id = ? AND comment_id IN ?", viewerID, ids).Find(&likes).Error; err != nil {
		return nil, nil, err
	}
	for _, lk := range likes {
		liked[lk.CommentID] = true
	}
	var dis []comment.ArticleCommentDislike
	if err := s.db.WithContext(ctx).Where("user_id = ? AND comment_id IN ?", viewerID, ids).Find(&dis).Error; err != nil {
		return nil, nil, err
	}
	for _, dk := range dis {
		disliked[dk.CommentID] = true
	}
	return liked, disliked, nil
}

// validateCommentContent is the shared content validation for all comment domains.
func (s *CommentService) validateCommentContent(content string) error {
	if n := utf8.RuneCountInString(content); n < 1 || n > 1000 {
		return ErrParamError
	}
	if s.sens != nil {
		if err := s.sens.Check(content); err != nil {
			if _, ok := err.(sensitive.ErrBlocked); ok {
				return ErrCommentSensitive
			}
			return ErrInternalError
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
		return 0, 0, ErrNotFound
	}
	if !ok {
		return 0, 0, ErrParamError
	}
	if level > 3 {
		level = 3
	}
	return parentID, level, nil
}

// collectCommentDescendants recursively collects descendant comment IDs via a per-domain pluck.
func (s *CommentService) collectCommentDescendants(tx *gorm.DB, pluck func(tx *gorm.DB, parentID uint64) ([]uint64, error), root uint64) []uint64 {
	var ids []uint64
	children, err := pluck(tx, root)
	if err == nil {
		for _, id := range children {
			ids = append(ids, id)
			ids = append(ids, s.collectCommentDescendants(tx, pluck, id)...)
		}
	}
	return ids
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
		return ErrNotFound
	}
	if !isOwner && ownerID != userID {
		return ErrForbidden
	}
	if !ad.cascade {
		_ = ad.deleteLikes(s.db.WithContext(ctx), []uint64{commentID})
		_ = ad.deleteDislikes(s.db.WithContext(ctx), []uint64{commentID})
		_, err := ad.deleteRows(s.db.WithContext(ctx), []uint64{commentID})
		return err
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		descIDs := ad.descendants(tx, commentID)
		allIDs := append([]uint64{commentID}, descIDs...)
		_ = ad.deleteLikes(tx, allIDs)
		_ = ad.deleteDislikes(tx, allIDs)
		affected, err := ad.deleteRows(tx, allIDs)
		if err != nil || affected == 0 {
			return ErrNotFound
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
		return false, ErrNotFound
	}
	tid, pinned, err := ad.fetch(ctx, commentID)
	if err != nil {
		return false, ErrNotFound
	}
	if tid != targetID {
		return false, ErrParamError
	}
	newPinned := !pinned
	if newPinned {
		_ = ad.unpinOthers(ctx, targetID)
	}
	if err := ad.update(ctx, commentID, newPinned); err != nil {
		return false, ErrInternalError
	}
	return newPinned, nil
}

// approveCommentGeneric marks a comment approved on the given model table.
func (s *CommentService) approveCommentGeneric(ctx context.Context, model interface{}, commentID uint64) error {
	return s.db.WithContext(ctx).Model(model).Where("id = ?", commentID).Update("approved", true).Error
}

// ignoreCommentGeneric marks a comment as curated-ignored (soft mark: kept in
// the database but not shown in public curated lists; visible in the creator
// panel's "ignored" tab).
func (s *CommentService) ignoreCommentGeneric(ctx context.Context, model interface{}, commentID uint64) error {
	return s.db.WithContext(ctx).Model(model).Where("id = ?", commentID).Update("curated_ignored", true).Error
}

// getCommentGeneric fetches a comment row by ID for the given model type.
func getCommentGeneric[T any](s *CommentService, ctx context.Context, id uint64) (*T, error) {
	var out T
	if err := s.db.WithContext(ctx).First(&out, id).Error; err != nil {
		return nil, err
	}
	return &out, nil
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
		return false, 0, ErrNotFound
	}
	_ = ad.clearDislike(ctx, userID, commentID)
	liked, err := ad.hasLike(ctx, userID, commentID)
	if err == nil && liked {
		_ = ad.deleteLike(ctx, userID, commentID)
		_ = ad.updateCount(ctx, commentID, -1)
		return false, int(ad.count(ctx, commentID)), nil
	}
	if err := ad.createLike(ctx, userID, commentID); err != nil {
		return false, 0, ErrInternalError
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
		return false, ErrNotFound
	}
	_ = ad.clearLike(ctx, userID, commentID)
	disliked, err := ad.hasDislike(ctx, userID, commentID)
	if err == nil && disliked {
		_ = ad.deleteDislike(ctx, userID, commentID)
		return false, nil
	}
	if err := ad.createDislike(ctx, userID, commentID); err != nil {
		return false, ErrInternalError
	}
	return true, nil
}

// videoReactionAdapter maps comment.CommentLike/CommentDislike to the shared toggle logic.
func (s *CommentService) videoReactionAdapter() commentReactionAdapter {
	return commentReactionAdapter{
		fetch: func(ctx context.Context, id uint64) (uint64, error) {
			var cm comment.Comment
			if err := s.db.WithContext(ctx).First(&cm, id).Error; err != nil {
				return 0, err
			}
			return cm.LikeCount, nil
		},
		clearDislike: func(ctx context.Context, userID, commentID uint64) error {
			return s.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).Delete(&comment.CommentDislike{}).Error
		},
		clearLike: func(ctx context.Context, userID, commentID uint64) error {
			return s.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).Delete(&comment.CommentLike{}).Error
		},
		hasLike: func(ctx context.Context, userID, commentID uint64) (bool, error) {
			var existing comment.CommentLike
			return s.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).First(&existing).Error == nil, nil
		},
		hasDislike: func(ctx context.Context, userID, commentID uint64) (bool, error) {
			var existing comment.CommentDislike
			return s.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).First(&existing).Error == nil, nil
		},
		createLike: func(ctx context.Context, userID, commentID uint64) error {
			return s.db.WithContext(ctx).Create(&comment.CommentLike{UserID: userID, CommentID: commentID}).Error
		},
		deleteLike: func(ctx context.Context, userID, commentID uint64) error {
			return s.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).Delete(&comment.CommentLike{}).Error
		},
		createDislike: func(ctx context.Context, userID, commentID uint64) error {
			return s.db.WithContext(ctx).Create(&comment.CommentDislike{UserID: userID, CommentID: commentID}).Error
		},
		deleteDislike: func(ctx context.Context, userID, commentID uint64) error {
			return s.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).Delete(&comment.CommentDislike{}).Error
		},
		updateCount: func(ctx context.Context, commentID uint64, delta int) error {
			if delta < 0 {
				return s.db.WithContext(ctx).Model(&comment.Comment{}).Where("id = ?", commentID).
					UpdateColumn("like_count", gorm.Expr("CASE WHEN like_count - ? < 0 THEN 0 ELSE like_count - ? END", -delta, -delta)).Error
			}
			return s.db.WithContext(ctx).Model(&comment.Comment{}).Where("id = ?", commentID).
				UpdateColumn("like_count", gorm.Expr("like_count + ?", delta)).Error
		},
		count: func(ctx context.Context, commentID uint64) uint64 {
			var u comment.Comment
			_ = s.db.WithContext(ctx).First(&u, commentID)
			return u.LikeCount
		},
		notifyLike: func(ctx context.Context, commentID, userID uint64) {
			if s.notifSvc == nil {
				return
			}
			var cm comment.Comment
			if err := s.db.WithContext(ctx).First(&cm, commentID).Error; err == nil {
				s.notifSvc.NotifyCommentLike(ctx, cm, userID)
			}
		},
	}
}

// articleReactionAdapter maps comment.ArticleCommentLike/Dislike to the shared toggle logic.
func (s *CommentService) articleReactionAdapter() commentReactionAdapter {
	return commentReactionAdapter{
		fetch: func(ctx context.Context, id uint64) (uint64, error) {
			var cm comment.ArticleComment
			if err := s.db.WithContext(ctx).First(&cm, id).Error; err != nil {
				return 0, err
			}
			return cm.LikeCount, nil
		},
		clearDislike: func(ctx context.Context, userID, commentID uint64) error {
			return s.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).Delete(&comment.ArticleCommentDislike{}).Error
		},
		clearLike: func(ctx context.Context, userID, commentID uint64) error {
			return s.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).Delete(&comment.ArticleCommentLike{}).Error
		},
		hasLike: func(ctx context.Context, userID, commentID uint64) (bool, error) {
			var existing comment.ArticleCommentLike
			return s.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).First(&existing).Error == nil, nil
		},
		hasDislike: func(ctx context.Context, userID, commentID uint64) (bool, error) {
			var existing comment.ArticleCommentDislike
			return s.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).First(&existing).Error == nil, nil
		},
		createLike: func(ctx context.Context, userID, commentID uint64) error {
			return s.db.WithContext(ctx).Create(&comment.ArticleCommentLike{UserID: userID, CommentID: commentID}).Error
		},
		deleteLike: func(ctx context.Context, userID, commentID uint64) error {
			return s.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).Delete(&comment.ArticleCommentLike{}).Error
		},
		createDislike: func(ctx context.Context, userID, commentID uint64) error {
			return s.db.WithContext(ctx).Create(&comment.ArticleCommentDislike{UserID: userID, CommentID: commentID}).Error
		},
		deleteDislike: func(ctx context.Context, userID, commentID uint64) error {
			return s.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).Delete(&comment.ArticleCommentDislike{}).Error
		},
		updateCount: func(ctx context.Context, commentID uint64, delta int) error {
			if delta < 0 {
				return s.db.WithContext(ctx).Model(&comment.ArticleComment{}).Where("id = ?", commentID).
					UpdateColumn("like_count", gorm.Expr("CASE WHEN like_count - ? < 0 THEN 0 ELSE like_count - ? END", -delta, -delta)).Error
			}
			return s.db.WithContext(ctx).Model(&comment.ArticleComment{}).Where("id = ?", commentID).
				UpdateColumn("like_count", gorm.Expr("like_count + ?", delta)).Error
		},
		count: func(ctx context.Context, commentID uint64) uint64 {
			var u comment.ArticleComment
			_ = s.db.WithContext(ctx).First(&u, commentID)
			return u.LikeCount
		},
	}
}

// dynamicReactionAdapter maps comment.DynamicCommentLike/Dislike to the shared toggle logic.
func (s *CommentService) dynamicReactionAdapter() commentReactionAdapter {
	return commentReactionAdapter{
		fetch: func(ctx context.Context, id uint64) (uint64, error) {
			var cm comment.DynamicComment
			if err := s.db.WithContext(ctx).First(&cm, id).Error; err != nil {
				return 0, err
			}
			return cm.LikeCount, nil
		},
		clearDislike: func(ctx context.Context, userID, commentID uint64) error {
			return s.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).Delete(&comment.DynamicCommentDislike{}).Error
		},
		clearLike: func(ctx context.Context, userID, commentID uint64) error {
			return s.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).Delete(&comment.DynamicCommentLike{}).Error
		},
		hasLike: func(ctx context.Context, userID, commentID uint64) (bool, error) {
			var existing comment.DynamicCommentLike
			return s.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).First(&existing).Error == nil, nil
		},
		hasDislike: func(ctx context.Context, userID, commentID uint64) (bool, error) {
			var existing comment.DynamicCommentDislike
			return s.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).First(&existing).Error == nil, nil
		},
		createLike: func(ctx context.Context, userID, commentID uint64) error {
			return s.db.WithContext(ctx).Create(&comment.DynamicCommentLike{UserID: userID, CommentID: commentID}).Error
		},
		deleteLike: func(ctx context.Context, userID, commentID uint64) error {
			return s.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).Delete(&comment.DynamicCommentLike{}).Error
		},
		createDislike: func(ctx context.Context, userID, commentID uint64) error {
			return s.db.WithContext(ctx).Create(&comment.DynamicCommentDislike{UserID: userID, CommentID: commentID}).Error
		},
		deleteDislike: func(ctx context.Context, userID, commentID uint64) error {
			return s.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).Delete(&comment.DynamicCommentDislike{}).Error
		},
		updateCount: func(ctx context.Context, commentID uint64, delta int) error {
			if delta < 0 {
				return s.db.WithContext(ctx).Model(&comment.DynamicComment{}).Where("id = ?", commentID).
					UpdateColumn("like_count", gorm.Expr("CASE WHEN like_count - ? < 0 THEN 0 ELSE like_count - ? END", -delta, -delta)).Error
			}
			return s.db.WithContext(ctx).Model(&comment.DynamicComment{}).Where("id = ?", commentID).
				UpdateColumn("like_count", gorm.Expr("like_count + ?", delta)).Error
		},
		count: func(ctx context.Context, commentID uint64) uint64 {
			var u comment.DynamicComment
			_ = s.db.WithContext(ctx).First(&u, commentID)
			return u.LikeCount
		},
	}
}
