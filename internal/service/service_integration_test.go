//go:build integration

package service

import (
	"context"
	"testing"
	
	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"minibili/internal/config"
	"minibili/internal/model"
)

// setupSQLiteDB creates an in-memory SQLite DB with auto-migration for tests.
func setupSQLiteDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.HotSearchOp{},
		&model.HotSearchDisplayLayout{},
		&model.DmConversation{},
		&model.AgentProfile{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

// ---------- SearchHotRecorder: TopWithScores, BoostKeyword, RemoveKeyword ----------

func TestSearchHotRecorder_TopWithScores(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	rec := &SearchHotRecorder{Rdb: rdb}
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
	rows, err = (*SearchHotRecorder)(nil).TopWithScores(ctx, 10)
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
	rec := &SearchHotRecorder{Rdb: rdb}
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
	if err := (*SearchHotRecorder)(nil).BoostKeyword(ctx, "x", 1); err != nil {
		t.Fatal(err)
	}
}

func TestSearchHotRecorder_RemoveKeyword(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	rec := &SearchHotRecorder{Rdb: rdb}
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
	if err := (*SearchHotRecorder)(nil).RemoveKeyword(ctx, "x"); err != nil {
		t.Fatal(err)
	}
}

// ---------- PlayCounter (skipped) ----------

func TestPlayCounter_Skip(t *testing.T) {
	t.Skip("requires Redis and DB for integration")
}

// ---------- DanmakuRelay (skipped) ----------

func TestDanmakuRelay_Skip(t *testing.T) {
	t.Skip("requires Redis and MQ for integration")
}

// ---------- AgentService: CheckQuota ----------

func TestAgentService_CheckQuota(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	log := zap.NewNop()

	t.Run("nil service returns true", func(t *testing.T) {
		var s *AgentService
		if !s.CheckQuota(context.Background(), 1) {
			t.Error("nil service should return true")
		}
	})

	t.Run("nil redis returns true", func(t *testing.T) {
		s := &AgentService{Redis: nil, Log: log}
		if !s.CheckQuota(context.Background(), 1) {
			t.Error("nil redis should return true")
		}
	})

	t.Run("nil cfg returns true", func(t *testing.T) {
		s := &AgentService{Redis: rdb, Log: log, Cfg: nil}
		if !s.CheckQuota(context.Background(), 1) {
			t.Error("nil cfg should return true")
		}
	})

	t.Run("quota zero means unlimited", func(t *testing.T) {
		s := &AgentService{Redis: rdb, Log: log, Cfg: &config.C{AgentDailyQuota: 0}}
		if !s.CheckQuota(context.Background(), 42) {
			t.Error("quota=0 should be unlimited (return true)")
		}
	})

	t.Run("under quota returns true", func(t *testing.T) {
		ctx := context.Background()
		s := &AgentService{Redis: rdb, Log: log, Cfg: &config.C{AgentDailyQuota: 10}}
		if !s.CheckQuota(ctx, 100) {
			t.Error("fresh user should have quota")
		}
	})

	t.Run("at quota returns true (equal)", func(t *testing.T) {
		ctx := context.Background()
		s := &AgentService{Redis: rdb, Log: log, Cfg: &config.C{AgentDailyQuota: 5}}
		key := s.quotaKey(200)
		_ = rdb.Set(ctx, key, 5, 0).Err()
		// CheckQuota: n >= quota returns false? Let's check: if s.RC != nil { quota = s.RC.GetInt(...) }
		// n < quota -> true; otherwise false (n >= quota -> false)
		// So at quota (n=5, quota=5) -> n < quota is false -> returns false
		// Wait, that means at quota it returns false? That doesn't seem right.
		// Let me re-read: if n < quota { return true } else { return false }
		// n=5, quota=5, n < quota = false, return false.
		// Hmm but the method returns true for "has quota available". If n >= quota, no quota available => false.
		// Actually let me re-read the implementation:
		// n, err := s.Redis.Get(ctx, s.quotaKey(userID)).Int()
		// if err == redis.Nil { return true }  // no key yet
		// return err != nil || n < quota
		// So n < quota => true (ok), n >= quota => false (exceeded)
		// At n=5, quota=5: n < quota is false, so returns false (not OK)
		// At n=5, quota=10: n < quota is true, so returns true (OK)
		// So correctly at quota means exceeded
		got := s.CheckQuota(ctx, 200)
		if got {
			t.Error("at quota (5==5) should return false (no quota)")
		}
	})

	t.Run("over quota returns false", func(t *testing.T) {
		ctx := context.Background()
		s := &AgentService{Redis: rdb, Log: log, Cfg: &config.C{AgentDailyQuota: 5}}
		key := s.quotaKey(300)
		_ = rdb.Set(ctx, key, 10, 0).Err()
		if s.CheckQuota(ctx, 300) {
			t.Error("over quota should return false")
		}
	})

	t.Run("no key yet returns true", func(t *testing.T) {
		ctx := context.Background()
		s := &AgentService{Redis: rdb, Log: log, Cfg: &config.C{AgentDailyQuota: 5}}
		if !s.CheckQuota(ctx, 999) {
			t.Error("no key (redis.Nil) should return true")
		}
	})
}

// ---------- AgentService: IsBotUser ----------

func TestAgentService_IsBotUser(t *testing.T) {
	log := zap.NewNop()

	t.Run("nil service", func(t *testing.T) {
		var s *AgentService
		if s.IsBotUser(1) {
			t.Error("nil service should return false")
		}
	})

	t.Run("nil db", func(t *testing.T) {
		s := &AgentService{DB: nil, Log: log}
		if s.IsBotUser(1) {
			t.Error("nil db should return false")
		}
	})

	t.Run("zero uid", func(t *testing.T) {
		db := setupSQLiteDB(t)
		s := &AgentService{DB: db, Log: log}
		if s.IsBotUser(0) {
			t.Error("zero uid should return false")
		}
	})

	t.Run("user not a bot", func(t *testing.T) {
		db := setupSQLiteDB(t)
		s := &AgentService{DB: db, Log: log}
		if s.IsBotUser(42) {
			t.Error("no agent profile should return false")
		}
	})

	t.Run("user is a bot", func(t *testing.T) {
		db := setupSQLiteDB(t)
		// Create an agent profile with BotUserID=100
		profile := model.AgentProfile{
			Slug:                "test-bot",
			BotUserID:           100,
			DisplayName:         "Test Bot",
			SystemPrompt:        "You are a test bot.",
			WelcomeMessagesJSON: `["Hello"]`,
			Enabled:             true,
		}
		if err := db.Create(&profile).Error; err != nil {
			t.Fatal(err)
		}
		s := &AgentService{DB: db, Log: log}
		if !s.IsBotUser(100) {
			t.Error("user 100 has an agent profile, should return true")
		}
		if s.IsBotUser(101) {
			t.Error("user 101 has no agent profile, should return false")
		}
	})
}

// ---------- AgentService: IsAgentConversation (already tested in service_extra_test.go) ----------

func TestAgentService_IsAgentConversationIntegration(t *testing.T) {
	s := &AgentService{}
	tests := []struct {
		name string
		conv *model.DmConversation
		want bool
	}{
		{"nil conv", nil, false},
		{"empty kind", &model.DmConversation{}, false},
		{"wrong kind", &model.DmConversation{Kind: "human"}, false},
		{"agent kind", &model.DmConversation{Kind: model.DmKindAgent}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := s.IsAgentConversation(tc.conv)
			if got != tc.want {
				t.Errorf("IsAgentConversation(%+v) = %v, want %v", tc.conv, got, tc.want)
			}
		})
	}
}


