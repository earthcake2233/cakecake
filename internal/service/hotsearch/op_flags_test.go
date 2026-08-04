package hotsearch

import (
	"cakecake/internal/model/admin"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

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
		_ = db.AutoMigrate(&admin.HotSearchOp{})
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
		_ = db.AutoMigrate(&admin.HotSearchOp{})
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
		_ = db.AutoMigrate(&admin.HotSearchOp{})
		db.Create(&admin.HotSearchOp{
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
		_ = db.AutoMigrate(&admin.HotSearchOp{})
		db.Create(&admin.HotSearchOp{
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
		_ = db.AutoMigrate(&admin.HotSearchOp{})
		db.Create(&admin.HotSearchOp{
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
		_ = db.AutoMigrate(&admin.HotSearchOp{})
		db.Create(&admin.HotSearchOp{
			OpType:  "block",
			Keyword: "word",
			Enabled: true,
		})
		db.Create(&admin.HotSearchOp{
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
		_ = db.AutoMigrate(&admin.HotSearchOp{})
		past := time.Now().Add(-2 * time.Hour)
		db.Select("OpType", "Keyword", "Enabled", "EndAt").Create(&admin.HotSearchOp{
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
		_ = db.AutoMigrate(&admin.HotSearchOp{})
		db.Create(&admin.HotSearchOp{
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
