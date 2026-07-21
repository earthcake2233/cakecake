package config

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"minibili/internal/model"
)

func setupRuntimeTest(t *testing.T) (*RuntimeConfig, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	_ = db.AutoMigrate(&model.SystemConfig{})
	defaults := map[string]string{
		"agent_enabled":     "true",
		"agent_daily_quota": "80",
		"rate_limit_rate":   "20.5",
	}
	rc := NewRuntimeConfig(db, defaults)
	rc.refresh(t.Context())
	return rc, db
}

func TestRuntimeConfig_Get_Fallback(t *testing.T) {
	rc, _ := setupRuntimeTest(t)
	// Key in defaults -> returns default value
	if got := rc.Get("agent_enabled", "false"); got != "true" {
		t.Errorf("expected true, got %s", got)
	}
	// Key missing from both cache and defaults -> returns fallback
	if got := rc.Get("nonexistent", "fallback"); got != "fallback" {
		t.Errorf("expected fallback, got %s", got)
	}
}

func TestRuntimeConfig_GetBool(t *testing.T) {
	rc, _ := setupRuntimeTest(t)
	if !rc.GetBool("agent_enabled", false) {
		t.Error("expected agent_enabled to be true")
	}
	if rc.GetBool("nonexistent", false) {
		t.Error("expected nonexistent key to fallback to false")
	}
}

func TestRuntimeConfig_GetInt(t *testing.T) {
	rc, _ := setupRuntimeTest(t)
	if got := rc.GetInt("agent_daily_quota", 10); got != 80 {
		t.Errorf("expected 80, got %d", got)
	}
	if got := rc.GetInt("nonexistent", 42); got != 42 {
		t.Errorf("expected 42, got %d", got)
	}
}

func TestRuntimeConfig_GetFloat(t *testing.T) {
	rc, _ := setupRuntimeTest(t)
	if got := rc.GetFloat("rate_limit_rate", 1.0); got != 20.5 {
		t.Errorf("expected 20.5, got %f", got)
	}
}

func TestRuntimeConfig_Set(t *testing.T) {
	rc, db := setupRuntimeTest(t)
	ctx := t.Context()
	if err := rc.Set(ctx, "agent_enabled", "false"); err != nil {
		t.Fatal(err)
	}
	if got := rc.GetBool("agent_enabled", true); got {
		t.Error("expected agent_enabled to become false after Set")
	}
	// Verify it persisted to DB
	var cfg model.SystemConfig
	if err := db.First(&cfg, "`key` = ?", "agent_enabled").Error; err != nil {
		t.Fatal(err)
	}
	if cfg.Value != "false" {
		t.Errorf("expected false in DB, got %s", cfg.Value)
	}
}

func TestRuntimeConfig_SeedDefaults(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	_ = db.AutoMigrate(&model.SystemConfig{})
	// Pre-insert one key
	db.Save(&model.SystemConfig{Key: "agent_enabled", Value: "false"})

	rc := NewRuntimeConfig(db, map[string]string{
		"agent_enabled":     "true",
		"agent_daily_quota": "80",
	})
	rc.refresh(t.Context())

	// Existing key should keep its value, not be overwritten by default
	if rc.Get("agent_enabled", "") != "false" {
		t.Error("existing key should not be overwritten by default")
	}
	// Missing key should be seeded
	if rc.Get("agent_daily_quota", "") != "80" {
		t.Error("new key should be seeded from defaults")
	}
}
