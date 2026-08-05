package service

import (
	"cakecake/internal/model/article"
	"cakecake/internal/model/dynamic"
	"context"

	"gorm.io/gorm"
)

// ArticleProviderImpl implements ArticleProvider using *gorm.DB.
type ArticleProviderImpl struct {
	db *gorm.DB
}

// NewArticleProvider creates a gorm-backed ArticleProvider implementation.
func NewArticleProvider(db *gorm.DB) *ArticleProviderImpl {
	return &ArticleProviderImpl{db: db}
}

// GetPublishedArticle loads a published article as an ArticleInfo projection.
func (p *ArticleProviderImpl) GetPublishedArticle(ctx context.Context, id uint64) (*ArticleInfo, error) {
	var a article.Article
	if err := p.db.WithContext(ctx).First(&a, id).Error; err != nil {
		return nil, err
	}
	if a.Status != article.StatusPublished {
		return nil, gorm.ErrRecordNotFound
	}
	return &ArticleInfo{
		ID: a.ID, UserID: a.UserID, Title: a.Title, Status: a.Status,
		CommentsClosed: a.CommentsClosed, CommentsCurated: a.CommentsCurated, CreatedAt: a.CreatedAt,
	}, nil
}

// GetArticleAuthor returns the owner id of an article.
func (p *ArticleProviderImpl) GetArticleAuthor(ctx context.Context, id uint64) (uint64, error) {
	var a article.Article
	if err := p.db.WithContext(ctx).Select("user_id").First(&a, id).Error; err != nil {
		return 0, err
	}
	return a.UserID, nil
}

// IncrCommentCount adjusts an article's comment count by delta.
func (p *ArticleProviderImpl) IncrCommentCount(ctx context.Context, id uint64, delta int) error {
	return p.db.WithContext(ctx).Model(&article.Article{}).Where("id = ?", id).
		UpdateColumn("comment_count", gorm.Expr("comment_count + ?", delta)).Error
}

// DynamicProviderImpl implements DynamicProvider using *gorm.DB.
type DynamicProviderImpl struct {
	db *gorm.DB
}

// NewDynamicProvider creates a gorm-backed DynamicProvider implementation.
func NewDynamicProvider(db *gorm.DB) *DynamicProviderImpl {
	return &DynamicProviderImpl{db: db}
}

// GetPublishedDynamic loads a dynamic as a DynamicInfo projection.
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

// GetDynamicAuthor returns the owner id of a dynamic.
func (p *DynamicProviderImpl) GetDynamicAuthor(ctx context.Context, id uint64) (uint64, error) {
	var d dynamic.UserDynamic
	if err := p.db.WithContext(ctx).Select("user_id").First(&d, id).Error; err != nil {
		return 0, err
	}
	return d.UserID, nil
}

// IncrCommentCount adjusts a dynamic's comment count by delta.
func (p *DynamicProviderImpl) IncrCommentCount(ctx context.Context, id uint64, delta int) error {
	return p.db.WithContext(ctx).Model(&dynamic.UserDynamic{}).Where("id = ?", id).
		UpdateColumn("comment_count", gorm.Expr("comment_count + ?", delta)).Error
}
