package handler

import (
	"cakecake/internal/model/article"
	"cakecake/internal/model/comment"
	"cakecake/internal/model/extra"
	"cakecake/internal/model/user"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"cakecake/internal/errcode"
	"cakecake/internal/middleware"
	"cakecake/internal/pkg/coverval"
	"cakecake/internal/pkg/markdown"
	"cakecake/internal/pkg/resp"
	"cakecake/internal/pkg/sensitive"
	"cakecake/internal/service"
)

const (
	maxArticleTitleRunes = 80
	maxArticleBodyRunes  = 100000
)

func (a *API) articleStatusAfterSubmit(publish bool) string {
	if !publish {
		return article.StatusDraft
	}
	if a.Cfg.ArticleReviewRequired {
		return article.StatusPendingReview
	}
	return article.StatusPublished
}

type articlePostJSON struct {
	Title    string   `json:"title"`
	BodyMD   string   `json:"body_md"`
	CoverURL string   `json:"cover_url"`
	Tags     []string `json:"tags"`
	Publish  bool     `json:"publish"`
}

type articlePatchJSON struct {
	Title    *string  `json:"title"`
	BodyMD   *string  `json:"body_md"`
	CoverURL *string  `json:"cover_url"`
	Tags     []string `json:"tags"`
	Publish  *bool    `json:"publish"`
}

type articleEngagement struct {
	FavoritedByMe bool
	CoinedByMe    bool
	MyCoinAmount  int
}

func toArticleEngagement(eng *service.ArticleEngagement) articleEngagement {
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

func articleEngagementByViewer(db *gorm.DB, viewer uint64, ids []uint64) map[uint64]articleEngagement {
	out := make(map[uint64]articleEngagement, len(ids))
	if viewer == 0 || len(ids) == 0 {
		return out
	}
	faved := map[uint64]bool{}
	var favRows []article.ArticleFavorite
	_ = db.Where("user_id = ? AND article_id IN ?", viewer, ids).Find(&favRows).Error
	for i := range favRows {
		faved[favRows[i].ArticleID] = true
	}
	var coinRows []article.ArticleCoin
	_ = db.Where("user_id = ? AND article_id IN ?", viewer, ids).Find(&coinRows).Error
	coinAmt := map[uint64]int{}
	for i := range coinRows {
		amt := coinRows[i].Amount
		if amt < 0 {
			amt = 0
		}
		if amt > 2 {
			amt = 2
		}
		coinAmt[coinRows[i].ArticleID] = amt
	}
	for _, id := range ids {
		amt := coinAmt[id]
		out[id] = articleEngagement{
			FavoritedByMe: faved[id],
			CoinedByMe:    amt > 0,
			MyCoinAmount:  amt,
		}
	}
	return out
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

type articlePlaybackResponse struct {
	CommentsClosed  bool `json:"comments_closed"`
	CommentsCurated bool `json:"comments_curated"`
}

type createArticleResponse struct {
	ID     uint64 `json:"id"`
	Status string `json:"status"`
}

type articleStatusResponse struct {
	ID     uint64 `json:"id"`
	Status string `json:"status"`
}

type articleViewCountResponse struct {
	ViewCount uint64 `json:"view_count"`
}

type myArticleListResponse struct {
	Items      []articleListItemDTO `json:"items"`
	Page       int                  `json:"page"`
	PageSize   int                  `json:"page_size"`
	Total      int64                `json:"total"`
	TotalPages int                  `json:"total_pages"`
	Counts     map[string]int64     `json:"counts"`
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

type articlePlaybackPatch struct {
	CommentsClosed  *bool `json:"comments_closed"`
	CommentsCurated *bool `json:"comments_curated"`
}

// PatchArticlePlayback toggles comment area settings for a published article (owner only).
func (a *API) PatchArticlePlayback(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	art, err := a.ArticleSvc.GetArticleByID(c.Request.Context(), id)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if art.UserID != uid {
		resp.Err(c, http.StatusForbidden, errcode.CodeForbidden)
		return
	}
	if art.Status != article.StatusPublished {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var req articlePlaybackPatch
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if req.CommentsClosed == nil && req.CommentsCurated == nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	updates := map[string]interface{}{}
	if req.CommentsClosed != nil {
		updates["comments_closed"] = *req.CommentsClosed
	}
	if req.CommentsCurated != nil {
		updates["comments_curated"] = *req.CommentsCurated
	}
	if err := a.ArticleSvc.UpdateArticle(c.Request.Context(), id, updates); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	updated, err := a.ArticleSvc.GetArticleByID(c.Request.Context(), id)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, articlePlaybackResponse{
		CommentsClosed:  updated.CommentsClosed,
		CommentsCurated: updated.CommentsCurated,
	})
}

func (a *API) checkArticleSensitive(title, body string) error {
	if a.Sens == nil {
		return nil
	}
	combined := title + "\n" + body
	if err := a.Sens.Check(combined); err != nil {
		return err
	}
	return nil
}

func validateArticleContent(title, bodyMD string, publish bool) bool {
	title = strings.TrimSpace(title)
	bodyMD = strings.TrimSpace(bodyMD)
	if publish {
		if utf8.RuneCountInString(title) < 1 || utf8.RuneCountInString(title) > maxArticleTitleRunes {
			return false
		}
		if utf8.RuneCountInString(bodyMD) < 1 || utf8.RuneCountInString(bodyMD) > maxArticleBodyRunes {
			return false
		}
		return true
	}
	// Draft: at least one of title or body must be filled to save.
	if title == "" && bodyMD == "" {
		return false
	}
	if utf8.RuneCountInString(title) > maxArticleTitleRunes {
		return false
	}
	if utf8.RuneCountInString(bodyMD) > maxArticleBodyRunes {
		return false
	}
	return true
}

// PostArticle creates or publishes a column article.
func (a *API) PostArticle(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	var req articlePostJSON
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	title := strings.TrimSpace(req.Title)
	bodyMD := strings.TrimSpace(req.BodyMD)
	if !validateArticleContent(title, bodyMD, req.Publish) {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if req.Publish {
		if err := a.checkArticleSensitive(title, bodyMD); err != nil {
			if _, ok := err.(sensitive.ErrBlocked); ok {
				resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
				return
			}
		}
	}
	tagsJSON, err := tagsJSONFromStringSlice(req.Tags)
	if err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	status := article.StatusDraft
	var publishedAt *time.Time
	if req.Publish {
		status = a.articleStatusAfterSubmit(true)
		if status == article.StatusPublished {
			now := time.Now()
			publishedAt = &now
		}
	}
	art := article.Article{
		UserID:      uid,
		Title:       title,
		BodyMD:      bodyMD,
		CoverURL:    strings.TrimSpace(req.CoverURL),
		Status:      status,
		TagsJSON:    tagsJSON,
		PublishedAt: publishedAt,
	}
	if err := a.ArticleSvc.CreateArticle(c.Request.Context(), &art); err != nil {
		a.Log.Error("create article", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if art.Status == article.StatusPublished {
		a.esIndexArticle(art.ID)
	} else {
		a.esDeleteArticle(art.ID)
	}
	resp.JSON(c, http.StatusCreated, errcode.CodeSuccess, createArticleResponse{
		ID:     art.ID,
		Status: art.Status,
	})
}

// PutMyArticle updates the current user's article.
func (a *API) PutMyArticle(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	art, err := a.ArticleSvc.GetArticleByID(c.Request.Context(), id)
	if err != nil || art.UserID != uid {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	var req articlePatchJSON
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	updates := map[string]interface{}{}
	publishNow := req.Publish != nil && *req.Publish
	if req.Title != nil {
		t := strings.TrimSpace(*req.Title)
		updates["title"] = t
		art.Title = t
	}
	if req.BodyMD != nil {
		b := strings.TrimSpace(*req.BodyMD)
		updates["body_md"] = b
		art.BodyMD = b
	}
	if !validateArticleContent(art.Title, art.BodyMD, publishNow) {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if req.CoverURL != nil {
		updates["cover_url"] = strings.TrimSpace(*req.CoverURL)
	}
	if req.Tags != nil {
		tagsJSON, err := tagsJSONFromStringSlice(req.Tags)
		if err != nil {
			resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
			return
		}
		updates["tags_json"] = tagsJSON
	}
	if publishNow {
		if err := a.checkArticleSensitive(art.Title, art.BodyMD); err != nil {
			if _, ok := err.(sensitive.ErrBlocked); ok {
				resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
				return
			}
		}
	}
	if req.Publish != nil {
		if *req.Publish {
			if art.Status != article.StatusPublished {
				st := a.articleStatusAfterSubmit(true)
				updates["status"] = st
				if st == article.StatusPublished {
					now := time.Now()
					updates["published_at"] = &now
				} else {
					updates["published_at"] = nil
					updates["fail_reason"] = ""
				}
			}
		} else {
			updates["status"] = article.StatusDraft
			updates["published_at"] = nil
		}
	}
	if len(updates) > 0 {
		if err := a.ArticleSvc.UpdateArticle(c.Request.Context(), id, updates); err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
	}
	art, _ = a.ArticleSvc.GetArticleByID(c.Request.Context(), id)
	if art.Status == article.StatusPublished {
		a.esIndexArticle(art.ID)
	} else {
		a.esDeleteArticle(art.ID)
	}
	resp.OK(c, articleStatusResponse{ID: art.ID, Status: art.Status})
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

func manuscriptArticleStatusToDB(st string) string {
	switch strings.TrimSpace(st) {
	case article.StatusDraft:
		return article.StatusDraft
	case article.StatusPassed, article.StatusPublished:
		return article.StatusPublished
	case article.StatusProcessing:
		return article.StatusPendingReview
	case article.StatusRejected, article.StatusFailed:
		return article.StatusRejected
	default:
		return ""
	}
}

func orderClauseForMyArticles(sort string) string {
	switch strings.TrimSpace(sort) {
	case "view":
		return "view_count DESC, id DESC"
	case "reply":
		return "comment_count DESC, id DESC"
	case "like":
		return "coin_count DESC, id DESC"
	case "fav":
		return "fav_count DESC, id DESC"
	default:
		return "COALESCE(published_at, created_at) DESC, id DESC"
	}
}

func (a *API) countMyArticlesByStatus(ctx context.Context, uid uint64) map[string]int64 {
	type row struct {
		Status string
		N      int64
	}
	var rows []row
	_ = a.ArticleSvc.CountArticlesByStatus(ctx, uid)
	out := map[string]int64{
		"draft":      0,
		"processing": 0,
		"passed":     0,
		"rejected":   0,
	}
	for _, r := range rows {
		switch r.Status {
		case article.StatusDraft:
			out["draft"] = r.N
		case article.StatusPublished:
			out["passed"] += r.N
		case article.StatusPendingReview:
			out["processing"] += r.N
		case article.StatusRejected:
			out["rejected"] = r.N
		}
	}
	dynN, _ := a.DynamicSvc.CountUserDynamics(ctx, uid)
	out["dynamics"] = dynN
	return out
}

// ListMyArticles lists the current user's column articles (content management).
// Query: page, page_size, sort(time|view|reply|like|fav), status(all|draft|passed|processing|rejected), q(title).
func (a *API) ListMyArticles(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	page, pageSize := parsePagination(c, 10)
	sortKey := strings.TrimSpace(c.DefaultQuery("sort", "time"))
	statusQ := strings.TrimSpace(c.Query("status"))
	titleQ := strings.TrimSpace(c.Query("q"))

	result, err := a.ArticleSvc.ListMyArticlesPage(c.Request.Context(), uid, page, pageSize, statusQ, titleQ, sortKey)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	list := result.Items
	total := result.Total
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	items := make([]articleListItemDTO, 0, len(list))
	for _, art := range list {
		items = append(items, articleListItem(art, "", articleEngagement{}))
	}
	resp.OK(c, myArticleListResponse{
		Items:      items,
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
		Counts:     a.countMyArticlesByStatus(c.Request.Context(), uid),
	})
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

// deleteArticleCascade removes one article and related engagement rows.
func deleteArticleCascade(tx *gorm.DB, articleID uint64) error {
	var cids []uint64
	if err := tx.Model(&comment.ArticleComment{}).Where("article_id = ?", articleID).Pluck("id", &cids).Error; err != nil {
		return err
	}
	if len(cids) > 0 {
		if err := tx.Where("comment_id IN ?", cids).Delete(&comment.ArticleCommentLike{}).Error; err != nil {
			return err
		}
		if err := tx.Where("comment_id IN ?", cids).Delete(&comment.ArticleCommentDislike{}).Error; err != nil {
			return err
		}
		if err := tx.Where("id IN ?", cids).Delete(&comment.ArticleComment{}).Error; err != nil {
			return err
		}
	}
	if err := tx.Where("article_id = ?", articleID).Delete(&article.ArticleFavorite{}).Error; err != nil {
		return err
	}
	if err := tx.Where("article_id = ?", articleID).Delete(&article.ArticleCoin{}).Error; err != nil {
		return err
	}
	if err := tx.Where("article_id = ?", articleID).Delete(&extra.ArticleViewHistory{}).Error; err != nil {
		return err
	}
	return tx.Where("id = ?", articleID).Delete(&article.Article{}).Error
}

// DeleteMyArticle removes an article owned by the current user.
func (a *API) DeleteMyArticle(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	art, err := a.ArticleSvc.GetArticleByID(c.Request.Context(), id)
	if err != nil || art.UserID != uid {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if err := a.ArticleSvc.DeleteArticle(c.Request.Context(), id); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	a.StorageSvc.PurgeArticle(*art)
	a.esDeleteArticle(id)
	resp.OK(c, okResponse{OK: true})
}

// UpdateArticleCover uploads/replaces article cover on OSS (same flow as video cover).
func (a *API) UpdateArticleCover(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	art, err := a.ArticleSvc.GetArticleByID(c.Request.Context(), id)
	if err != nil || art.UserID != uid {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if err := c.Request.ParseMultipartForm(12 << 20); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	fh, err := c.FormFile("cover")
	if err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if code := coverval.ValidateCoverHeader(fh); code != 0 {
		resp.Err(c, http.StatusBadRequest, code)
		return
	}
	if !a.StorageSvc.Enabled() {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if err := os.MkdirAll(a.Cfg.TempUploadDir, 0o755); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	tmp := filepath.Join(a.Cfg.TempUploadDir, uuid.NewString()+filepath.Ext(fh.Filename))
	if err := saveUploadedFile(fh, tmp); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	defer os.Remove(tmp)
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(fh.Filename)), ".")
	if ext == "jpeg" {
		ext = "jpg"
	}
	key := fmt.Sprintf("article-covers/%d.%s", art.ID, ext)
	if err := a.StorageSvc.UploadFile(key, tmp); err != nil {
		a.Log.Error("oss article cover upload", zap.Error(err), zap.Uint64("article_id", art.ID))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	url := a.Cfg.OSSObjectURL(key)
	if err := a.ArticleSvc.UpdateArticle(c.Request.Context(), id, map[string]interface{}{"cover_url": url}); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, imageURLResponse{ImageURL: url})
}
