package service

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"minibili/internal/model"
)

// ---------- IsAgentConversation ----------

func TestIsAgentConversation(t *testing.T) {
	s := &AgentService{}
	tests := []struct {
		name string
		conv *model.DmConversation
		want bool
	}{
		{"nil conv", nil, false},
		{"empty kind", &model.DmConversation{}, false},
		{"wrong kind", &model.DmConversation{Kind: "normal"}, false},
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

// ---------- DedupKeyForTest ----------

func TestDedupKeyForTest(t *testing.T) {
	tests := []struct {
		name      string
		userID    uint64
		clientKey string
		norm      string
		want      string
	}{
		{"logged in user", 42, "1.2.3.4", "hello", "hotsearch:dedup:u:42:hello"},
		{"anonymous with ip", 0, "1.2.3.4", "world", "hotsearch:dedup:ip:1.2.3.4:world"},
		{"anonymous no ip", 0, "", "test", "hotsearch:dedup:anon:test"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := DedupKeyForTest(tc.userID, tc.clientKey, tc.norm)
			if got != tc.want {
				t.Errorf("DedupKeyForTest(%d, %q, %q) = %q, want %q",
					tc.userID, tc.clientKey, tc.norm, got, tc.want)
			}
		})
	}
}

// ---------- NormalizeSearchKeyword more edge cases ----------

func TestNormalizeSearchKeywordEdge(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"TAB\tHERE", "tabhere"},
		{"new\nline", "newline"},
		{"  mixed\tSPACES\nHERE  ", "mixedspaceshere"},
		{"a", "a"},
		{"Hello123", "hello123"},
		{"  ", ""},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := NormalizeSearchKeyword(tc.input)
			if got != tc.want {
				t.Errorf("NormalizeSearchKeyword(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// ---------- HotSearchOpActive more edge cases ----------

func TestHotSearchOpActiveEdge(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name  string
		start *time.Time
		end   *time.Time
		want  bool
	}{
		{"both set in middle", ptrTime(now.Add(-2 * time.Hour)), ptrTime(now.Add(2 * time.Hour)), true},
		{"start just passed end", ptrTime(now.Add(-3 * time.Hour)), ptrTime(now.Add(-2 * time.Hour)), false},
		{"same start and end equals now", ptrTime(now), ptrTime(now), true},
		{"start after end", ptrTime(now.Add(2 * time.Hour)), ptrTime(now.Add(-2 * time.Hour)), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := hotSearchOpActive(now, tc.start, tc.end)
			if got != tc.want {
				t.Errorf("hotSearchOpActive(%v, %v, %v) = %v, want %v",
					now, tc.start, tc.end, got, tc.want)
			}
		})
	}
}

// ---------- ActiveHotSearchOpFlags ----------

func TestActiveHotSearchOpFlags(t *testing.T) {
	t.Run("nil db returns empty", func(t *testing.T) {
		m := ActiveHotSearchOpFlags(nil)
		if m == nil {
			t.Error("expected non-nil map")
		}
		if len(m) != 0 {
			t.Errorf("expected empty map, got %d entries", len(m))
		}
	})

	t.Run("empty db returns empty", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		if err != nil {
			t.Fatal(err)
		}
		_ = db.AutoMigrate(&model.HotSearchOp{})
		m := ActiveHotSearchOpFlags(db)
		if len(m) != 0 {
			t.Errorf("expected empty map, got %d entries", len(m))
		}
	})

	t.Run("disabled op ignored", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		if err != nil {
			t.Fatal(err)
		}
		_ = db.AutoMigrate(&model.HotSearchOp{})
		// Use raw SQL to ensure Enabled=false is persisted
		db.Exec("INSERT INTO hot_search_ops (op_type, keyword, enabled) VALUES (?, ?, ?)",
			"block", "badword", false)
		m := ActiveHotSearchOpFlags(db)
		if _, ok := m["badword"]; ok {
			t.Error("disabled op should not appear")
		}
	})

	t.Run("block op", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		if err != nil {
			t.Fatal(err)
		}
		_ = db.AutoMigrate(&model.HotSearchOp{})
		db.Create(&model.HotSearchOp{
			OpType:  "block",
			Keyword: "spam",
			Enabled: true,
		})
		m := ActiveHotSearchOpFlags(db)
		f, ok := m["spam"]
		if !ok {
			t.Fatal("expected spam entry")
		}
		if !f.Blocked {
			t.Error("expected Blocked=true")
		}
		if f.Pin || f.Manual {
			t.Error("expected Pin/Manual=false")
		}
		if f.OpType != "block" {
			t.Errorf("OpType want block, got %q", f.OpType)
		}
	})

	t.Run("pin op", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		if err != nil {
			t.Fatal(err)
		}
		_ = db.AutoMigrate(&model.HotSearchOp{})
		db.Create(&model.HotSearchOp{
			OpType:  "pin",
			Keyword: "important",
			Enabled: true,
		})
		m := ActiveHotSearchOpFlags(db)
		f := m["important"]
		if !f.Pin {
			t.Error("expected Pin=true")
		}
		if f.Blocked || f.Manual {
			t.Error("expected Blocked/Manual=false")
		}
	})

	t.Run("manual op", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		if err != nil {
			t.Fatal(err)
		}
		_ = db.AutoMigrate(&model.HotSearchOp{})
		db.Create(&model.HotSearchOp{
			OpType:  "manual",
			Keyword: "curated",
			Enabled: true,
		})
		m := ActiveHotSearchOpFlags(db)
		f := m["curated"]
		if !f.Manual {
			t.Error("expected Manual=true")
		}
	})

	t.Run("multiple ops for same keyword", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		if err != nil {
			t.Fatal(err)
		}
		_ = db.AutoMigrate(&model.HotSearchOp{})
		db.Create(&model.HotSearchOp{
			OpType:  "block",
			Keyword: "word",
			Enabled: true,
		})
		db.Create(&model.HotSearchOp{
			OpType:  "pin",
			Keyword: "word",
			Enabled: true,
		})
		m := ActiveHotSearchOpFlags(db)
		f := m["word"]
		if !f.Blocked || !f.Pin {
			t.Error("expected both Blocked and Pin for keyword 'word'")
		}
	})

	t.Run("expired op skipped", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		if err != nil {
			t.Fatal(err)
		}
		_ = db.AutoMigrate(&model.HotSearchOp{})
		past := time.Now().Add(-2 * time.Hour)
		db.Select("OpType", "Keyword", "Enabled", "EndAt").Create(&model.HotSearchOp{
			OpType:  "block",
			Keyword: "old",
			Enabled: true,
			EndAt:   &past,
		})
		m := ActiveHotSearchOpFlags(db)
		if _, ok := m["old"]; ok {
			t.Error("expired op should not appear")
		}
	})

	t.Run("normalize keyword for lookup", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		if err != nil {
			t.Fatal(err)
		}
		_ = db.AutoMigrate(&model.HotSearchOp{})
		db.Create(&model.HotSearchOp{
			OpType:  "block",
			Keyword: "  Hello World  ",
			Enabled: true,
		})
		m := ActiveHotSearchOpFlags(db)
		f, ok := m["helloworld"]
		if !ok {
			t.Fatal("expected normalized key 'helloworld'")
		}
		if !f.Blocked {
			t.Error("expected Blocked=true")
		}
	})
}



