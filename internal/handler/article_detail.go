package handler

import (
	"cakecake/internal/errcode"
	"cakecake/internal/middleware"
	"cakecake/internal/model/article"
	"cakecake/internal/model/user"
	"cakecake/internal/pkg/markdown"
	"cakecake/internal/pkg/resp"
	artsvc "cakecake/internal/service/article"
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type articleEngagement struct {
	FavoritedByMe bool
	CoinedByMe    bool
	MyCoinAmount  int
}

func toArticleEngagement(eng *artsvc.ArticleEngagement) articleEngagement {
	if eng == nil {
		return articleEngagement{}
	}
	return articleEngagement{
		FavoritedByMe: eng.FavoritedByMe,
		CoinedByMe:    eng.CoinedByMe,
		MyCoinAmount:  eng.MyCoinAmount,
	}
}

func loadPublishedArticle(ctx context.Context, a *API, id uint64) (article.Article, bool) {
	artInfo, err := a.ArticleSvc.GetPublishedArticle(ctx, id)
	if err != nil {
		return article.Article{}, false
	}
	return article.Article{
		ID: artInfo.ID, UserID: artInfo.UserID, Title: artInfo.Title,
		Status: artInfo.Status,
	}, true
}

func parseArticleTagsJSON(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil
	}
	var arr []string
	if err := json.Unmarshal([]byte(raw), &arr); err != nil {
		return nil
	}
	return arr
}

type articleDetailDTO struct {
	ID              uint64              `json:"id"`
	CVID            uint64              `json:"cv_id"`
	UserID          uint64              `json:"user_id"`
	Title           string              `json:"title"`
	CoverURL        string              `json:"cover_url"`
	BodyMD          string              `json:"body_md"`
	BodyHTML        string              `json:"body_html"`
	TOC             []markdown.TocEntry `json:"toc"`
	Tags            []string            `json:"tags"`
	Status          string              `json:"status"`
	FailReason      string              `json:"fail_reason"`
	ViewCount       uint64              `json:"view_count"`
	CommentCount    uint64              `json:"comment_count"`
	CoinCount       uint64              `json:"coin_count"`
	FavCount        uint64              `json:"fav_count"`
	ForwardCount    uint64              `json:"forward_count"`
	PublishedAt     string              `json:"published_at"`
	CreatedAt       string              `json:"created_at"`
	AuthorName      string              `json:"author_name"`
	AuthorAvatar    string              `json:"author_avatar"`
	FavoritedByMe   bool                `json:"favorited_by_me"`
	CoinedByMe      bool                `json:"coined_by_me"`
	MyCoinAmount    int                 `json:"my_coin_amount"`
	IsAuthor        bool                `json:"is_author"`
	CommentsClosed  bool                `json:"comments_closed"`
	CommentsCurated bool                `json:"comments_curated"`
}

type articleListItemDTO struct {
	ID              uint64 `json:"id"`
	Title           string `json:"title"`
	CoverURL        string `json:"cover_url"`
	Status          string `json:"status"`
	FailReason      string `json:"fail_reason"`
	ViewCount       uint64 `json:"view_count"`
	CommentCount    uint64 `json:"comment_count"`
	CoinCount       uint64 `json:"coin_count"`
	FavCount        uint64 `json:"fav_count"`
	ForwardCount    uint64 `json:"forward_count"`
	PublishedAt     string `json:"published_at"`
	CreatedAt       string `json:"created_at"`
	AuthorName      string `json:"author_name"`
	FavoritedByMe   bool   `json:"favorited_by_me"`
	CommentsClosed  bool   `json:"comments_closed"`
	CommentsCurated bool   `json:"comments_curated"`
	FavoritedAt     string `json:"favorited_at,omitempty"`
	Unavailable     bool   `json:"unavailable,omitempty"`
}

type articleViewCountResponse struct {
	ViewCount uint64 `json:"view_count"`
}

type articleCursorListResponse struct {
	Items      []articleListItemDTO `json:"items"`
	NextCursor string               `json:"next_cursor"`
}

func articleDetailPayload(a *API, art *article.Article, author *user.User, eng articleEngagement, viewer uint64) articleDetailDTO {
	bodyHTML, toc, _ := markdown.Render(art.BodyMD)
	upName := ""
	avatar := ""
	if author != nil {
		upName = user.DisplayUsername(author)
		if author.Nickname != "" && !user.IsUserAnonymized(author) {
			upName = strings.TrimSpace(author.Nickname)
		}
		avatar = uploaderAvatarForAPI(author)
	}
	pubAt := ""
	if art.PublishedAt != nil {
		pubAt = art.PublishedAt.Format("2006-01-02 15:04:05")
	}
	return articleDetailDTO{
		ID:              art.ID,
		CVID:            art.ID,
		UserID:          art.UserID,
		Title:           art.Title,
		CoverURL:        art.CoverURL,
		BodyMD:          art.BodyMD,
		BodyHTML:        bodyHTML,
		TOC:             toc,
		Tags:            parseArticleTagsJSON(art.TagsJSON),
		Status:          art.Status,
		FailReason:      strings.TrimSpace(art.FailReason),
		ViewCount:       art.ViewCount,
		CommentCount:    art.CommentCount,
		CoinCount:       art.CoinCount,
		FavCount:        art.FavCount,
		ForwardCount:    art.ForwardCount,
		PublishedAt:     pubAt,
		CreatedAt:       art.CreatedAt.Format("2006-01-02 15:04:05"),
		AuthorName:      upName,
		AuthorAvatar:    avatar,
		FavoritedByMe:   eng.FavoritedByMe,
		CoinedByMe:      eng.CoinedByMe,
		MyCoinAmount:    eng.MyCoinAmount,
		IsAuthor:        viewer > 0 && viewer == art.UserID,
		CommentsClosed:  art.CommentsClosed,
		CommentsCurated: art.CommentsCurated,
	}
}

func articleListItem(art article.Article, authorName string, eng articleEngagement) articleListItemDTO {
	pubAt := ""
	if art.PublishedAt != nil {
		pubAt = art.PublishedAt.Format("2006-01-02 15:04:05")
	}
	return articleListItemDTO{
		ID:              art.ID,
		Title:           art.Title,
		CoverURL:        art.CoverURL,
		Status:          art.Status,
		FailReason:      strings.TrimSpace(art.FailReason),
		ViewCount:       art.ViewCount,
		CommentCount:    art.CommentCount,
		CoinCount:       art.CoinCount,
		FavCount:        art.FavCount,
		ForwardCount:    art.ForwardCount,
		PublishedAt:     pubAt,
		CreatedAt:       art.CreatedAt.Format("2006-01-02 15:04:05"),
		AuthorName:      authorName,
		FavoritedByMe:   eng.FavoritedByMe,
		CommentsClosed:  art.CommentsClosed,
		CommentsCurated: art.CommentsCurated,
	}
}

// GetArticle returns a published article for reading.
// GetArticle godoc
// @Summary      Get article detail
// @Description  Get full article content by ID
// @Tags         Articles
// @Produce      json
// @Param        id path int true "Article ID"
// @Success      200 {object} map[string]interface{}
// @Router       /articles/{id} [get]
func (a *API) GetArticle(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	art, err := a.ArticleSvc.GetPublishedArticle(c.Request.Context(), id)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	var author user.User
	userPub3, _ := a.UserSvc.GetUserPublic(c.Request.Context(), art.UserID)
	if userPub3 != nil {
		author = user.User{ID: userPub3.ID, Username: userPub3.Username, AvatarURL: userPub3.AvatarURL}
	}
	var viewer uint64
	if uid, ok := middleware.UserID(c); ok {
		viewer = uid
	}
	eng := toArticleEngagement(a.ArticleSvc.BatchArticleEngagementByViewer(c.Request.Context(), viewer, []uint64{id})[id])
	resp.OK(c, articleDetailPayload(a, art, &author, eng, viewer))
}

// PostArticleView increments view count (best-effort).
// PostArticleView godoc
// @Summary      Record article view
// @Description  Increment the view count for an article
// @Tags         Articles
// @Produce      json
// @Param        id path int true "Article ID"
// @Success      200 {object} map[string]interface{}
// @Router       /articles/{id}/view [post]
func (a *API) PostArticleView(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if _, ok := loadPublishedArticle(c.Request.Context(), a, id); !ok {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	_ = a.ArticleSvc.IncrementArticleView(c.Request.Context(), id)
	if uid, ok := middleware.UserID(c); ok {
		a.RecordArticleViewHistory(c.Request.Context(), uid, id, "web")
	}
	var art article.Article
	art2, _ := a.ArticleSvc.GetArticleByID(c.Request.Context(), id)
	if art2 != nil {
		art = *art2
	}
	resp.OK(c, articleViewCountResponse{ViewCount: art.ViewCount})
}

// ListUserPublishedArticles lists published articles in a user's space.
// ListUserPublishedArticles godoc
// @Summary      List user articles
// @Description  Get paginated published articles for a user space
// @Tags         Articles
// @Produce      json
// @Param        userId path int true "User ID"
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Page size" default(20)
// @Success      200 {object} map[string]interface{}
// @Router       /space/{userId}/articles [get]
func (a *API) ListUserPublishedArticles(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil || userID == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	limit := parseLimit(c, 20, 50)
	curID, _ := strconv.ParseUint(c.Query("cursor"), 10, 64)
	result, err := a.ArticleSvc.ListUserPublishedArticlesCursor(c.Request.Context(), userID, curID, limit)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	list := result.Items
	hasMore := result.HasMore
	var author user.User
	userPub, _ := a.UserSvc.GetUserPublic(c.Request.Context(), userID)
	if userPub != nil {
		author = user.User{ID: userPub.ID, Username: userPub.Username, AvatarURL: userPub.AvatarURL}
	}
	name := user.DisplayUsername(&author)
	if author.Nickname != "" && !user.IsUserAnonymized(&author) {
		name = strings.TrimSpace(author.Nickname)
	}
	var viewer uint64
	if uid, ok := middleware.UserID(c); ok {
		viewer = uid
	}
	ids := make([]uint64, 0, len(list))
	for _, art := range list {
		ids = append(ids, art.ID)
	}
	engMapSvc := a.ArticleSvc.BatchArticleEngagementByViewer(c.Request.Context(), viewer, ids)
	engMap := make(map[uint64]articleEngagement, len(engMapSvc))
	for id, e := range engMapSvc {
		engMap[id] = toArticleEngagement(e)
	}
	items := make([]articleListItemDTO, 0, len(list))
	for _, art := range list {
		items = append(items, articleListItem(art, name, engMap[art.ID]))
	}
	next := ""
	if hasMore && len(list) > 0 {
		next = strconv.FormatUint(list[len(list)-1].ID, 10)
	}
	resp.OK(c, articleCursorListResponse{Items: items, NextCursor: next})
}
