package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// AllResult matches the bilibili-vue search store shape (F10 search page).
type AllResult struct {
	Result       SearchResultBuckets `json:"result"`
	TopTlist     TopTlist            `json:"top_tlist"`
	NumResults   int64               `json:"numResults,omitempty"`
	Page         int                 `json:"page,omitempty"`
	PageSize     int                 `json:"page_size,omitempty"`
	SearchStatus string              `json:"search_status,omitempty"` // ok | empty | unavailable
}

type SearchResultBuckets struct {
	Video         []VideoHit   `json:"video"`
	MediaBangumi  []any        `json:"media_bangumi"`
	MediaFt       []any        `json:"media_ft"`
	Live          []any        `json:"live"`
	Article       []ArticleHit `json:"article"`
	Topic         []any        `json:"topic"`
	BiliUser      []UserHit    `json:"bili_user"`
	Photo         []any        `json:"photo"`
}

type TopTlist struct {
	Video         int `json:"video"`
	MediaBangumi  int `json:"media_bangumi"`
	Movie         int `json:"movie"`
	Live          int `json:"live"`
	Article       int `json:"article"`
	Topic         int `json:"topic"`
	BiliUser      int `json:"bili_user"`
	Photo         int `json:"photo"`
}

// VideoHit is one video card on /search/all.
type VideoHit struct {
	Aid          uint64 `json:"aid"`
	Title        string `json:"title"`
	Pic          string `json:"pic"`
	Duration     string `json:"duration"`
	Play         uint64 `json:"play"`
	VideoReview  uint64 `json:"video_review"`
	Pubdate      int64  `json:"pubdate"`
	Author       string `json:"author"`
	Mid          uint64 `json:"mid"`
	Description  string `json:"description,omitempty"`
	TypeName     string `json:"type_name,omitempty"`
	InWatchLater bool   `json:"in_watch_later,omitempty"`
}

// ArticleHit matches bilibili-vue /search/article list cards.
type ArticleHit struct {
	ID           uint64 `json:"id"`
	Title        string `json:"title"`
	Desc         string `json:"desc"`
	CoverURL     string `json:"cover_url"`
	View         uint64 `json:"view"`
	Like         uint64 `json:"like"`
	Reply        uint64 `json:"reply"`
	Pubdate      int64  `json:"pubdate"`
	Mid          uint64 `json:"mid"`
	Author       string `json:"author"`
	Face         string `json:"face"`
	CategoryName string `json:"category_name"`
}

// ArticleSearchPayload is returned for type=article searches.
type ArticleSearchPayload struct {
	Result     []ArticleHit `json:"result"`
	NumResults int64        `json:"numResults"`
	Page       int          `json:"page"`
	PageSize   int          `json:"page_size"`
}

// UserArchiveItem is one recent submission on the user search card.
type UserArchiveItem struct {
	Aid     uint64 `json:"aid"`
	Title   string `json:"title"`
	Pic     string `json:"pic"`
	Pubdate int64  `json:"pubdate"`
	Rtype   string `json:"rtype"` // video | article
}

// UserHit matches bilibili-vue user search cards (/search/all, /search/upuser).
type UserHit struct {
	Mid          uint64            `json:"mid"`
	Uname        string            `json:"uname"`
	Usign        string            `json:"usign"`
	Face         string            `json:"face"`
	Archives     int               `json:"archives"`
	Fans         int               `json:"fans"`
	Level        int               `json:"level"`
	FollowedByMe bool              `json:"followed_by_me"`
	Recent       []UserArchiveItem `json:"recent"`
}

// SearchParams controls GET /api/v1/search.
type SearchParams struct {
	Keyword   string
	Highlight bool
	Page      int
	PageSize  int
	Sort      string // article tab: default | pubdate | click | like | reply
	Type      string // all | article | video | user
	Video     VideoFilter
}

// SearchArticles returns the article-tab payload (full list + total).
func (c *Client) SearchArticles(ctx context.Context, p SearchParams) (*ArticleSearchPayload, error) {
	if !c.Enabled() {
		return nil, fmt.Errorf("elasticsearch disabled")
	}
	keyword := strings.TrimSpace(p.Keyword)
	if keyword == "" {
		return &ArticleSearchPayload{Result: []ArticleHit{}, Page: 1, PageSize: 20}, nil
	}
	page := p.Page
	if page < 1 {
		page = 1
	}
	size := p.PageSize
	if size < 1 {
		size = 20
	}
	if size > 50 {
		size = 50
	}
	from := (page - 1) * size
	articles, total, err := c.searchArticles(ctx, keyword, p.Highlight, p.Sort, from, size)
	if err != nil {
		return nil, err
	}
	return &ArticleSearchPayload{
		Result:     articles,
		NumResults: total,
		Page:       page,
		PageSize:   size,
	}, nil
}

// SearchAll queries videos, articles, and users for the composite search page.
func (c *Client) SearchAll(ctx context.Context, p SearchParams) (*AllResult, error) {
	if !c.Enabled() {
		return nil, fmt.Errorf("elasticsearch disabled")
	}
	if t := strings.ToLower(strings.TrimSpace(p.Type)); t == "user" || t == "upuser" {
		page := p.Page
		if page < 1 {
			page = 1
		}
		size := p.PageSize
		if size < 1 {
			size = 20
		}
		if size > 50 {
			size = 50
		}
		from := (page - 1) * size
		users, userTotal, err := c.searchUsers(ctx, p.Keyword, p.Highlight, from, size)
		if err != nil {
			return nil, err
		}
		return &AllResult{
			Result: SearchResultBuckets{
				BiliUser:     users,
				Video:        []VideoHit{},
				Article:      []ArticleHit{},
				MediaBangumi: []any{}, MediaFt: []any{}, Live: []any{}, Topic: []any{}, Photo: []any{},
			},
			TopTlist:   TopTlist{BiliUser: int(userTotal)},
			NumResults: userTotal,
			Page:       page,
			PageSize:   size,
		}, nil
	}
	if strings.EqualFold(strings.TrimSpace(p.Type), "video") {
		page := p.Page
		if page < 1 {
			page = 1
		}
		size := p.PageSize
		if size < 1 {
			size = 20
		}
		if size > 50 {
			size = 50
		}
		from := (page - 1) * size
		videos, videoTotal, err := c.searchVideosFiltered(ctx, p.Keyword, p.Highlight, p.Video, from, size)
		if err != nil {
			return nil, err
		}
		return &AllResult{
			Result: SearchResultBuckets{
				Video:        videos,
				Article:      []ArticleHit{},
				BiliUser:     []UserHit{},
				MediaBangumi: []any{}, MediaFt: []any{}, Live: []any{}, Topic: []any{}, Photo: []any{},
			},
			TopTlist:     TopTlist{Video: int(videoTotal)},
			NumResults:   videoTotal,
			Page:         page,
			PageSize:     size,
			SearchStatus: searchStatusFromCount(videoTotal),
		}, nil
	}
	if strings.EqualFold(strings.TrimSpace(p.Type), "article") {
		ap, err := c.SearchArticles(ctx, p)
		if err != nil {
			return nil, err
		}
		return &AllResult{
			Result: SearchResultBuckets{
				Article:      ap.Result,
				Video:        []VideoHit{},
				BiliUser:     []UserHit{},
				MediaBangumi: []any{}, MediaFt: []any{}, Live: []any{}, Topic: []any{}, Photo: []any{},
			},
			TopTlist: TopTlist{Article: int(ap.NumResults)},
			NumResults: ap.NumResults,
			Page:       ap.Page,
			PageSize:   ap.PageSize,
		}, nil
	}
	keyword := strings.TrimSpace(p.Keyword)
	if keyword == "" {
		return &AllResult{
			Result: SearchResultBuckets{
				Video: []VideoHit{}, Article: []ArticleHit{}, BiliUser: []UserHit{},
				MediaBangumi: []any{}, MediaFt: []any{}, Live: []any{}, Topic: []any{}, Photo: []any{},
			},
			TopTlist: TopTlist{},
		}, nil
	}
	page := p.Page
	if page < 1 {
		page = 1
	}
	size := p.PageSize
	if size < 1 {
		size = 20
	}
	if size > 50 {
		size = 50
	}
	from := (page - 1) * size

	videos, videoTotal, err := c.searchVideosFiltered(ctx, keyword, p.Highlight, p.Video, from, size)
	if err != nil {
		return nil, err
	}
	articles, articleTotal, err := c.searchArticles(ctx, keyword, p.Highlight, "", 0, 10)
	if err != nil {
		return nil, err
	}
	users, userTotal, err := c.searchUsers(ctx, keyword, p.Highlight, 0, 10)
	if err != nil {
		return nil, err
	}

	return &AllResult{
		Result: SearchResultBuckets{
			Video:        videos,
			Article:      articles,
			BiliUser:     users,
			MediaBangumi: []any{},
			MediaFt:      []any{},
			Live:         []any{},
			Topic:        []any{},
			Photo:        []any{},
		},
		TopTlist: TopTlist{
			Video:        int(videoTotal),
			Article:      int(articleTotal),
			BiliUser:     int(userTotal),
			MediaBangumi: 0,
			Movie:        0,
			Live:         0,
			Topic:        0,
			Photo:        0,
		},
	}, nil
}

func searchStatusFromCount(n int64) string {
	if n <= 0 {
		return "empty"
	}
	return "ok"
}

func (c *Client) searchVideosFiltered(ctx context.Context, keyword string, highlight bool, vf VideoFilter, from, size int) ([]VideoHit, int64, error) {
	body := buildVideoQuery(keyword, highlight, vf, from, size)
	return c.parseVideoHits(ctx, body)
}

func buildVideoQuery(keyword string, highlight bool, vf VideoFilter, from, size int) map[string]any {
	should := []any{
		map[string]any{
			"multi_match": map[string]any{
				"query":  keyword,
				"fields": []string{"title^3", "description", "tags", "uploader"},
			},
		},
	}
	if id, err := strconv.ParseUint(keyword, 10, 64); err == nil && id > 0 {
		should = append(should, map[string]any{"term": map[string]any{"id": id}})
	}
	filters := []any{
		map[string]any{"term": map[string]any{"status": "published"}},
	}
	filters = appendVideoFilters(filters, vf)
	q := map[string]any{
		"from": from,
		"size": size,
		"query": map[string]any{
			"bool": map[string]any{
				"filter":               filters,
				"should":               should,
				"minimum_should_match": 1,
			},
		},
		"sort": videoSortClause(vf.Order),
	}
	if highlight {
		q["highlight"] = map[string]any{
			"pre_tags":  []string{`<em class="keyword">`},
			"post_tags": []string{"</em>"},
			"fields": map[string]any{
				"title":       map[string]any{},
				"description": map[string]any{"fragment_size": 80, "number_of_fragments": 1},
			},
		}
	}
	return q
}

func (c *Client) parseVideoHits(ctx context.Context, query map[string]any) ([]VideoHit, int64, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, 0, err
	}
	res, err := c.es.Search(
		c.es.Search.WithContext(ctx),
		c.es.Search.WithIndex(IndexVideos),
		c.es.Search.WithBody(&buf),
	)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()
	if res.IsError() {
		msg, _ := io.ReadAll(res.Body)
		return nil, 0, fmt.Errorf("search videos: %s %s", res.Status(), strings.TrimSpace(string(msg)))
	}
	var parsed esSearchResponse
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return nil, 0, err
	}
	out := make([]VideoHit, 0, len(parsed.Hits.Hits))
	for _, h := range parsed.Hits.Hits {
		var d videoDoc
		if err := json.Unmarshal(h.Source, &d); err != nil {
			continue
		}
		title := d.Title
		if frags, ok := h.Highlight["title"]; ok && len(frags) > 0 {
			title = frags[0]
		}
		desc := d.Description
		if frags, ok := h.Highlight["description"]; ok && len(frags) > 0 {
			desc = stripHTML(frags[0])
		}
		out = append(out, VideoHit{
			Aid:         d.ID,
			Title:       title,
			Pic:         d.CoverURL,
			Duration:    formatDuration(d.DurationSec),
			Play:        d.PlayCount,
			VideoReview: d.DanmakuCount,
			Pubdate:     d.CreatedAt.Unix(),
			Author:      d.Uploader,
			Mid:         d.UserID,
			Description: desc,
			TypeName:    videoSearchTypeName(d.Zone),
		})
	}
	return out, parsed.Hits.Total.Value, nil
}

func articleSortClause(sort string) []any {
	switch strings.TrimSpace(sort) {
	case "pubdate":
		return []any{
			map[string]any{"published_at": map[string]any{"order": "desc", "missing": "_last"}},
			map[string]any{"_score": map[string]any{"order": "desc"}},
		}
	case "click":
		return []any{
			map[string]any{"view_count": map[string]any{"order": "desc"}},
			map[string]any{"_score": map[string]any{"order": "desc"}},
		}
	case "like":
		return []any{
			map[string]any{"fav_count": map[string]any{"order": "desc"}},
			map[string]any{"_score": map[string]any{"order": "desc"}},
		}
	case "reply":
		return []any{
			map[string]any{"comment_count": map[string]any{"order": "desc"}},
			map[string]any{"_score": map[string]any{"order": "desc"}},
		}
	default:
		return []any{
			map[string]any{"_score": map[string]any{"order": "desc"}},
			map[string]any{"published_at": map[string]any{"order": "desc", "missing": "_last"}},
		}
	}
}

func (c *Client) searchArticles(ctx context.Context, keyword string, highlight bool, sort string, from, size int) ([]ArticleHit, int64, error) {
	should := []any{
		map[string]any{
			"multi_match": map[string]any{
				"query":  keyword,
				"fields": []string{"title^3", "excerpt", "body", "author", "tags"},
			},
		},
	}
	if id, err := strconv.ParseUint(keyword, 10, 64); err == nil && id > 0 {
		should = append(should, map[string]any{"term": map[string]any{"id": id}})
	}
	q := map[string]any{
		"from": from,
		"size": size,
		"query": map[string]any{
			"bool": map[string]any{
				"filter": []any{
					map[string]any{"term": map[string]any{"status": "published"}},
				},
				"should":               should,
				"minimum_should_match": 1,
			},
		},
		"sort": articleSortClause(sort),
	}
	if highlight {
		q["highlight"] = map[string]any{
			"pre_tags":  []string{`<em class="keyword">`},
			"post_tags": []string{"</em>"},
			"fields": map[string]any{
				"title":   map[string]any{},
				"excerpt": map[string]any{"fragment_size": 100, "number_of_fragments": 1},
			},
		}
	}
	var buf bytes.Buffer
	_ = json.NewEncoder(&buf).Encode(q)
	res, err := c.es.Search(
		c.es.Search.WithContext(ctx),
		c.es.Search.WithIndex(IndexArticles),
		c.es.Search.WithBody(&buf),
	)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()
	if res.IsError() {
		msg, _ := io.ReadAll(res.Body)
		return nil, 0, fmt.Errorf("search articles: %s %s", res.Status(), strings.TrimSpace(string(msg)))
	}
	var parsed esSearchResponse
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return nil, 0, err
	}
	out := make([]ArticleHit, 0, len(parsed.Hits.Hits))
	for _, h := range parsed.Hits.Hits {
		var d articleDoc
		if err := json.Unmarshal(h.Source, &d); err != nil {
			continue
		}
		title := d.Title
		if frags, ok := h.Highlight["title"]; ok && len(frags) > 0 {
			title = frags[0]
		}
		desc := d.Excerpt
		if desc == "" {
			desc = articleExcerpt(d.Body, 120)
		}
		if frags, ok := h.Highlight["excerpt"]; ok && len(frags) > 0 {
			desc = stripHTML(frags[0])
		}
		pub := int64(0)
		if d.PublishedAt != nil {
			pub = d.PublishedAt.Unix()
		}
		cat := d.Category
		if cat == "" {
			cat = "专栏"
		}
		out = append(out, ArticleHit{
			ID:           d.ID,
			Title:        title,
			Desc:         desc,
			CoverURL:     d.CoverURL,
			View:         d.ViewCount,
			Like:         d.FavCount,
			Reply:        d.CommentCount,
			Author:       d.Author,
			Mid:          d.UserID,
			Pubdate:      pub,
			Face:         d.AuthorAvatar,
			CategoryName: cat,
		})
	}
	return out, parsed.Hits.Total.Value, nil
}

func (c *Client) searchUsers(ctx context.Context, keyword string, highlight bool, from, size int) ([]UserHit, int64, error) {
	kw := strings.TrimSpace(keyword)
	should := []any{
		map[string]any{
			"multi_match": map[string]any{
				"query":  kw,
				"fields": []string{"nickname^2", "username", "cake_id", "sign"},
			},
		},
	}
	if kw != "" {
		// 支持昵称/用户名子串（如 earthcake 搜 cake）
		should = append(should, map[string]any{
			"query_string": map[string]any{
				"query":            "*" + escapeQueryString(kw) + "*",
				"fields":           []string{"nickname", "username", "sign"},
				"analyze_wildcard": true,
				"default_operator": "OR",
			},
		})
		if id, err := strconv.ParseUint(kw, 10, 64); err == nil && id > 0 {
			should = append(should, map[string]any{"term": map[string]any{"id": id}})
		}
	}
	q := map[string]any{
		"from": from,
		"size": size,
		"query": map[string]any{
			"bool": map[string]any{
				"filter": []any{
					map[string]any{"term": map[string]any{"status": "active"}},
				},
				"should":               should,
				"minimum_should_match": 1,
			},
		},
	}
	if highlight {
		q["highlight"] = map[string]any{
			"pre_tags":  []string{`<em class="keyword">`},
			"post_tags": []string{"</em>"},
			"fields":    map[string]any{"nickname": map[string]any{}, "username": map[string]any{}},
		}
	}
	var buf bytes.Buffer
	_ = json.NewEncoder(&buf).Encode(q)
	res, err := c.es.Search(
		c.es.Search.WithContext(ctx),
		c.es.Search.WithIndex(IndexUsers),
		c.es.Search.WithBody(&buf),
	)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()
	if res.IsError() {
		msg, _ := io.ReadAll(res.Body)
		return nil, 0, fmt.Errorf("search users: %s %s", res.Status(), strings.TrimSpace(string(msg)))
	}
	var parsed esSearchResponse
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return nil, 0, err
	}
	out := make([]UserHit, 0, len(parsed.Hits.Hits))
	for _, h := range parsed.Hits.Hits {
		var d userDoc
		if err := json.Unmarshal(h.Source, &d); err != nil {
			continue
		}
		name := d.Nickname
		if name == "" {
			name = d.Username
		}
		if frags, ok := h.Highlight["nickname"]; ok && len(frags) > 0 {
			name = frags[0]
		} else if frags, ok := h.Highlight["username"]; ok && len(frags) > 0 {
			name = frags[0]
		}
		out = append(out, UserHit{
			Mid:   d.ID,
			Uname: name,
			Usign: strings.TrimSpace(d.Sign),
		})
	}
	return out, parsed.Hits.Total.Value, nil
}

type esSearchResponse struct {
	Hits struct {
		Total struct {
			Value int64 `json:"value"`
		} `json:"total"`
		Hits []struct {
			Source    json.RawMessage   `json:"_source"`
			Highlight map[string][]string `json:"highlight"`
		} `json:"hits"`
	} `json:"hits"`
}

func formatDuration(sec float64) string {
	s := int(sec + 0.5)
	if s < 0 {
		s = 0
	}
	if s < 3600 {
		m := s / 60
		r := s % 60
		return fmt.Sprintf("%02d:%02d", m, r)
	}
	h := s / 3600
	m := (s % 3600) / 60
	r := s % 60
	return fmt.Sprintf("%d:%02d:%02d", h, m, r)
}

var htmlTagRe = regexp.MustCompile(`<[^>]+>`)

func stripHTML(s string) string {
	return strings.TrimSpace(htmlTagRe.ReplaceAllString(s, ""))
}

// escapeQueryString escapes Lucene special chars for wildcard query_string.
func escapeQueryString(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '\\', '+', '-', '!', '(', ')', ':', '^', '[', ']', '"', '{', '}', '~', '*', '?', '|', '&', '/', '<', '>':
			b.WriteRune('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// ValidateKeyword checks search query length.
func ValidateKeyword(keyword string) error {
	k := strings.TrimSpace(keyword)
	if k == "" {
		return fmt.Errorf("empty keyword")
	}
	if utf8.RuneCountInString(k) > 50 {
		return fmt.Errorf("keyword too long")
	}
	return nil
}
