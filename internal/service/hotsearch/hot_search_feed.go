package hotsearch

import (
	"cakecake/internal/model/admin"
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
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

func hotSearchDisplayTitle(op *admin.HotSearchOp) string {
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
