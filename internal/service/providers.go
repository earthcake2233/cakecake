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

func NewArticleProvider(db *gorm.DB) *ArticleProviderImpl {
	return &ArticleProviderImpl{db: db}
}

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
