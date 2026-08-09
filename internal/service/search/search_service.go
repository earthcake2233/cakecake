package search

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	searchclient "cakecake/internal/search"
)

// SearchService owns the Elasticsearch client and the DB reads used for
// indexing/enrichment, so handlers never touch ES or DB directly.
type SearchService struct {
	es    *searchclient.Client
	store SearchStore
	rdb   *redis.Client
	log   *zap.Logger
}

// NewSearchService creates a SearchService with an optional ES client, storage,
// cache, and logger.
func NewSearchService(es *searchclient.Client, db *gorm.DB, rdb *redis.Client, log *zap.Logger) *SearchService {
	return &SearchService{es: es, store: NewSearchStore(db), rdb: rdb, log: log}
}

// SearchStore is the DB-backed slice of the search domain (Phase 1: *gorm.DB impl).
type SearchStore interface {
	EnrichUserHits(ctx context.Context, viewer uint64, hits []searchclient.UserHit) []searchclient.UserHit
	IndexVideoFromDB(ctx context.Context, es *searchclient.Client, videoID uint64) error
	IndexArticleFromDB(ctx context.Context, es *searchclient.Client, articleID uint64) error
	IndexUserFromDB(ctx context.Context, es *searchclient.Client, userID uint64) error
}

// SearchStoreImpl implements SearchStore using *gorm.DB (Phase 1 monolith).
type SearchStoreImpl struct {
	db *gorm.DB
}

var _ SearchStore = (*SearchStoreImpl)(nil)

// NewSearchStore creates a gorm-backed SearchStore implementation.
func NewSearchStore(db *gorm.DB) *SearchStoreImpl {
	return &SearchStoreImpl{db: db}
}

// EnrichUserHits fills profile stats, follow state, and recent archives for search hits.
func (p *SearchStoreImpl) EnrichUserHits(ctx context.Context, viewer uint64, hits []searchclient.UserHit) []searchclient.UserHit {
	if p.db == nil {
		return hits
	}
	return searchclient.EnrichUserHits(p.db, viewer, hits)
}

// IndexVideoFromDB indexes a video from the database.
func (p *SearchStoreImpl) IndexVideoFromDB(ctx context.Context, es *searchclient.Client, videoID uint64) error {
	return es.IndexVideoFromDB(ctx, p.db, videoID)
}

// IndexArticleFromDB indexes an article from the database.
func (p *SearchStoreImpl) IndexArticleFromDB(ctx context.Context, es *searchclient.Client, articleID uint64) error {
	return es.IndexArticleFromDB(ctx, p.db, articleID)
}

// IndexUserFromDB indexes a user from the database.
func (p *SearchStoreImpl) IndexUserFromDB(ctx context.Context, es *searchclient.Client, userID uint64) error {
	return es.IndexUserFromDB(ctx, p.db, userID)
}

// Enabled reports whether the Elasticsearch client is configured and enabled.
func (s *SearchService) Enabled() bool {
	return s.es != nil && s.es.Enabled()
}

// SearchAll runs a full multi-domain search when Elasticsearch is enabled.
func (s *SearchService) SearchAll(ctx context.Context, params searchclient.SearchParams) (*searchclient.AllResult, error) {
	if !s.Enabled() {
		return nil, nil
	}
	return s.es.SearchAll(ctx, params)
}

// EnrichUserHits fills user profile data into search hit rows.
func (s *SearchService) EnrichUserHits(ctx context.Context, viewer uint64, hits []searchclient.UserHit) []searchclient.UserHit {
	return s.store.EnrichUserHits(ctx, viewer, hits)
}

// CacheGet reads a search result cache entry (best-effort).
func (s *SearchService) CacheGet(ctx context.Context, key string) (string, error) {
	if s.rdb == nil {
		return "", nil
	}
	return s.rdb.Get(ctx, key).Result()
}

// CacheSet writes a search result cache entry (best-effort).
func (s *SearchService) CacheSet(ctx context.Context, key, value string, ttl time.Duration) error {
	if s.rdb == nil {
		return nil
	}
	return s.rdb.Set(ctx, key, value, ttl).Err()
}

// IndexVideoFromDB indexes a video by id.
func (s *SearchService) IndexVideoFromDB(ctx context.Context, videoID uint64) error {
	if !s.Enabled() {
		return nil
	}
	return s.store.IndexVideoFromDB(ctx, s.es, videoID)
}

// DeleteVideo removes a video document from the index.
func (s *SearchService) DeleteVideo(ctx context.Context, videoID uint64) error {
	if !s.Enabled() {
		return nil
	}
	return s.es.DeleteVideo(ctx, videoID)
}

// IndexArticleFromDB indexes an article by id.
func (s *SearchService) IndexArticleFromDB(ctx context.Context, articleID uint64) error {
	if !s.Enabled() {
		return nil
	}
	return s.store.IndexArticleFromDB(ctx, s.es, articleID)
}

// DeleteArticle removes an article document from the index.
func (s *SearchService) DeleteArticle(ctx context.Context, articleID uint64) error {
	if !s.Enabled() {
		return nil
	}
	return s.es.DeleteArticle(ctx, articleID)
}

// IndexUserFromDB indexes a user by id.
func (s *SearchService) IndexUserFromDB(ctx context.Context, userID uint64) error {
	if !s.Enabled() {
		return nil
	}
	return s.store.IndexUserFromDB(ctx, s.es, userID)
}
