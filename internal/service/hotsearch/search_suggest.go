package hotsearch

import (
	"cakecake/internal/model/admin"
	"cakecake/internal/model/extra"
	"context"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"
)

// SearchSuggestTag is one row for search box autocomplete (Bilibili-style suggest.tag).
type SearchSuggestTag struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// SearchSuggest builds keyword suggestions from hot-search Redis, ops rules, and optional user history.
func SearchSuggest(ctx context.Context, db *gorm.DB, rec *SearchHotRecorder, userID uint64, term string, limit int) []SearchSuggestTag {
	limit = clampSuggestLimit(limit)
	term = strings.TrimSpace(term)
	coll := newSuggestCollector(normalizeSearchKeyword(term))
	collectUserHistorySuggestions(ctx, db, userID, coll)
	collectOpsRuleSuggestions(ctx, db, time.Now(), coll)
	collectHotRankSuggestions(ctx, rec, coll)
	if len(coll.cands) == 0 {
		fallbackSuggestions(ctx, db, rec, limit, coll)
	}
	return finalizeSuggestions(coll, term, limit)
}

func clampSuggestLimit(limit int) int {
	if limit <= 0 {
		return 10
	}
	if limit > 20 {
		return 20
	}
	return limit
}

type suggestCandidate struct {
	display string
	score   int
}

type suggestCollector struct {
	termNorm string
	seen     map[string]struct{}
	cands    []suggestCandidate
}

func newSuggestCollector(termNorm string) *suggestCollector {
	return &suggestCollector{termNorm: termNorm, seen: make(map[string]struct{})}
}

// add appends a candidate when it matches the term and has not been seen.
func (c *suggestCollector) add(display string, score int) {
	d := strings.TrimSpace(display)
	if d == "" {
		return
	}
	norm := normalizeSearchKeyword(d)
	if norm == "" {
		return
	}
	if c.seenNorm(norm) {
		return
	}
	if c.termNorm != "" && !keywordMatchesSuggest(c.termNorm, norm, d) {
		return
	}
	c.markNorm(norm)
	c.cands = append(c.cands, suggestCandidate{display: d, score: score})
}

// addUnchecked appends a candidate without term matching (fallback path).
func (c *suggestCollector) addUnchecked(display string, score int) {
	c.cands = append(c.cands, suggestCandidate{display: display, score: score})
}

func (c *suggestCollector) seenNorm(norm string) bool {
	_, ok := c.seen[norm]
	return ok
}

func (c *suggestCollector) markNorm(norm string) {
	c.seen[norm] = struct{}{}
}

func collectUserHistorySuggestions(ctx context.Context, db *gorm.DB, userID uint64, coll *suggestCollector) {
	if userID == 0 || db == nil {
		return
	}
	var rows []extra.UserSearchHistory
	_ = db.Where("user_id = ?", userID).
		Order("updated_at DESC, id DESC").
		Limit(40).
		Find(&rows).Error
	for i, r := range rows {
		coll.add(r.Keyword, 1000-i)
	}
}

func collectOpsRuleSuggestions(ctx context.Context, db *gorm.DB, now time.Time, coll *suggestCollector) {
	if db == nil {
		return
	}
	var ops []admin.HotSearchOp
	_ = db.Where("enabled = ?", true).Find(&ops).Error
	for i := range ops {
		op := ops[i]
		if !hotSearchOpActive(now, op.StartAt, op.EndAt) {
			continue
		}
		if op.OpType == "block" {
			continue
		}
		coll.add(hotSearchDisplayTitle(&op), 500-i)
	}
}

func collectHotRankSuggestions(ctx context.Context, rec *SearchHotRecorder, coll *suggestCollector) {
	if rec == nil || rec.Rdb == nil {
		return
	}
	zs, err := rec.Rdb.ZRevRangeWithScores(ctx, keyHotSearchRank, 0, 299).Result()
	if err != nil || len(zs) == 0 {
		return
	}
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
		if coll.termNorm != "" && strings.HasPrefix(normalizeSearchKeyword(title), coll.termNorm) {
			score += 10000
		}
		coll.add(title, score)
	}
}

// fallbackSuggestions shows the top merged hot-search items when nothing matched.
func fallbackSuggestions(ctx context.Context, db *gorm.DB, rec *SearchHotRecorder, limit int, coll *suggestCollector) {
	items, _ := ListHotSearchMerged(ctx, db, rec, limit)
	for i, it := range items {
		title := strings.TrimSpace(it.Title)
		if title == "" {
			continue
		}
		norm := normalizeSearchKeyword(title)
		if norm == "" {
			continue
		}
		if coll.seenNorm(norm) {
			continue
		}
		coll.markNorm(norm)
		coll.addUnchecked(title, 100-i)
	}
}

func finalizeSuggestions(coll *suggestCollector, term string, limit int) []SearchSuggestTag {
	sort.Slice(coll.cands, func(i, j int) bool {
		if coll.cands[i].score != coll.cands[j].score {
			return coll.cands[i].score > coll.cands[j].score
		}
		return coll.cands[i].display < coll.cands[j].display
	})
	if len(coll.cands) > limit {
		coll.cands = coll.cands[:limit]
	}
	out := make([]SearchSuggestTag, 0, len(coll.cands))
	for _, c := range coll.cands {
		out = append(out, SearchSuggestTag{
			Name:  highlightSuggestKeyword(c.display, term),
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
	dn := normalizeSearchKeyword(display)
	if strings.HasPrefix(dn, termNorm) || strings.Contains(dn, termNorm) {
		return true
	}
	return false
}

// highlightSuggestKeyword wraps matched substring for suggest UI.
func highlightSuggestKeyword(display, term string) string {
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
