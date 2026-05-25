package service

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestSearchHotRecorder_DedupAndTop(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	rec := &SearchHotRecorder{Rdb: rdb, Sens: nil}
	ctx := context.Background()

	if err := rec.Record(ctx, 1, "", "  Spring Anime  "); err != nil {
		t.Fatal(err)
	}
	if err := rec.Record(ctx, 1, "", "spring anime"); err != nil {
		t.Fatal(err)
	}
	n, _ := rdb.ZScore(ctx, keyHotSearchRank, "springanime").Result()
	if n != 1 {
		t.Fatalf("want 1 count after dedup, got %v", n)
	}
	top, err := rec.Top(ctx, 5)
	if err != nil || len(top) != 1 || top[0].Title != "Spring Anime" {
		t.Fatalf("top=%v err=%v", top, err)
	}
}

func TestSearchHotRecorder_DedupTTLExpires(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	rec := &SearchHotRecorder{Rdb: rdb}
	ctx := context.Background()
	_ = rec.Record(ctx, 2, "1.2.3.4", "keyword")
	mr.FastForward(HotSearchDedupTTL + time.Second)
	_ = rec.Record(ctx, 2, "1.2.3.4", "keyword")
	n, _ := rdb.ZScore(ctx, keyHotSearchRank, "keyword").Result()
	if n != 2 {
		t.Fatalf("want 2 after ttl, got %v", n)
	}
}
