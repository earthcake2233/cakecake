package service

import (
	"context"
	"minibili/internal/model/article"
	"minibili/internal/model/dynamic"
	"minibili/internal/model/video"

	"gorm.io/gorm"
)

// VideoProviderImpl implements VideoProvider using *gorm.DB.
type VideoProviderImpl struct {
	db *gorm.DB
}

func NewVideoProvider(db *gorm.DB) *VideoProviderImpl {
	return &VideoProviderImpl{db: db}
}

func (p *VideoProviderImpl) GetPublishedVideo(ctx context.Context, id uint64) (*VideoInfo, error) {
	var v video.Video
	if err := p.db.WithContext(ctx).First(&v, id).Error; err != nil {
		return nil, err
	}
	if v.Status != "published" {
		return nil, gorm.ErrRecordNotFound
	}
	return &VideoInfo{
		ID: v.ID, UserID: v.UserID, Title: v.Title, CoverURL: v.CoverURL,
		PlayCount: v.PlayCount, DanmakuCount: v.DanmakuCount, CommentCount: v.CommentCount, DurationSec: v.DurationSec,
		FavCount: v.FavCount, Status: v.Status,
		CommentsClosed: v.CommentsClosed, CommentsCurated: v.CommentsCurated,
		DanmakuClosed: v.DanmakuClosed, CreatedAt: v.CreatedAt,
	}, nil
}

func (p *VideoProviderImpl) GetVideoAuthor(ctx context.Context, id uint64) (uint64, error) {
	var v video.Video
	if err := p.db.WithContext(ctx).Select("user_id").First(&v, id).Error; err != nil {
		return 0, err
	}
	return v.UserID, nil
}

func (p *VideoProviderImpl) IncrCommentCount(ctx context.Context, id uint64, delta int) error {
	return p.db.WithContext(ctx).Model(&video.Video{}).Where("id = ?", id).
		UpdateColumn("comment_count", gorm.Expr("comment_count + ?", delta)).Error
}

func (p *VideoProviderImpl) IncrFavCount(ctx context.Context, id uint64, delta int) error {
	q := p.db.WithContext(ctx).Model(&video.Video{}).Where("id = ?", id)
	if delta < 0 {
		abs := -delta
		return q.UpdateColumn("fav_count",
			gorm.Expr("CASE WHEN fav_count < ? THEN 0 ELSE fav_count - ? END", abs, abs)).Error
	}
	return q.UpdateColumn("fav_count", gorm.Expr("fav_count + ?", delta)).Error
}

func (p *VideoProviderImpl) BatchGetPublishedVideos(ctx context.Context, ids []uint64) (map[uint64]*VideoInfo, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var videos []video.Video
	if err := p.db.WithContext(ctx).Where("id IN ? AND status = ?", ids, "published").Find(&videos).Error; err != nil {
		return nil, err
	}
	result := make(map[uint64]*VideoInfo, len(videos))
	for i := range videos {
		v := &videos[i]
		result[v.ID] = &VideoInfo{
			ID: v.ID, UserID: v.UserID, Title: v.Title, CoverURL: v.CoverURL,
			PlayCount: v.PlayCount, DanmakuCount: v.DanmakuCount, CommentCount: v.CommentCount, DurationSec: v.DurationSec,
			FavCount: v.FavCount, Status: v.Status,
			CommentsClosed: v.CommentsClosed, CommentsCurated: v.CommentsCurated,
			DanmakuClosed: v.DanmakuClosed, CreatedAt: v.CreatedAt,
		}
	}
	return result, nil
}

// ArticleProviderImpl implements ArticleProvider using *gorm.DB.
type ArticleProviderImpl struct {
	db *gorm.DB
}

func NewArticleProvider(db *gorm.DB) *ArticleProviderImpl {
	return &ArticleProviderImpl{db: db}
}

func (p *ArticleProviderImpl) GetPublishedArticle(ctx context.Context, id uint64) (*ArticleInfo, error) {
	var a article.Article
	if err := p.db.WithContext(ctx).First(&a, id).Error; err != nil {
		return nil, err
	}
	if a.Status != "published" {
		return nil, gorm.ErrRecordNotFound
	}
	return &ArticleInfo{
		ID: a.ID, UserID: a.UserID, Title: a.Title, Status: a.Status,
		CommentsClosed: a.CommentsClosed, CommentsCurated: a.CommentsCurated, CreatedAt: a.CreatedAt,
	}, nil
}

func (p *ArticleProviderImpl) GetArticleAuthor(ctx context.Context, id uint64) (uint64, error) {
	var a article.Article
	if err := p.db.WithContext(ctx).Select("user_id").First(&a, id).Error; err != nil {
		return 0, err
	}
	return a.UserID, nil
}

func (p *ArticleProviderImpl) IncrCommentCount(ctx context.Context, id uint64, delta int) error {
	return p.db.WithContext(ctx).Model(&article.Article{}).Where("id = ?", id).
		UpdateColumn("comment_count", gorm.Expr("comment_count + ?", delta)).Error
}

// DynamicProviderImpl implements DynamicProvider using *gorm.DB.
type DynamicProviderImpl struct {
	db *gorm.DB
}

func NewDynamicProvider(db *gorm.DB) *DynamicProviderImpl {
	return &DynamicProviderImpl{db: db}
}

func (p *DynamicProviderImpl) GetPublishedDynamic(ctx context.Context, id uint64) (*DynamicInfo, error) {
	var d dynamic.UserDynamic
	if err := p.db.WithContext(ctx).First(&d, id).Error; err != nil {
		return nil, err
	}
	return &DynamicInfo{
		ID: d.ID, UserID: d.UserID, Status: "published",
		CommentsClosed: d.CommentsClosed, CommentsCurated: d.CommentsCurated, CreatedAt: d.CreatedAt,
	}, nil
}

func (p *DynamicProviderImpl) GetDynamicAuthor(ctx context.Context, id uint64) (uint64, error) {
	var d dynamic.UserDynamic
	if err := p.db.WithContext(ctx).Select("user_id").First(&d, id).Error; err != nil {
		return 0, err
	}
	return d.UserID, nil
}

func (p *DynamicProviderImpl) IncrCommentCount(ctx context.Context, id uint64, delta int) error {
	return p.db.WithContext(ctx).Model(&dynamic.UserDynamic{}).Where("id = ?", id).
		UpdateColumn("comment_count", gorm.Expr("comment_count + ?", delta)).Error
}
