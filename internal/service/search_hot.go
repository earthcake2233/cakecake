package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/redis/go-redis/v9"

	"minibili/internal/pkg/sensitive"
)

const (
	keyHotSearchRank   = "hotsearch:rank"
	keyHotSearchLabel  = "hotsearch:label"
	prefixHotSearchDed = "hotsearch:dedup:"
	prefixHotSearchNew = "hotsearch:badge:new:"

	// HotSearchDedupTTL 同一用户/IP 在窗口内重复搜索同一词只计一次。
	HotSearchDedupTTL = 10 * time.Minute
	hotSearchNewTTL   = 48 * time.Hour
)

// HotSearchItem is one row for GET /api/v1/hot-search.
type HotSearchItem struct {
	Rank  int    `json:"rank"`
	Title string `json:"title"`
	Badge string `json:"badge"`
}

// HotSearchRedisRow is one row from Redis ZSET with score (admin).
type HotSearchRedisRow struct {
	Rank    int     `json:"rank"`
	Title   string  `json:"title"`
	Keyword string  `json:"keyword"`
	Score   float64 `json:"score"`
	Badge   string  `json:"badge"`
}

// HotSearchMergedDetail includes intervention source for admin dashboard.
type HotSearchMergedDetail struct {
	Rank    int    `json:"rank"`
	Title   string `json:"title"`
	Badge   string `json:"badge"`
	Source  string `json:"source"` // pin | manual | auto
	Keyword string `json:"keyword,omitempty"`
	OpID    uint64 `json:"op_id,omitempty"`
}

// SearchHotRecorder aggregates search keywords in Redis (ZSET + dedup keys).
type SearchHotRecorder struct {
	Rdb  *redis.Client
	Sens *sensitive.Filter
}

// NormalizeSearchKeyword lowercases and strips spaces for dedup / ZSET member.
func NormalizeSearchKeyword(keyword string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(keyword)) {
		if unicode.IsSpace(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func dedupIdentity(userID uint64, clientKey string) string {
	if userID > 0 {
		return "u:" + strconv.FormatUint(userID, 10)
	}
	ip := strings.TrimSpace(clientKey)
	if ip == "" {
		return "anon"
	}
	return "ip:" + ip
}

// Record counts a search keyword if it passes sensitive filter and dedup window.
func (r *SearchHotRecorder) Record(ctx context.Context, userID uint64, clientKey, keyword string) error {
	if r == nil || r.Rdb == nil {
		return nil
	}
	kw := strings.TrimSpace(keyword)
	if kw == "" || utf8.RuneCountInString(kw) > 50 {
		return nil
	}
	if r.Sens != nil {
		if err := r.Sens.Check(kw); err != nil {
			return nil
		}
	}
	norm := NormalizeSearchKeyword(kw)
	if norm == "" {
		return nil
	}
	dedKey := prefixHotSearchDed + dedupIdentity(userID, clientKey) + ":" + norm
	ok, err := r.Rdb.SetNX(ctx, dedKey, "1", HotSearchDedupTTL).Result()
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	before, err := r.Rdb.ZScore(ctx, keyHotSearchRank, norm).Result()
	if err != nil && err != redis.Nil {
		return err
	}
	if err == redis.Nil {
		before = 0
	}
	pipe := r.Rdb.Pipeline()
	pipe.ZIncrBy(ctx, keyHotSearchRank, 1, norm)
	pipe.HSet(ctx, keyHotSearchLabel, norm, kw)
	if before == 0 {
		pipe.Set(ctx, prefixHotSearchNew+norm, "1", hotSearchNewTTL)
	}
	_, err = pipe.Exec(ctx)
	return err
}

// Top returns top keywords by search count.
func (r *SearchHotRecorder) Top(ctx context.Context, limit int) ([]HotSearchItem, error) {
	if r == nil || r.Rdb == nil || limit <= 0 {
		return nil, nil
	}
	zs, err := r.Rdb.ZRevRangeWithScores(ctx, keyHotSearchRank, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, err
	}
	if len(zs) == 0 {
		return []HotSearchItem{}, nil
	}
	norms := make([]string, 0, len(zs))
	for _, z := range zs {
		norms = append(norms, z.Member.(string))
	}
	labels, err := r.Rdb.HMGet(ctx, keyHotSearchLabel, norms...).Result()
	if err != nil {
		return nil, err
	}
	out := make([]HotSearchItem, 0, len(zs))
	for i, z := range zs {
		norm, _ := z.Member.(string)
		title := norm
		if i < len(labels) && labels[i] != nil {
			if s, ok := labels[i].(string); ok && strings.TrimSpace(s) != "" {
				title = strings.TrimSpace(s)
			}
		}
		if title == "" {
			continue
		}
		rank := len(out) + 1
		badge := ""
		switch rank {
		case 1:
			badge = "热"
		default:
			if r.Rdb.Exists(ctx, prefixHotSearchNew+norm).Val() > 0 {
				badge = "新"
			}
		}
		out = append(out, HotSearchItem{
			Rank:  rank,
			Title: title,
			Badge: badge,
		})
	}
	return out, nil
}

// TopWithScores returns Redis auto rank with search counts (admin).
func (r *SearchHotRecorder) TopWithScores(ctx context.Context, limit int) ([]HotSearchRedisRow, error) {
	if r == nil || r.Rdb == nil || limit <= 0 {
		return nil, nil
	}
	zs, err := r.Rdb.ZRevRangeWithScores(ctx, keyHotSearchRank, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, err
	}
	if len(zs) == 0 {
		return []HotSearchRedisRow{}, nil
	}
	norms := make([]string, 0, len(zs))
	for _, z := range zs {
		norms = append(norms, z.Member.(string))
	}
	labels, err := r.Rdb.HMGet(ctx, keyHotSearchLabel, norms...).Result()
	if err != nil {
		return nil, err
	}
	out := make([]HotSearchRedisRow, 0, len(zs))
	for i, z := range zs {
		norm, _ := z.Member.(string)
		title := norm
		if i < len(labels) && labels[i] != nil {
			if s, ok := labels[i].(string); ok && strings.TrimSpace(s) != "" {
				title = strings.TrimSpace(s)
			}
		}
		if title == "" {
			continue
		}
		rank := len(out) + 1
		badge := ""
		switch rank {
		case 1:
			badge = "热"
		default:
			if r.Rdb.Exists(ctx, prefixHotSearchNew+norm).Val() > 0 {
				badge = "新"
			}
		}
		out = append(out, HotSearchRedisRow{
			Rank:    rank,
			Title:   title,
			Keyword: norm,
			Score:   z.Score,
			Badge:   badge,
		})
	}
	return out, nil
}

// BoostKeyword increases Redis hot-search score for a keyword.
func (r *SearchHotRecorder) BoostKeyword(ctx context.Context, keyword string, delta float64) error {
	if r == nil || r.Rdb == nil || delta == 0 {
		return nil
	}
	kw := strings.TrimSpace(keyword)
	norm := NormalizeSearchKeyword(kw)
	if norm == "" {
		return nil
	}
	if kw == "" {
		kw = norm
	}
	pipe := r.Rdb.Pipeline()
	pipe.ZIncrBy(ctx, keyHotSearchRank, delta, norm)
	pipe.HSet(ctx, keyHotSearchLabel, norm, kw)
	_, err := pipe.Exec(ctx)
	return err
}

// RemoveKeyword deletes a keyword from Redis hot-search rank.
func (r *SearchHotRecorder) RemoveKeyword(ctx context.Context, keyword string) error {
	if r == nil || r.Rdb == nil {
		return nil
	}
	norm := NormalizeSearchKeyword(keyword)
	if norm == "" {
		return nil
	}
	pipe := r.Rdb.Pipeline()
	pipe.ZRem(ctx, keyHotSearchRank, norm)
	pipe.HDel(ctx, keyHotSearchLabel, norm)
	pipe.Del(ctx, prefixHotSearchNew+norm)
	_, err := pipe.Exec(ctx)
	return err
}

// DedupKeyForTest exposes dedup key format for tests.
func DedupKeyForTest(userID uint64, clientKey, norm string) string {
	return fmt.Sprintf("%s%s:%s", prefixHotSearchDed, dedupIdentity(userID, clientKey), norm)
}
