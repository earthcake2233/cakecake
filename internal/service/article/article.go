package article

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

	"cakecake/internal/pkg/usercoin"
	"cakecake/internal/search"
)

// ArticleService handles article business logic.
type ArticleService struct {
	store ArticleStore
	rdb   *redis.Client
	log   *zap.Logger
	es    *search.Client
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

// ListArticlesCursorResult is a cursor-paginated article list.
type ListArticlesCursorResult struct {
	Items   []article.Article
	HasMore bool
}

// NewArticleService creates an ArticleService with storage, cache, logger,
// and optional search client.
func NewArticleService(db *gorm.DB, rdb *redis.Client, log *zap.Logger, es *search.Client) *ArticleService {
	return &ArticleService{store: NewArticleStore(db), rdb: rdb, log: log, es: es}
}

// Publish marks an article published and re-indexes search (post-review or direct publish).
func (s *ArticleService) Publish(ctx context.Context, articleID uint64, adminID *uint64) error {
	return s.store.PublishArticle(ctx, s.es, s.log, articleID, adminID)
}

// CreateArticle inserts a new article.
func (s *ArticleService) CreateArticle(ctx context.Context, art *article.Article) error {
	return s.store.CreateArticle(ctx, art)
}

// GetArticleByID returns an article by ID, regardless of status.
func (s *ArticleService) GetArticleByID(ctx context.Context, id uint64) (*article.Article, error) {
	return s.store.GetArticleByID(ctx, id)
}

// GetPublishedArticle returns a published article by ID.
func (s *ArticleService) GetPublishedArticle(ctx context.Context, id uint64) (*article.Article, error) {
	return s.store.GetPublishedArticle(ctx, id)
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
	return s.store.UpdateArticleByID(ctx, id, updates)
}

// IncrementArticleView increments the view count for an article.
func (s *ArticleService) IncrementArticleView(ctx context.Context, id uint64) error {
	return s.store.IncrArticleView(ctx, id)
}

// BatchArticleEngagementByViewer returns engagement state for multiple articles for a viewer.
func (s *ArticleService) BatchArticleEngagementByViewer(ctx context.Context, viewerID uint64, articleIDs []uint64) map[uint64]*ArticleEngagement {
	out := make(map[uint64]*ArticleEngagement, len(articleIDs))
	if viewerID == 0 || len(articleIDs) == 0 {
		return out
	}
	faved := map[uint64]bool{}
	favRows, _ := s.store.BatchFavRows(ctx, viewerID, articleIDs)
	for i := range favRows {
		faved[favRows[i].ArticleID] = true
	}
	coinAmt := map[uint64]int{}
	coinRows, _ := s.store.BatchCoinRows(ctx, viewerID, articleIDs)
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
	return s.store.CountArticlesByStatus(ctx, userID)
}

// ToggleArticleFavorite toggles favorite on an article. Returns favorited state and updated FavCount.
func (s *ArticleService) ToggleArticleFavorite(ctx context.Context, userID, articleID uint64) (bool, uint64, error) {
	return s.store.ToggleArticleFavorite(ctx, userID, articleID)
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
	return s.store.DeleteArticleCascadeTx(tx, articleID)
}

// DeleteArticle deletes an article with cascade in a new transaction.
func (s *ArticleService) DeleteArticle(ctx context.Context, id uint64) error {
	return s.store.DeleteArticle(ctx, id)
}

// ListArticlesCursor returns published articles with cursor-based pagination.
func (s *ArticleService) ListArticlesCursor(ctx context.Context, cursor uint64, limit int, sort string) (*ListArticlesCursorResult, error) {
	return s.store.ListArticlesCursor(ctx, cursor, limit, sort)
}

// ListMyArticlesCursor returns a user's articles with cursor pagination and filtering.
func (s *ArticleService) ListMyArticlesCursor(ctx context.Context, userID uint64, cursor uint64, limit int, status, titleQ, sortKey string) (*ListArticlesCursorResult, error) {
	return s.store.ListMyArticlesCursor(ctx, userID, cursor, limit, status, titleQ, sortKey)
}

// ListMyArticlesPage returns user articles with page-based pagination and filtering.
func (s *ArticleService) ListMyArticlesPage(ctx context.Context, userID uint64, page, pageSize int, status, titleQ, sortKey string) (*ListMyArticlesPageResult, error) {
	return s.store.ListMyArticlesPage(ctx, userID, page, pageSize, status, titleQ, sortKey)
}

// ListUserPublishedArticlesCursor returns published articles for a user space with cursor pagination.
func (s *ArticleService) ListUserPublishedArticlesCursor(ctx context.Context, userID uint64, cursor uint64, limit int) (*ListArticlesCursorResult, error) {
	return s.store.ListUserPublishedArticlesCursor(ctx, userID, cursor, limit)
}

// ListFavoritedArticlesV2 returns favorited articles for a user with cursor pagination.
func (s *ArticleService) ListFavoritedArticlesV2(ctx context.Context, userID, cursor uint64, limit int) ([]article.ArticleFavorite, bool, error) {
	return s.store.ListFavoritedArticlesV2(ctx, userID, cursor, limit)
}

// BatchFetchArticles returns a map of article ID to article pointer.
func (s *ArticleService) BatchFetchArticles(ctx context.Context, ids []uint64, onlyPublished bool) map[uint64]*article.Article {
	return s.store.BatchFetchArticles(ctx, ids, onlyPublished)
}

// CountFavoritedArticles counts favorited articles for a user.
func (s *ArticleService) CountFavoritedArticles(ctx context.Context, userID uint64, onlyPublished bool) (int64, error) {
	return s.store.CountFavoritedArticles(ctx, userID, onlyPublished)
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
	return s.store.HasArticleCoin(ctx, userID, articleID)
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
	art, err := s.store.GetArticleByID(ctx, articleID)
	if err != nil {
		return nil, err
	}

	exist, err := s.HasArticleCoin(ctx, userID, articleID)
	if err != nil {
		return nil, err
	}
	coinBefore := s.store.CoinProgress(userID)

	var spentAmount int
	var myCoinAmount int

	if exist != nil {
		if exist.Amount >= 2 {
			return &PostArticleCoinResult{AlreadyCoined: true}, nil
		}
		spentAmount = 1
		myCoinAmount = 2
		if err := s.store.UpdateArticleCoinTx(ctx, userID, art.UserID, articleID, exist); err != nil {
			if errors.Is(err, usercoin.ErrInsufficientCoins) {
				return &PostArticleCoinResult{Insufficient: true}, nil
			}
			return nil, err
		}
	} else {
		spentAmount = amount
		myCoinAmount = amount
		if err := s.store.CreateArticleCoinTx(ctx, userID, art.UserID, articleID, amount); err != nil {
			if errors.Is(err, usercoin.ErrInsufficientCoins) {
				return &PostArticleCoinResult{Insufficient: true}, nil
			}
			return nil, err
		}
	}

	coinAfter := s.store.CoinProgress(userID)
	_ = s.store.GrantCoinExp(userID, coinBefore, coinAfter)

	if reloaded, err := s.store.GetArticleByID(ctx, articleID); err == nil {
		art = reloaded
	}

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
	return s.store.CountByStatus(ctx, status)
}

// AdminUpdateArticle updates article fields by ID.
func (s *ArticleService) AdminUpdateArticle(ctx context.Context, id uint64, updates map[string]interface{}) error {
	return s.store.UpdateArticleByID(ctx, id, updates)
}

// AdminListArticlesResult holds paginated admin article list results.
type AdminListArticlesResult struct {
	Total        int64
	Rows         []article.Article
	PendingCount int64
}

// AdminListArticles returns paginated articles with filters for admin panel.
func (s *ArticleService) AdminListArticles(ctx context.Context, statuses []string, titleQ string, page, pageSize int) (*AdminListArticlesResult, error) {
	return s.store.AdminListArticles(ctx, statuses, titleQ, page, pageSize)
}

// AdminDeleteArticleCascade deletes an article within a transaction with a custom function.
func (s *ArticleService) AdminDeleteArticleCascade(ctx context.Context, id uint64, fn func(tx *gorm.DB) error) error {
	return s.store.AdminDeleteArticleCascade(ctx, id, fn)
}
