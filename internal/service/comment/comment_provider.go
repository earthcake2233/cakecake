package comment

import (
	"cakecake/internal/model/comment"
	"cakecake/internal/service/queryutil"
	"context"

	"gorm.io/gorm"
)

// CommentKind identifies which comment table a storage operation targets.
type CommentKind int

// CommentVideo is the video comment kind.
const (
	CommentVideo CommentKind = iota
	CommentArticle
	CommentDynamic
)

// CommentProvider is the comment domain storage boundary.
// Phase 1: *gorm.DB impl. Phase 2+: replaced by gRPC client / per-domain store.
type CommentProvider interface {
	// WithTx runs fn inside a database transaction (Phase 1 monolith seam).
	WithTx(ctx context.Context, fn func(tx *gorm.DB) error) error
	// ListComments returns the target's comment rows (approved-filtered when curated).
	ListComments(ctx context.Context, kind CommentKind, targetID uint64, curated bool) ([]commentListRow, error)

	// Typed creates.
	CreateVideoComment(ctx context.Context, cm *comment.Comment) error
	CreateArticleComment(ctx context.Context, cm *comment.ArticleComment) error
	CreateDynamicComment(ctx context.Context, cm *comment.DynamicComment) error

	// Typed fetches.
	GetVideoComment(ctx context.Context, id uint64) (*comment.Comment, error)
	GetArticleComment(ctx context.Context, id uint64) (*comment.ArticleComment, error)
	GetDynamicComment(ctx context.Context, id uint64) (*comment.DynamicComment, error)

	// GetCommentParent returns a parent comment's target id and level.
	GetCommentParent(ctx context.Context, kind CommentKind, id uint64) (targetID uint64, level int, err error)
	// GetCommentForDelete returns a comment's owner and target ids.
	GetCommentForDelete(ctx context.Context, kind CommentKind, id uint64) (ownerID, targetID uint64, err error)
	// GetCommentPin returns a comment's target id and pinned flag.
	GetCommentPin(ctx context.Context, kind CommentKind, id uint64) (targetID uint64, pinned bool, err error)
	UnpinComments(ctx context.Context, kind CommentKind, targetID uint64) error
	UpdateCommentPinned(ctx context.Context, kind CommentKind, id uint64, pinned bool) error
	ApproveComment(ctx context.Context, kind CommentKind, id uint64) error
	IgnoreComment(ctx context.Context, kind CommentKind, id uint64) error

	// Cascade-delete primitives (run inside a WithTx callback).
	CollectCommentDescendants(tx *gorm.DB, kind CommentKind, root uint64) []uint64
	DeleteCommentLikesTx(tx *gorm.DB, kind CommentKind, ids []uint64) error
	DeleteCommentDislikesTx(tx *gorm.DB, kind CommentKind, ids []uint64) error
	DeleteCommentRowsTx(tx *gorm.DB, kind CommentKind, ids []uint64) (int64, error)

	// Reactions.
	LoadCommentLikes(ctx context.Context, kind CommentKind, viewerID uint64, ids []uint64) (map[uint64]bool, error)
	LoadCommentDislikes(ctx context.Context, kind CommentKind, viewerID uint64, ids []uint64) (map[uint64]bool, error)
	GetCommentLikeCount(ctx context.Context, kind CommentKind, id uint64) (uint64, error)
	HasCommentLike(ctx context.Context, kind CommentKind, userID, commentID uint64) (bool, error)
	HasCommentDislike(ctx context.Context, kind CommentKind, userID, commentID uint64) (bool, error)
	ClearCommentLike(ctx context.Context, kind CommentKind, userID, commentID uint64) error
	ClearCommentDislike(ctx context.Context, kind CommentKind, userID, commentID uint64) error
	CreateCommentLike(ctx context.Context, kind CommentKind, userID, commentID uint64) error
	DeleteCommentLike(ctx context.Context, kind CommentKind, userID, commentID uint64) error
	CreateCommentDislike(ctx context.Context, kind CommentKind, userID, commentID uint64) error
	DeleteCommentDislike(ctx context.Context, kind CommentKind, userID, commentID uint64) error
	IncrCommentLikeCount(ctx context.Context, kind CommentKind, commentID uint64, delta int) error
	CountCommentLikes(ctx context.Context, kind CommentKind, commentID uint64) uint64
}

// CommentProviderImpl implements CommentProvider using *gorm.DB (Phase 1 monolith).
type CommentProviderImpl struct {
	db *gorm.DB
}

var _ CommentProvider = (*CommentProviderImpl)(nil)

// NewCommentProvider creates a gorm-backed CommentProvider implementation.
func NewCommentProvider(db *gorm.DB) *CommentProviderImpl {
	return &CommentProviderImpl{db: db}
}

// WithTx runs fn inside a transaction.
func (p *CommentProviderImpl) WithTx(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return p.db.WithContext(ctx).Transaction(fn)
}

func commentTargetColumn(kind CommentKind) string {
	switch kind {
	case CommentArticle:
		return "article_id"
	case CommentDynamic:
		return "dynamic_id"
	default:
		return "video_id"
	}
}

func commentModel(kind CommentKind) interface{} {
	switch kind {
	case CommentArticle:
		return &comment.ArticleComment{}
	case CommentDynamic:
		return &comment.DynamicComment{}
	default:
		return &comment.Comment{}
	}
}

func commentLikeModel(kind CommentKind) interface{} {
	switch kind {
	case CommentArticle:
		return &comment.ArticleCommentLike{}
	case CommentDynamic:
		return &comment.DynamicCommentLike{}
	default:
		return &comment.CommentLike{}
	}
}

func commentDislikeModel(kind CommentKind) interface{} {
	switch kind {
	case CommentArticle:
		return &comment.ArticleCommentDislike{}
	case CommentDynamic:
		return &comment.DynamicCommentDislike{}
	default:
		return &comment.CommentDislike{}
	}
}

// ListComments lists comments of the given kind for a target media id,
// optionally filtering to curated-approved rows.
func (p *CommentProviderImpl) ListComments(ctx context.Context, kind CommentKind, targetID uint64, curated bool) ([]commentListRow, error) {
	col := commentTargetColumn(kind)
	q := p.db.WithContext(ctx).Where(col+" = ?", targetID)
	if curated {
		q = q.Where("approved = ?", true)
	}
	switch kind {
	case CommentArticle:
		var list []comment.ArticleComment
		if err := q.Order("id ASC").Find(&list).Error; err != nil {
			return nil, err
		}
		return toArticleCommentRows(list), nil
	case CommentDynamic:
		var list []comment.DynamicComment
		if err := q.Order("id ASC").Find(&list).Error; err != nil {
			return nil, err
		}
		return toDynamicCommentRows(list), nil
	default:
		var list []comment.Comment
		if err := q.Order("id ASC").Find(&list).Error; err != nil {
			return nil, err
		}
		return toCommentRows(list), nil
	}
}

// CreateVideoComment inserts a video comment row.
func (p *CommentProviderImpl) CreateVideoComment(ctx context.Context, cm *comment.Comment) error {
	return p.db.WithContext(ctx).Create(cm).Error
}

// CreateArticleComment inserts an article comment row.
func (p *CommentProviderImpl) CreateArticleComment(ctx context.Context, cm *comment.ArticleComment) error {
	return p.db.WithContext(ctx).Create(cm).Error
}

// CreateDynamicComment inserts a dynamic comment row.
func (p *CommentProviderImpl) CreateDynamicComment(ctx context.Context, cm *comment.DynamicComment) error {
	return p.db.WithContext(ctx).Create(cm).Error
}

// GetVideoComment loads a video comment by id.
func (p *CommentProviderImpl) GetVideoComment(ctx context.Context, id uint64) (*comment.Comment, error) {
	return queryutil.FirstByID[comment.Comment](ctx, p.db, id)
}

// GetArticleComment loads an article comment by id.
func (p *CommentProviderImpl) GetArticleComment(ctx context.Context, id uint64) (*comment.ArticleComment, error) {
	return queryutil.FirstByID[comment.ArticleComment](ctx, p.db, id)
}

// GetDynamicComment loads a dynamic comment by id.
func (p *CommentProviderImpl) GetDynamicComment(ctx context.Context, id uint64) (*comment.DynamicComment, error) {
	return queryutil.FirstByID[comment.DynamicComment](ctx, p.db, id)
}

// GetCommentParent returns the target media id and level of a parent comment.
func (p *CommentProviderImpl) GetCommentParent(ctx context.Context, kind CommentKind, id uint64) (targetID uint64, level int, err error) {
	switch kind {
	case CommentArticle:
		var parent comment.ArticleComment
		if err := p.db.WithContext(ctx).First(&parent, id).Error; err != nil {
			return 0, 0, err
		}
		return parent.ArticleID, parent.Level, nil
	case CommentDynamic:
		var parent comment.DynamicComment
		if err := p.db.WithContext(ctx).First(&parent, id).Error; err != nil {
			return 0, 0, err
		}
		return parent.DynamicID, parent.Level, nil
	default:
		var parent comment.Comment
		if err := p.db.WithContext(ctx).First(&parent, id).Error; err != nil {
			return 0, 0, err
		}
		return parent.VideoID, parent.Level, nil
	}
}

// GetCommentForDelete returns the owner id and target media id of a comment.
func (p *CommentProviderImpl) GetCommentForDelete(ctx context.Context, kind CommentKind, id uint64) (ownerID, targetID uint64, err error) {
	switch kind {
	case CommentArticle:
		var cm comment.ArticleComment
		if err := p.db.WithContext(ctx).First(&cm, id).Error; err != nil {
			return 0, 0, err
		}
		return cm.UserID, cm.ArticleID, nil
	case CommentDynamic:
		var cm comment.DynamicComment
		if err := p.db.WithContext(ctx).First(&cm, id).Error; err != nil {
			return 0, 0, err
		}
		return cm.UserID, cm.DynamicID, nil
	default:
		var cm comment.Comment
		if err := p.db.WithContext(ctx).First(&cm, id).Error; err != nil {
			return 0, 0, err
		}
		return cm.UserID, cm.VideoID, nil
	}
}

// GetCommentPin returns the target media id and pinned state of a comment.
func (p *CommentProviderImpl) GetCommentPin(ctx context.Context, kind CommentKind, id uint64) (targetID uint64, pinned bool, err error) {
	switch kind {
	case CommentArticle:
		var cm comment.ArticleComment
		if err := p.db.WithContext(ctx).First(&cm, id).Error; err != nil {
			return 0, false, err
		}
		return cm.ArticleID, cm.Pinned, nil
	case CommentDynamic:
		var cm comment.DynamicComment
		if err := p.db.WithContext(ctx).First(&cm, id).Error; err != nil {
			return 0, false, err
		}
		return cm.DynamicID, cm.Pinned, nil
	default:
		var cm comment.Comment
		if err := p.db.WithContext(ctx).First(&cm, id).Error; err != nil {
			return 0, false, err
		}
		return cm.VideoID, cm.Pinned, nil
	}
}

// UnpinComments clears the pinned flag on all comments of a target media id.
func (p *CommentProviderImpl) UnpinComments(ctx context.Context, kind CommentKind, targetID uint64) error {
	col := commentTargetColumn(kind)
	return p.db.WithContext(ctx).Model(commentModel(kind)).
		Where(col+" = ? AND pinned = ?", targetID, true).Update("pinned", false).Error
}

// UpdateCommentPinned sets the pinned flag on a single comment.
func (p *CommentProviderImpl) UpdateCommentPinned(ctx context.Context, kind CommentKind, id uint64, pinned bool) error {
	return p.db.WithContext(ctx).Model(commentModel(kind)).Where("id = ?", id).Update("pinned", pinned).Error
}

// ApproveComment marks a comment as curated-approved.
func (p *CommentProviderImpl) ApproveComment(ctx context.Context, kind CommentKind, id uint64) error {
	return p.db.WithContext(ctx).Model(commentModel(kind)).Where("id = ?", id).Update("approved", true).Error
}

// IgnoreComment soft-marks a comment as curated-ignored.
func (p *CommentProviderImpl) IgnoreComment(ctx context.Context, kind CommentKind, id uint64) error {
	return p.db.WithContext(ctx).Model(commentModel(kind)).Where("id = ?", id).Update("curated_ignored", true).Error
}

// CollectCommentDescendants returns all descendant comment ids under a root comment.
func (p *CommentProviderImpl) CollectCommentDescendants(tx *gorm.DB, kind CommentKind, root uint64) []uint64 {
	var ids []uint64
	children, err := p.pluckCommentChildren(tx, kind, root)
	if err == nil {
		for _, id := range children {
			ids = append(ids, id)
			ids = append(ids, p.CollectCommentDescendants(tx, kind, id)...)
		}
	}
	return ids
}

func (p *CommentProviderImpl) pluckCommentChildren(tx *gorm.DB, kind CommentKind, parentID uint64) ([]uint64, error) {
	var ids []uint64
	var err error
	switch kind {
	case CommentArticle:
		err = tx.Model(&comment.ArticleComment{}).Where("parent_id = ?", parentID).Pluck("id", &ids).Error
	case CommentDynamic:
		err = tx.Model(&comment.DynamicComment{}).Where("parent_id = ?", parentID).Pluck("id", &ids).Error
	default:
		err = tx.Model(&comment.Comment{}).Where("parent_id = ?", parentID).Pluck("id", &ids).Error
	}
	return ids, err
}

// DeleteCommentLikesTx deletes like rows for the given comment ids inside a transaction.
func (p *CommentProviderImpl) DeleteCommentLikesTx(tx *gorm.DB, kind CommentKind, ids []uint64) error {
	return tx.Where("comment_id IN ?", ids).Delete(commentLikeModel(kind)).Error
}

// DeleteCommentDislikesTx deletes dislike rows for the given comment ids inside a transaction.
func (p *CommentProviderImpl) DeleteCommentDislikesTx(tx *gorm.DB, kind CommentKind, ids []uint64) error {
	return tx.Where("comment_id IN ?", ids).Delete(commentDislikeModel(kind)).Error
}

// DeleteCommentRowsTx deletes comment rows for the given ids inside a transaction.
func (p *CommentProviderImpl) DeleteCommentRowsTx(tx *gorm.DB, kind CommentKind, ids []uint64) (int64, error) {
	res := tx.Where("id IN ?", ids).Delete(commentModel(kind))
	return res.RowsAffected, res.Error
}

// LoadCommentLikes returns which of the given comment ids are liked by the viewer.
func (p *CommentProviderImpl) LoadCommentLikes(ctx context.Context, kind CommentKind, viewerID uint64, ids []uint64) (map[uint64]bool, error) {
	result := make(map[uint64]bool)
	if viewerID == 0 || len(ids) == 0 {
		return result, nil
	}
	switch kind {
	case CommentArticle:
		var likes []comment.ArticleCommentLike
		if err := p.db.WithContext(ctx).Where("user_id = ? AND comment_id IN ?", viewerID, ids).Find(&likes).Error; err != nil {
			return nil, err
		}
		for _, lk := range likes {
			result[lk.CommentID] = true
		}
	case CommentDynamic:
		var likes []comment.DynamicCommentLike
		if err := p.db.WithContext(ctx).Where("user_id = ? AND comment_id IN ?", viewerID, ids).Find(&likes).Error; err != nil {
			return nil, err
		}
		for _, lk := range likes {
			result[lk.CommentID] = true
		}
	default:
		var likes []comment.CommentLike
		if err := p.db.WithContext(ctx).Where("user_id = ? AND comment_id IN ?", viewerID, ids).Find(&likes).Error; err != nil {
			return nil, err
		}
		for _, lk := range likes {
			result[lk.CommentID] = true
		}
	}
	return result, nil
}

// LoadCommentDislikes returns which of the given comment ids are disliked by the viewer.
func (p *CommentProviderImpl) LoadCommentDislikes(ctx context.Context, kind CommentKind, viewerID uint64, ids []uint64) (map[uint64]bool, error) {
	result := make(map[uint64]bool)
	if viewerID == 0 || len(ids) == 0 {
		return result, nil
	}
	switch kind {
	case CommentArticle:
		var dis []comment.ArticleCommentDislike
		if err := p.db.WithContext(ctx).Where("user_id = ? AND comment_id IN ?", viewerID, ids).Find(&dis).Error; err != nil {
			return nil, err
		}
		for _, dk := range dis {
			result[dk.CommentID] = true
		}
	case CommentDynamic:
		var dis []comment.DynamicCommentDislike
		if err := p.db.WithContext(ctx).Where("user_id = ? AND comment_id IN ?", viewerID, ids).Find(&dis).Error; err != nil {
			return nil, err
		}
		for _, dk := range dis {
			result[dk.CommentID] = true
		}
	default:
		var dis []comment.CommentDislike
		if err := p.db.WithContext(ctx).Where("user_id = ? AND comment_id IN ?", viewerID, ids).Find(&dis).Error; err != nil {
			return nil, err
		}
		for _, dk := range dis {
			result[dk.CommentID] = true
		}
	}
	return result, nil
}

// GetCommentLikeCount returns the stored like count of a comment.
func (p *CommentProviderImpl) GetCommentLikeCount(ctx context.Context, kind CommentKind, id uint64) (uint64, error) {
	switch kind {
	case CommentArticle:
		var cm comment.ArticleComment
		if err := p.db.WithContext(ctx).First(&cm, id).Error; err != nil {
			return 0, err
		}
		return cm.LikeCount, nil
	case CommentDynamic:
		var cm comment.DynamicComment
		if err := p.db.WithContext(ctx).First(&cm, id).Error; err != nil {
			return 0, err
		}
		return cm.LikeCount, nil
	default:
		var cm comment.Comment
		if err := p.db.WithContext(ctx).First(&cm, id).Error; err != nil {
			return 0, err
		}
		return cm.LikeCount, nil
	}
}

// HasCommentLike reports whether the user liked the comment.
func (p *CommentProviderImpl) HasCommentLike(ctx context.Context, kind CommentKind, userID, commentID uint64) (bool, error) {
	err := p.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).First(commentLikeModel(kind)).Error
	return err == nil, nil
}

// HasCommentDislike reports whether the user disliked the comment.
func (p *CommentProviderImpl) HasCommentDislike(ctx context.Context, kind CommentKind, userID, commentID uint64) (bool, error) {
	err := p.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).First(commentDislikeModel(kind)).Error
	return err == nil, nil
}

// ClearCommentLike removes the user's like on the comment.
func (p *CommentProviderImpl) ClearCommentLike(ctx context.Context, kind CommentKind, userID, commentID uint64) error {
	return p.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).Delete(commentLikeModel(kind)).Error
}

// ClearCommentDislike removes the user's dislike on the comment.
func (p *CommentProviderImpl) ClearCommentDislike(ctx context.Context, kind CommentKind, userID, commentID uint64) error {
	return p.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).Delete(commentDislikeModel(kind)).Error
}

// CreateCommentLike records a like from the user on the comment.
func (p *CommentProviderImpl) CreateCommentLike(ctx context.Context, kind CommentKind, userID, commentID uint64) error {
	switch kind {
	case CommentArticle:
		return p.db.WithContext(ctx).Create(&comment.ArticleCommentLike{UserID: userID, CommentID: commentID}).Error
	case CommentDynamic:
		return p.db.WithContext(ctx).Create(&comment.DynamicCommentLike{UserID: userID, CommentID: commentID}).Error
	default:
		return p.db.WithContext(ctx).Create(&comment.CommentLike{UserID: userID, CommentID: commentID}).Error
	}
}

// DeleteCommentLike removes the user's like row on the comment.
func (p *CommentProviderImpl) DeleteCommentLike(ctx context.Context, kind CommentKind, userID, commentID uint64) error {
	return p.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).Delete(commentLikeModel(kind)).Error
}

// CreateCommentDislike records a dislike from the user on the comment.
func (p *CommentProviderImpl) CreateCommentDislike(ctx context.Context, kind CommentKind, userID, commentID uint64) error {
	switch kind {
	case CommentArticle:
		return p.db.WithContext(ctx).Create(&comment.ArticleCommentDislike{UserID: userID, CommentID: commentID}).Error
	case CommentDynamic:
		return p.db.WithContext(ctx).Create(&comment.DynamicCommentDislike{UserID: userID, CommentID: commentID}).Error
	default:
		return p.db.WithContext(ctx).Create(&comment.CommentDislike{UserID: userID, CommentID: commentID}).Error
	}
}

// DeleteCommentDislike removes the user's dislike row on the comment.
func (p *CommentProviderImpl) DeleteCommentDislike(ctx context.Context, kind CommentKind, userID, commentID uint64) error {
	return p.db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).Delete(commentDislikeModel(kind)).Error
}

// IncrCommentLikeCount adjusts a comment's like count by delta (negative clamps at zero).
func (p *CommentProviderImpl) IncrCommentLikeCount(ctx context.Context, kind CommentKind, commentID uint64, delta int) error {
	if delta < 0 {
		return p.db.WithContext(ctx).Model(commentModel(kind)).Where("id = ?", commentID).
			UpdateColumn("like_count", gorm.Expr("CASE WHEN like_count - ? < 0 THEN 0 ELSE like_count - ? END", -delta, -delta)).Error
	}
	return p.db.WithContext(ctx).Model(commentModel(kind)).Where("id = ?", commentID).
		UpdateColumn("like_count", gorm.Expr("like_count + ?", delta)).Error
}

// CountCommentLikes returns the stored like count of a comment (zero on lookup failure).
func (p *CommentProviderImpl) CountCommentLikes(ctx context.Context, kind CommentKind, commentID uint64) uint64 {
	switch kind {
	case CommentArticle:
		var cm comment.ArticleComment
		_ = p.db.WithContext(ctx).First(&cm, commentID)
		return cm.LikeCount
	case CommentDynamic:
		var cm comment.DynamicComment
		_ = p.db.WithContext(ctx).First(&cm, commentID)
		return cm.LikeCount
	default:
		var cm comment.Comment
		_ = p.db.WithContext(ctx).First(&cm, commentID)
		return cm.LikeCount
	}
}
