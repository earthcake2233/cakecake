package handler

import (
	"cakecake/internal/model/article"
	"cakecake/internal/model/comment"
	"cakecake/internal/model/dynamic"
	"cakecake/internal/model/user"
	"cakecake/internal/model/video"
	cs "cakecake/internal/service/comment"
	"context"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"cakecake/internal/errcode"
	"cakecake/internal/middleware"
	"cakecake/internal/pkg/resp"
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
	switch strings.TrimSpace(c.Query("media")) {
	case "article":
		a.listCreatorArticleComments(c, uid)
		return
	case "dynamic":
		a.listCreatorDynamicComments(c, uid)
		return
	}
	q, code := parseCreatorCommentQuery(c)
	if code != 0 {
		resp.Err(c, http.StatusBadRequest, code)
		return
	}
	viewerID, _ := middleware.UserID(c)
	result, err := a.CreatorCommentSvc.ListCreatorVideoComments(c.Request.Context(), cs.CreatorVideoCommentQuery{
		UserID: uid, Page: q.page, PageSize: q.pageSize, SortKey: q.sortKey,
		Pending: q.pending, PendingStatus: q.pendingStatus, PendingScope: q.pendingScope,
		Keyword: q.keyword, FilterVideoID: q.filterVideoID, ViewerID: viewerID,
	})
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	total := result.Total
	if total > creatorCommentsMaxTotal {
		total = creatorCommentsMaxTotal
	}
	ctx := a.loadCreatorCommentContext(c.Request.Context(), result)
	items := buildCreatorVideoCommentItems(result.Comments, ctx)
	resp.OK(c, creatorVideoCommentListResponse{
		Items: items, Page: q.page, PageSize: q.pageSize,
		Total: total, TotalPages: totalPagesFor(total, q.pageSize),
	})
}

type creatorCommentQuery struct {
	page          int
	pageSize      int
	sortKey       string
	pending       bool
	pendingStatus string
	pendingScope  string
	keyword       string
	filterVideoID uint64
}

func parseCreatorCommentQuery(c *gin.Context) (creatorCommentQuery, int) {
	q := creatorCommentQuery{}
	q.page, q.pageSize = parsePagination(c, 10)
	q.sortKey = strings.TrimSpace(c.Query("sort"))
	if q.sortKey == "" {
		q.sortKey = "recent"
	}
	q.pending = strings.TrimSpace(c.Query("pending")) == "1"
	q.pendingStatus = strings.TrimSpace(c.Query("pending_status"))
	if q.pendingStatus == "" {
		q.pendingStatus = "unprocessed"
	}
	q.pendingScope = strings.TrimSpace(c.Query("scope"))
	if q.pendingScope == "" {
		q.pendingScope = "all"
	}
	q.keyword = strings.TrimSpace(c.Query("q"))
	if v := strings.TrimSpace(c.Query("video_id")); v != "" {
		n, err := strconv.ParseUint(v, 10, 64)
		if err != nil || n == 0 {
			return q, errcode.CodeParamError
		}
		q.filterVideoID = n
	}
	return q, 0
}

// creatorCommentContext holds enrichment maps shared by creator comment list builders.
type creatorCommentContext struct {
	videos        map[uint64]video.Video
	names         map[uint64]string
	avatars       map[uint64]string
	parents       map[uint64]creatorCommentParentDTO
	replyCounts   map[uint64]uint64
	likedByViewer map[uint64]bool
}

// loadCreatorCommentContext batch-fetches videos/users/parents/reply counts.
func (a *API) loadCreatorCommentContext(ctx context.Context, result *cs.CreatorVideoCommentResult) *creatorCommentContext {
	cc := &creatorCommentContext{
		videos:        map[uint64]video.Video{},
		names:         map[uint64]string{},
		avatars:       map[uint64]string{},
		parents:       map[uint64]creatorCommentParentDTO{},
		replyCounts:   a.CreatorCommentSvc.CommentReplyCounts(ctx, result.CommentIDs),
		likedByViewer: result.LikedByViewer,
	}
	if cc.likedByViewer == nil {
		cc.likedByViewer = map[uint64]bool{}
	}
	if len(result.VideoIDs) > 0 {
		if vmap, err := a.CreatorCommentSvc.BatchFetchVideos(ctx, result.VideoIDs); err == nil {
			cc.videos = vmap
		}
	}
	if len(result.UserIDs) > 0 {
		if umap, err := a.CreatorCommentSvc.BatchFetchUsers(ctx, result.UserIDs); err == nil {
			for id, u := range umap {
				cc.names[id] = user.DisplayUsername(&u)
				cc.avatars[id] = uploaderAvatarForAPI(&u)
			}
		}
	}
	if len(result.ParentIDs) > 0 {
		cc.parents = a.creatorParentMap(ctx, result.ParentIDs)
	}
	return cc
}

// creatorParentMap builds the parent-comment DTO lookup for a set of parent IDs.
func (a *API) creatorParentMap(ctx context.Context, parentIDs []uint64) map[uint64]creatorCommentParentDTO {
	out := map[uint64]creatorCommentParentDTO{}
	pmap, err := a.CreatorCommentSvc.BatchFetchComments(ctx, parentIDs)
	if err != nil {
		return out
	}
	var parentUserIDs []uint64
	for _, p := range pmap {
		parentUserIDs = append(parentUserIDs, p.UserID)
	}
	if len(parentUserIDs) == 0 {
		return out
	}
	pumap, _ := a.CreatorCommentSvc.BatchFetchUsers(ctx, parentUserIDs)
	for id, p := range pmap {
		if p.UserID == 0 {
			continue
		}
		pname := ""
		if pu, ok2 := pumap[p.UserID]; ok2 {
			pname = user.DisplayUsername(&pu)
		}
		out[id] = creatorCommentParentDTO{
			ID:       p.ID,
			UserID:   p.UserID,
			Username: pname,
			Content:  previewCommentContent(p.Content, 80),
		}
	}
	return out
}

func buildCreatorVideoCommentItems(list []comment.Comment, cc *creatorCommentContext) []creatorVideoCommentItem {
	items := make([]creatorVideoCommentItem, 0, len(list))
	for _, cm := range list {
		v := cc.videos[cm.VideoID]
		item := creatorVideoCommentItem{
			ID: cm.ID, VideoID: cm.VideoID, UserID: cm.UserID,
			Username: cc.names[cm.UserID], AvatarURL: cc.avatars[cm.UserID],
			ParentID: cm.ParentID, Content: cm.Content,
			LikeCount: cm.LikeCount, LikedByMe: cc.likedByViewer[cm.ID],
			ReplyCount: cc.replyCounts[cm.ID],
			CreatedAt:  cm.CreatedAt.Format("2006-01-02 15:04:05"),
			Approved:   cm.Approved, CuratedIgnored: cm.CuratedIgnored,
			Video: creatorCommentMediaRef{
				ID: v.ID, Title: v.Title, CoverURL: v.CoverURL,
			},
		}
		if cm.ParentID > 0 {
			if p, ok := cc.parents[cm.ParentID]; ok {
				item.Parent = &p
			}
		}
		items = append(items, item)
	}
	return items
}

func totalPagesFor(total int64, pageSize int) int {
	if total <= 0 {
		return 0
	}
	return int(math.Ceil(float64(total) / float64(pageSize)))
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
	result, err := a.CreatorCommentSvc.ListCreatorArticleComments(c.Request.Context(), cs.CreatorArticleCommentQuery{
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
		amap, err := a.CreatorCommentSvc.BatchFetchArticles(c.Request.Context(), result.ArticleIDs)
		if err == nil {
			articles = amap
		}
	}
	names := map[uint64]string{}
	avatars := map[uint64]string{}
	if len(result.UserIDs) > 0 {
		umap, err := a.CreatorCommentSvc.BatchFetchUsers(c.Request.Context(), result.UserIDs)
		if err == nil {
			for id, u := range umap {
				names[id] = user.DisplayUsername(&u)
				avatars[id] = uploaderAvatarForAPI(&u)
			}
		}
	}
	parents := map[uint64]creatorCommentParentDTO{}
	if len(result.ParentIDs) > 0 {
		pmap, err := a.CreatorCommentSvc.BatchFetchArticleComments(c.Request.Context(), result.ParentIDs)
		if err == nil {
			var parentUserIDs []uint64
			for _, p := range pmap {
				parentUserIDs = append(parentUserIDs, p.UserID)
			}
			if len(parentUserIDs) > 0 {
				pumap, _ := a.CreatorCommentSvc.BatchFetchUsers(c.Request.Context(), parentUserIDs)
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
	replyCounts := a.CreatorCommentSvc.ArticleCommentReplyCounts(c.Request.Context(), result.CommentIDs)
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
	result, err := a.CreatorCommentSvc.ListCreatorDynamicComments(c.Request.Context(), cs.CreatorDynamicCommentQuery{
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
		dmap, err := a.CreatorCommentSvc.BatchFetchDynamics(c.Request.Context(), result.DynamicIDs)
		if err == nil {
			dynamics = dmap
		}
	}
	names := map[uint64]string{}
	avatars := map[uint64]string{}
	if len(result.UserIDs) > 0 {
		umap, err := a.CreatorCommentSvc.BatchFetchUsers(c.Request.Context(), result.UserIDs)
		if err == nil {
			for id, u := range umap {
				names[id] = user.DisplayUsername(&u)
				avatars[id] = uploaderAvatarForAPI(&u)
			}
		}
	}
	parents := map[uint64]creatorCommentParentDTO{}
	if len(result.ParentIDs) > 0 {
		pmap, err := a.CreatorCommentSvc.BatchFetchDynamicComments(c.Request.Context(), result.ParentIDs)
		if err == nil {
			var parentUserIDs []uint64
			for _, p := range pmap {
				parentUserIDs = append(parentUserIDs, p.UserID)
			}
			if len(parentUserIDs) > 0 {
				pumap, _ := a.CreatorCommentSvc.BatchFetchUsers(c.Request.Context(), parentUserIDs)
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
	replyCounts := a.CreatorCommentSvc.DynamicCommentReplyCounts(c.Request.Context(), result.CommentIDs)
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
