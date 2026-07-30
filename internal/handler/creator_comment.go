package handler

import (
	"minibili/internal/model/article"
	"minibili/internal/model/dynamic"
	"minibili/internal/model/user"
	"minibili/internal/model/video"
	"context"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"minibili/internal/errcode"
	"minibili/internal/middleware"
	"minibili/internal/pkg/resp"
	"minibili/internal/service"
)

const creatorCommentsMaxTotal = 50000

// ListCreatorComments lists comments on the authenticated uploader's videos or articles (创作中心 · 评论管理).
func (a *API) ListCreatorComments(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	media := strings.TrimSpace(c.Query("media"))
	if media == "article" {
		a.listCreatorArticleComments(c, uid)
		return
	}
	if media == "dynamic" {
		a.listCreatorDynamicComments(c, uid)
		return
	}
	page := queryIntDefault(c.Query("page"), 1)
	if page < 1 { page = 1 }
	pageSize := queryIntDefault(c.Query("page_size"), 10)
	if pageSize < 1 { pageSize = 10 }
	if pageSize > 50 { pageSize = 50 }
	sortKey := strings.TrimSpace(c.Query("sort"))
	if sortKey == "" { sortKey = "recent" }
	pending := strings.TrimSpace(c.Query("pending")) == "1"
	pendingStatus := strings.TrimSpace(c.Query("pending_status"))
	if pendingStatus == "" { pendingStatus = "unprocessed" }
	pendingScope := strings.TrimSpace(c.Query("scope"))
	if pendingScope == "" { pendingScope = "all" }
	keyword := strings.TrimSpace(c.Query("q"))
	var filterVideoID uint64
	if v := strings.TrimSpace(c.Query("video_id")); v != "" {
		n, err := strconv.ParseUint(v, 10, 64)
		if err != nil || n == 0 {
			resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
			return
		}
		filterVideoID = n
	}
	viewerID, _ := middleware.UserID(c)
	result, err := a.CreatorCommentSvc.ListCreatorVideoComments(context.Background(), service.CreatorVideoCommentQuery{
		UserID: uid, Page: page, PageSize: pageSize, SortKey: sortKey,
		Pending: pending, PendingStatus: pendingStatus, PendingScope: pendingScope,
		Keyword: keyword, FilterVideoID: filterVideoID, ViewerID: viewerID,
	})
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	list := result.Comments
	total := result.Total
	if total > creatorCommentsMaxTotal { total = creatorCommentsMaxTotal }
	videos := map[uint64]video.Video{}
	if len(result.VideoIDs) > 0 {
		vmap, err := a.CreatorCommentSvc.BatchFetchVideos(context.Background(), result.VideoIDs)
		if err == nil { videos = vmap }
	}
	names := map[uint64]string{}
	avatars := map[uint64]string{}
	if len(result.UserIDs) > 0 {
		umap, err := a.CreatorCommentSvc.BatchFetchUsers(context.Background(), result.UserIDs)
		if err == nil {
			for id, u := range umap {
				names[id] = user.DisplayUsername(&u)
				avatars[id] = uploaderAvatarForAPI(&u)
			}
		}
	}
	parents := map[uint64]gin.H{}
	if len(result.ParentIDs) > 0 {
		pmap, err := a.CreatorCommentSvc.BatchFetchComments(context.Background(), result.ParentIDs)
		if err == nil {
			var parentUserIDs []uint64
			for _, p := range pmap {
				parentUserIDs = append(parentUserIDs, p.UserID)
			}
			if len(parentUserIDs) > 0 {
				pumap, _ := a.CreatorCommentSvc.BatchFetchUsers(context.Background(), parentUserIDs)
				for id, p := range pmap {
					if p.UserID > 0 {
						pname := ""
						if pu, ok2 := pumap[p.UserID]; ok2 { pname = user.DisplayUsername(&pu) }
						parents[id] = gin.H{
							"id": p.ID, "user_id": p.UserID,
							"username": pname,
							"content":  previewCommentContent(p.Content, 80),
						}
					}
				}
			}
		}
	}
	replyCounts := a.CreatorCommentSvc.CommentReplyCounts(context.Background(), result.CommentIDs)
	likedByViewer := result.LikedByViewer
	if likedByViewer == nil { likedByViewer = map[uint64]bool{} }
	items := make([]gin.H, 0, len(list))
	for _, cm := range list {
		v := videos[cm.VideoID]
		item := gin.H{
			"id": cm.ID, "video_id": cm.VideoID, "user_id": cm.UserID,
			"username": names[cm.UserID], "avatar_url": avatars[cm.UserID],
			"parent_id": cm.ParentID, "content": cm.Content,
			"like_count": cm.LikeCount, "liked_by_me": likedByViewer[cm.ID],
			"reply_count": replyCounts[cm.ID],
			"created_at": cm.CreatedAt.Format("2006-01-02 15:04:05"),
			"approved": cm.Approved, "curated_ignored": cm.CuratedIgnored,
			"video": gin.H{
				"id": v.ID, "title": v.Title, "cover_url": v.CoverURL,
			},
		}
		if cm.ParentID > 0 {
			if p, ok := parents[cm.ParentID]; ok {
				item["parent"] = p
			}
		}
		items = append(items, item)
	}
	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(pageSize)))
	}
	resp.OK(c, gin.H{
		"items": items, "page": page, "page_size": pageSize,
		"total": total, "total_pages": totalPages,
	})
}

func queryIntDefault(s string, def int) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func previewCommentContent(s string, maxRunes int) string {
	s = strings.TrimSpace(s)
	if s == "" || maxRunes <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes]) + "…"
}

// listCreatorArticleComments lists comments on the authenticated user's published articles (专栏评论).
func (a *API) listCreatorArticleComments(c *gin.Context, uid uint64) {
	page := queryIntDefault(c.Query("page"), 1)
	if page < 1 { page = 1 }
	pageSize := queryIntDefault(c.Query("page_size"), 10)
	if pageSize < 1 { pageSize = 10 }
	if pageSize > 50 { pageSize = 50 }
	sortKey := strings.TrimSpace(c.Query("sort"))
	if sortKey == "" { sortKey = "recent" }
	pending := strings.TrimSpace(c.Query("pending")) == "1"
	pendingStatus := strings.TrimSpace(c.Query("pending_status"))
	if pendingStatus == "" { pendingStatus = "unprocessed" }
	pendingScope := strings.TrimSpace(c.Query("scope"))
	if pendingScope == "" { pendingScope = "all" }
	keyword := strings.TrimSpace(c.Query("q"))
	var filterArticleID uint64
	if v := strings.TrimSpace(c.Query("article_id")); v != "" {
		n, err := strconv.ParseUint(v, 10, 64)
		if err != nil || n == 0 {
			resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
			return
		}
		filterArticleID = n
	}
	viewerID, _ := middleware.UserID(c)
	result, err := a.CreatorCommentSvc.ListCreatorArticleComments(context.Background(), service.CreatorArticleCommentQuery{
		UserID: uid, Page: page, PageSize: pageSize, SortKey: sortKey,
		Pending: pending, PendingStatus: pendingStatus, PendingScope: pendingScope,
		Keyword: keyword, FilterArticleID: filterArticleID, ViewerID: viewerID,
	})
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	list := result.Comments
	total := result.Total
	if total > creatorCommentsMaxTotal { total = creatorCommentsMaxTotal }
	articles := map[uint64]article.Article{}
	if len(result.ArticleIDs) > 0 {
		amap, err := a.CreatorCommentSvc.BatchFetchArticles(context.Background(), result.ArticleIDs)
		if err == nil { articles = amap }
	}
	names := map[uint64]string{}
	avatars := map[uint64]string{}
	if len(result.UserIDs) > 0 {
		umap, err := a.CreatorCommentSvc.BatchFetchUsers(context.Background(), result.UserIDs)
		if err == nil {
			for id, u := range umap {
				names[id] = user.DisplayUsername(&u)
				avatars[id] = uploaderAvatarForAPI(&u)
			}
		}
	}
	parents := map[uint64]gin.H{}
	if len(result.ParentIDs) > 0 {
		pmap, err := a.CreatorCommentSvc.BatchFetchArticleComments(context.Background(), result.ParentIDs)
		if err == nil {
			var parentUserIDs []uint64
			for _, p := range pmap {
				parentUserIDs = append(parentUserIDs, p.UserID)
			}
			if len(parentUserIDs) > 0 {
				pumap, _ := a.CreatorCommentSvc.BatchFetchUsers(context.Background(), parentUserIDs)
				for id, p := range pmap {
					if p.UserID > 0 {
						pname := ""
						if pu, ok2 := pumap[p.UserID]; ok2 { pname = user.DisplayUsername(&pu) }
						parents[id] = gin.H{
							"id": p.ID, "user_id": p.UserID,
							"username": pname,
							"content":  previewCommentContent(p.Content, 80),
						}
					}
				}
			}
		}
	}
	replyCounts := a.CreatorCommentSvc.ArticleCommentReplyCounts(context.Background(), result.CommentIDs)
	likedByViewer := result.LikedByViewer
	if likedByViewer == nil { likedByViewer = map[uint64]bool{} }
	items := make([]gin.H, 0, len(list))
	for _, cm := range list {
		a := articles[cm.ArticleID]
		item := gin.H{
			"id": cm.ID, "article_id": cm.ArticleID, "user_id": cm.UserID,
			"username": names[cm.UserID], "avatar_url": avatars[cm.UserID],
			"parent_id": cm.ParentID, "content": cm.Content,
			"like_count": cm.LikeCount, "liked_by_me": likedByViewer[cm.ID],
			"reply_count": replyCounts[cm.ID],
			"created_at": cm.CreatedAt.Format("2006-01-02 15:04:05"),
			"approved": cm.Approved, "curated_ignored": cm.CuratedIgnored,
			"article": gin.H{"id": a.ID, "title": a.Title},
		}
		if cm.ParentID > 0 {
			if p, ok := parents[cm.ParentID]; ok {
				item["parent"] = p
			}
		}
		items = append(items, item)
	}
	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(pageSize)))
	}
	resp.OK(c, gin.H{
		"items": items, "page": page, "page_size": pageSize,
		"total": total, "total_pages": totalPages,
	})
}

func dynamicDisplayTitle(d *dynamic.UserDynamic) string {
	if d == nil {
		return ""
	}
	if t := strings.TrimSpace(d.Title); t != "" {
		return t
	}
	c := strings.TrimSpace(d.Content)
	if c == "" {
		return "图文动态"
	}
	return previewCommentContent(c, 40)
}

func dynamicCoverURL(d *dynamic.UserDynamic) string {
	if d == nil {
		return ""
	}
	imgs := parseDynamicImagesJSON(d.ImagesJSON)
	if len(imgs) > 0 {
		return imgs[0]
	}
	return ""
}

// listCreatorDynamicComments lists comments on the authenticated user's image/text dynamics.
func (a *API) listCreatorDynamicComments(c *gin.Context, uid uint64) {
	page := queryIntDefault(c.Query("page"), 1)
	if page < 1 { page = 1 }
	pageSize := queryIntDefault(c.Query("page_size"), 10)
	if pageSize < 1 { pageSize = 10 }
	if pageSize > 50 { pageSize = 50 }
	sortKey := strings.TrimSpace(c.Query("sort"))
	if sortKey == "" { sortKey = "recent" }
	pending := strings.TrimSpace(c.Query("pending")) == "1"
	pendingStatus := strings.TrimSpace(c.Query("pending_status"))
	if pendingStatus == "" { pendingStatus = "unprocessed" }
	pendingScope := strings.TrimSpace(c.Query("scope"))
	if pendingScope == "" { pendingScope = "all" }
	keyword := strings.TrimSpace(c.Query("q"))
	var filterDynamicID uint64
	if v := strings.TrimSpace(c.Query("dynamic_id")); v != "" {
		n, err := strconv.ParseUint(v, 10, 64)
		if err != nil || n == 0 {
			resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
			return
		}
		filterDynamicID = n
	}
	viewerID, _ := middleware.UserID(c)
	result, err := a.CreatorCommentSvc.ListCreatorDynamicComments(context.Background(), service.CreatorDynamicCommentQuery{
		UserID: uid, Page: page, PageSize: pageSize, SortKey: sortKey,
		Pending: pending, PendingStatus: pendingStatus, PendingScope: pendingScope,
		Keyword: keyword, FilterDynamicID: filterDynamicID, ViewerID: viewerID,
	})
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	list := result.Comments
	total := result.Total
	if total > creatorCommentsMaxTotal { total = creatorCommentsMaxTotal }
	dynamics := map[uint64]dynamic.UserDynamic{}
	if len(result.DynamicIDs) > 0 {
		dmap, err := a.CreatorCommentSvc.BatchFetchDynamics(context.Background(), result.DynamicIDs)
		if err == nil { dynamics = dmap }
	}
	names := map[uint64]string{}
	avatars := map[uint64]string{}
	if len(result.UserIDs) > 0 {
		umap, err := a.CreatorCommentSvc.BatchFetchUsers(context.Background(), result.UserIDs)
		if err == nil {
			for id, u := range umap {
				names[id] = user.DisplayUsername(&u)
				avatars[id] = uploaderAvatarForAPI(&u)
			}
		}
	}
	parents := map[uint64]gin.H{}
	if len(result.ParentIDs) > 0 {
		pmap, err := a.CreatorCommentSvc.BatchFetchDynamicComments(context.Background(), result.ParentIDs)
		if err == nil {
			var parentUserIDs []uint64
			for _, p := range pmap {
				parentUserIDs = append(parentUserIDs, p.UserID)
			}
			if len(parentUserIDs) > 0 {
				pumap, _ := a.CreatorCommentSvc.BatchFetchUsers(context.Background(), parentUserIDs)
				for id, p := range pmap {
					if p.UserID > 0 {
						pname := ""
						if pu, ok2 := pumap[p.UserID]; ok2 { pname = user.DisplayUsername(&pu) }
						parents[id] = gin.H{
							"id": p.ID, "user_id": p.UserID,
							"username": pname,
							"content":  previewCommentContent(p.Content, 80),
						}
					}
				}
			}
		}
	}
	replyCounts := a.CreatorCommentSvc.DynamicCommentReplyCounts(context.Background(), result.CommentIDs)
	likedByViewer := result.LikedByViewer
	if likedByViewer == nil { likedByViewer = map[uint64]bool{} }
	items := make([]gin.H, 0, len(list))
	for _, cm := range list {
		d := dynamics[cm.DynamicID]
		item := gin.H{
			"id": cm.ID, "dynamic_id": cm.DynamicID, "user_id": cm.UserID,
			"username": names[cm.UserID], "avatar_url": avatars[cm.UserID],
			"parent_id": cm.ParentID, "content": cm.Content,
			"like_count": cm.LikeCount, "liked_by_me": likedByViewer[cm.ID],
			"reply_count": replyCounts[cm.ID],
			"created_at": cm.CreatedAt.Format("2006-01-02 15:04:05"),
			"approved": cm.Approved, "curated_ignored": cm.CuratedIgnored,
			"dynamic": gin.H{"id": d.ID, "title": dynamicDisplayTitle(&d)},
		}
		if cm.ParentID > 0 {
			if p, ok := parents[cm.ParentID]; ok {
				item["parent"] = p
			}
		}
		items = append(items, item)
	}
	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(pageSize)))
	}
	resp.OK(c, gin.H{
		"items": items, "page": page, "page_size": pageSize,
		"total": total, "total_pages": totalPages,
	})
}
