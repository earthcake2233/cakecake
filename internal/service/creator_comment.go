package service

import (
	"minibili/internal/model/article"
	"minibili/internal/model/comment"
	"minibili/internal/model/dynamic"
	"minibili/internal/model/user"
	"minibili/internal/model/video"
	"context"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

)

// CreatorCommentService handles creator comment management business logic.
type CreatorCommentService struct {
	db  *gorm.DB
	rdb *redis.Client
	log *zap.Logger
}

func NewCreatorCommentService(db *gorm.DB, rdb *redis.Client, log *zap.Logger) *CreatorCommentService {
	return &CreatorCommentService{db: db, rdb: rdb, log: log}
}

// CreatorVideoCommentQuery holds filter params for listing video comments.
type CreatorVideoCommentQuery struct {
	UserID         uint64
	Page           int
	PageSize       int
	SortKey        string
	Pending        bool
	PendingStatus  string
	PendingScope   string
	Keyword        string
	FilterVideoID  uint64
	ViewerID       uint64
}

// CreatorVideoCommentResult holds query results for video comments.
type CreatorVideoCommentResult struct {
	Comments    []comment.Comment
	Total       int64
	VideoIDs    []uint64
	UserIDs     []uint64
	ParentIDs   []uint64
	CommentIDs  []uint64
	LikedByViewer map[uint64]bool
}

// ListCreatorVideoComments lists comments on the creator's videos with filters.
func (s *CreatorCommentService) ListCreatorVideoComments(ctx context.Context, q CreatorVideoCommentQuery) (*CreatorVideoCommentResult, error) {
	base := s.db.WithContext(ctx).Model(&comment.Comment{}).
		Joins("INNER JOIN videos ON videos.id = comments.video_id AND videos.user_id = ?", q.UserID)

	if q.Pending {
		switch q.PendingStatus {
		case "ignored":
			base = base.Where("comments.curated_ignored = ?", true).
				Where("comments.approved = ?", false)
		default:
			base = base.Where("videos.comments_curated = ?", true).
				Where("comments.approved = ?", false)
			switch q.PendingStatus {
			case "all":
			default: // unprocessed
				base = base.Where("comments.curated_ignored = ?", false)
			}
		}
		switch q.PendingScope {
		case "root":
			base = base.Where("comments.parent_id = ?", 0)
		case "reply":
			base = base.Where("comments.parent_id > ?", 0)
		}
	} else {
		base = base.Where("comments.approved = ?", true)
	}

	if q.FilterVideoID > 0 {
		base = base.Where("comments.video_id = ?", q.FilterVideoID)
	}
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		base = base.Where("comments.content LIKE ? OR videos.title LIKE ?", like, like)
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, err
	}

	order := "comments.created_at DESC, comments.id DESC"
	switch q.SortKey {
	case "earliest":
		order = "comments.created_at ASC, comments.id ASC"
	case "likes":
		order = "comments.like_count DESC, comments.id DESC"
	case "replies":
		order = "(SELECT COUNT(*) FROM comments AS r WHERE r.parent_id = comments.id) DESC, comments.id DESC"
	}

	offset := (q.Page - 1) * q.PageSize
	var list []comment.Comment
	if err := base.Order(order).Offset(offset).Limit(q.PageSize).Find(&list).Error; err != nil {
		return nil, err
	}

	result := &CreatorVideoCommentResult{
		Comments:   list,
		Total:      total,
		VideoIDs:   make([]uint64, 0, len(list)),
		UserIDs:    make([]uint64, 0, len(list)),
		ParentIDs:  make([]uint64, 0),
		CommentIDs: make([]uint64, len(list)),
		LikedByViewer: make(map[uint64]bool),
	}
	for i, cm := range list {
		result.CommentIDs[i] = cm.ID
		result.VideoIDs = append(result.VideoIDs, cm.VideoID)
		result.UserIDs = append(result.UserIDs, cm.UserID)
		if cm.ParentID > 0 {
			result.ParentIDs = append(result.ParentIDs, cm.ParentID)
		}
	}
	if q.ViewerID > 0 && len(list) > 0 {
		var likes []comment.CommentLike
		_ = s.db.WithContext(ctx).Where("user_id = ? AND comment_id IN ?", q.ViewerID, result.CommentIDs).Find(&likes).Error
		for _, lk := range likes {
			result.LikedByViewer[lk.CommentID] = true
		}
	}
	return result, nil
}

// BatchFetchVideos returns a map of video id to Video for the given ids.
func (s *CreatorCommentService) BatchFetchVideos(ctx context.Context, ids []uint64) (map[uint64]video.Video, error) {
	result := make(map[uint64]video.Video, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	var list []video.Video
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&list).Error; err != nil {
		return nil, err
	}
	for i := range list {
		result[list[i].ID] = list[i]
	}
	return result, nil
}

// BatchFetchUsers returns a map of user id to User for the given ids.
func (s *CreatorCommentService) BatchFetchUsers(ctx context.Context, ids []uint64) (map[uint64]user.User, error) {
	result := make(map[uint64]user.User, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	var list []user.User
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&list).Error; err != nil {
		return nil, err
	}
	for i := range list {
		result[list[i].ID] = list[i]
	}
	return result, nil
}

// BatchFetchComments returns a map of comment id to Comment for the given ids.
func (s *CreatorCommentService) BatchFetchComments(ctx context.Context, ids []uint64) (map[uint64]comment.Comment, error) {
	result := make(map[uint64]comment.Comment, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	var list []comment.Comment
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&list).Error; err != nil {
		return nil, err
	}
	for i := range list {
		result[list[i].ID] = list[i]
	}
	return result, nil
}

// BatchFetchArticleComments returns a map of comment id to ArticleComment for the given ids.
func (s *CreatorCommentService) BatchFetchArticleComments(ctx context.Context, ids []uint64) (map[uint64]comment.ArticleComment, error) {
	result := make(map[uint64]comment.ArticleComment, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	var list []comment.ArticleComment
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&list).Error; err != nil {
		return nil, err
	}
	for i := range list {
		result[list[i].ID] = list[i]
	}
	return result, nil
}

// BatchFetchDynamicComments returns a map of comment id to DynamicComment for the given ids.
func (s *CreatorCommentService) BatchFetchDynamicComments(ctx context.Context, ids []uint64) (map[uint64]comment.DynamicComment, error) {
	result := make(map[uint64]comment.DynamicComment, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	var list []comment.DynamicComment
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&list).Error; err != nil {
		return nil, err
	}
	for i := range list {
		result[list[i].ID] = list[i]
	}
	return result, nil
}

// BatchFetchUserLikesForComments returns a map of comment id to liked-by-user status.
func (s *CreatorCommentService) BatchFetchUserLikesForComments(ctx context.Context, userID uint64, commentIDs []uint64) (map[uint64]bool, error) {
	result := make(map[uint64]bool, len(commentIDs))
	if len(commentIDs) == 0 || userID == 0 {
		return result, nil
	}
	var likes []comment.CommentLike
	if err := s.db.WithContext(ctx).Where("user_id = ? AND comment_id IN ?", userID, commentIDs).Find(&likes).Error; err != nil {
		return nil, err
	}
	for _, lk := range likes {
		result[lk.CommentID] = true
	}
	return result, nil
}

// BatchFetchUserLikesForArticleComments returns a map of article comment id to liked-by-user status.
func (s *CreatorCommentService) BatchFetchUserLikesForArticleComments(ctx context.Context, userID uint64, commentIDs []uint64) (map[uint64]bool, error) {
	result := make(map[uint64]bool, len(commentIDs))
	if len(commentIDs) == 0 || userID == 0 {
		return result, nil
	}
	var likes []comment.ArticleCommentLike
	if err := s.db.WithContext(ctx).Where("user_id = ? AND comment_id IN ?", userID, commentIDs).Find(&likes).Error; err != nil {
		return nil, err
	}
	for _, lk := range likes {
		result[lk.CommentID] = true
	}
	return result, nil
}

// BatchFetchUserLikesForDynamicComments returns a map of dynamic comment id to liked-by-user status.
func (s *CreatorCommentService) BatchFetchUserLikesForDynamicComments(ctx context.Context, userID uint64, commentIDs []uint64) (map[uint64]bool, error) {
	result := make(map[uint64]bool, len(commentIDs))
	if len(commentIDs) == 0 || userID == 0 {
		return result, nil
	}
	var likes []comment.DynamicCommentLike
	if err := s.db.WithContext(ctx).Where("user_id = ? AND comment_id IN ?", userID, commentIDs).Find(&likes).Error; err != nil {
		return nil, err
	}
	for _, lk := range likes {
		result[lk.CommentID] = true
	}
	return result, nil
}

// CommentReplyCounts returns reply count per parent comment id.
func (s *CreatorCommentService) CommentReplyCounts(ctx context.Context, ids []uint64) map[uint64]uint64 {
	out := make(map[uint64]uint64, len(ids))
	if len(ids) == 0 {
		return out
	}
	type row struct {
		ParentID uint64
		C        int64
	}
	var rows []row
	_ = s.db.WithContext(ctx).Model(&comment.Comment{}).
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
func (s *CreatorCommentService) ArticleCommentReplyCounts(ctx context.Context, ids []uint64) map[uint64]uint64 {
	out := make(map[uint64]uint64, len(ids))
	if len(ids) == 0 {
		return out
	}
	type row struct {
		ParentID uint64
		C        int64
	}
	var rows []row
	_ = s.db.WithContext(ctx).Model(&comment.ArticleComment{}).
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
func (s *CreatorCommentService) DynamicCommentReplyCounts(ctx context.Context, ids []uint64) map[uint64]uint64 {
	out := make(map[uint64]uint64, len(ids))
	if len(ids) == 0 {
		return out
	}
	type row struct {
		ParentID uint64
		C        int64
	}
	var rows []row
	_ = s.db.WithContext(ctx).Model(&comment.DynamicComment{}).
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
func (s *CreatorCommentService) CheckVideoOwnership(ctx context.Context, videoID, userID uint64) (bool, error) {
	var owned video.Video
	err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", videoID, userID).First(&owned).Error
	if err == gorm.ErrRecordNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// CheckArticleOwnership checks if an article with the given id belongs to the given user.
func (s *CreatorCommentService) CheckArticleOwnership(ctx context.Context, articleID, userID uint64) (bool, error) {
	var owned article.Article
	err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", articleID, userID).First(&owned).Error
	if err == gorm.ErrRecordNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// CheckDynamicOwnership checks if a dynamic with the given id belongs to the given user.
func (s *CreatorCommentService) CheckDynamicOwnership(ctx context.Context, dynamicID, userID uint64) (bool, error) {
	var owned dynamic.UserDynamic
	err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", dynamicID, userID).First(&owned).Error
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
	UserID           uint64
	Page             int
	PageSize         int
	SortKey          string
	Pending          bool
	PendingStatus    string
	PendingScope     string
	Keyword          string
	FilterArticleID  uint64
	ViewerID       uint64
}

// CreatorArticleCommentResult holds query results for article comments.
type CreatorArticleCommentResult struct {
	Comments    []comment.ArticleComment
	Total       int64
	ArticleIDs  []uint64
	UserIDs     []uint64
	ParentIDs   []uint64
	CommentIDs  []uint64
	LikedByViewer map[uint64]bool
}

// ListCreatorArticleComments lists comments on the creator's articles with filters.
func (s *CreatorCommentService) ListCreatorArticleComments(ctx context.Context, q CreatorArticleCommentQuery) (*CreatorArticleCommentResult, error) {
	base := s.db.WithContext(ctx).Model(&comment.ArticleComment{}).
		Joins("INNER JOIN articles ON articles.id = article_comments.article_id AND articles.user_id = ?", q.UserID)

	if q.Pending {
		switch q.PendingStatus {
		case "ignored":
			base = base.Where("article_comments.curated_ignored = ?", true).
				Where("article_comments.approved = ?", false)
		default:
			base = base.Where("articles.comments_curated = ?", true).
				Where("article_comments.approved = ?", false)
			switch q.PendingStatus {
			case "all":
			default:
				base = base.Where("article_comments.curated_ignored = ?", false)
			}
		}
		switch q.PendingScope {
		case "root":
			base = base.Where("article_comments.parent_id = ?", 0)
		case "reply":
			base = base.Where("article_comments.parent_id > ?", 0)
		}
	} else {
		base = base.Where("article_comments.approved = ?", true)
	}

	if q.FilterArticleID > 0 {
		base = base.Where("article_comments.article_id = ?", q.FilterArticleID)
	}
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		base = base.Where("article_comments.content LIKE ? OR articles.title LIKE ?", like, like)
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, err
	}

	order := "article_comments.created_at DESC, article_comments.id DESC"
	switch q.SortKey {
	case "earliest":
		order = "article_comments.created_at ASC, article_comments.id ASC"
	case "likes":
		order = "article_comments.like_count DESC, article_comments.id DESC"
	case "replies":
		order = "(SELECT COUNT(*) FROM article_comments AS r WHERE r.parent_id = article_comments.id) DESC, article_comments.id DESC"
	}

	offset := (q.Page - 1) * q.PageSize
	var list []comment.ArticleComment
	if err := base.Order(order).Offset(offset).Limit(q.PageSize).Find(&list).Error; err != nil {
		return nil, err
	}

	result := &CreatorArticleCommentResult{
		Comments:   list,
		Total:      total,
		ArticleIDs: make([]uint64, 0, len(list)),
		UserIDs:    make([]uint64, 0, len(list)),
		ParentIDs:  make([]uint64, 0),
		CommentIDs: make([]uint64, len(list)),
	}
	for i, cm := range list {
		result.CommentIDs[i] = cm.ID
		result.ArticleIDs = append(result.ArticleIDs, cm.ArticleID)
		result.UserIDs = append(result.UserIDs, cm.UserID)
		if cm.ParentID > 0 {
			result.ParentIDs = append(result.ParentIDs, cm.ParentID)
		}
	}

	if q.ViewerID > 0 && len(list) > 0 {
		var likes []comment.ArticleCommentLike
		_ = s.db.WithContext(ctx).Where("user_id = ? AND comment_id IN ?", q.ViewerID, result.CommentIDs).Find(&likes).Error
		for _, lk := range likes {
			result.LikedByViewer[lk.CommentID] = true
		}
	}
	return result, nil
}

// BatchFetchArticles returns a map of article id to Article for the given ids.
func (s *CreatorCommentService) BatchFetchArticles(ctx context.Context, ids []uint64) (map[uint64]article.Article, error) {
	result := make(map[uint64]article.Article, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	var list []article.Article
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&list).Error; err != nil {
		return nil, err
	}
	for i := range list {
		result[list[i].ID] = list[i]
	}

	return result, nil
}

// CreatorDynamicCommentQuery holds filter params for listing dynamic comments.
type CreatorDynamicCommentQuery struct {
	UserID            uint64
	Page              int
	PageSize          int
	SortKey           string
	Pending           bool
	PendingStatus     string
	PendingScope      string
	Keyword           string
	FilterDynamicID   uint64
	ViewerID          uint64
}

// CreatorDynamicCommentResult holds query results for dynamic comments.
type CreatorDynamicCommentResult struct {
	Comments    []comment.DynamicComment
	Total       int64
	DynamicIDs  []uint64
	UserIDs     []uint64
	ParentIDs   []uint64
	CommentIDs  []uint64
	LikedByViewer map[uint64]bool
}

// ListCreatorDynamicComments lists comments on the creator's dynamics with filters.
func (s *CreatorCommentService) ListCreatorDynamicComments(ctx context.Context, q CreatorDynamicCommentQuery) (*CreatorDynamicCommentResult, error) {
	base := s.db.WithContext(ctx).Model(&comment.DynamicComment{}).
		Joins("INNER JOIN user_dynamics ON user_dynamics.id = dynamic_comments.dynamic_id AND user_dynamics.user_id = ?", q.UserID)

	if q.Pending {
		switch q.PendingStatus {
		case "ignored":
			base = base.Where("dynamic_comments.curated_ignored = ?", true).
				Where("dynamic_comments.approved = ?", false)
		default:
			base = base.Where("user_dynamics.comments_curated = ?", true).
				Where("dynamic_comments.approved = ?", false)
			switch q.PendingStatus {
			case "all":
			default:
				base = base.Where("dynamic_comments.curated_ignored = ?", false)
			}
		}
		switch q.PendingScope {
		case "root":
			base = base.Where("dynamic_comments.parent_id = ?", 0)
		case "reply":
			base = base.Where("dynamic_comments.parent_id > ?", 0)
		}
	} else {
		base = base.Where("dynamic_comments.approved = ?", true)
	}

	if q.FilterDynamicID > 0 {
		base = base.Where("dynamic_comments.dynamic_id = ?", q.FilterDynamicID)
	}
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		base = base.Where("dynamic_comments.content LIKE ? OR user_dynamics.title LIKE ? OR user_dynamics.content LIKE ?", like, like, like)
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, err
	}

	order := "dynamic_comments.created_at DESC, dynamic_comments.id DESC"
	switch q.SortKey {
	case "earliest":
		order = "dynamic_comments.created_at ASC, dynamic_comments.id ASC"
	case "likes":
		order = "dynamic_comments.like_count DESC, dynamic_comments.id DESC"
	case "replies":
		order = "(SELECT COUNT(*) FROM dynamic_comments AS r WHERE r.parent_id = dynamic_comments.id) DESC, dynamic_comments.id DESC"
	}

	offset := (q.Page - 1) * q.PageSize
	var list []comment.DynamicComment
	if err := base.Order(order).Offset(offset).Limit(q.PageSize).Find(&list).Error; err != nil {
		return nil, err
	}

	result := &CreatorDynamicCommentResult{
		Comments:   list,
		Total:      total,
		DynamicIDs: make([]uint64, 0, len(list)),
		UserIDs:    make([]uint64, 0, len(list)),
		ParentIDs:  make([]uint64, 0),
		CommentIDs: make([]uint64, len(list)),
	}
	for i, cm := range list {
		result.CommentIDs[i] = cm.ID
		result.DynamicIDs = append(result.DynamicIDs, cm.DynamicID)
		result.UserIDs = append(result.UserIDs, cm.UserID)
		if cm.ParentID > 0 {
			result.ParentIDs = append(result.ParentIDs, cm.ParentID)
		}
	}

	if q.ViewerID > 0 && len(list) > 0 {
		var likes []comment.DynamicCommentLike
		_ = s.db.WithContext(ctx).Where("user_id = ? AND comment_id IN ?", q.ViewerID, result.CommentIDs).Find(&likes).Error
		for _, lk := range likes {
			result.LikedByViewer[lk.CommentID] = true
		}
	}
	return result, nil
}

// BatchFetchDynamics returns a map of dynamic id to UserDynamic for the given ids.
func (s *CreatorCommentService) BatchFetchDynamics(ctx context.Context, ids []uint64) (map[uint64]dynamic.UserDynamic, error) {
	result := make(map[uint64]dynamic.UserDynamic, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	var list []dynamic.UserDynamic
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&list).Error; err != nil {
		return nil, err
	}
	for i := range list {
		result[list[i].ID] = list[i]
	}
	return result, nil
}
