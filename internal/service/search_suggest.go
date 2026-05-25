package service

import (
	"context"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"minibili/internal/model"
)

// SearchSuggestTag is one row for search box autocomplete (B 站 suggest.tag).
type SearchSuggestTag struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// SearchSuggest builds keyword suggestions from hot-search Redis, ops rules, and optional user history.
func SearchSuggest(ctx context.Context, db *gorm.DB, rec *SearchHotRecorder, userID uint64, term string, limit int) []SearchSuggestTag {
	if limit <= 0 {
		limit = 10
	}
	if limit > 20 {
		limit = 20
	}
	term = strings.TrimSpace(term)
	termNorm := NormalizeSearchKeyword(term)

	type cand struct {
		display string
		score   int
	}
	seen := make(map[string]struct{})
	var cands []cand
	add := func(display string, score int) {
		d := strings.TrimSpace(display)
		if d == "" {
			return
		}
		norm := NormalizeSearchKeyword(d)
		if norm == "" {
			return
		}
		if _, ok := seen[norm]; ok {
			return
		}
		if termNorm != "" && !keywordMatchesSuggest(termNorm, norm, d) {
			return
		}
		seen[norm] = struct{}{}
		cands = append(cands, cand{display: d, score: score})
	}

	if userID > 0 && db != nil {
		var rows []model.UserSearchHistory
		_ = db.Where("user_id = ?", userID).
			Order("updated_at DESC, id DESC").
			Limit(40).
			Find(&rows).Error
		for i, r := range rows {
			add(r.Keyword, 1000-i)
		}
	}

	if db != nil {
		now := time.Now()
		var ops []model.HotSearchOp
		_ = db.Where("enabled = ?", true).Find(&ops).Error
		for i := range ops {
			op := ops[i]
			if !hotSearchOpActive(now, op.StartAt, op.EndAt) {
				continue
			}
			if op.OpType == "block" {
				continue
			}
			add(hotSearchDisplayTitle(&op), 500-i)
		}
	}

	if rec != nil && rec.Rdb != nil {
		zs, err := rec.Rdb.ZRevRangeWithScores(ctx, keyHotSearchRank, 0, 299).Result()
		if err == nil && len(zs) > 0 {
			norms := make([]string, 0, len(zs))
			for _, z := range zs {
				norms = append(norms, z.Member.(string))
			}
			labels, _ := rec.Rdb.HMGet(ctx, keyHotSearchLabel, norms...).Result()
			for i, z := range zs {
				norm, _ := z.Member.(string)
				title := norm
				if i < len(labels) && labels[i] != nil {
					if s, ok := labels[i].(string); ok && strings.TrimSpace(s) != "" {
						title = strings.TrimSpace(s)
					}
				}
				score := int(z.Score)
				if termNorm != "" && strings.HasPrefix(NormalizeSearchKeyword(title), termNorm) {
					score += 10000
				}
				add(title, score)
			}
		}
	}

	// 无匹配（或库内暂无词条）时回退展示合并热搜榜前 N 条。
	if len(cands) == 0 {
		items, _ := ListHotSearchMerged(ctx, db, rec, limit)
		for i, it := range items {
			title := strings.TrimSpace(it.Title)
			if title == "" {
				continue
			}
			norm := NormalizeSearchKeyword(title)
			if norm == "" {
				continue
			}
			if _, ok := seen[norm]; ok {
				continue
			}
			seen[norm] = struct{}{}
			cands = append(cands, cand{display: title, score: 100 - i})
		}
	}

	sort.Slice(cands, func(i, j int) bool {
		if cands[i].score != cands[j].score {
			return cands[i].score > cands[j].score
		}
		return cands[i].display < cands[j].display
	})

	if len(cands) > limit {
		cands = cands[:limit]
	}
	out := make([]SearchSuggestTag, 0, len(cands))
	for _, c := range cands {
		out = append(out, SearchSuggestTag{
			Name:  HighlightSuggestKeyword(c.display, term),
			Value: c.display,
		})
	}
	return out
}

func keywordMatchesSuggest(termNorm, kwNorm, display string) bool {
	if termNorm == "" {
		return true
	}
	if strings.HasPrefix(kwNorm, termNorm) {
		return true
	}
	if strings.Contains(kwNorm, termNorm) {
		return true
	}
	dn := NormalizeSearchKeyword(display)
	if strings.HasPrefix(dn, termNorm) || strings.Contains(dn, termNorm) {
		return true
	}
	return false
}

// HighlightSuggestKeyword wraps matched substring for suggest UI.
func HighlightSuggestKeyword(display, term string) string {
	d := strings.TrimSpace(display)
	t := strings.TrimSpace(term)
	if d == "" {
		return ""
	}
	if t == "" {
		return escapeHTML(d)
	}
	lowerD := strings.ToLower(d)
	lowerT := strings.ToLower(t)
	idx := strings.Index(lowerD, lowerT)
	if idx < 0 {
		return escapeHTML(d)
	}
	end := idx + len(t)
	if end > len(d) {
		end = len(d)
	}
	var b strings.Builder
	b.WriteString(escapeHTML(d[:idx]))
	b.WriteString(`<em class="suggest_high_light">`)
	b.WriteString(escapeHTML(d[idx:end]))
	b.WriteString(`</em>`)
	b.WriteString(escapeHTML(d[end:]))
	return b.String()
}

func escapeHTML(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return r.Replace(s)
}

// ValidateSuggestTerm returns false if term is too long for suggest API.
func ValidateSuggestTerm(term string) bool {
	return utf8.RuneCountInString(strings.TrimSpace(term)) <= 50
}
