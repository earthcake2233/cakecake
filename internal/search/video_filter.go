package search

import "strings"

// VideoFilter from search UI (综合/视频 Tab).
type VideoFilter struct {
	Order      string // default | click | pubdate | dm | fav
	Duration   string // all | lt10 | m10_30 | m30_60 | gt60
	Zone       string // UI label e.g. 动画、全部分区
}

func ParseVideoFilter(order, duration, zone string) VideoFilter {
	return VideoFilter{
		Order:    strings.TrimSpace(order),
		Duration: strings.TrimSpace(duration),
		Zone:     strings.TrimSpace(zone),
	}
}

func (f VideoFilter) durationRange() (minSec, maxSec float64, ok bool) {
	switch f.Duration {
	case "", "all", "全部时长":
		return 0, 0, false
	case "lt10", "10分钟以下":
		return 0, 600, true
	case "m10_30", "10-30分钟":
		return 600, 1800, true
	case "m30_60", "30-60分钟":
		return 1800, 3600, true
	case "gt60", "60分钟以上":
		return 3600, 0, true
	default:
		return 0, 0, false
	}
}

// zoneParentChild maps bilibili-vue search filter label to ES zone_parent / full zone.
func zoneParentChild(label string) (parent, fullZone string) {
	label = strings.TrimSpace(label)
	if label == "" || label == "全部分区" {
		return "", ""
	}
	switch label {
	case "番剧相关":
		return "番剧", ""
	case "纪录片":
		return "放映厅", "放映厅-纪录片"
	case "电影":
		return "放映厅", "放映厅-电影"
	case "电视剧":
		return "放映厅", "放映厅-电视剧"
	default:
		return label, ""
	}
}

func videoSortClause(order string) []any {
	switch strings.TrimSpace(order) {
	case "click", "最多点击":
		return []any{
			map[string]any{"play_count": map[string]any{"order": "desc"}},
			map[string]any{"_score": map[string]any{"order": "desc"}},
		}
	case "pubdate", "最新发布":
		return []any{
			map[string]any{"created_at": map[string]any{"order": "desc"}},
			map[string]any{"_score": map[string]any{"order": "desc"}},
		}
	case "dm", "最多弹幕":
		return []any{
			map[string]any{"danmaku_count": map[string]any{"order": "desc"}},
			map[string]any{"_score": map[string]any{"order": "desc"}},
		}
	case "fav", "最多收藏":
		return []any{
			map[string]any{"fav_count": map[string]any{"order": "desc"}},
			map[string]any{"_score": map[string]any{"order": "desc"}},
		}
	default:
		return []any{
			map[string]any{"_score": map[string]any{"order": "desc"}},
			map[string]any{"play_count": map[string]any{"order": "desc"}},
		}
	}
}

// ES may store zone as keyword (our mapping) or text with a .keyword subfield (dynamic mapping).
func zoneKeywordTerm(field, value string) map[string]any {
	should := []any{
		map[string]any{"term": map[string]any{field: value}},
	}
	if !strings.HasSuffix(field, ".keyword") {
		should = append(should, map[string]any{
			"term": map[string]any{field + ".keyword": value},
		})
	}
	return map[string]any{
		"bool": map[string]any{
			"should":               should,
			"minimum_should_match": 1,
		},
	}
}

func zoneParentMatchFilter(parent string) map[string]any {
	should := []any{
		zoneKeywordTerm("zone_parent", parent),
		zoneKeywordTerm("zone", parent),
		map[string]any{
			"bool": map[string]any{
				"should": []any{
					map[string]any{"prefix": map[string]any{"zone.keyword": parent + "-"}},
					map[string]any{"prefix": map[string]any{"zone": parent + "-"}},
				},
				"minimum_should_match": 1,
			},
		},
	}
	return map[string]any{
		"bool": map[string]any{
			"should":               should,
			"minimum_should_match": 1,
		},
	}
}

func appendVideoFilters(filters []any, vf VideoFilter) []any {
	if min, max, ok := vf.durationRange(); ok {
		rng := map[string]any{}
		if min > 0 {
			rng["gte"] = min
		}
		if max > 0 {
			rng["lt"] = max
		}
		filters = append(filters, map[string]any{
			"range": map[string]any{"duration_sec": rng},
		})
	}
	parent, full := zoneParentChild(vf.Zone)
	if full != "" {
		filters = append(filters, zoneKeywordTerm("zone", full))
	} else if parent != "" {
		filters = append(filters, zoneParentMatchFilter(parent))
	}
	return filters
}
