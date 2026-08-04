package handler

import (
	"cakecake/internal/model/comment"
	"cakecake/internal/model/dynamic"
	"cakecake/internal/model/user"
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

// ListCreatorComments lists comments on the authenticated uploader's videos, articles, or
// dynamics (Creator Hub comment management). The media=article|dynamic query switches domains.
func (a *API) ListCreatorComments(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	switch strings.TrimSpace(c.Query("media")) {
	case "article":
		listCreatorCommentsFor(a, c, uid, articleCreatorCommentKind(a))
		return
	case "dynamic":
		listCreatorCommentsFor(a, c, uid, dynamicCreatorCommentKind(a))
		return
	}
	listCreatorCommentsFor(a, c, uid, videoCreatorCommentKind(a))
}

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

// creatorCommentQuery holds filter params shared by all creator comment list variants.
type creatorCommentQuery struct {
	page          int
	pageSize      int
	sortKey       string
	pending       bool
	pendingStatus string
	pendingScope  string
	keyword       string
	filterMediaID uint64
}

func parseCreatorCommentQuery(c *gin.Context) (creatorCommentQuery, int) {
	return parseCreatorCommentQueryFor(c, "video_id")
}

// parseCreatorCommentQueryFor parses the shared creator comment filters plus one media filter param.
func parseCreatorCommentQueryFor(c *gin.Context, filterParam string) (creatorCommentQuery, int) {
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
	if v := strings.TrimSpace(c.Query(filterParam)); v != "" {
		n, err := strconv.ParseUint(v, 10, 64)
		if err != nil || n == 0 {
			return q, errcode.CodeParamError
		}
		q.filterMediaID = n
	}
	return q, 0
}

// creatorCommentListPayload is the domain-agnostic result of a creator comment query.
type creatorCommentListPayload[R any] struct {
	comments      []R
	total         int64
	mediaIDs      []uint64
	userIDs       []uint64
	parentIDs     []uint64
	commentIDs    []uint64
	likedByViewer map[uint64]bool
}

// creatorCommentKind binds one media domain (video/article/dynamic) to the shared list flow.
type creatorCommentKind[R any, Item any] struct {
	parseQuery   func(c *gin.Context) (creatorCommentQuery, int)
	fetch        func(ctx context.Context, uid uint64, q creatorCommentQuery, viewerID uint64) (*creatorCommentListPayload[R], error)
	fetchMedia   func(ctx context.Context, ids []uint64) map[uint64]creatorCommentMediaRef
	fetchParents func(ctx context.Context, ids []uint64) map[uint64]creatorCommentParentDTO
	replyCounts  func(ctx context.Context, ids []uint64) map[uint64]uint64
	buildItems   func(list []R, cc *creatorCommentContext) []Item
	respond      func(c *gin.Context, items []Item, q creatorCommentQuery, total int64)
}

// listCreatorCommentsFor runs the shared creator comment list flow for one media domain.
func listCreatorCommentsFor[R any, Item any](a *API, c *gin.Context, uid uint64, kind creatorCommentKind[R, Item]) {
	q, code := kind.parseQuery(c)
	if code != 0 {
		resp.Err(c, http.StatusBadRequest, code)
		return
	}
	viewerID, _ := middleware.UserID(c)
	payload, err := kind.fetch(c.Request.Context(), uid, q, viewerID)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	total := payload.total
	if total > creatorCommentsMaxTotal {
		total = creatorCommentsMaxTotal
	}
	cc := loadCreatorCommentListContext(a, c.Request.Context(), payload, kind.fetchMedia, kind.fetchParents, kind.replyCounts)
	items := kind.buildItems(payload.comments, cc)
	kind.respond(c, items, q, total)
}

// creatorCommentContext holds enrichment maps shared by creator comment list builders.
type creatorCommentContext struct {
	media         map[uint64]creatorCommentMediaRef
	names         map[uint64]string
	avatars       map[uint64]string
	parents       map[uint64]creatorCommentParentDTO
	replyCounts   map[uint64]uint64
	likedByViewer map[uint64]bool
}

// loadCreatorCommentListContext batch-fetches media/users/parents/reply counts/likes for one list payload.
func loadCreatorCommentListContext[R any](a *API, ctx context.Context, payload *creatorCommentListPayload[R], fetchMedia func(context.Context, []uint64) map[uint64]creatorCommentMediaRef, fetchParents func(context.Context, []uint64) map[uint64]creatorCommentParentDTO, replyCounts func(context.Context, []uint64) map[uint64]uint64) *creatorCommentContext {
	cc := &creatorCommentContext{
		media:         fetchMedia(ctx, payload.mediaIDs),
		names:         map[uint64]string{},
		avatars:       map[uint64]string{},
		parents:       fetchParents(ctx, payload.parentIDs),
		replyCounts:   replyCounts(ctx, payload.commentIDs),
		likedByViewer: payload.likedByViewer,
	}
	if cc.likedByViewer == nil {
		cc.likedByViewer = map[uint64]bool{}
	}
	if len(payload.userIDs) > 0 {
		umap, err := a.CreatorCommentSvc.BatchFetchUsers(ctx, payload.userIDs)
		if err == nil {
			for id, u := range umap {
				cc.names[id] = user.DisplayUsername(&u)
				cc.avatars[id] = uploaderAvatarForAPI(&u)
			}
		}
	}
	return cc
}

// creatorParentMapFor builds the parent-comment DTO lookup for a set of parent IDs of one domain.
func creatorParentMapFor[R any](a *API, ctx context.Context, parentIDs []uint64, fetch func(context.Context, []uint64) (map[uint64]R, error), idOf, userIDOf func(R) uint64, contentOf func(R) string) map[uint64]creatorCommentParentDTO {
	out := map[uint64]creatorCommentParentDTO{}
	if len(parentIDs) == 0 {
		return out
	}
	pmap, err := fetch(ctx, parentIDs)
	if err != nil {
		return out
	}
	parentUserIDs := make([]uint64, 0, len(pmap))
	for _, p := range pmap {
		parentUserIDs = append(parentUserIDs, userIDOf(p))
	}
	if len(parentUserIDs) == 0 {
		return out
	}
	pumap, _ := a.CreatorCommentSvc.BatchFetchUsers(ctx, parentUserIDs)
	for id, p := range pmap {
		uid := userIDOf(p)
		if uid == 0 {
			continue
		}
		pname := ""
		if pu, ok2 := pumap[uid]; ok2 {
			pname = user.DisplayUsername(&pu)
		}
		out[id] = creatorCommentParentDTO{
			ID:       idOf(p),
			UserID:   uid,
			Username: pname,
			Content:  previewCommentContent(contentOf(p), 80),
		}
	}
	return out
}

func (a *API) fetchCreatorCommentVideos(ctx context.Context, ids []uint64) map[uint64]creatorCommentMediaRef {
	out := map[uint64]creatorCommentMediaRef{}
	if len(ids) == 0 {
		return out
	}
	vm, err := a.CreatorCommentSvc.BatchFetchVideos(ctx, ids)
	if err != nil {
		return out
	}
	for id, v := range vm {
		out[id] = creatorCommentMediaRef{ID: v.ID, Title: v.Title, CoverURL: v.CoverURL}
	}
	return out
}

func (a *API) fetchCreatorCommentArticles(ctx context.Context, ids []uint64) map[uint64]creatorCommentMediaRef {
	out := map[uint64]creatorCommentMediaRef{}
	if len(ids) == 0 {
		return out
	}
	am, err := a.CreatorCommentSvc.BatchFetchArticles(ctx, ids)
	if err != nil {
		return out
	}
	for id, art := range am {
		out[id] = creatorCommentMediaRef{ID: art.ID, Title: art.Title}
	}
	return out
}

func (a *API) fetchCreatorCommentDynamics(ctx context.Context, ids []uint64) map[uint64]creatorCommentMediaRef {
	out := map[uint64]creatorCommentMediaRef{}
	if len(ids) == 0 {
		return out
	}
	dm, err := a.CreatorCommentSvc.BatchFetchDynamics(ctx, ids)
	if err != nil {
		return out
	}
	for id, d := range dm {
		out[id] = creatorCommentMediaRef{ID: d.ID, Title: dynamicDisplayTitle(&d)}
	}
	return out
}

func attachCreatorCommentParent(parent **creatorCommentParentDTO, parentID uint64, cc *creatorCommentContext) {
	if parentID == 0 {
		return
	}
	if p, ok := cc.parents[parentID]; ok {
		*parent = &p
	}
}

func buildCreatorVideoCommentItems(list []comment.Comment, cc *creatorCommentContext) []creatorVideoCommentItem {
	items := make([]creatorVideoCommentItem, 0, len(list))
	for _, cm := range list {
		m := cc.media[cm.VideoID]
		item := creatorVideoCommentItem{
			ID: cm.ID, VideoID: cm.VideoID, UserID: cm.UserID,
			Username: cc.names[cm.UserID], AvatarURL: cc.avatars[cm.UserID],
			ParentID: cm.ParentID, Content: cm.Content,
			LikeCount: cm.LikeCount, LikedByMe: cc.likedByViewer[cm.ID],
			ReplyCount: cc.replyCounts[cm.ID],
			CreatedAt:  cm.CreatedAt.Format("2006-01-02 15:04:05"),
			Approved:   cm.Approved, CuratedIgnored: cm.CuratedIgnored,
			Video: creatorCommentMediaRef{ID: m.ID, Title: m.Title, CoverURL: m.CoverURL},
		}
		attachCreatorCommentParent(&item.Parent, cm.ParentID, cc)
		items = append(items, item)
	}
	return items
}

func buildCreatorArticleCommentItems(list []comment.ArticleComment, cc *creatorCommentContext) []creatorArticleCommentItem {
	items := make([]creatorArticleCommentItem, 0, len(list))
	for _, cm := range list {
		m := cc.media[cm.ArticleID]
		item := creatorArticleCommentItem{
			ID: cm.ID, ArticleID: cm.ArticleID, UserID: cm.UserID,
			Username: cc.names[cm.UserID], AvatarURL: cc.avatars[cm.UserID],
			ParentID: cm.ParentID, Content: cm.Content,
			LikeCount: cm.LikeCount, LikedByMe: cc.likedByViewer[cm.ID],
			ReplyCount: cc.replyCounts[cm.ID],
			CreatedAt:  cm.CreatedAt.Format("2006-01-02 15:04:05"),
			Approved:   cm.Approved, CuratedIgnored: cm.CuratedIgnored,
			Article: creatorCommentMediaRef{ID: m.ID, Title: m.Title},
		}
		attachCreatorCommentParent(&item.Parent, cm.ParentID, cc)
		items = append(items, item)
	}
	return items
}

func buildCreatorDynamicCommentItems(list []comment.DynamicComment, cc *creatorCommentContext) []creatorDynamicCommentItem {
	items := make([]creatorDynamicCommentItem, 0, len(list))
	for _, cm := range list {
		m := cc.media[cm.DynamicID]
		item := creatorDynamicCommentItem{
			ID: cm.ID, DynamicID: cm.DynamicID, UserID: cm.UserID,
			Username: cc.names[cm.UserID], AvatarURL: cc.avatars[cm.UserID],
			ParentID: cm.ParentID, Content: cm.Content,
			LikeCount: cm.LikeCount, LikedByMe: cc.likedByViewer[cm.ID],
			ReplyCount: cc.replyCounts[cm.ID],
			CreatedAt:  cm.CreatedAt.Format("2006-01-02 15:04:05"),
			Approved:   cm.Approved, CuratedIgnored: cm.CuratedIgnored,
			Dynamic: creatorCommentMediaRef{ID: m.ID, Title: m.Title},
		}
		attachCreatorCommentParent(&item.Parent, cm.ParentID, cc)
		items = append(items, item)
	}
	return items
}

func respondCreatorVideoComments(c *gin.Context, items []creatorVideoCommentItem, q creatorCommentQuery, total int64) {
	resp.OK(c, creatorVideoCommentListResponse{
		Items: items, Page: q.page, PageSize: q.pageSize,
		Total: total, TotalPages: totalPagesFor(total, q.pageSize),
	})
}

func respondCreatorArticleComments(c *gin.Context, items []creatorArticleCommentItem, q creatorCommentQuery, total int64) {
	resp.OK(c, creatorArticleCommentListResponse{
		Items: items, Page: q.page, PageSize: q.pageSize,
		Total: total, TotalPages: totalPagesFor(total, q.pageSize),
	})
}

func respondCreatorDynamicComments(c *gin.Context, items []creatorDynamicCommentItem, q creatorCommentQuery, total int64) {
	resp.OK(c, creatorDynamicCommentListResponse{
		Items: items, Page: q.page, PageSize: q.pageSize,
		Total: total, TotalPages: totalPagesFor(total, q.pageSize),
	})
}

func videoCreatorCommentKind(a *API) creatorCommentKind[comment.Comment, creatorVideoCommentItem] {
	return creatorCommentKind[comment.Comment, creatorVideoCommentItem]{
		parseQuery: parseCreatorCommentQuery,
		fetch: func(ctx context.Context, uid uint64, q creatorCommentQuery, viewerID uint64) (*creatorCommentListPayload[comment.Comment], error) {
			result, err := a.CreatorCommentSvc.ListCreatorVideoComments(ctx, cs.CreatorVideoCommentQuery{
				UserID: uid, Page: q.page, PageSize: q.pageSize, SortKey: q.sortKey,
				Pending: q.pending, PendingStatus: q.pendingStatus, PendingScope: q.pendingScope,
				Keyword: q.keyword, FilterVideoID: q.filterMediaID, ViewerID: viewerID,
			})
			if err != nil {
				return nil, err
			}
			return &creatorCommentListPayload[comment.Comment]{
				comments: result.Comments, total: result.Total,
				mediaIDs: result.VideoIDs, userIDs: result.UserIDs,
				parentIDs: result.ParentIDs, commentIDs: result.CommentIDs,
				likedByViewer: result.LikedByViewer,
			}, nil
		},
		fetchMedia: a.fetchCreatorCommentVideos,
		fetchParents: func(ctx context.Context, ids []uint64) map[uint64]creatorCommentParentDTO {
			return creatorParentMapFor(a, ctx, ids, a.CreatorCommentSvc.BatchFetchComments,
				func(cm comment.Comment) uint64 { return cm.ID },
				func(cm comment.Comment) uint64 { return cm.UserID },
				func(cm comment.Comment) string { return cm.Content })
		},
		replyCounts: a.CreatorCommentSvc.CommentReplyCounts,
		buildItems:  buildCreatorVideoCommentItems,
		respond:     respondCreatorVideoComments,
	}
}

func articleCreatorCommentKind(a *API) creatorCommentKind[comment.ArticleComment, creatorArticleCommentItem] {
	return creatorCommentKind[comment.ArticleComment, creatorArticleCommentItem]{
		parseQuery: func(c *gin.Context) (creatorCommentQuery, int) {
			return parseCreatorCommentQueryFor(c, "article_id")
		},
		fetch: func(ctx context.Context, uid uint64, q creatorCommentQuery, viewerID uint64) (*creatorCommentListPayload[comment.ArticleComment], error) {
			result, err := a.CreatorCommentSvc.ListCreatorArticleComments(ctx, cs.CreatorArticleCommentQuery{
				UserID: uid, Page: q.page, PageSize: q.pageSize, SortKey: q.sortKey,
				Pending: q.pending, PendingStatus: q.pendingStatus, PendingScope: q.pendingScope,
				Keyword: q.keyword, FilterArticleID: q.filterMediaID, ViewerID: viewerID,
			})
			if err != nil {
				return nil, err
			}
			return &creatorCommentListPayload[comment.ArticleComment]{
				comments: result.Comments, total: result.Total,
				mediaIDs: result.ArticleIDs, userIDs: result.UserIDs,
				parentIDs: result.ParentIDs, commentIDs: result.CommentIDs,
				likedByViewer: result.LikedByViewer,
			}, nil
		},
		fetchMedia: a.fetchCreatorCommentArticles,
		fetchParents: func(ctx context.Context, ids []uint64) map[uint64]creatorCommentParentDTO {
			return creatorParentMapFor(a, ctx, ids, a.CreatorCommentSvc.BatchFetchArticleComments,
				func(cm comment.ArticleComment) uint64 { return cm.ID },
				func(cm comment.ArticleComment) uint64 { return cm.UserID },
				func(cm comment.ArticleComment) string { return cm.Content })
		},
		replyCounts: a.CreatorCommentSvc.ArticleCommentReplyCounts,
		buildItems:  buildCreatorArticleCommentItems,
		respond:     respondCreatorArticleComments,
	}
}

func dynamicCreatorCommentKind(a *API) creatorCommentKind[comment.DynamicComment, creatorDynamicCommentItem] {
	return creatorCommentKind[comment.DynamicComment, creatorDynamicCommentItem]{
		parseQuery: func(c *gin.Context) (creatorCommentQuery, int) {
			return parseCreatorCommentQueryFor(c, "dynamic_id")
		},
		fetch: func(ctx context.Context, uid uint64, q creatorCommentQuery, viewerID uint64) (*creatorCommentListPayload[comment.DynamicComment], error) {
			result, err := a.CreatorCommentSvc.ListCreatorDynamicComments(ctx, cs.CreatorDynamicCommentQuery{
				UserID: uid, Page: q.page, PageSize: q.pageSize, SortKey: q.sortKey,
				Pending: q.pending, PendingStatus: q.pendingStatus, PendingScope: q.pendingScope,
				Keyword: q.keyword, FilterDynamicID: q.filterMediaID, ViewerID: viewerID,
			})
			if err != nil {
				return nil, err
			}
			return &creatorCommentListPayload[comment.DynamicComment]{
				comments: result.Comments, total: result.Total,
				mediaIDs: result.DynamicIDs, userIDs: result.UserIDs,
				parentIDs: result.ParentIDs, commentIDs: result.CommentIDs,
				likedByViewer: result.LikedByViewer,
			}, nil
		},
		fetchMedia: a.fetchCreatorCommentDynamics,
		fetchParents: func(ctx context.Context, ids []uint64) map[uint64]creatorCommentParentDTO {
			return creatorParentMapFor(a, ctx, ids, a.CreatorCommentSvc.BatchFetchDynamicComments,
				func(cm comment.DynamicComment) uint64 { return cm.ID },
				func(cm comment.DynamicComment) uint64 { return cm.UserID },
				func(cm comment.DynamicComment) string { return cm.Content })
		},
		replyCounts: a.CreatorCommentSvc.DynamicCommentReplyCounts,
		buildItems:  buildCreatorDynamicCommentItems,
		respond:     respondCreatorDynamicComments,
	}
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
