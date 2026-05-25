package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"minibili/internal/errcode"
	"minibili/internal/middleware"
	"minibili/internal/pkg/resp"
	"minibili/internal/search"
)

// SearchAll implements GET /api/v1/search for the bilibili-vue search page.
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
	if a.SearchHot != nil {
		recCtx, recCancel := context.WithTimeout(c.Request.Context(), 500*time.Millisecond)
		if err := a.SearchHot.Record(recCtx, viewer, c.ClientIP(), keyword); err != nil {
			a.Log.Warn("record search hot", zap.Error(err), zap.String("keyword", keyword))
		}
		recCancel()
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
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	searchType := strings.TrimSpace(c.DefaultQuery("type", "all"))
	sort := strings.TrimSpace(c.Query("sort"))
	videoFilter := search.ParseVideoFilter(
		c.DefaultQuery("order", c.Query("video_order")),
		c.DefaultQuery("duration", ""),
		c.DefaultQuery("zone", ""),
	)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	out, err := a.ES.SearchAll(ctx, search.SearchParams{
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
		later := watchLaterByViewer(a.DB, viewer, ids)
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
