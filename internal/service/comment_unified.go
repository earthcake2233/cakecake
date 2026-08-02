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
