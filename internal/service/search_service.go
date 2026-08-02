package service

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"cakecake/internal/search"
)

// SearchService owns the Elasticsearch client and the DB reads used for
// indexing/enrichment, so handlers never touch ES or DB directly.
type SearchService struct {
	es  *search.Client
	db  *gorm.DB
	rdb *redis.Client
	log *zap.Logger
}

func NewSearchService(es *search.Client, db *gorm.DB, rdb *redis.Client, log *zap.Logger) *SearchService {
	return &SearchService{es: es, db: db, rdb: rdb, log: log}
}

func (s *SearchService) Enabled() bool {
	return s.es != nil && s.es.Enabled()
}

func (s *SearchService) SearchAll(ctx context.Context, params search.SearchParams) (*search.AllResult, error) {
	if !s.Enabled() {
		return nil, nil
	}
	return s.es.SearchAll(ctx, params)
}

func (s *SearchService) EnrichUserHits(ctx context.Context, viewer uint64, hits []search.UserHit) []search.UserHit {
	if s.db == nil {
		return hits
	}
	return search.EnrichUserHits(s.db, viewer, hits)
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

func (s *SearchService) IndexVideoFromDB(ctx context.Context, videoID uint64) error {
	if !s.Enabled() {
		return nil
	}
	return s.es.IndexVideoFromDB(ctx, s.db, videoID)
}

func (s *SearchService) DeleteVideo(ctx context.Context, videoID uint64) error {
	if !s.Enabled() {
		return nil
	}
	return s.es.DeleteVideo(ctx, videoID)
}

func (s *SearchService) IndexArticleFromDB(ctx context.Context, articleID uint64) error {
	if !s.Enabled() {
		return nil
	}
	return s.es.IndexArticleFromDB(ctx, s.db, articleID)
}

func (s *SearchService) DeleteArticle(ctx context.Context, articleID uint64) error {
	if !s.Enabled() {
		return nil
	}
	return s.es.DeleteArticle(ctx, articleID)
}

func (s *SearchService) IndexUserFromDB(ctx context.Context, userID uint64) error {
	if !s.Enabled() {
		return nil
	}
	return s.es.IndexUserFromDB(ctx, s.db, userID)
}
