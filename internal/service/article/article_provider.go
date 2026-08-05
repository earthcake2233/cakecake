package article

import (
	"cakecake/internal/model/article"
	"cakecake/internal/pkg/dailyreward"
	"cakecake/internal/pkg/usercoin"
	"cakecake/internal/search"
	"context"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ArticleStore is the article-domain storage boundary.
// Phase 1: *gorm.DB impl. Phase 2+: replaced by gRPC client / per-domain store.
type ArticleStore interface {
	CreateArticle(ctx context.Context, art *article.Article) error
	GetArticleByID(ctx context.Context, id uint64) (*article.Article, error)
	GetPublishedArticle(ctx context.Context, id uint64) (*article.Article, error)
	UpdateArticleByID(ctx context.Context, id uint64, updates map[string]interface{}) error
	IncrArticleView(ctx context.Context, id uint64) error
	BatchFavRows(ctx context.Context, viewerID uint64, articleIDs []uint64) ([]article.ArticleFavorite, error)
	BatchCoinRows(ctx context.Context, viewerID uint64, articleIDs []uint64) ([]article.ArticleCoin, error)
	CountArticlesByStatus(ctx context.Context, userID uint64) map[string]int64
	ToggleArticleFavorite(ctx context.Context, userID, articleID uint64) (bool, uint64, error)
	DeleteArticle(ctx context.Context, id uint64) error
	DeleteArticleCascadeTx(tx *gorm.DB, articleID uint64) error
	ListArticlesCursor(ctx context.Context, cursor uint64, limit int, sort string) (*ListArticlesCursorResult, error)
	ListMyArticlesCursor(ctx context.Context, userID uint64, cursor uint64, limit int, status, titleQ, sortKey string) (*ListArticlesCursorResult, error)
	ListMyArticlesPage(ctx context.Context, userID uint64, page, pageSize int, status, titleQ, sortKey string) (*ListMyArticlesPageResult, error)
	ListUserPublishedArticlesCursor(ctx context.Context, userID uint64, cursor uint64, limit int) (*ListArticlesCursorResult, error)
	ListFavoritedArticlesV2(ctx context.Context, userID, cursor uint64, limit int) ([]article.ArticleFavorite, bool, error)
	BatchFetchArticles(ctx context.Context, ids []uint64, onlyPublished bool) map[uint64]*article.Article
	CountFavoritedArticles(ctx context.Context, userID uint64, onlyPublished bool) (int64, error)
	HasArticleCoin(ctx context.Context, userID, articleID uint64) (*article.ArticleCoin, error)
	UpdateArticleCoinTx(ctx context.Context, userID, authorID, articleID uint64, exist *article.ArticleCoin) error
	CreateArticleCoinTx(ctx context.Context, userID, authorID, articleID uint64, amount int) error
	CoinProgress(uid uint64) int
	GrantCoinExp(uid uint64, before, after int) error
	CountByStatus(ctx context.Context, status string) (int64, error)
	AdminListArticles(ctx context.Context, statuses []string, titleQ string, page, pageSize int) (*AdminListArticlesResult, error)
	AdminDeleteArticleCascade(ctx context.Context, id uint64, fn func(tx *gorm.DB) error) error
	PublishArticle(ctx context.Context, esc *search.Client, log *zap.Logger, articleID uint64, adminID *uint64) error
}

// ArticleStoreImpl implements ArticleStore using *gorm.DB (Phase 1 monolith).
type ArticleStoreImpl struct {
	db *gorm.DB
}

// NewArticleStore creates a gorm-backed ArticleStore implementation.
func NewArticleStore(db *gorm.DB) *ArticleStoreImpl {
	return &ArticleStoreImpl{db: db}
}

// CreateArticle inserts an article row.
func (p *ArticleStoreImpl) CreateArticle(ctx context.Context, art *article.Article) error {
	return p.db.WithContext(ctx).Create(art).Error
}

// GetArticleByID loads an article by id.
func (p *ArticleStoreImpl) GetArticleByID(ctx context.Context, id uint64) (*article.Article, error) {
	var a article.Article
	if err := p.db.WithContext(ctx).First(&a, id).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// GetPublishedArticle loads an article that is published.
func (p *ArticleStoreImpl) GetPublishedArticle(ctx context.Context, id uint64) (*article.Article, error) {
	var a article.Article
	if err := p.db.WithContext(ctx).Where("id = ? AND status = ?", id, article.StatusPublished).First(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// UpdateArticleByID applies partial updates to an article.
func (p *ArticleStoreImpl) UpdateArticleByID(ctx context.Context, id uint64, updates map[string]interface{}) error {
	return p.db.WithContext(ctx).Model(&article.Article{}).Where("id = ?", id).Updates(updates).Error
}

// IncrArticleView increments an article's view count.
func (p *ArticleStoreImpl) IncrArticleView(ctx context.Context, id uint64) error {
	return p.db.WithContext(ctx).Model(&article.Article{}).Where("id = ?", id).
		UpdateColumn("view_count", gorm.Expr("view_count + 1")).Error
}

// BatchFavRows loads a viewer's favorite rows for the given articles.
func (p *ArticleStoreImpl) BatchFavRows(ctx context.Context, viewerID uint64, articleIDs []uint64) ([]article.ArticleFavorite, error) {
	var favRows []article.ArticleFavorite
	if err := p.db.WithContext(ctx).Where("user_id = ? AND article_id IN ?", viewerID, articleIDs).Find(&favRows).Error; err != nil {
		return nil, err
	}
	return favRows, nil
}

// BatchCoinRows loads a viewer's coin rows for the given articles.
func (p *ArticleStoreImpl) BatchCoinRows(ctx context.Context, viewerID uint64, articleIDs []uint64) ([]article.ArticleCoin, error) {
	var coinRows []article.ArticleCoin
	if err := p.db.WithContext(ctx).Where("user_id = ? AND article_id IN ?", viewerID, articleIDs).Find(&coinRows).Error; err != nil {
		return nil, err
	}
	return coinRows, nil
}

// CountArticlesByStatus counts a user's articles per status.
func (p *ArticleStoreImpl) CountArticlesByStatus(ctx context.Context, userID uint64) map[string]int64 {
	type statusRow struct {
		Status string
		N      int64
	}
	var rows []statusRow
	_ = p.db.WithContext(ctx).Model(&article.Article{}).
		Select("status, COUNT(*) AS n").
		Where("user_id = ?", userID).
		Group("status").
		Scan(&rows).Error
	out := map[string]int64{
		"draft":      0,
		"processing": 0,
		"passed":     0,
		"rejected":   0,
	}
	for _, r := range rows {
		switch r.Status {
		case article.StatusDraft:
			out["draft"] = r.N
		case article.StatusProcessing:
			out["processing"] = r.N
		case article.StatusPublished:
			out["passed"] = r.N
		case article.StatusRejected:
			out["rejected"] = r.N
		}
	}
	return out
}

// ToggleArticleFavorite toggles a user's favorite on an article, returning the new state and fav count.
func (p *ArticleStoreImpl) ToggleArticleFavorite(ctx context.Context, userID, articleID uint64) (bool, uint64, error) {
	var favorited bool
	var favCount uint64
	err := p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing article.ArticleFavorite
		res := tx.Where("user_id = ? AND article_id = ?", userID, articleID).Limit(1).Find(&existing)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected > 0 {
			favorited = false
			if err := tx.Where("user_id = ? AND article_id = ?", userID, articleID).Delete(&article.ArticleFavorite{}).Error; err != nil {
				return err
			}
			if err := tx.Model(&article.Article{}).Where("id = ?", articleID).UpdateColumn("fav_count",
				gorm.Expr("CASE WHEN fav_count - ? < 0 THEN 0 ELSE fav_count - ? END", 1, 1)).Error; err != nil {
				return err
			}
		} else {
			favorited = true
			if err := tx.Create(&article.ArticleFavorite{UserID: userID, ArticleID: articleID}).Error; err != nil {
				return err
			}
			if err := tx.Model(&article.Article{}).Where("id = ?", articleID).UpdateColumn("fav_count",
				gorm.Expr("fav_count + ?", 1)).Error; err != nil {
				return err
			}
		}
		var art article.Article
		if err := tx.First(&art, articleID).Error; err != nil {
			return err
		}
		favCount = art.FavCount
		return nil
	})
	return favorited, favCount, err
}

// DeleteArticle deletes an article with its cascade rows atomically.
func (p *ArticleStoreImpl) DeleteArticle(ctx context.Context, id uint64) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return deleteArticleCascadeTx(tx, id)
	})
}

// DeleteArticleCascadeTx runs the article cascade delete inside a transaction.
func (p *ArticleStoreImpl) DeleteArticleCascadeTx(tx *gorm.DB, articleID uint64) error {
	return deleteArticleCascadeTx(tx, articleID)
}

// ListArticlesCursor pages published articles with a keyset cursor.
func (p *ArticleStoreImpl) ListArticlesCursor(ctx context.Context, cursor uint64, limit int, sort string) (*ListArticlesCursorResult, error) {
	q := p.db.WithContext(ctx).Model(&article.Article{}).Where("status = ?", article.StatusPublished)
	if cursor > 0 {
		q = q.Where("id < ?", cursor)
	}
	order := "id DESC"
	if sort == "hot" {
		order = "view_count DESC, id DESC"
	}
	var list []article.Article
	if err := q.Order(order).Limit(limit + 1).Find(&list).Error; err != nil {
		return nil, err
	}
	hasMore := len(list) > limit
	if hasMore {
		list = list[:limit]
	}
	return &ListArticlesCursorResult{Items: list, HasMore: hasMore}, nil
}

// ListMyArticlesCursor pages the caller's articles with status/title/sort filters.
func (p *ArticleStoreImpl) ListMyArticlesCursor(ctx context.Context, userID uint64, cursor uint64, limit int, status, titleQ, sortKey string) (*ListArticlesCursorResult, error) {
	q := p.db.WithContext(ctx).Model(&article.Article{}).Where("user_id = ?", userID)
	if st := manuscriptStatusToDB(status); st != "" {
		q = q.Where("status = ?", st)
	}
	if titleQ != "" {
		q = q.Where("title LIKE ?", "%"+titleQ+"%")
	}
	if cursor > 0 {
		q = q.Where("id < ?", cursor)
	}
	order := "id DESC"
	switch sortKey {
	case "time":
		order = "id DESC"
	case "view":
		order = "view_count DESC, id DESC"
	case "reply":
		order = "comment_count DESC, id DESC"
	case "like", "fav":
		order = "fav_count DESC, id DESC"
	}
	var list []article.Article
	if err := q.Order(order).Limit(limit + 1).Find(&list).Error; err != nil {
		return nil, err
	}
	hasMore := len(list) > limit
	if hasMore {
		list = list[:limit]
	}
	return &ListArticlesCursorResult{Items: list, HasMore: hasMore}, nil
}

// ListMyArticlesPage pages the caller's non-draft articles with filters.
func (p *ArticleStoreImpl) ListMyArticlesPage(ctx context.Context, userID uint64, page, pageSize int, status, titleQ, sortKey string) (*ListMyArticlesPageResult, error) {
	q := p.db.WithContext(ctx).Model(&article.Article{}).Where("user_id = ?", userID)
	if st := manuscriptStatusToDB(status); st != "" {
		q = q.Where("status = ?", st)
	} else {
		q = q.Where("status <> ?", article.StatusDraft)
	}
	if titleQ != "" {
		q = q.Where("title LIKE ?", "%"+titleQ+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}
	offset := (page - 1) * pageSize
	order := "id DESC"
	switch sortKey {
	case "time":
		order = "created_at DESC, id DESC"
	case "view":
		order = "view_count DESC, id DESC"
	case "reply":
		order = "comment_count DESC, id DESC"
	case "like", "fav":
		order = "fav_count DESC, id DESC"
	}
	var list []article.Article
	if err := q.Order(order).Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, err
	}
	return &ListMyArticlesPageResult{Items: list, Total: total}, nil
}

// ListUserPublishedArticlesCursor pages a user's published articles with a keyset cursor.
func (p *ArticleStoreImpl) ListUserPublishedArticlesCursor(ctx context.Context, userID uint64, cursor uint64, limit int) (*ListArticlesCursorResult, error) {
	q := p.db.WithContext(ctx).Model(&article.Article{}).Where("user_id = ? AND status = ?", userID, article.StatusPublished)
	if cursor > 0 {
		q = q.Where("id < ?", cursor)
	}
	var list []article.Article
	if err := q.Order("id DESC").Limit(limit + 1).Find(&list).Error; err != nil {
		return nil, err
	}
	hasMore := len(list) > limit
	if hasMore {
		list = list[:limit]
	}
	return &ListArticlesCursorResult{Items: list, HasMore: hasMore}, nil
}

// ListFavoritedArticlesV2 pages a user's favorited article rows with a keyset cursor.
func (p *ArticleStoreImpl) ListFavoritedArticlesV2(ctx context.Context, userID, cursor uint64, limit int) ([]article.ArticleFavorite, bool, error) {
	q := p.db.WithContext(ctx).Model(&article.ArticleFavorite{}).Where("user_id = ?", userID)
	if cursor > 0 {
		q = q.Where("id < ?", cursor)
	}
	var favs []article.ArticleFavorite
	if err := q.Order("id DESC").Limit(limit + 1).Find(&favs).Error; err != nil {
		return nil, false, err
	}
	hasMore := len(favs) > limit
	if hasMore {
		favs = favs[:limit]
	}
	return favs, hasMore, nil
}

// BatchFetchArticles loads articles by ids, optionally published only.
func (p *ArticleStoreImpl) BatchFetchArticles(ctx context.Context, ids []uint64, onlyPublished bool) map[uint64]*article.Article {
	out := make(map[uint64]*article.Article, len(ids))
	if len(ids) == 0 {
		return out
	}
	q := p.db.WithContext(ctx).Where("id IN ?", ids)
	if onlyPublished {
		q = q.Where("status = ?", article.StatusPublished)
	}
	var rows []article.Article
	if err := q.Find(&rows).Error; err != nil {
		return out
	}
	for i := range rows {
		out[rows[i].ID] = &rows[i]
	}
	return out
}

// CountFavoritedArticles counts a user's favorited articles.
func (p *ArticleStoreImpl) CountFavoritedArticles(ctx context.Context, userID uint64, onlyPublished bool) (int64, error) {
	q := p.db.WithContext(ctx).Table("article_favorites").
		Joins("INNER JOIN articles ON articles.id = article_favorites.article_id")
	if onlyPublished {
		q = q.Where("articles.status = ?", article.StatusPublished)
	}
	q = q.Where("article_favorites.user_id = ?", userID)
	var total int64
	err := q.Count(&total).Error
	return total, err
}

// HasArticleCoin returns a user's existing coin row for an article, if any.
func (p *ArticleStoreImpl) HasArticleCoin(ctx context.Context, userID, articleID uint64) (*article.ArticleCoin, error) {
	var exist article.ArticleCoin
	res := p.db.WithContext(ctx).Where("user_id = ? AND article_id = ?", userID, articleID).Limit(1).Find(&exist)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, nil
	}
	return &exist, nil
}

// UpdateArticleCoinTx upgrades a user's article coin to max amount and updates counts.
func (p *ArticleStoreImpl) UpdateArticleCoinTx(ctx context.Context, userID, authorID, articleID uint64, exist *article.ArticleCoin) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := usercoin.SpendOnArticleCoin(tx, userID, authorID, articleID, 1); err != nil {
			return err
		}
		if err := tx.Model(exist).Update("amount", 2).Error; err != nil {
			return err
		}
		return tx.Model(&article.Article{}).Where("id = ?", articleID).
			UpdateColumn("coin_count", gorm.Expr("coin_count + ?", 1)).Error
	})
}

// CreateArticleCoinTx spends coins and records a new article coin row.
func (p *ArticleStoreImpl) CreateArticleCoinTx(ctx context.Context, userID, authorID, articleID uint64, amount int) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := usercoin.SpendOnArticleCoin(tx, userID, authorID, articleID, amount); err != nil {
			return err
		}
		row := article.ArticleCoin{UserID: userID, ArticleID: articleID, Amount: amount}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return tx.Model(&article.Article{}).Where("id = ?", articleID).
			UpdateColumn("coin_count", gorm.Expr("coin_count + ?", amount)).Error
	})
}

// CoinProgress returns the user's daily coin-task EXP progress.
func (p *ArticleStoreImpl) CoinProgress(uid uint64) int {
	return dailyreward.CoinProgress(p.db, uid)
}

// GrantCoinExp grants daily coin-task EXP to the user.
func (p *ArticleStoreImpl) GrantCoinExp(uid uint64, before, after int) error {
	return dailyreward.GrantCoinExp(p.db, uid, before, after)
}

// CountByStatus counts articles with the given status.
func (p *ArticleStoreImpl) CountByStatus(ctx context.Context, status string) (int64, error) {
	var cnt int64
	err := p.db.WithContext(ctx).Model(&article.Article{}).Where("status = ?", status).Count(&cnt).Error
	return cnt, err
}

// AdminListArticles pages all articles for the admin panel.
func (p *ArticleStoreImpl) AdminListArticles(ctx context.Context, statuses []string, titleQ string, page, pageSize int) (*AdminListArticlesResult, error) {
	q := p.db.WithContext(ctx).Model(&article.Article{})
	if len(statuses) > 0 {
		q = q.Where("status IN ?", statuses)
	}
	if titleQ != "" {
		q = q.Where("title LIKE ?", "%"+titleQ+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}
	offset := (page - 1) * pageSize
	var rows []article.Article
	if err := q.Order("created_at DESC, id DESC").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	var pending int64
	_ = p.db.WithContext(ctx).Model(&article.Article{}).Where("status = ?", article.StatusPendingReview).Count(&pending).Error
	return &AdminListArticlesResult{Total: total, Rows: rows, PendingCount: pending}, nil
}

// PublishArticle marks an article published and indexes it in Elasticsearch.
func (p *ArticleStoreImpl) PublishArticle(ctx context.Context, esc *search.Client, log *zap.Logger, articleID uint64, adminID *uint64) error {
	var art article.Article
	if err := p.db.WithContext(ctx).First(&art, articleID).Error; err != nil {
		return err
	}
	if art.Status == article.StatusPublished {
		return nil
	}
	now := time.Now()
	updates := map[string]any{
		"status":       article.StatusPublished,
		"published_at": now,
		"reviewed_at":  now,
		"fail_reason":  "",
	}
	if adminID != nil && *adminID > 0 {
		updates["reviewed_by_admin_id"] = *adminID
	}
	if err := p.db.WithContext(ctx).Model(&art).Updates(updates).Error; err != nil {
		return err
	}
	if esc != nil && esc.Enabled() {
		ictx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		if err := esc.IndexArticleFromDB(ictx, p.db, articleID); err != nil && log != nil {
			log.Warn("elasticsearch index article on publish", zap.Uint64("article_id", articleID), zap.Error(err))
		}
	}
	return nil
}

// AdminDeleteArticleCascade runs the cascade delete callback inside a transaction.
func (p *ArticleStoreImpl) AdminDeleteArticleCascade(ctx context.Context, id uint64, fn func(tx *gorm.DB) error) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := fn(tx); err != nil {
			return err
		}
		return nil
	})
}
