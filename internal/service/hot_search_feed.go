package service

import (
	"context"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"minibili/internal/model"
)

func hotSearchOpActive(now time.Time, start, end *time.Time) bool {
	if start != nil && now.Before(*start) {
		return false
	}
	if end != nil && now.After(*end) {
		return false
	}
	return true
}

func hotSearchDisplayTitle(op *model.HotSearchOp) string {
	if t := strings.TrimSpace(op.DisplayTitle); t != "" {
		return t
	}
	return strings.TrimSpace(op.Keyword)
}

// ListHotSearchMerged merges DB ops (pin/block/manual) with Redis auto rank.
func ListHotSearchMerged(ctx context.Context, db *gorm.DB, rec *SearchHotRecorder, limit int) ([]HotSearchItem, error) {
	details, err := ListHotSearchMergedDetail(ctx, db, rec, limit)
	if err != nil {
		return nil, err
	}
	out := make([]HotSearchItem, 0, len(details))
	for _, d := range details {
		out = append(out, HotSearchItem{
			Rank:  d.Rank,
			Title: d.Title,
			Badge: d.Badge,
		})
	}
	return out, nil
}

func listHotSearchMergedLegacy(ctx context.Context, db *gorm.DB, rec *SearchHotRecorder, limit int) ([]HotSearchItem, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 20 {
		limit = 20
	}
	now := time.Now()
	var ops []model.HotSearchOp
	if db != nil {
		_ = db.Where("enabled = ?", true).Order("pin_rank ASC, id ASC").Find(&ops).Error
	}
	active := make([]model.HotSearchOp, 0, len(ops))
	blocked := make(map[string]struct{})
	for i := range ops {
		op := ops[i]
		if !hotSearchOpActive(now, op.StartAt, op.EndAt) {
			continue
		}
		active = append(active, op)
		norm := NormalizeSearchKeyword(op.Keyword)
		if norm == "" {
			continue
		}
		if op.OpType == "block" {
			blocked[norm] = struct{}{}
		}
	}

	type slot struct {
		rank  int
		title string
		badge string
		norm  string
	}
	manualSlots := make([]slot, 0)
	for _, op := range active {
		if op.OpType != "pin" && op.OpType != "manual" {
			continue
		}
		title := hotSearchDisplayTitle(&op)
		if title == "" {
			continue
		}
		norm := NormalizeSearchKeyword(op.Keyword)
		if norm == "" {
			continue
		}
		r := op.PinRank
		if r <= 0 {
			r = 999
		}
		manualSlots = append(manualSlots, slot{
			rank:  r,
			title: title,
			badge: strings.TrimSpace(op.Badge),
			norm:  norm,
		})
	}
	sort.Slice(manualSlots, func(i, j int) bool {
		if manualSlots[i].rank != manualSlots[j].rank {
			return manualSlots[i].rank < manualSlots[j].rank
		}
		return manualSlots[i].title < manualSlots[j].title
	})

	out := make([]HotSearchItem, 0, limit)
	usedNorm := make(map[string]struct{})
	usedTitle := make(map[string]struct{})
	for _, s := range manualSlots {
		if len(out) >= limit {
			break
		}
		if _, ok := usedNorm[s.norm]; ok {
			continue
		}
		usedNorm[s.norm] = struct{}{}
		usedTitle[strings.ToLower(s.title)] = struct{}{}
		out = append(out, HotSearchItem{
			Rank:  len(out) + 1,
			Title: s.title,
			Badge: s.badge,
		})
	}

	auto, _ := rec.Top(ctx, limit*2)
	for _, it := range auto {
		if len(out) >= limit {
			break
		}
		norm := NormalizeSearchKeyword(it.Title)
		if norm != "" {
			if _, ok := blocked[norm]; ok {
				continue
			}
			if _, ok := usedNorm[norm]; ok {
				continue
			}
		}
		low := strings.ToLower(strings.TrimSpace(it.Title))
		if low != "" {
			if _, ok := usedTitle[low]; ok {
				continue
			}
			usedTitle[low] = struct{}{}
		}
		if norm != "" {
			usedNorm[norm] = struct{}{}
		}
		badge := it.Badge
		out = append(out, HotSearchItem{
			Rank:  len(out) + 1,
			Title: it.Title,
			Badge: badge,
		})
	}
	return out, nil
}
