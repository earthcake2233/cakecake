package handler

import (
	"cakecake/internal/errcode"
	"cakecake/internal/middleware"
	"cakecake/internal/model/article"
	"cakecake/internal/model/comment"
	"cakecake/internal/model/extra"
	"cakecake/internal/pkg/coverval"
	"cakecake/internal/pkg/resp"
	"cakecake/internal/pkg/sensitive"
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"cakecake/internal/pkg/dbtx"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
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

type myArticleListResponse struct {
	Items      []articleListItemDTO `json:"items"`
	Page       int                  `json:"page"`
	PageSize   int                  `json:"page_size"`
	Total      int64                `json:"total"`
	TotalPages int                  `json:"total_pages"`
	Counts     map[string]int64     `json:"counts"`
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

// deleteArticleCascade removes one article and related engagement rows.
func deleteArticleCascade(tx dbtx.Tx, articleID uint64) error {
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
