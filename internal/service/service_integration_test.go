//go:build integration

package service

import (
	"cakecake/internal/model/admin"
	"cakecake/internal/model/agent"
	"cakecake/internal/model/dm"
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// setupSQLiteDB creates an in-memory SQLite DB with auto-migration for tests.
func setupSQLiteDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&admin.HotSearchOp{},
		&admin.HotSearchDisplayLayout{},
		&dm.DmConversation{},
		&agent.AgentProfile{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

// ---------- hotsearch.SearchHotRecorder: TopWithScores, BoostKeyword, RemoveKeyword ----------

func TestSearchHotRecorder_TopWithScores(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	rec := &hotsearch.SearchHotRecorder{Rdb: rdb}
	ctx := context.Background()

	// empty initially
	rows, err := rec.TopWithScores(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected empty, got %d rows", len(rows))
	}

	// seed records — use different user IDs to avoid dedup
	_ = rec.Record(ctx, 1, "", "golang")
	_ = rec.Record(ctx, 2, "", "rust")
	_ = rec.Record(ctx, 3, "", "rust")

	rows, err = rec.TopWithScores(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0].Keyword != "rust" || rows[0].Score != 2 {
		t.Errorf("top: want rust score=2, got %s score=%f", rows[0].Keyword, rows[0].Score)
	}
	if rows[1].Keyword != "golang" || rows[1].Score != 1 {
		t.Errorf("second: want golang score=1, got %s score=%f", rows[1].Keyword, rows[1].Score)
	}
	if rows[0].Rank != 1 || rows[1].Rank != 2 {
		t.Errorf("rank: want 1,2 got %d,%d", rows[0].Rank, rows[1].Rank)
	}

	// limit < count
	rows, err = rec.TopWithScores(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("limit=1: expected 1 row, got %d", len(rows))
	}

	// nil/zero receiver
	rows, err = (*hotsearch.SearchHotRecorder)(nil).TopWithScores(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if rows != nil {
		t.Fatal("nil receiver should return nil")
	}

	// limit <= 0
	rows, err = rec.TopWithScores(ctx, 0)
	if err != nil {
		t.Fatal(err)
	}
	if rows != nil {
		t.Fatal("limit=0 should return nil")
	}
}

func TestSearchHotRecorder_BoostKeyword(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	rec := &hotsearch.SearchHotRecorder{Rdb: rdb}
	ctx := context.Background()

	// boost a keyword
	if err := rec.BoostKeyword(ctx, "feature-x", 10); err != nil {
		t.Fatal(err)
	}
	rows, err := rec.TopWithScores(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Score != 10 {
		t.Fatalf("want 1 row score=10, got %d score=%f", len(rows), rows[0].Score)
	}

	// boost again to cumulate
	if err := rec.BoostKeyword(ctx, "feature-x", 5); err != nil {
		t.Fatal(err)
	}
	rows, err = rec.TopWithScores(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Score != 15 {
		t.Fatalf("want score=15, got %f", rows[0].Score)
	}

	// boost with display title
	_ = rec.BoostKeyword(ctx, "Feature X", 3)
	rows, _ = rec.TopWithScores(ctx, 10)
	for _, r := range rows {
		if r.Keyword == "featurex" && r.Title != "Feature X" {
			t.Errorf("want title='Feature X', got %q", r.Title)
		}
	}

	// zero delta
	if err := rec.BoostKeyword(ctx, "feature-x", 0); err != nil {
		t.Fatal(err)
	}
	rows, _ = rec.TopWithScores(ctx, 10)
	if rows[0].Score != 15 {
		t.Errorf("delta=0 should not change score, got %f", rows[0].Score)
	}

	// empty keyword
	if err := rec.BoostKeyword(ctx, "  ", 5); err != nil {
		t.Fatal(err)
	}

	// nil receiver
	if err := (*hotsearch.SearchHotRecorder)(nil).BoostKeyword(ctx, "x", 1); err != nil {
		t.Fatal(err)
	}
}

func TestSearchHotRecorder_RemoveKeyword(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	rec := &hotsearch.SearchHotRecorder{Rdb: rdb}
	ctx := context.Background()

	// seed
	_ = rec.Record(ctx, 1, "", "remove-me")
	_ = rec.Record(ctx, 1, "", "keep-me")

	rows, err := rec.TopWithScores(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows before remove, got %d", len(rows))
	}

	// remove one
	if err := rec.RemoveKeyword(ctx, "remove-me"); err != nil {
		t.Fatal(err)
	}
	rows, err = rec.TopWithScores(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row after remove, got %d", len(rows))
	}
	if rows[0].Keyword != "keep-me" {
		t.Errorf("remaining should be 'keep-me', got %q", rows[0].Keyword)
	}

	// remove non-existent
	if err := rec.RemoveKeyword(ctx, "nonexistent"); err != nil {
		t.Fatal(err)
	}

	// empty keyword
	if err := rec.RemoveKeyword(ctx, "  "); err != nil {
		t.Fatal(err)
	}

	// nil receiver
	if err := (*hotsearch.SearchHotRecorder)(nil).RemoveKeyword(ctx, "x"); err != nil {
		t.Fatal(err)
	}
}

// ---------- playcount.PlayCounter (skipped) ----------

func TestPlayCounter_Skip(t *testing.T) {
	t.Skip("requires Redis and DB for integration")
}
