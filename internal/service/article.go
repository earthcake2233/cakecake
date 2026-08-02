package service

import (
	"cakecake/internal/model/article"
	"cakecake/internal/model/comment"
	"cakecake/internal/model/extra"
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"cakecake/internal/pkg/dailyreward"
	"cakecake/internal/pkg/usercoin"
	"cakecake/internal/search"
)

// ArticleService handles article business logic.
type ArticleService struct {
	db  *gorm.DB
	rdb *redis.Client
	log *zap.Logger
	us  *UserService
	es  *search.Client
}

// ArticleEngagement holds per-viewer engagement state for an article.
type ArticleEngagement struct {
	FavoritedByMe bool
	CoinedByMe    bool
	MyCoinAmount  int
}

// ListArticlesCursorResult contains cursor-paginated article list result.

// ListMyArticlesPageResult contains page-based article list result.
type ListMyArticlesPageResult struct {
	Items []article.Article
	Total int64
}
type ListArticlesCursorResult struct {
	Items   []article.Article
	HasMore bool
}

func NewArticleService(db *gorm.DB, rdb *redis.Client, log *zap.Logger, us *UserService, es *search.Client) *ArticleService {
	return &ArticleService{db: db, rdb: rdb, log: log, us: us, es: es}
}

// Publish marks an article published and re-indexes search (post-review or direct publish).
func (s *ArticleService) Publish(ctx context.Context, articleID uint64, adminID *uint64) error {
	return PublishArticle(ctx, s.db, s.es, s.log, articleID, adminID)
}

// CreateArticle inserts a new article.
func (s *ArticleService) CreateArticle(ctx context.Context, art *article.Article) error {
	return s.db.WithContext(ctx).Create(art).Error
}

// GetArticleByID returns an article by ID, regardless of status.
func (s *ArticleService) GetArticleByID(ctx context.Context, id uint64) (*article.Article, error) {
	var a article.Article
	if err := s.db.WithContext(ctx).First(&a, id).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// GetPublishedArticle returns a published article by ID.
func (s *ArticleService) GetPublishedArticle(ctx context.Context, id uint64) (*article.Article, error) {
	var a article.Article
	if err := s.db.WithContext(ctx).Where("id = ? AND status = ?", id, article.StatusPublished).First(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// GetOwnedArticle returns an article owned by the given user.
func (s *ArticleService) GetOwnedArticle(ctx context.Context, id, userID uint64) (*article.Article, error) {
	art, err := s.GetArticleByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if art.UserID != userID {
		return nil, fmt.Errorf("article %d not owned by user %d", id, userID)
	}
	return art, nil
}

// UpdateArticle updates article fields by ID.
func (s *ArticleService) UpdateArticle(ctx context.Context, id uint64, updates map[string]interface{}) error {
	return s.db.WithContext(ctx).Model(&article.Article{}).Where("id = ?", id).Updates(updates).Error
}

// IncrementArticleView increments the view count for an article.
func (s *ArticleService) IncrementArticleView(ctx context.Context, id uint64) error {
	return s.db.WithContext(ctx).Model(&article.Article{}).Where("id = ?", id).
		UpdateColumn("view_count", gorm.Expr("view_count + 1")).Error
}

// BatchArticleEngagementByViewer returns engagement state for multiple articles for a viewer.
func (s *ArticleService) BatchArticleEngagementByViewer(ctx context.Context, viewerID uint64, articleIDs []uint64) map[uint64]*ArticleEngagement {
	out := make(map[uint64]*ArticleEngagement, len(articleIDs))
	if viewerID == 0 || len(articleIDs) == 0 {
		return out
	}
	faved := map[uint64]bool{}
	var favRows []article.ArticleFavorite
	s.db.WithContext(ctx).Where("user_id = ? AND article_id IN ?", viewerID, articleIDs).Find(&favRows)
	for i := range favRows {
		faved[favRows[i].ArticleID] = true
	}
	coinAmt := map[uint64]int{}
	var coinRows []article.ArticleCoin
	s.db.WithContext(ctx).Where("user_id = ? AND article_id IN ?", viewerID, articleIDs).Find(&coinRows)
	for i := range coinRows {
		amt := coinRows[i].Amount
		if amt < 0 {
			amt = 0
		}
		if amt > 2 {
			amt = 2
		}
		coinAmt[coinRows[i].ArticleID] = amt
	}
	for _, id := range articleIDs {
		amt := coinAmt[id]
		out[id] = &ArticleEngagement{
			FavoritedByMe: faved[id],
			CoinedByMe:    amt > 0,
			MyCoinAmount:  amt,
		}
	}
	return out
}

// CountArticlesByStatus returns article counts grouped by status for a user.
func (s *ArticleService) CountArticlesByStatus(ctx context.Context, userID uint64) map[string]int64 {
	type statusRow struct {
		Status string
		N      int64
	}
	var rows []statusRow
	_ = s.db.WithContext(ctx).Model(&article.Article{}).
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

// ToggleArticleFavorite toggles favorite on an article. Returns favorited state and updated FavCount.
func (s *ArticleService) ToggleArticleFavorite(ctx context.Context, userID, articleID uint64) (bool, uint64, error) {
	var favorited bool
	var favCount uint64
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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

func deleteArticleCascadeTx(tx *gorm.DB, articleID uint64) error {
	var cids []uint64
	if err := tx.Model(&comment.ArticleComment{}).Where("article_id = ?", articleID).Pluck("id", &cids).Error; err != nil {
		return err
	}
	if len(cids) > 0 {
		if err := tx.Where("comment_id IN ?", cids).Delete(&comment.ArticleCommentLike{}).Error; err != nil {
			return err
		}
		if err := tx.Where("comment_id IN ?", cids).Delete(&comment.ArticleCommentDislike{}).Error; err != nil {
			return err
		}
		if err := tx.Where("id IN ?", cids).Delete(&comment.ArticleComment{}).Error; err != nil {
			return err
		}
	}
	if err := tx.Where("article_id = ?", articleID).Delete(&article.ArticleFavorite{}).Error; err != nil {
		return err
	}
	if err := tx.Where("article_id = ?", articleID).Delete(&article.ArticleCoin{}).Error; err != nil {
		return err
	}
	if err := tx.Where("article_id = ?", articleID).Delete(&extra.ArticleViewHistory{}).Error; err != nil {
		return err
	}
	return tx.Where("id = ?", articleID).Delete(&article.Article{}).Error
}

// DeleteArticleCascadeTx performs cascade delete of an article within an existing transaction.
func (s *ArticleService) DeleteArticleCascadeTx(tx *gorm.DB, articleID uint64) error {
	return deleteArticleCascadeTx(tx, articleID)
}

// DeleteArticle deletes an article with cascade in a new transaction.
func (s *ArticleService) DeleteArticle(ctx context.Context, id uint64) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return deleteArticleCascadeTx(tx, id)
	})
}

// ListArticlesCursor returns published articles with cursor-based pagination.
func (s *ArticleService) ListArticlesCursor(ctx context.Context, cursor uint64, limit int, sort string) (*ListArticlesCursorResult, error) {
	q := s.db.WithContext(ctx).Model(&article.Article{}).Where("status = ?", article.StatusPublished)
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

// ListMyArticlesCursor returns a user's articles with cursor pagination and filtering.
func (s *ArticleService) ListMyArticlesCursor(ctx context.Context, userID uint64, cursor uint64, limit int, status, titleQ, sortKey string) (*ListArticlesCursorResult, error) {
	q := s.db.WithContext(ctx).Model(&article.Article{}).Where("user_id = ?", userID)
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

// ListMyArticlesPage returns user articles with page-based pagination and filtering.
func (s *ArticleService) ListMyArticlesPage(ctx context.Context, userID uint64, page, pageSize int, status, titleQ, sortKey string) (*ListMyArticlesPageResult, error) {
	q := s.db.WithContext(ctx).Model(&article.Article{}).Where("user_id = ?", userID)
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

// ListUserPublishedArticlesCursor returns published articles for a user space with cursor pagination.
func (s *ArticleService) ListUserPublishedArticlesCursor(ctx context.Context, userID uint64, cursor uint64, limit int) (*ListArticlesCursorResult, error) {
	q := s.db.WithContext(ctx).Model(&article.Article{}).Where("user_id = ? AND status = ?", userID, article.StatusPublished)
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

// ListFavoritedArticlesV2 returns favorited articles for a user with cursor pagination.
func (s *ArticleService) ListFavoritedArticlesV2(ctx context.Context, userID, cursor uint64, limit int) ([]article.ArticleFavorite, bool, error) {
	q := s.db.WithContext(ctx).Model(&article.ArticleFavorite{}).Where("user_id = ?", userID)
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

// BatchFetchArticles returns a map of article ID to article pointer.
func (s *ArticleService) BatchFetchArticles(ctx context.Context, ids []uint64, onlyPublished bool) map[uint64]*article.Article {
	out := make(map[uint64]*article.Article, len(ids))
	if len(ids) == 0 {
		return out
	}
	q := s.db.WithContext(ctx).Where("id IN ?", ids)
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

// CountFavoritedArticles counts favorited articles for a user.
func (s *ArticleService) CountFavoritedArticles(ctx context.Context, userID uint64, onlyPublished bool) (int64, error) {
	q := s.db.WithContext(ctx).Table("article_favorites").
		Joins("INNER JOIN articles ON articles.id = article_favorites.article_id")
	if onlyPublished {
		q = q.Where("articles.status = ?", article.StatusPublished)
	}
	q = q.Where("article_favorites.user_id = ?", userID)
	var total int64
	err := q.Count(&total).Error
	return total, err
}

// SetUserService sets UserService for cross-domain queries.

func manuscriptStatusToDB(status string) string {
	switch status {
	case "all", "":
		return ""
	case article.StatusDraft:
		return article.StatusDraft
	case article.StatusProcessing:
		return article.StatusProcessing
	case article.StatusPassed:
		return article.StatusPublished
	case article.StatusRejected:
		return article.StatusRejected
	default:
		return ""
	}
}

// HasArticleCoin checks if a user has already coined on an article and returns the existing record.
func (s *ArticleService) HasArticleCoin(ctx context.Context, userID, articleID uint64) (*article.ArticleCoin, error) {
	var exist article.ArticleCoin
	res := s.db.WithContext(ctx).Where("user_id = ? AND article_id = ?", userID, articleID).Limit(1).Find(&exist)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, nil
	}
	return &exist, nil
}

// PostArticleCoinResult holds the result of a coin operation.
type PostArticleCoinResult struct {
	Coined        bool
	CoinCount     uint64
	Amount        int
	MyCoinAmount  int
	CoinBalance   float64
	DailyProgress int
	AlreadyCoined bool
	Insufficient  bool
}

// PostArticleCoin handles the full article coin transaction.
func (s *ArticleService) PostArticleCoin(ctx context.Context, userID, articleID uint64, amount int) (*PostArticleCoinResult, error) {
	if amount != 1 && amount != 2 {
		amount = 1
	}

	// Load article to get author ID
	var art article.Article
	if err := s.db.WithContext(ctx).First(&art, articleID).Error; err != nil {
		return nil, err
	}

	exist, err := s.HasArticleCoin(ctx, userID, articleID)
	if err != nil {
		return nil, err
	}
	coinBefore := dailyreward.CoinProgress(s.db, userID)

	var spentAmount int
	var myCoinAmount int

	if exist != nil {
		if exist.Amount >= 2 {
			return &PostArticleCoinResult{AlreadyCoined: true}, nil
		}
		spentAmount = 1
		myCoinAmount = 2
		if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := usercoin.SpendOnArticleCoin(tx, userID, art.UserID, articleID, spentAmount); err != nil {
				return err
			}
			if err := tx.Model(&exist).Update("amount", 2).Error; err != nil {
				return err
			}
			return tx.Model(&article.Article{}).Where("id = ?", articleID).
				UpdateColumn("coin_count", gorm.Expr("coin_count + ?", 1)).Error
		}); err != nil {
			if errors.Is(err, usercoin.ErrInsufficientCoins) {
				return &PostArticleCoinResult{Insufficient: true}, nil
			}
			return nil, err
		}
	} else {
		spentAmount = amount
		myCoinAmount = amount
		if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := usercoin.SpendOnArticleCoin(tx, userID, art.UserID, articleID, spentAmount); err != nil {
				return err
			}
			row := article.ArticleCoin{UserID: userID, ArticleID: articleID, Amount: amount}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
			return tx.Model(&article.Article{}).Where("id = ?", articleID).
				UpdateColumn("coin_count", gorm.Expr("coin_count + ?", amount)).Error
		}); err != nil {
			if errors.Is(err, usercoin.ErrInsufficientCoins) {
				return &PostArticleCoinResult{Insufficient: true}, nil
			}
			return nil, err
		}
	}

	coinAfter := dailyreward.CoinProgress(s.db, userID)
	_ = dailyreward.GrantCoinExp(s.db, userID, coinBefore, coinAfter)

	_ = s.db.WithContext(ctx).First(&art, articleID)

	return &PostArticleCoinResult{
		Coined:        true,
		CoinCount:     art.CoinCount,
		Amount:        spentAmount,
		MyCoinAmount:  myCoinAmount,
		CoinBalance:   usercoin.BalanceFloat(0),
		DailyProgress: coinAfter,
	}, nil
}

// CountByStatus returns article count by status.
func (s *ArticleService) CountByStatus(ctx context.Context, status string) (int64, error) {
	var cnt int64
	err := s.db.WithContext(ctx).Model(&article.Article{}).Where("status = ?", status).Count(&cnt).Error
	return cnt, err
}

// AdminUpdateArticle updates article fields by ID.
func (s *ArticleService) AdminUpdateArticle(ctx context.Context, id uint64, updates map[string]interface{}) error {
	return s.db.WithContext(ctx).Model(&article.Article{}).Where("id = ?", id).Updates(updates).Error
}

// AdminListArticlesResult holds paginated admin article list results.
type AdminListArticlesResult struct {
	Total        int64
	Rows         []article.Article
	PendingCount int64
}

// AdminListArticles returns paginated articles with filters for admin panel.
func (s *ArticleService) AdminListArticles(ctx context.Context, statuses []string, titleQ string, page, pageSize int) (*AdminListArticlesResult, error) {
	q := s.db.WithContext(ctx).Model(&article.Article{})
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
	pending, _ := s.CountByStatus(ctx, article.StatusPendingReview)
	return &AdminListArticlesResult{
		Total:        total,
		Rows:         rows,
		PendingCount: pending,
	}, nil
}

// AdminDeleteArticleCascade deletes an article within a transaction with a custom function.
func (s *ArticleService) AdminDeleteArticleCascade(ctx context.Context, id uint64, fn func(tx *gorm.DB) error) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := fn(tx); err != nil {
			return err
		}
		return nil
	})
}
