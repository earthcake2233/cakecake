package service

import (
	"context"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"minibili/internal/model"
)

// ListHotSearchMergedDetail merges ops + Redis and annotates each row's source.
func ListHotSearchMergedDetail(ctx context.Context, db *gorm.DB, rec *SearchHotRecorder, limit int) ([]HotSearchMergedDetail, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 20 {
		limit = 20
	}
	if out, ok := mergeHotSearchFromLayout(ctx, db, rec, limit); ok {
		return out, nil
	}
	return listHotSearchMergedDetailLegacy(ctx, db, rec, limit)
}

func listHotSearchMergedDetailLegacy(ctx context.Context, db *gorm.DB, rec *SearchHotRecorder, limit int) ([]HotSearchMergedDetail, error) {
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
		rank   int
		title  string
		badge  string
		norm   string
		source string
		opID   uint64
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
		src := op.OpType
		if src == "" {
			src = "manual"
		}
		manualSlots = append(manualSlots, slot{
			rank:   r,
			title:  title,
			badge:  strings.TrimSpace(op.Badge),
			norm:   norm,
			source: src,
			opID:   op.ID,
		})
	}
	sort.Slice(manualSlots, func(i, j int) bool {
		if manualSlots[i].rank != manualSlots[j].rank {
			return manualSlots[i].rank < manualSlots[j].rank
		}
		return manualSlots[i].title < manualSlots[j].title
	})

	out := make([]HotSearchMergedDetail, 0, limit)
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
		out = append(out, HotSearchMergedDetail{
			Rank:    len(out) + 1,
			Title:   s.title,
			Badge:   s.badge,
			Source:  s.source,
			Keyword: s.norm,
			OpID:    s.opID,
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
		out = append(out, HotSearchMergedDetail{
			Rank:    len(out) + 1,
			Title:   it.Title,
			Badge:   it.Badge,
			Source:  "auto",
			Keyword: norm,
		})
	}
	return out, nil
}

// HotSearchOpFlags describes intervention state for a normalized keyword.
type HotSearchOpFlags struct {
	Blocked bool
	Pin     bool
	Manual  bool
	OpID    uint64
	OpType  string
}

// ActiveHotSearchOpFlags maps normalized keyword -> intervention flags.
func ActiveHotSearchOpFlags(db *gorm.DB) map[string]HotSearchOpFlags {
	out := make(map[string]HotSearchOpFlags)
	if db == nil {
		return out
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
		f := out[norm]
		f.OpID = op.ID
		f.OpType = op.OpType
		switch op.OpType {
		case "block":
			f.Blocked = true
		case "pin":
			f.Pin = true
		case "manual":
			f.Manual = true
		}
		out[norm] = f
	}
	return out
}
