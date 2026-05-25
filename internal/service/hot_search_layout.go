package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"gorm.io/gorm"

	"minibili/internal/model"
)

const hotSearchLayoutSingletonID uint64 = 1

// HotSearchLayoutEntry is one slot in admin drag order.
type HotSearchLayoutEntry struct {
	Keyword string `json:"keyword"`
	Title   string `json:"title"`
}

type hotSearchMergePools struct {
	blocked  map[string]struct{}
	opByNorm map[string]model.HotSearchOp
	autoBy   map[string]HotSearchItem
}

func loadHotSearchLayout(db *gorm.DB) []HotSearchLayoutEntry {
	if db == nil {
		return nil
	}
	var row model.HotSearchDisplayLayout
	if err := db.First(&row, hotSearchLayoutSingletonID).Error; err != nil {
		return nil
	}
	raw := strings.TrimSpace(row.OrderJSON)
	if raw == "" || raw == "[]" {
		return nil
	}
	var entries []HotSearchLayoutEntry
	if err := json.Unmarshal([]byte(raw), &entries); err != nil {
		return nil
	}
	out := make([]HotSearchLayoutEntry, 0, len(entries))
	for _, e := range entries {
		kw := strings.TrimSpace(e.Keyword)
		if kw == "" {
			kw = strings.TrimSpace(e.Title)
		}
		if kw == "" {
			continue
		}
		title := strings.TrimSpace(e.Title)
		if title == "" {
			title = kw
		}
		out = append(out, HotSearchLayoutEntry{Keyword: kw, Title: title})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// SaveHotSearchDisplayLayout persists drag order without changing op types.
func SaveHotSearchDisplayLayout(db *gorm.DB, entries []HotSearchLayoutEntry) error {
	if db == nil {
		return nil
	}
	b, err := json.Marshal(entries)
	if err != nil {
		return err
	}
	row := model.HotSearchDisplayLayout{
		ID:        hotSearchLayoutSingletonID,
		OrderJSON: string(b),
		UpdatedAt: time.Now(),
	}
	return db.Save(&row).Error
}

// ClearHotSearchDisplayLayout removes custom drag order.
func ClearHotSearchDisplayLayout(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	return db.Delete(&model.HotSearchDisplayLayout{}, hotSearchLayoutSingletonID).Error
}

// HasHotSearchDisplayLayout reports whether a custom drag order exists.
func HasHotSearchDisplayLayout(db *gorm.DB) bool {
	return len(loadHotSearchLayout(db)) > 0
}

func buildHotSearchMergePools(ctx context.Context, db *gorm.DB, rec *SearchHotRecorder, limit int) hotSearchMergePools {
	pools := hotSearchMergePools{
		blocked:  make(map[string]struct{}),
		opByNorm: make(map[string]model.HotSearchOp),
		autoBy:   make(map[string]HotSearchItem),
	}
	if db == nil {
		return pools
	}
	now := time.Now()
	var ops []model.HotSearchOp
	_ = db.Where("enabled = ?", true).Find(&ops).Error
	for i := range ops {
		op := ops[i]
		if !hotSearchOpActive(now, op.StartAt, op.EndAt) {
			continue
		}
		norm := NormalizeSearchKeyword(op.Keyword)
		if norm == "" {
			continue
		}
		if op.OpType == "block" {
			pools.blocked[norm] = struct{}{}
			continue
		}
		if op.OpType == "pin" || op.OpType == "manual" {
			pools.opByNorm[norm] = op
		}
	}
	if rec != nil {
		autoLimit := limit * 3
		if autoLimit < 30 {
			autoLimit = 30
		}
		auto, _ := rec.Top(ctx, autoLimit)
		for _, it := range auto {
			norm := NormalizeSearchKeyword(it.Title)
			if norm == "" {
				continue
			}
			pools.autoBy[norm] = it
		}
	}
	return pools
}

func resolveHotSearchEntry(norm, fallbackTitle string, pools hotSearchMergePools) (HotSearchMergedDetail, bool) {
	if norm == "" {
		return HotSearchMergedDetail{}, false
	}
	if _, blocked := pools.blocked[norm]; blocked {
		return HotSearchMergedDetail{}, false
	}
	if op, ok := pools.opByNorm[norm]; ok {
		title := hotSearchDisplayTitle(&op)
		if title == "" {
			title = fallbackTitle
		}
		src := op.OpType
		if src == "" {
			src = "manual"
		}
		return HotSearchMergedDetail{
			Title:   title,
			Badge:   strings.TrimSpace(op.Badge),
			Source:  src,
			Keyword: norm,
			OpID:    op.ID,
		}, true
	}
	if auto, ok := pools.autoBy[norm]; ok {
		return HotSearchMergedDetail{
			Title:   auto.Title,
			Badge:   auto.Badge,
			Source:  "auto",
			Keyword: norm,
		}, true
	}
	title := strings.TrimSpace(fallbackTitle)
	if title == "" {
		return HotSearchMergedDetail{}, false
	}
	return HotSearchMergedDetail{
		Title:   title,
		Badge:   "",
		Source:  "auto",
		Keyword: norm,
	}, true
}

func mergeHotSearchFromLayout(ctx context.Context, db *gorm.DB, rec *SearchHotRecorder, limit int) ([]HotSearchMergedDetail, bool) {
	entries := loadHotSearchLayout(db)
	if len(entries) == 0 {
		return nil, false
	}
	pools := buildHotSearchMergePools(ctx, db, rec, limit)
	out := make([]HotSearchMergedDetail, 0, limit)
	usedNorm := make(map[string]struct{})
	for _, e := range entries {
		if len(out) >= limit {
			break
		}
		norm := NormalizeSearchKeyword(e.Keyword)
		if norm == "" {
			norm = NormalizeSearchKeyword(e.Title)
		}
		if norm == "" {
			continue
		}
		if _, ok := usedNorm[norm]; ok {
			continue
		}
		item, ok := resolveHotSearchEntry(norm, e.Title, pools)
		if !ok {
			continue
		}
		usedNorm[norm] = struct{}{}
		item.Rank = len(out) + 1
		out = append(out, item)
	}
	if len(out) == 0 {
		return nil, false
	}
	return out, true
}

// ApplyHotSearchLayoutMove reorders one keyword in saved drag layout (no-op if layout empty).
func ApplyHotSearchLayoutMove(db *gorm.DB, keyword, title string, targetRank int) error {
	if db == nil {
		return nil
	}
	entries := loadHotSearchLayout(db)
	if len(entries) == 0 {
		return nil
	}
	norm := NormalizeSearchKeyword(keyword)
	if norm == "" {
		return nil
	}
	if targetRank <= 0 {
		targetRank = 1
	}
	title = strings.TrimSpace(title)
	if title == "" {
		title = strings.TrimSpace(keyword)
	}
	rest := make([]HotSearchLayoutEntry, 0, len(entries))
	for _, e := range entries {
		n := NormalizeSearchKeyword(e.Keyword)
		if n == "" {
			n = NormalizeSearchKeyword(e.Title)
		}
		if n == norm {
			continue
		}
		rest = append(rest, e)
	}
	entry := HotSearchLayoutEntry{Keyword: strings.TrimSpace(keyword), Title: title}
	if entry.Keyword == "" {
		entry.Keyword = title
	}
	idx := targetRank - 1
	if idx < 0 {
		idx = 0
	}
	if idx > len(rest) {
		idx = len(rest)
	}
	out := make([]HotSearchLayoutEntry, 0, len(rest)+1)
	out = append(out, rest[:idx]...)
	out = append(out, entry)
	out = append(out, rest[idx:]...)
	return SaveHotSearchDisplayLayout(db, out)
}

// RemoveHotSearchLayoutEntry drops one keyword from saved drag layout.
func RemoveHotSearchLayoutEntry(db *gorm.DB, keyword string) error {
	if db == nil {
		return nil
	}
	entries := loadHotSearchLayout(db)
	if len(entries) == 0 {
		return nil
	}
	norm := NormalizeSearchKeyword(keyword)
	if norm == "" {
		return nil
	}
	out := make([]HotSearchLayoutEntry, 0, len(entries))
	for _, e := range entries {
		n := NormalizeSearchKeyword(e.Keyword)
		if n == "" {
			n = NormalizeSearchKeyword(e.Title)
		}
		if n == norm {
			continue
		}
		out = append(out, e)
	}
	if len(out) == 0 {
		return ClearHotSearchDisplayLayout(db)
	}
	return SaveHotSearchDisplayLayout(db, out)
}

// EnsureHotSearchLayoutFromMerged seeds layout when absent (used before first pin under custom flow).
func EnsureHotSearchLayoutFromMerged(ctx context.Context, db *gorm.DB, rec *SearchHotRecorder, limit int) error {
	if db == nil || HasHotSearchDisplayLayout(db) {
		return nil
	}
	items, err := listHotSearchMergedDetailLegacy(ctx, db, rec, limit)
	if err != nil {
		return err
	}
	entries := make([]HotSearchLayoutEntry, 0, len(items))
	for _, it := range items {
		kw := strings.TrimSpace(it.Keyword)
		if kw == "" {
			kw = strings.TrimSpace(it.Title)
		}
		if kw == "" {
			continue
		}
		entries = append(entries, HotSearchLayoutEntry{Keyword: kw, Title: it.Title})
	}
	if len(entries) == 0 {
		return nil
	}
	return SaveHotSearchDisplayLayout(db, entries)
}
