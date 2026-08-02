package handler

import (
	"cakecake/internal/model/article"
	"cakecake/internal/model/dynamic"
	"cakecake/internal/model/user"
	"cakecake/internal/model/video"
	"context"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"cakecake/internal/errcode"
	"cakecake/internal/middleware"
	"cakecake/internal/pkg/resp"
	"cakecake/internal/service"
)

const creatorCommentsMaxTotal = 50000

// ListCreatorComments lists comments on the authenticated uploader's videos or articles (Creator Hub comment management).
type creatorCommentParentDTO struct {
	ID       uint64 `json:"id"`
	UserID   uint64 `json:"user_id"`
	Username string `json:"username"`
	Content  string `json:"content"`
}

type creatorCommentMediaRef struct {
	ID       uint64 `json:"id"`
	Title    string `json:"title"`
	CoverURL string `json:"cover_url,omitempty"`
}

type creatorVideoCommentItem struct {
	ID             uint64                   `json:"id"`
	VideoID        uint64                   `json:"video_id"`
	UserID         uint64                   `json:"user_id"`
	Username       string                   `json:"username"`
	AvatarURL      string                   `json:"avatar_url"`
	ParentID       uint64                   `json:"parent_id"`
	Content        string                   `json:"content"`
	LikeCount      uint64                   `json:"like_count"`
	LikedByMe      bool                     `json:"liked_by_me"`
	ReplyCount     uint64                   `json:"reply_count"`
	CreatedAt      string                   `json:"created_at"`
	Approved       bool                     `json:"approved"`
	CuratedIgnored bool                     `json:"curated_ignored"`
	Video          creatorCommentMediaRef   `json:"video"`
	Parent         *creatorCommentParentDTO `json:"parent,omitempty"`
}

type creatorArticleCommentItem struct {
	ID             uint64                   `json:"id"`
	ArticleID      uint64                   `json:"article_id"`
	UserID         uint64                   `json:"user_id"`
	Username       string                   `json:"username"`
	AvatarURL      string                   `json:"avatar_url"`
	ParentID       uint64                   `json:"parent_id"`
	Content        string                   `json:"content"`
	LikeCount      uint64                   `json:"like_count"`
	LikedByMe      bool                     `json:"liked_by_me"`
	ReplyCount     uint64                   `json:"reply_count"`
	CreatedAt      string                   `json:"created_at"`
	Approved       bool                     `json:"approved"`
	CuratedIgnored bool                     `json:"curated_ignored"`
	Article        creatorCommentMediaRef   `json:"article"`
	Parent         *creatorCommentParentDTO `json:"parent,omitempty"`
}

type creatorDynamicCommentItem struct {
	ID             uint64                   `json:"id"`
	DynamicID      uint64                   `json:"dynamic_id"`
	UserID         uint64                   `json:"user_id"`
	Username       string                   `json:"username"`
	AvatarURL      string                   `json:"avatar_url"`
	ParentID       uint64                   `json:"parent_id"`
	Content        string                   `json:"content"`
	LikeCount      uint64                   `json:"like_count"`
	LikedByMe      bool                     `json:"liked_by_me"`
	ReplyCount     uint64                   `json:"reply_count"`
	CreatedAt      string                   `json:"created_at"`
	Approved       bool                     `json:"approved"`
	CuratedIgnored bool                     `json:"curated_ignored"`
	Dynamic        creatorCommentMediaRef   `json:"dynamic"`
	Parent         *creatorCommentParentDTO `json:"parent,omitempty"`
}

type creatorVideoCommentListResponse struct {
	Items      []creatorVideoCommentItem `json:"items"`
	Page       int                       `json:"page"`
	PageSize   int                       `json:"page_size"`
	Total      int64                     `json:"total"`
	TotalPages int                       `json:"total_pages"`
}

type creatorArticleCommentListResponse struct {
	Items      []creatorArticleCommentItem `json:"items"`
	Page       int                         `json:"page"`
	PageSize   int                         `json:"page_size"`
	Total      int64                       `json:"total"`
	TotalPages int                         `json:"total_pages"`
}

type creatorDynamicCommentListResponse struct {
	Items      []creatorDynamicCommentItem `json:"items"`
	Page       int                         `json:"page"`
	PageSize   int                         `json:"page_size"`
	Total      int64                       `json:"total"`
	TotalPages int                         `json:"total_pages"`
}

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
	page, pageSize := parsePagination(c, 10)
	sortKey := strings.TrimSpace(c.Query("sort"))
	if sortKey == "" {
		sortKey = "recent"
	}
	pending := strings.TrimSpace(c.Query("pending")) == "1"
	pendingStatus := strings.TrimSpace(c.Query("pending_status"))
	if pendingStatus == "" {
		pendingStatus = "unprocessed"
	}
	pendingScope := strings.TrimSpace(c.Query("scope"))
	if pendingScope == "" {
		pendingScope = "all"
	}
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
	if total > creatorCommentsMaxTotal {
		total = creatorCommentsMaxTotal
	}
	videos := map[uint64]video.Video{}
	if len(result.VideoIDs) > 0 {
		vmap, err := a.CreatorCommentSvc.BatchFetchVideos(context.Background(), result.VideoIDs)
		if err == nil {
			videos = vmap
		}
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
	parents := map[uint64]creatorCommentParentDTO{}
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
						if pu, ok2 := pumap[p.UserID]; ok2 {
							pname = user.DisplayUsername(&pu)
						}
						parents[id] = creatorCommentParentDTO{
							ID:       p.ID,
							UserID:   p.UserID,
							Username: pname,
							Content:  previewCommentContent(p.Content, 80),
						}
					}
				}
			}
		}
	}
	replyCounts := a.CreatorCommentSvc.CommentReplyCounts(context.Background(), result.CommentIDs)
	likedByViewer := result.LikedByViewer
	if likedByViewer == nil {
		likedByViewer = map[uint64]bool{}
	}
	items := make([]creatorVideoCommentItem, 0, len(list))
	for _, cm := range list {
		v := videos[cm.VideoID]
		item := creatorVideoCommentItem{
			ID: cm.ID, VideoID: cm.VideoID, UserID: cm.UserID,
			Username: names[cm.UserID], AvatarURL: avatars[cm.UserID],
			ParentID: cm.ParentID, Content: cm.Content,
			LikeCount: cm.LikeCount, LikedByMe: likedByViewer[cm.ID],
			ReplyCount: replyCounts[cm.ID],
			CreatedAt:  cm.CreatedAt.Format("2006-01-02 15:04:05"),
			Approved:   cm.Approved, CuratedIgnored: cm.CuratedIgnored,
			Video: creatorCommentMediaRef{
				ID: v.ID, Title: v.Title, CoverURL: v.CoverURL,
			},
		}
		if cm.ParentID > 0 {
			if p, ok := parents[cm.ParentID]; ok {
				item.Parent = &p
			}
		}
		items = append(items, item)
	}
	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(pageSize)))
	}
	resp.OK(c, creatorVideoCommentListResponse{
		Items: items, Page: page, PageSize: pageSize,
		Total: total, TotalPages: totalPages,
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

// listCreatorArticleComments lists comments on the authenticated user's published articles.
func (a *API) listCreatorArticleComments(c *gin.Context, uid uint64) {
	page, pageSize := parsePagination(c, 10)
	sortKey := strings.TrimSpace(c.Query("sort"))
	if sortKey == "" {
		sortKey = "recent"
	}
	pending := strings.TrimSpace(c.Query("pending")) == "1"
	pendingStatus := strings.TrimSpace(c.Query("pending_status"))
	if pendingStatus == "" {
		pendingStatus = "unprocessed"
	}
	pendingScope := strings.TrimSpace(c.Query("scope"))
	if pendingScope == "" {
		pendingScope = "all"
	}
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
	if total > creatorCommentsMaxTotal {
		total = creatorCommentsMaxTotal
	}
	articles := map[uint64]article.Article{}
	if len(result.ArticleIDs) > 0 {
		amap, err := a.CreatorCommentSvc.BatchFetchArticles(context.Background(), result.ArticleIDs)
		if err == nil {
			articles = amap
		}
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
	parents := map[uint64]creatorCommentParentDTO{}
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
						if pu, ok2 := pumap[p.UserID]; ok2 {
							pname = user.DisplayUsername(&pu)
						}
						parents[id] = creatorCommentParentDTO{
							ID:       p.ID,
							UserID:   p.UserID,
							Username: pname,
							Content:  previewCommentContent(p.Content, 80),
						}
					}
				}
			}
		}
	}
	replyCounts := a.CreatorCommentSvc.ArticleCommentReplyCounts(context.Background(), result.CommentIDs)
	likedByViewer := result.LikedByViewer
	if likedByViewer == nil {
		likedByViewer = map[uint64]bool{}
	}
	items := make([]creatorArticleCommentItem, 0, len(list))
	for _, cm := range list {
		a := articles[cm.ArticleID]
		item := creatorArticleCommentItem{
			ID: cm.ID, ArticleID: cm.ArticleID, UserID: cm.UserID,
			Username: names[cm.UserID], AvatarURL: avatars[cm.UserID],
			ParentID: cm.ParentID, Content: cm.Content,
			LikeCount: cm.LikeCount, LikedByMe: likedByViewer[cm.ID],
			ReplyCount: replyCounts[cm.ID],
			CreatedAt:  cm.CreatedAt.Format("2006-01-02 15:04:05"),
			Approved:   cm.Approved, CuratedIgnored: cm.CuratedIgnored,
			Article: creatorCommentMediaRef{ID: a.ID, Title: a.Title},
		}
		if cm.ParentID > 0 {
			if p, ok := parents[cm.ParentID]; ok {
				item.Parent = &p
			}
		}
		items = append(items, item)
	}
	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(pageSize)))
	}
	resp.OK(c, creatorArticleCommentListResponse{
		Items: items, Page: page, PageSize: pageSize,
		Total: total, TotalPages: totalPages,
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
	page, pageSize := parsePagination(c, 10)
	sortKey := strings.TrimSpace(c.Query("sort"))
	if sortKey == "" {
		sortKey = "recent"
	}
	pending := strings.TrimSpace(c.Query("pending")) == "1"
	pendingStatus := strings.TrimSpace(c.Query("pending_status"))
	if pendingStatus == "" {
		pendingStatus = "unprocessed"
	}
	pendingScope := strings.TrimSpace(c.Query("scope"))
	if pendingScope == "" {
		pendingScope = "all"
	}
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
	if total > creatorCommentsMaxTotal {
		total = creatorCommentsMaxTotal
	}
	dynamics := map[uint64]dynamic.UserDynamic{}
	if len(result.DynamicIDs) > 0 {
		dmap, err := a.CreatorCommentSvc.BatchFetchDynamics(context.Background(), result.DynamicIDs)
		if err == nil {
			dynamics = dmap
		}
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
	parents := map[uint64]creatorCommentParentDTO{}
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
						if pu, ok2 := pumap[p.UserID]; ok2 {
							pname = user.DisplayUsername(&pu)
						}
						parents[id] = creatorCommentParentDTO{
							ID:       p.ID,
							UserID:   p.UserID,
							Username: pname,
							Content:  previewCommentContent(p.Content, 80),
						}
					}
				}
			}
		}
	}
	replyCounts := a.CreatorCommentSvc.DynamicCommentReplyCounts(context.Background(), result.CommentIDs)
	likedByViewer := result.LikedByViewer
	if likedByViewer == nil {
		likedByViewer = map[uint64]bool{}
	}
	items := make([]creatorDynamicCommentItem, 0, len(list))
	for _, cm := range list {
		d := dynamics[cm.DynamicID]
		item := creatorDynamicCommentItem{
			ID: cm.ID, DynamicID: cm.DynamicID, UserID: cm.UserID,
			Username: names[cm.UserID], AvatarURL: avatars[cm.UserID],
			ParentID: cm.ParentID, Content: cm.Content,
			LikeCount: cm.LikeCount, LikedByMe: likedByViewer[cm.ID],
			ReplyCount: replyCounts[cm.ID],
			CreatedAt:  cm.CreatedAt.Format("2006-01-02 15:04:05"),
			Approved:   cm.Approved, CuratedIgnored: cm.CuratedIgnored,
			Dynamic: creatorCommentMediaRef{ID: d.ID, Title: dynamicDisplayTitle(&d)},
		}
		if cm.ParentID > 0 {
			if p, ok := parents[cm.ParentID]; ok {
				item.Parent = &p
			}
		}
		items = append(items, item)
	}
	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(pageSize)))
	}
	resp.OK(c, creatorDynamicCommentListResponse{
		Items: items, Page: page, PageSize: pageSize,
		Total: total, TotalPages: totalPages,
	})
}
