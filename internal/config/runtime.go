package config

import (
	"context"
	"strconv"
	"sync"
	"time"

	"gorm.io/gorm"

	"minibili/internal/model"
)

const refreshInterval = 30 * time.Second

// RuntimeConfig provides an in-memory cache of DB-stored system configuration,
// refreshed periodically. Fallback defaults (from env) are used when the DB
// has no value for a key.
type RuntimeConfig struct {
	db       *gorm.DB
	defaults map[string]string
	cache    map[string]string
	mu       sync.RWMutex
	stopCh   chan struct{}
	stopped  bool
}

// NewRuntimeConfig creates a RuntimeConfig with env-provided fallback defaults.
// Call Start() to begin periodic DB polling.
func NewRuntimeConfig(db *gorm.DB, defaults map[string]string) *RuntimeConfig {
	if defaults == nil {
		defaults = make(map[string]string)
	}
	return &RuntimeConfig{
		db:       db,
		defaults: defaults,
		cache:    make(map[string]string),
		stopCh:   make(chan struct{}),
	}
}

// Start loads initial values from DB and starts a background goroutine that
// refreshes every 30s. It also seeds any default key not yet in the DB.
func (rc *RuntimeConfig) Start(ctx context.Context) {
	rc.refresh(ctx)
	go rc.loop(ctx)
}

// Stop signals the background goroutine to exit.
func (rc *RuntimeConfig) Stop() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	if !rc.stopped {
		close(rc.stopCh)
		rc.stopped = true
	}
}

func (rc *RuntimeConfig) loop(ctx context.Context) {
	ticker := time.NewTicker(refreshInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			rc.refresh(ctx)
		case <-rc.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (rc *RuntimeConfig) refresh(ctx context.Context) {
	var list []model.SystemConfig
	if err := rc.db.Find(&list).Error; err != nil {
		return
	}
	rc.mu.Lock()
	rc.cache = make(map[string]string, len(list)+len(rc.defaults))
	for _, cfg := range list {
		rc.cache[cfg.Key] = cfg.Value
	}
	// Seed defaults for keys missing in DB
	for k, v := range rc.defaults {
		if _, ok := rc.cache[k]; !ok {
			rc.cache[k] = v
			if err := rc.db.Save(&model.SystemConfig{
				Key:   k,
				Value: v,
			}).Error; err != nil {
				// Non-fatal; next refresh will retry
			}
		}
	}
	rc.mu.Unlock()
}

// Get returns the cached value for key, falling back to defaults.
// If the key is not in DB or defaults, returns fallback.
func (rc *RuntimeConfig) Get(key, fallback string) string {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	if v, ok := rc.cache[key]; ok {
		return v
	}
	if v, ok := rc.defaults[key]; ok {
		return v
	}
	return fallback
}

// GetBool parses the cached value as bool. Falls back to env default then the
// provided fallback. Truthy values: 1, true, yes, on.
func (rc *RuntimeConfig) GetBool(key string, fallback bool) bool {
	rc.mu.RLock()
	raw, inCache := rc.cache[key]
	rc.mu.RUnlock()
	if !inCache {
		return fallback
	}
	switch raw {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// GetInt parses the cached value as int. Falls back to env default then fallback.
func (rc *RuntimeConfig) GetInt(key string, fallback int) int {
	rc.mu.RLock()
	raw, inCache := rc.cache[key]
	rc.mu.RUnlock()
	if !inCache {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return n
}

// GetFloat parses the cached value as float64. Falls back to env default then fallback.
func (rc *RuntimeConfig) GetFloat(key string, fallback float64) float64 {
	rc.mu.RLock()
	raw, inCache := rc.cache[key]
	rc.mu.RUnlock()
	if !inCache {
		return fallback
	}
	f, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fallback
	}
	return f
}

// Set writes a config key to DB and immediately updates the in-memory cache.
func (rc *RuntimeConfig) Set(ctx context.Context, key, value string) error {
	if err := rc.db.Save(&model.SystemConfig{
		Key:   key,
		Value: value,
	}).Error; err != nil {
		return err
	}
	rc.mu.Lock()
	rc.cache[key] = value
	rc.mu.Unlock()
	return nil
}
