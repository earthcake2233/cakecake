package comment

import (
	"cakecake/internal/model/article"
	"cakecake/internal/model/comment"
	"cakecake/internal/model/dynamic"
	"cakecake/internal/model/user"
	"cakecake/internal/model/video"
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"cakecake/internal/logger"
)

// CreatorCommentService handles creator comment management business logic.
type CreatorCommentService struct {
	store CreatorCommentProvider
	rdb   *redis.Client
	log   *zap.Logger
}

// NewCreatorCommentService creates a creator-panel comment service.
func NewCreatorCommentService(db *gorm.DB, rdb *redis.Client, log *zap.Logger) *CreatorCommentService {
	return &CreatorCommentService{store: NewCreatorCommentProvider(db), rdb: rdb, log: log}
}

// CreatorCommentProvider is the creator-panel storage boundary.
// Phase 1: *gorm.DB impl. Phase 2+: replaced by gRPC client / per-domain store.
type CreatorCommentProvider interface {
	ListCreatorVideoComments(ctx context.Context, q CreatorVideoCommentQuery) (*CreatorVideoCommentResult, error)
	ListCreatorArticleComments(ctx context.Context, q CreatorArticleCommentQuery) (*CreatorArticleCommentResult, error)
	ListCreatorDynamicComments(ctx context.Context, q CreatorDynamicCommentQuery) (*CreatorDynamicCommentResult, error)
	BatchFetchVideos(ctx context.Context, ids []uint64) (map[uint64]video.Video, error)
	BatchFetchUsers(ctx context.Context, ids []uint64) (map[uint64]user.User, error)
	BatchFetchComments(ctx context.Context, ids []uint64) (map[uint64]comment.Comment, error)
	BatchFetchArticleComments(ctx context.Context, ids []uint64) (map[uint64]comment.ArticleComment, error)
	BatchFetchDynamicComments(ctx context.Context, ids []uint64) (map[uint64]comment.DynamicComment, error)
	BatchFetchArticles(ctx context.Context, ids []uint64) (map[uint64]article.Article, error)
	BatchFetchDynamics(ctx context.Context, ids []uint64) (map[uint64]dynamic.UserDynamic, error)
	BatchFetchUserLikesForComments(ctx context.Context, userID uint64, commentIDs []uint64) (map[uint64]bool, error)
	BatchFetchUserLikesForArticleComments(ctx context.Context, userID uint64, commentIDs []uint64) (map[uint64]bool, error)
	BatchFetchUserLikesForDynamicComments(ctx context.Context, userID uint64, commentIDs []uint64) (map[uint64]bool, error)
	CommentReplyCounts(ctx context.Context, ids []uint64) map[uint64]uint64
	ArticleCommentReplyCounts(ctx context.Context, ids []uint64) map[uint64]uint64
	DynamicCommentReplyCounts(ctx context.Context, ids []uint64) map[uint64]uint64
	CheckVideoOwnership(ctx context.Context, videoID, userID uint64) (bool, error)
	CheckArticleOwnership(ctx context.Context, articleID, userID uint64) (bool, error)
	CheckDynamicOwnership(ctx context.Context, dynamicID, userID uint64) (bool, error)
}

// CreatorCommentProviderImpl implements CreatorCommentProvider using *gorm.DB (Phase 1 monolith).
type CreatorCommentProviderImpl struct {
	db *gorm.DB
}

// NewCreatorCommentProvider creates a gorm-backed creator comment store.
func NewCreatorCommentProvider(db *gorm.DB) *CreatorCommentProviderImpl {
	return &CreatorCommentProviderImpl{db: db}
}

// CreatorVideoCommentQuery holds filter params for listing video comments.
type CreatorVideoCommentQuery struct {
	UserID        uint64
	Page          int
	PageSize      int
	SortKey       string
	Pending       bool
	PendingStatus string
	PendingScope  string
	Keyword       string
	FilterVideoID uint64
	ViewerID      uint64
}

// CreatorVideoCommentResult holds query results for video comments.
type CreatorVideoCommentResult struct {
	Comments      []comment.Comment
	Total         int64
	VideoIDs      []uint64
	UserIDs       []uint64
	ParentIDs     []uint64
	CommentIDs    []uint64
	LikedByViewer map[uint64]bool
}

// creatorCommentListQuery is the shared filter surface for creator-panel comment lists.
type creatorCommentListQuery struct {
	UserID        uint64
	Page          int
	PageSize      int
	SortKey       string
	Pending       bool
	PendingStatus string
	PendingScope  string
	Keyword       string
	FilterMediaID uint64
	ViewerID      uint64
}

// creatorCommentListResult is the row-level query result before per-domain projection.
type creatorCommentListResult[R any] struct {
	Comments      []R
	Total         int64
	MediaIDs      []uint64
	UserIDs       []uint64
	ParentIDs     []uint64
	CommentIDs    []uint64
	LikedByViewer map[uint64]bool
}

// creatorCommentListSpec parameterizes the shared creator list query for one comment domain.
type creatorCommentListSpec[R any, L any] struct {
	rowModel        R
	table           string // comments / article_comments / dynamic_comments
	mediaFK         string // owner media FK column, e.g. comments.video_id
	mediaJoin       string // join to owner media; must contain one ? for the creator user id
	curatedColumn   string // media table comments_curated column, e.g. videos.comments_curated
	keywordWhere    string // content/title LIKE expression with ? placeholders
	keywordArgCount int
	idOf            func(R) uint64
	mediaIDOf       func(R) uint64
	userIDOf        func(R) uint64
	parentIDOf      func(R) uint64
	likeIDOf        func(L) uint64
}

// listCreatorComments runs the shared creator-panel comment query (ownership join, pending
// filter, keyword, sort, pagination, viewer-like annotation) for one comment domain.
func listCreatorComments[R any, L any](ctx context.Context, db *gorm.DB, q creatorCommentListQuery, spec creatorCommentListSpec[R, L]) (*creatorCommentListResult[R], error) {
	base := db.WithContext(ctx).Model(&spec.rowModel).Joins(spec.mediaJoin, q.UserID)

	if q.Pending {
		switch q.PendingStatus {
		case "ignored":
			base = base.Where(spec.table+".curated_ignored = ?", true).
				Where(spec.table+".approved = ?", false)
		default:
			base = base.Where(spec.curatedColumn+" = ?", true).
				Where(spec.table+".approved = ?", false)
			switch q.PendingStatus {
			case "all":
			default: // unprocessed
				base = base.Where(spec.table+".curated_ignored = ?", false)
			}
		}
		switch q.PendingScope {
		case "root":
			base = base.Where(spec.table+".parent_id = ?", 0)
		case "reply":
			base = base.Where(spec.table+".parent_id > ?", 0)
		}
	} else {
		base = base.Where(spec.table+".approved = ?", true)
	}

	if q.FilterMediaID > 0 {
		base = base.Where(spec.mediaFK+" = ?", q.FilterMediaID)
	}
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		args := make([]any, spec.keywordArgCount)
		for i := range args {
			args[i] = like
		}
		base = base.Where(spec.keywordWhere, args...)
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, err
	}

	order := spec.table + ".created_at DESC, " + spec.table + ".id DESC"
	switch q.SortKey {
	case "earliest":
		order = spec.table + ".created_at ASC, " + spec.table + ".id ASC"
	case "likes":
		order = spec.table + ".like_count DESC, " + spec.table + ".id DESC"
	case "replies":
		order = fmt.Sprintf("(SELECT COUNT(*) FROM %s AS r WHERE r.parent_id = %s.id) DESC, %s.id DESC", spec.table, spec.table, spec.table)
	}

	offset := (q.Page - 1) * q.PageSize
	var list []R
	if err := base.Order(order).Offset(offset).Limit(q.PageSize).Find(&list).Error; err != nil {
		return nil, err
	}

	result := &creatorCommentListResult[R]{
		Comments:      list,
		Total:         total,
		MediaIDs:      make([]uint64, 0, len(list)),
		UserIDs:       make([]uint64, 0, len(list)),
		ParentIDs:     make([]uint64, 0),
		CommentIDs:    make([]uint64, len(list)),
		LikedByViewer: make(map[uint64]bool),
	}
	for i := range list {
		result.CommentIDs[i] = spec.idOf(list[i])
		result.MediaIDs = append(result.MediaIDs, spec.mediaIDOf(list[i]))
		result.UserIDs = append(result.UserIDs, spec.userIDOf(list[i]))
		if pid := spec.parentIDOf(list[i]); pid > 0 {
			result.ParentIDs = append(result.ParentIDs, pid)
		}
	}
	if q.ViewerID > 0 && len(list) > 0 {
		var likes []L
		if err := db.WithContext(ctx).Where("user_id = ? AND comment_id IN ?", q.ViewerID, result.CommentIDs).Find(&likes).Error; err != nil && logger.L != nil {
			logger.L.Warn("creator comment: load viewer likes failed", zap.Uint64("viewer_id", q.ViewerID), zap.Error(err))
		}
		for _, lk := range likes {
			result.LikedByViewer[spec.likeIDOf(lk)] = true
		}
	}
	return result, nil
}

// ListCreatorVideoComments lists comments on the creator's videos with filters.
func (p *CreatorCommentProviderImpl) ListCreatorVideoComments(ctx context.Context, q CreatorVideoCommentQuery) (*CreatorVideoCommentResult, error) {
	res, err := listCreatorComments(ctx, p.db, creatorCommentListQuery{
		UserID: q.UserID, Page: q.Page, PageSize: q.PageSize, SortKey: q.SortKey,
		Pending: q.Pending, PendingStatus: q.PendingStatus, PendingScope: q.PendingScope,
		Keyword: q.Keyword, FilterMediaID: q.FilterVideoID, ViewerID: q.ViewerID,
	}, creatorCommentListSpec[comment.Comment, comment.CommentLike]{
		rowModel: comment.Comment{},
		table:    "comments", mediaFK: "comments.video_id",
		mediaJoin:       "INNER JOIN videos ON videos.id = comments.video_id AND videos.user_id = ?",
		curatedColumn:   "videos.comments_curated",
		keywordWhere:    "comments.content LIKE ? OR videos.title LIKE ?",
		keywordArgCount: 2,
		idOf:            func(cm comment.Comment) uint64 { return cm.ID },
		mediaIDOf:       func(cm comment.Comment) uint64 { return cm.VideoID },
		userIDOf:        func(cm comment.Comment) uint64 { return cm.UserID },
		parentIDOf:      func(cm comment.Comment) uint64 { return cm.ParentID },
		likeIDOf:        func(lk comment.CommentLike) uint64 { return lk.CommentID },
	})
	if err != nil {
		return nil, err
	}
	return &CreatorVideoCommentResult{
		Comments: res.Comments, Total: res.Total, VideoIDs: res.MediaIDs,
		UserIDs: res.UserIDs, ParentIDs: res.ParentIDs, CommentIDs: res.CommentIDs,
		LikedByViewer: res.LikedByViewer,
	}, nil
}

// BatchFetchVideos returns a map of video id to Video for the given ids.
func (p *CreatorCommentProviderImpl) BatchFetchVideos(ctx context.Context, ids []uint64) (map[uint64]video.Video, error) {
	result := make(map[uint64]video.Video, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	var list []video.Video
	if err := p.db.WithContext(ctx).Where("id IN ?", ids).Find(&list).Error; err != nil {
		return nil, err
	}
	for i := range list {
		result[list[i].ID] = list[i]
	}
	return result, nil
}

// BatchFetchUsers returns a map of user id to User for the given ids.
func (p *CreatorCommentProviderImpl) BatchFetchUsers(ctx context.Context, ids []uint64) (map[uint64]user.User, error) {
	result := make(map[uint64]user.User, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	var list []user.User
	if err := p.db.WithContext(ctx).Where("id IN ?", ids).Find(&list).Error; err != nil {
		return nil, err
	}
	for i := range list {
		result[list[i].ID] = list[i]
	}
	return result, nil
}

// BatchFetchComments returns a map of comment id to Comment for the given ids.
func (p *CreatorCommentProviderImpl) BatchFetchComments(ctx context.Context, ids []uint64) (map[uint64]comment.Comment, error) {
	result := make(map[uint64]comment.Comment, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	var list []comment.Comment
	if err := p.db.WithContext(ctx).Where("id IN ?", ids).Find(&list).Error; err != nil {
		return nil, err
	}
	for i := range list {
		result[list[i].ID] = list[i]
	}
	return result, nil
}

// BatchFetchArticleComments returns a map of comment id to ArticleComment for the given ids.
func (p *CreatorCommentProviderImpl) BatchFetchArticleComments(ctx context.Context, ids []uint64) (map[uint64]comment.ArticleComment, error) {
	result := make(map[uint64]comment.ArticleComment, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	var list []comment.ArticleComment
	if err := p.db.WithContext(ctx).Where("id IN ?", ids).Find(&list).Error; err != nil {
		return nil, err
	}
	for i := range list {
		result[list[i].ID] = list[i]
	}
	return result, nil
}

// BatchFetchDynamicComments returns a map of comment id to DynamicComment for the given ids.
func (p *CreatorCommentProviderImpl) BatchFetchDynamicComments(ctx context.Context, ids []uint64) (map[uint64]comment.DynamicComment, error) {
	result := make(map[uint64]comment.DynamicComment, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	var list []comment.DynamicComment
	if err := p.db.WithContext(ctx).Where("id IN ?", ids).Find(&list).Error; err != nil {
		return nil, err
	}
	for i := range list {
		result[list[i].ID] = list[i]
	}
	return result, nil
}

// BatchFetchUserLikesForComments returns a map of comment id to liked-by-user status.
func (p *CreatorCommentProviderImpl) BatchFetchUserLikesForComments(ctx context.Context, userID uint64, commentIDs []uint64) (map[uint64]bool, error) {
	result := make(map[uint64]bool, len(commentIDs))
	if len(commentIDs) == 0 || userID == 0 {
		return result, nil
	}
	var likes []comment.CommentLike
	if err := p.db.WithContext(ctx).Where("user_id = ? AND comment_id IN ?", userID, commentIDs).Find(&likes).Error; err != nil {
		return nil, err
	}
	for _, lk := range likes {
		result[lk.CommentID] = true
	}
	return result, nil
}

// BatchFetchUserLikesForArticleComments returns a map of article comment id to liked-by-user status.
func (p *CreatorCommentProviderImpl) BatchFetchUserLikesForArticleComments(ctx context.Context, userID uint64, commentIDs []uint64) (map[uint64]bool, error) {
	result := make(map[uint64]bool, len(commentIDs))
	if len(commentIDs) == 0 || userID == 0 {
		return result, nil
	}
	var likes []comment.ArticleCommentLike
	if err := p.db.WithContext(ctx).Where("user_id = ? AND comment_id IN ?", userID, commentIDs).Find(&likes).Error; err != nil {
		return nil, err
	}
	for _, lk := range likes {
		result[lk.CommentID] = true
	}
	return result, nil
}

// BatchFetchUserLikesForDynamicComments returns a map of dynamic comment id to liked-by-user status.
func (p *CreatorCommentProviderImpl) BatchFetchUserLikesForDynamicComments(ctx context.Context, userID uint64, commentIDs []uint64) (map[uint64]bool, error) {
	result := make(map[uint64]bool, len(commentIDs))
	if len(commentIDs) == 0 || userID == 0 {
		return result, nil
	}
	var likes []comment.DynamicCommentLike
	if err := p.db.WithContext(ctx).Where("user_id = ? AND comment_id IN ?", userID, commentIDs).Find(&likes).Error; err != nil {
		return nil, err
	}
	for _, lk := range likes {
		result[lk.CommentID] = true
	}
	return result, nil
}

// CommentReplyCounts returns reply count per parent comment id.
func (p *CreatorCommentProviderImpl) CommentReplyCounts(ctx context.Context, ids []uint64) map[uint64]uint64 {
	out := make(map[uint64]uint64, len(ids))
	if len(ids) == 0 {
		return out
	}
	type row struct {
		ParentID uint64
		C        int64
	}
	var rows []row
	_ = p.db.WithContext(ctx).Model(&comment.Comment{}).
		Select("parent_id, COUNT(*) AS c").
		Where("parent_id IN ?", ids).
		Group("parent_id").
		Scan(&rows).Error
	for _, r := range rows {
		if r.C > 0 {
			out[r.ParentID] = uint64(r.C)
		}
	}
	return out
}

// ArticleCommentReplyCounts returns reply count per parent article comment id.
func (p *CreatorCommentProviderImpl) ArticleCommentReplyCounts(ctx context.Context, ids []uint64) map[uint64]uint64 {
	out := make(map[uint64]uint64, len(ids))
	if len(ids) == 0 {
		return out
	}
	type row struct {
		ParentID uint64
		C        int64
	}
	var rows []row
	_ = p.db.WithContext(ctx).Model(&comment.ArticleComment{}).
		Select("parent_id, COUNT(*) AS c").
		Where("parent_id IN ?", ids).
		Group("parent_id").
		Scan(&rows).Error
	for _, r := range rows {
		if r.C > 0 {
			out[r.ParentID] = uint64(r.C)
		}
	}
	return out
}

// DynamicCommentReplyCounts returns reply count per parent dynamic comment id.
func (p *CreatorCommentProviderImpl) DynamicCommentReplyCounts(ctx context.Context, ids []uint64) map[uint64]uint64 {
	out := make(map[uint64]uint64, len(ids))
	if len(ids) == 0 {
		return out
	}
	type row struct {
		ParentID uint64
		C        int64
	}
	var rows []row
	_ = p.db.WithContext(ctx).Model(&comment.DynamicComment{}).
		Select("parent_id, COUNT(*) AS c").
		Where("parent_id IN ?", ids).
		Group("parent_id").
		Scan(&rows).Error
	for _, r := range rows {
		if r.C > 0 {
			out[r.ParentID] = uint64(r.C)
		}
	}
	return out
}

// CheckVideoOwnership checks if a video with the given id belongs to the given user.
func (p *CreatorCommentProviderImpl) CheckVideoOwnership(ctx context.Context, videoID, userID uint64) (bool, error) {
	var owned video.Video
	err := p.db.WithContext(ctx).Where("id = ? AND user_id = ?", videoID, userID).First(&owned).Error
	if err == gorm.ErrRecordNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// CheckArticleOwnership checks if an article with the given id belongs to the given user.
func (p *CreatorCommentProviderImpl) CheckArticleOwnership(ctx context.Context, articleID, userID uint64) (bool, error) {
	var owned article.Article
	err := p.db.WithContext(ctx).Where("id = ? AND user_id = ?", articleID, userID).First(&owned).Error
	if err == gorm.ErrRecordNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// CheckDynamicOwnership checks if a dynamic with the given id belongs to the given user.
func (p *CreatorCommentProviderImpl) CheckDynamicOwnership(ctx context.Context, dynamicID, userID uint64) (bool, error) {
	var owned dynamic.UserDynamic
	err := p.db.WithContext(ctx).Where("id = ? AND user_id = ?", dynamicID, userID).First(&owned).Error
	if err == gorm.ErrRecordNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// CreatorArticleCommentQuery holds filter params for listing article comments.
type CreatorArticleCommentQuery struct {
	UserID          uint64
	Page            int
	PageSize        int
	SortKey         string
	Pending         bool
	PendingStatus   string
	PendingScope    string
	Keyword         string
	FilterArticleID uint64
	ViewerID        uint64
}

// CreatorArticleCommentResult holds query results for article comments.
type CreatorArticleCommentResult struct {
	Comments      []comment.ArticleComment
	Total         int64
	ArticleIDs    []uint64
	UserIDs       []uint64
	ParentIDs     []uint64
	CommentIDs    []uint64
	LikedByViewer map[uint64]bool
}

// ListCreatorArticleComments lists comments on the creator's articles with filters.
func (p *CreatorCommentProviderImpl) ListCreatorArticleComments(ctx context.Context, q CreatorArticleCommentQuery) (*CreatorArticleCommentResult, error) {
	res, err := listCreatorComments(ctx, p.db, creatorCommentListQuery{
		UserID: q.UserID, Page: q.Page, PageSize: q.PageSize, SortKey: q.SortKey,
		Pending: q.Pending, PendingStatus: q.PendingStatus, PendingScope: q.PendingScope,
		Keyword: q.Keyword, FilterMediaID: q.FilterArticleID, ViewerID: q.ViewerID,
	}, creatorCommentListSpec[comment.ArticleComment, comment.ArticleCommentLike]{
		rowModel: comment.ArticleComment{},
		table:    "article_comments", mediaFK: "article_comments.article_id",
		mediaJoin:       "INNER JOIN articles ON articles.id = article_comments.article_id AND articles.user_id = ?",
		curatedColumn:   "articles.comments_curated",
		keywordWhere:    "article_comments.content LIKE ? OR articles.title LIKE ?",
		keywordArgCount: 2,
		idOf:            func(cm comment.ArticleComment) uint64 { return cm.ID },
		mediaIDOf:       func(cm comment.ArticleComment) uint64 { return cm.ArticleID },
		userIDOf:        func(cm comment.ArticleComment) uint64 { return cm.UserID },
		parentIDOf:      func(cm comment.ArticleComment) uint64 { return cm.ParentID },
		likeIDOf:        func(lk comment.ArticleCommentLike) uint64 { return lk.CommentID },
	})
	if err != nil {
		return nil, err
	}
	return &CreatorArticleCommentResult{
		Comments: res.Comments, Total: res.Total, ArticleIDs: res.MediaIDs,
		UserIDs: res.UserIDs, ParentIDs: res.ParentIDs, CommentIDs: res.CommentIDs,
		LikedByViewer: res.LikedByViewer,
	}, nil
}

// BatchFetchArticles returns a map of article id to Article for the given ids.
func (p *CreatorCommentProviderImpl) BatchFetchArticles(ctx context.Context, ids []uint64) (map[uint64]article.Article, error) {
	result := make(map[uint64]article.Article, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	var list []article.Article
	if err := p.db.WithContext(ctx).Where("id IN ?", ids).Find(&list).Error; err != nil {
		return nil, err
	}
	for i := range list {
		result[list[i].ID] = list[i]
	}

	return result, nil
}

// CreatorDynamicCommentQuery holds filter params for listing dynamic comments.
type CreatorDynamicCommentQuery struct {
	UserID          uint64
	Page            int
	PageSize        int
	SortKey         string
	Pending         bool
	PendingStatus   string
	PendingScope    string
	Keyword         string
	FilterDynamicID uint64
	ViewerID        uint64
}

// CreatorDynamicCommentResult holds query results for dynamic comments.
type CreatorDynamicCommentResult struct {
	Comments      []comment.DynamicComment
	Total         int64
	DynamicIDs    []uint64
	UserIDs       []uint64
	ParentIDs     []uint64
	CommentIDs    []uint64
	LikedByViewer map[uint64]bool
}

// ListCreatorDynamicComments lists comments on the creator's dynamics with filters.
func (p *CreatorCommentProviderImpl) ListCreatorDynamicComments(ctx context.Context, q CreatorDynamicCommentQuery) (*CreatorDynamicCommentResult, error) {
	res, err := listCreatorComments(ctx, p.db, creatorCommentListQuery{
		UserID: q.UserID, Page: q.Page, PageSize: q.PageSize, SortKey: q.SortKey,
		Pending: q.Pending, PendingStatus: q.PendingStatus, PendingScope: q.PendingScope,
		Keyword: q.Keyword, FilterMediaID: q.FilterDynamicID, ViewerID: q.ViewerID,
	}, creatorCommentListSpec[comment.DynamicComment, comment.DynamicCommentLike]{
		rowModel: comment.DynamicComment{},
		table:    "dynamic_comments", mediaFK: "dynamic_comments.dynamic_id",
		mediaJoin:       "INNER JOIN user_dynamics ON user_dynamics.id = dynamic_comments.dynamic_id AND user_dynamics.user_id = ?",
		curatedColumn:   "user_dynamics.comments_curated",
		keywordWhere:    "dynamic_comments.content LIKE ? OR user_dynamics.title LIKE ? OR user_dynamics.content LIKE ?",
		keywordArgCount: 3,
		idOf:            func(cm comment.DynamicComment) uint64 { return cm.ID },
		mediaIDOf:       func(cm comment.DynamicComment) uint64 { return cm.DynamicID },
		userIDOf:        func(cm comment.DynamicComment) uint64 { return cm.UserID },
		parentIDOf:      func(cm comment.DynamicComment) uint64 { return cm.ParentID },
		likeIDOf:        func(lk comment.DynamicCommentLike) uint64 { return lk.CommentID },
	})
	if err != nil {
		return nil, err
	}
	return &CreatorDynamicCommentResult{
		Comments: res.Comments, Total: res.Total, DynamicIDs: res.MediaIDs,
		UserIDs: res.UserIDs, ParentIDs: res.ParentIDs, CommentIDs: res.CommentIDs,
		LikedByViewer: res.LikedByViewer,
	}, nil
}

// BatchFetchDynamics returns a map of dynamic id to UserDynamic for the given ids.
func (p *CreatorCommentProviderImpl) BatchFetchDynamics(ctx context.Context, ids []uint64) (map[uint64]dynamic.UserDynamic, error) {
	result := make(map[uint64]dynamic.UserDynamic, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	var list []dynamic.UserDynamic
	if err := p.db.WithContext(ctx).Where("id IN ?", ids).Find(&list).Error; err != nil {
		return nil, err
	}
	for i := range list {
		result[list[i].ID] = list[i]
	}
	return result, nil
}

// ListCreatorVideoComments lists comments on the creator's videos with filters.
func (s *CreatorCommentService) ListCreatorVideoComments(ctx context.Context, q CreatorVideoCommentQuery) (*CreatorVideoCommentResult, error) {
	return s.store.ListCreatorVideoComments(ctx, q)
}

// ListCreatorArticleComments lists comments on the creator's articles with filters.
func (s *CreatorCommentService) ListCreatorArticleComments(ctx context.Context, q CreatorArticleCommentQuery) (*CreatorArticleCommentResult, error) {
	return s.store.ListCreatorArticleComments(ctx, q)
}

// ListCreatorDynamicComments lists comments on the creator's dynamics with filters.
func (s *CreatorCommentService) ListCreatorDynamicComments(ctx context.Context, q CreatorDynamicCommentQuery) (*CreatorDynamicCommentResult, error) {
	return s.store.ListCreatorDynamicComments(ctx, q)
}

// BatchFetchVideos returns a map of video id to Video for the given ids.
func (s *CreatorCommentService) BatchFetchVideos(ctx context.Context, ids []uint64) (map[uint64]video.Video, error) {
	return s.store.BatchFetchVideos(ctx, ids)
}

// BatchFetchUsers returns a map of user id to User for the given ids.
func (s *CreatorCommentService) BatchFetchUsers(ctx context.Context, ids []uint64) (map[uint64]user.User, error) {
	return s.store.BatchFetchUsers(ctx, ids)
}

// BatchFetchComments returns a map of comment id to Comment for the given ids.
func (s *CreatorCommentService) BatchFetchComments(ctx context.Context, ids []uint64) (map[uint64]comment.Comment, error) {
	return s.store.BatchFetchComments(ctx, ids)
}

// BatchFetchArticleComments returns a map of comment id to ArticleComment for the given ids.
func (s *CreatorCommentService) BatchFetchArticleComments(ctx context.Context, ids []uint64) (map[uint64]comment.ArticleComment, error) {
	return s.store.BatchFetchArticleComments(ctx, ids)
}

// BatchFetchDynamicComments returns a map of comment id to DynamicComment for the given ids.
func (s *CreatorCommentService) BatchFetchDynamicComments(ctx context.Context, ids []uint64) (map[uint64]comment.DynamicComment, error) {
	return s.store.BatchFetchDynamicComments(ctx, ids)
}

// BatchFetchArticles returns a map of article id to Article for the given ids.
func (s *CreatorCommentService) BatchFetchArticles(ctx context.Context, ids []uint64) (map[uint64]article.Article, error) {
	return s.store.BatchFetchArticles(ctx, ids)
}

// BatchFetchDynamics returns a map of dynamic id to UserDynamic for the given ids.
func (s *CreatorCommentService) BatchFetchDynamics(ctx context.Context, ids []uint64) (map[uint64]dynamic.UserDynamic, error) {
	return s.store.BatchFetchDynamics(ctx, ids)
}

// BatchFetchUserLikesForComments returns a map of comment id to liked-by-user status.
func (s *CreatorCommentService) BatchFetchUserLikesForComments(ctx context.Context, userID uint64, commentIDs []uint64) (map[uint64]bool, error) {
	return s.store.BatchFetchUserLikesForComments(ctx, userID, commentIDs)
}

// BatchFetchUserLikesForArticleComments returns a map of article comment id to liked-by-user status.
func (s *CreatorCommentService) BatchFetchUserLikesForArticleComments(ctx context.Context, userID uint64, commentIDs []uint64) (map[uint64]bool, error) {
	return s.store.BatchFetchUserLikesForArticleComments(ctx, userID, commentIDs)
}

// BatchFetchUserLikesForDynamicComments returns a map of dynamic comment id to liked-by-user status.
func (s *CreatorCommentService) BatchFetchUserLikesForDynamicComments(ctx context.Context, userID uint64, commentIDs []uint64) (map[uint64]bool, error) {
	return s.store.BatchFetchUserLikesForDynamicComments(ctx, userID, commentIDs)
}

// CommentReplyCounts returns reply count per parent comment id.
func (s *CreatorCommentService) CommentReplyCounts(ctx context.Context, ids []uint64) map[uint64]uint64 {
	return s.store.CommentReplyCounts(ctx, ids)
}

// ArticleCommentReplyCounts returns reply count per parent article comment id.
func (s *CreatorCommentService) ArticleCommentReplyCounts(ctx context.Context, ids []uint64) map[uint64]uint64 {
	return s.store.ArticleCommentReplyCounts(ctx, ids)
}

// DynamicCommentReplyCounts returns reply count per parent dynamic comment id.
func (s *CreatorCommentService) DynamicCommentReplyCounts(ctx context.Context, ids []uint64) map[uint64]uint64 {
	return s.store.DynamicCommentReplyCounts(ctx, ids)
}

// CheckVideoOwnership checks if a video with the given id belongs to the given user.
func (s *CreatorCommentService) CheckVideoOwnership(ctx context.Context, videoID, userID uint64) (bool, error) {
	return s.store.CheckVideoOwnership(ctx, videoID, userID)
}

// CheckArticleOwnership checks if an article with the given id belongs to the given user.
func (s *CreatorCommentService) CheckArticleOwnership(ctx context.Context, articleID, userID uint64) (bool, error) {
	return s.store.CheckArticleOwnership(ctx, articleID, userID)
}

// CheckDynamicOwnership checks if a dynamic with the given id belongs to the given user.
func (s *CreatorCommentService) CheckDynamicOwnership(ctx context.Context, dynamicID, userID uint64) (bool, error) {
	return s.store.CheckDynamicOwnership(ctx, dynamicID, userID)
}
