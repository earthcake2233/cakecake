package service

import (
	"context"

	"gorm.io/gorm"

	"minibili/internal/model"
)

// VideoProviderImpl implements VideoProvider using *gorm.DB.
type VideoProviderImpl struct {
	db *gorm.DB
}

func NewVideoProvider(db *gorm.DB) *VideoProviderImpl {
	return &VideoProviderImpl{db: db}
}

func (p *VideoProviderImpl) GetPublishedVideo(ctx context.Context, id uint64) (*VideoInfo, error) {
	var v model.Video
	if err := p.db.WithContext(ctx).First(&v, id).Error; err != nil { return nil, err }
	if v.Status != "published" { return nil, gorm.ErrRecordNotFound }
	return &VideoInfo{
		ID: v.ID, UserID: v.UserID, Title: v.Title, Status: v.Status,
		CommentsClosed: v.CommentsClosed, CommentsCurated: v.CommentsCurated,
		DanmakuClosed: v.DanmakuClosed, CreatedAt: v.CreatedAt,
	}, nil
}

func (p *VideoProviderImpl) GetVideoAuthor(ctx context.Context, id uint64) (uint64, error) {
	var v model.Video
	if err := p.db.WithContext(ctx).Select("user_id").First(&v, id).Error; err != nil { return 0, err }
	return v.UserID, nil
}

// ArticleProviderImpl implements ArticleProvider using *gorm.DB.
type ArticleProviderImpl struct {
	db *gorm.DB
}

func NewArticleProvider(db *gorm.DB) *ArticleProviderImpl {
	return &ArticleProviderImpl{db: db}
}

func (p *ArticleProviderImpl) GetPublishedArticle(ctx context.Context, id uint64) (*ArticleInfo, error) {
	var a model.Article
	if err := p.db.WithContext(ctx).First(&a, id).Error; err != nil { return nil, err }
	if a.Status != "published" { return nil, gorm.ErrRecordNotFound }
	return &ArticleInfo{
		ID: a.ID, UserID: a.UserID, Title: a.Title, Status: a.Status,
		CommentsClosed: a.CommentsClosed, CommentsCurated: a.CommentsCurated, CreatedAt: a.CreatedAt,
	}, nil
}

func (p *ArticleProviderImpl) GetArticleAuthor(ctx context.Context, id uint64) (uint64, error) {
	var a model.Article
	if err := p.db.WithContext(ctx).Select("user_id").First(&a, id).Error; err != nil { return 0, err }
	return a.UserID, nil
}

// DynamicProviderImpl implements DynamicProvider using *gorm.DB.
type DynamicProviderImpl struct {
	db *gorm.DB
}

func NewDynamicProvider(db *gorm.DB) *DynamicProviderImpl {
	return &DynamicProviderImpl{db: db}
}

func (p *DynamicProviderImpl) GetPublishedDynamic(ctx context.Context, id uint64) (*DynamicInfo, error) {
	var d model.UserDynamic
	if err := p.db.WithContext(ctx).First(&d, id).Error; err != nil { return nil, err }
	return &DynamicInfo{
		ID: d.ID, UserID: d.UserID, Status: "published",
		CommentsClosed: d.CommentsClosed, CommentsCurated: d.CommentsCurated, CreatedAt: d.CreatedAt,
	}, nil
}

func (p *DynamicProviderImpl) GetDynamicAuthor(ctx context.Context, id uint64) (uint64, error) {
	var d model.UserDynamic
	if err := p.db.WithContext(ctx).Select("user_id").First(&d, id).Error; err != nil { return 0, err }
	return d.UserID, nil
}