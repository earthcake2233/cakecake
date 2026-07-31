package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"minibili/internal/errcode"
	"minibili/internal/middleware"
	"minibili/internal/pkg/resp"
	"minibili/internal/search"
)

// hotRecordReq is a fire-and-forget hot search record request.
type hotRecordReq struct {
	userID   uint64
	clientIP string
	keyword  string
}

// InitHotRecorder starts a background worker to asynchronously record hot searches.
// Call once during app startup after SearchHot is initialized.
func (a *API) InitHotRecorder(buffer int) {
	if a.SearchHot == nil {
		return
	}
	ch := make(chan hotRecordReq, buffer)
	a.hotRecCh = ch
	go func() {
		for req := range ch {
			ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			err := a.SearchHot.Record(ctx, req.userID, req.clientIP, req.keyword)
			if err != nil {
				a.Log.Warn("async hot record", zap.Error(err), zap.String("keyword", req.keyword))
			}
			cancel()
		}
	}()
}

const searchCacheTTL = 30 * time.Second

func searchCacheKey(keyword, searchType, sort string, page, pageSize int) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%s|%d|%d", keyword, searchType, sort, page, pageSize)))
	return "search:cache:" + hex.EncodeToString(h[:])
}

// SearchAll implements GET /api/v1/search with Redis caching.
func (a *API) SearchAll(c *gin.Context) {
	keyword := strings.TrimSpace(c.Query("keyword"))
	if err := search.ValidateKeyword(keyword); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var viewer uint64
	if uid, ok := middleware.UserID(c); ok {
		viewer = uid
	}

	// Record hot search (async, fire-and-forget via channel)
	if a.hotRecCh != nil {
		select {
		case a.hotRecCh <- hotRecordReq{userID: viewer, clientIP: c.ClientIP(), keyword: keyword}:
		default:
			a.Log.Warn("hot record channel full, dropping")
		}
	}

	if a.ES == nil || !a.ES.Enabled() {
		if strings.TrimSpace(a.Cfg.ElasticsearchURL) != "" {
			resp.Err(c, http.StatusServiceUnavailable, errcode.CodeSearchUnavailable)
			return
		}
		out := emptySearchResult()
		out.SearchStatus = "unavailable"
		resp.OK(c, out)
		return
	}

	highlight := c.Query("highlight") == "1" || strings.EqualFold(c.Query("highlight"), "true")
	page, pageSize := parsePagination(c, 20)
	searchType := strings.TrimSpace(c.DefaultQuery("type", "all"))
	sort := strings.TrimSpace(c.Query("sort"))
	videoFilter := search.ParseVideoFilter(
		c.DefaultQuery("order", c.Query("video_order")),
		c.DefaultQuery("duration", ""),
		c.DefaultQuery("zone", ""),
	)

	// ── Redis cache lookup (skip for highlighted queries) ──
	var out *search.AllResult
	cacheHit := false
	if !highlight && a.Redis != nil {
		cacheKey := searchCacheKey(keyword, searchType, sort, page, pageSize)
		if cached, err := a.Redis.Get(c.Request.Context(), cacheKey).Result(); err == nil && cached != "" {
			if err := json.Unmarshal([]byte(cached), &out); err == nil {
				cacheHit = true
			}
		}
	}

	// ── Cache miss: call Elasticsearch ──
	if !cacheHit {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		var err error
		out, err = a.ES.SearchAll(ctx, search.SearchParams{
			Keyword:   keyword,
			Highlight: highlight,
			Page:      page,
			PageSize:  pageSize,
			Type:      searchType,
			Sort:      sort,
			Video:     videoFilter,
		})
		if err != nil {
			a.Log.Error("search all", zap.Error(err), zap.String("keyword", keyword))
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}

		// Write to cache (async, best-effort)
		if !highlight && a.Redis != nil {
			if data, err := json.Marshal(out); err == nil {
				cacheKey := searchCacheKey(keyword, searchType, sort, page, pageSize)
				if err := a.Redis.Set(c.Request.Context(), cacheKey, string(data), searchCacheTTL).Err(); err != nil {
					a.Log.Warn("search cache write", zap.Error(err))
				}
			}
		}
	}

	// ── Viewer enrichment (always run, even on cache hit) ──
	if len(out.Result.BiliUser) > 0 {
		out.Result.BiliUser = search.EnrichUserHits(a.DB, viewer, out.Result.BiliUser)
	}
	if len(out.Result.Video) > 0 && viewer > 0 {
		ids := make([]uint64, 0, len(out.Result.Video))
		for _, v := range out.Result.Video {
			if v.Aid > 0 {
				ids = append(ids, v.Aid)
			}
		}
		later := a.EngagementSvc.BatchWatchLater(context.Background(), viewer, ids)
		for i := range out.Result.Video {
			out.Result.Video[i].InWatchLater = later[out.Result.Video[i].Aid]
		}
	}
	if out.SearchStatus == "" {
		if searchResultEmpty(out) {
			out.SearchStatus = "empty"
		} else {
			out.SearchStatus = "ok"
		}
	}

	if cacheHit {
		a.Log.Debug("search cache hit", zap.String("keyword", keyword))
	}
	resp.OK(c, out)
}

func searchResultEmpty(out *search.AllResult) bool {
	if out == nil {
		return true
	}
	r := out.Result
	return len(r.Video) == 0 &&
		len(r.Article) == 0 &&
		len(r.BiliUser) == 0 &&
		len(r.MediaBangumi) == 0 &&
		len(r.MediaFt) == 0 &&
		len(r.Live) == 0 &&
		len(r.Topic) == 0 &&
		len(r.Photo) == 0
}

func emptySearchResult() *search.AllResult {
	return &search.AllResult{
		Result: search.SearchResultBuckets{
			Video:        []search.VideoHit{},
			Article:      []search.ArticleHit{},
			BiliUser:     []search.UserHit{},
			MediaBangumi: []any{},
			MediaFt:      []any{},
			Live:         []any{},
			Topic:        []any{},
			Photo:        []any{},
		},
		TopTlist:     search.TopTlist{},
		SearchStatus: "empty",
	}
}
