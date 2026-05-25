package handler

import (
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

	"minibili/internal/errcode"
	"minibili/internal/middleware"
	"minibili/internal/model"
	"minibili/internal/pkg/coverval"
	"minibili/internal/pkg/markdown"
	"minibili/internal/pkg/resp"
	"minibili/internal/pkg/sensitive"
)

const (
	articleStatusDraft          = "draft"
	articleStatusPublished      = "published"
	articleStatusPendingReview  = "pending_review"
	articleStatusRejected       = "rejected"
	maxArticleTitleRunes        = 80
	maxArticleBodyRunes         = 100000
)

func (a *API) articleStatusAfterSubmit(publish bool) string {
	if !publish {
		return articleStatusDraft
	}
	if a.Cfg.ArticleReviewRequired {
		return articleStatusPendingReview
	}
	return articleStatusPublished
}

type articlePostJSON struct {
	Title   string   `json:"title"`
	BodyMD  string   `json:"body_md"`
	CoverURL string  `json:"cover_url"`
	Tags    []string `json:"tags"`
	Publish bool     `json:"publish"`
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

func loadPublishedArticle(a *API, id uint64) (model.Article, bool) {
	var art model.Article
	if err := a.DB.First(&art, id).Error; err != nil || art.Status != articleStatusPublished {
		return model.Article{}, false
	}
	return art, true
}

func articleEngagementByViewer(db *gorm.DB, viewer uint64, ids []uint64) map[uint64]articleEngagement {
	out := make(map[uint64]articleEngagement, len(ids))
	if viewer == 0 || len(ids) == 0 {
		return out
	}
	faved := map[uint64]bool{}
	var favRows []model.ArticleFavorite
	_ = db.Where("user_id = ? AND article_id IN ?", viewer, ids).Find(&favRows).Error
	for i := range favRows {
		faved[favRows[i].ArticleID] = true
	}
	var coinRows []model.ArticleCoin
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

func articleDetailPayload(a *API, art *model.Article, author *model.User, eng articleEngagement, viewer uint64) gin.H {
	bodyHTML, toc, _ := markdown.Render(art.BodyMD)
	upName := ""
	avatar := ""
	if author != nil {
		upName = model.DisplayUsername(author)
		if author.Nickname != "" && !model.IsUserAnonymized(author) {
			upName = strings.TrimSpace(author.Nickname)
		}
		avatar = uploaderAvatarForAPI(author)
	}
	pubAt := ""
	if art.PublishedAt != nil {
		pubAt = art.PublishedAt.Format("2006-01-02 15:04:05")
	}
	return gin.H{
		"id":              art.ID,
		"cv_id":           art.ID,
		"user_id":         art.UserID,
		"title":           art.Title,
		"cover_url":       art.CoverURL,
		"body_md":         art.BodyMD,
		"body_html":       bodyHTML,
		"toc":             toc,
		"tags":            parseArticleTagsJSON(art.TagsJSON),
		"status":          art.Status,
		"fail_reason":     strings.TrimSpace(art.FailReason),
		"view_count":      art.ViewCount,
		"comment_count":   art.CommentCount,
		"coin_count":      art.CoinCount,
		"fav_count":       art.FavCount,
		"forward_count":   art.ForwardCount,
		"published_at":    pubAt,
		"created_at":      art.CreatedAt.Format("2006-01-02 15:04:05"),
		"author_name":     upName,
		"author_avatar":   avatar,
		"favorited_by_me":  eng.FavoritedByMe,
		"coined_by_me":     eng.CoinedByMe,
		"my_coin_amount":   eng.MyCoinAmount,
		"is_author":        viewer > 0 && viewer == art.UserID,
		"comments_closed":  art.CommentsClosed,
		"comments_curated": art.CommentsCurated,
	}
}

func articleListItem(art model.Article, authorName string, eng articleEngagement) gin.H {
	pubAt := ""
	if art.PublishedAt != nil {
		pubAt = art.PublishedAt.Format("2006-01-02 15:04:05")
	}
	return gin.H{
		"id":              art.ID,
		"title":           art.Title,
		"cover_url":       art.CoverURL,
		"status":          art.Status,
		"fail_reason":     strings.TrimSpace(art.FailReason),
		"view_count":      art.ViewCount,
		"comment_count":   art.CommentCount,
		"coin_count":      art.CoinCount,
		"fav_count":       art.FavCount,
		"forward_count":   art.ForwardCount,
		"published_at":    pubAt,
		"created_at":      art.CreatedAt.Format("2006-01-02 15:04:05"),
		"author_name":      authorName,
		"favorited_by_me":  eng.FavoritedByMe,
		"comments_closed":  art.CommentsClosed,
		"comments_curated": art.CommentsCurated,
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
	var art model.Article
	if err := a.DB.First(&art, id).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if art.UserID != uid {
		resp.Err(c, http.StatusForbidden, errcode.CodeForbidden)
		return
	}
	if art.Status != articleStatusPublished {
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
	if err := a.DB.Model(&art).Updates(updates).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if err := a.DB.First(&art, id).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{
		"comments_closed":  art.CommentsClosed,
		"comments_curated": art.CommentsCurated,
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
	// 草稿：标题与正文至少填一项即可保存
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

// PostArticle creates or publishes a column article (创作中心专栏投稿).
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
	status := articleStatusDraft
	var publishedAt *time.Time
	if req.Publish {
		status = a.articleStatusAfterSubmit(true)
		if status == articleStatusPublished {
			now := time.Now()
			publishedAt = &now
		}
	}
	art := model.Article{
		UserID:    uid,
		Title:     title,
		BodyMD:    bodyMD,
		CoverURL:  strings.TrimSpace(req.CoverURL),
		Status:    status,
		TagsJSON:  tagsJSON,
		PublishedAt: publishedAt,
	}
	if err := a.DB.Create(&art).Error; err != nil {
		a.Log.Error("create article", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if art.Status == articleStatusPublished {
		a.esIndexArticle(art.ID)
	} else {
		a.esDeleteArticle(art.ID)
	}
	resp.JSON(c, http.StatusCreated, errcode.CodeSuccess, gin.H{
		"id":     art.ID,
		"status": art.Status,
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
	var art model.Article
	if err := a.DB.First(&art, id).Error; err != nil || art.UserID != uid {
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
			if art.Status != articleStatusPublished {
				st := a.articleStatusAfterSubmit(true)
				updates["status"] = st
				if st == articleStatusPublished {
					now := time.Now()
					updates["published_at"] = &now
				} else {
					updates["published_at"] = nil
					updates["fail_reason"] = ""
				}
			}
		} else {
			updates["status"] = articleStatusDraft
			updates["published_at"] = nil
		}
	}
	if len(updates) > 0 {
		if err := a.DB.Model(&art).Updates(updates).Error; err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
	}
	_ = a.DB.First(&art, id).Error
	if art.Status == articleStatusPublished {
		a.esIndexArticle(art.ID)
	} else {
		a.esDeleteArticle(art.ID)
	}
	resp.OK(c, gin.H{"id": art.ID, "status": art.Status})
}

// GetArticle returns a published article for reading.
func (a *API) GetArticle(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	art, ok := loadPublishedArticle(a, id)
	if !ok {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	var author model.User
	_ = a.DB.First(&author, art.UserID).Error
	var viewer uint64
	if uid, ok := middleware.UserID(c); ok {
		viewer = uid
	}
	eng := articleEngagementByViewer(a.DB, viewer, []uint64{id})[id]
	resp.OK(c, articleDetailPayload(a, &art, &author, eng, viewer))
}

// PostArticleView increments view count (best-effort).
func (a *API) PostArticleView(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if _, ok := loadPublishedArticle(a, id); !ok {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	_ = a.DB.Model(&model.Article{}).Where("id = ?", id).
		UpdateColumn("view_count", gorm.Expr("view_count + ?", 1)).Error
	if uid, ok := middleware.UserID(c); ok {
		a.RecordArticleViewHistory(uid, id, "web")
	}
	var art model.Article
	_ = a.DB.First(&art, id).Error
	resp.OK(c, gin.H{"view_count": art.ViewCount})
}

func manuscriptArticleStatusToDB(st string) string {
	switch strings.TrimSpace(st) {
	case "draft":
		return "draft"
	case "passed", "published":
		return "published"
	case "processing":
		return "pending_review"
	case "rejected", "failed":
		return "rejected"
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

func (a *API) countMyArticlesByStatus(uid uint64) gin.H {
	type row struct {
		Status string
		N      int64
	}
	var rows []row
	_ = a.DB.Model(&model.Article{}).
		Select("status, COUNT(*) AS n").
		Where("user_id = ?", uid).
		Group("status").
		Scan(&rows).Error
	out := gin.H{
		"draft":      int64(0),
		"processing": int64(0),
		"passed":     int64(0),
		"rejected":   int64(0),
	}
	for _, r := range rows {
		switch r.Status {
		case "draft":
			out["draft"] = r.N
		case "published":
			out["passed"] = out["passed"].(int64) + r.N
		case "pending_review":
			out["processing"] = out["processing"].(int64) + r.N
		case "rejected":
			out["rejected"] = r.N
		}
	}
	var dynN int64
	_ = a.DB.Model(&model.UserDynamic{}).Where("user_id = ?", uid).Count(&dynN).Error
	out["dynamics"] = dynN
	return out
}

// ListMyArticles lists the current user's column articles (稿件管理).
// Query: page, page_size, sort(time|view|reply|like|fav), status(all|draft|passed|processing|rejected), q(title).
func (a *API) ListMyArticles(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}
	sortKey := strings.TrimSpace(c.DefaultQuery("sort", "time"))
	statusQ := strings.TrimSpace(c.Query("status"))
	titleQ := strings.TrimSpace(c.Query("q"))

	base := a.DB.Model(&model.Article{}).Where("user_id = ?", uid)
	filtered := base
	if dbSt := manuscriptArticleStatusToDB(statusQ); dbSt != "" {
		filtered = filtered.Where("status = ?", dbSt)
	} else {
		filtered = filtered.Where("status <> ?", articleStatusDraft)
	}
	if titleQ != "" {
		filtered = filtered.Where("title LIKE ?", "%"+titleQ+"%")
	}
	var total int64
	if err := filtered.Count(&total).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	offset := (page - 1) * pageSize
	var list []model.Article
	if err := filtered.Order(orderClauseForMyArticles(sortKey)).
		Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	items := make([]gin.H, 0, len(list))
	for _, art := range list {
		items = append(items, articleListItem(art, "", articleEngagement{}))
	}
	resp.OK(c, gin.H{
		"items":       items,
		"page":        page,
		"page_size":   pageSize,
		"total":       total,
		"total_pages": totalPages,
		"counts":      a.countMyArticlesByStatus(uid),
	})
}

// ListUserPublishedArticles lists published articles in a user's space.
func (a *API) ListUserPublishedArticles(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil || userID == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	limit := 20
	if s := c.Query("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 50 {
			limit = n
		}
	}
	curID, _ := strconv.ParseUint(c.Query("cursor"), 10, 64)
	q := a.DB.Model(&model.Article{}).
		Where("user_id = ? AND status = ?", userID, articleStatusPublished)
	if curID > 0 {
		q = q.Where("id < ?", curID)
	}
	var list []model.Article
	if err := q.Order("id DESC").Limit(limit + 1).Find(&list).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	hasMore := len(list) > limit
	if hasMore {
		list = list[:limit]
	}
	var author model.User
	_ = a.DB.First(&author, userID).Error
	name := model.DisplayUsername(&author)
	if author.Nickname != "" && !model.IsUserAnonymized(&author) {
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
	engMap := articleEngagementByViewer(a.DB, viewer, ids)
	items := make([]gin.H, 0, len(list))
	for _, art := range list {
		items = append(items, articleListItem(art, name, engMap[art.ID]))
	}
	next := ""
	if hasMore && len(list) > 0 {
		next = strconv.FormatUint(list[len(list)-1].ID, 10)
	}
	resp.OK(c, gin.H{"items": items, "next_cursor": next})
}

// deleteArticleCascade removes one article and related engagement rows.
func deleteArticleCascade(tx *gorm.DB, articleID uint64) error {
	var cids []uint64
	if err := tx.Model(&model.ArticleComment{}).Where("article_id = ?", articleID).Pluck("id", &cids).Error; err != nil {
		return err
	}
	if len(cids) > 0 {
		if err := tx.Where("comment_id IN ?", cids).Delete(&model.ArticleCommentLike{}).Error; err != nil {
			return err
		}
		if err := tx.Where("comment_id IN ?", cids).Delete(&model.ArticleCommentDislike{}).Error; err != nil {
			return err
		}
		if err := tx.Where("id IN ?", cids).Delete(&model.ArticleComment{}).Error; err != nil {
			return err
		}
	}
	if err := tx.Where("article_id = ?", articleID).Delete(&model.ArticleFavorite{}).Error; err != nil {
		return err
	}
	if err := tx.Where("article_id = ?", articleID).Delete(&model.ArticleCoin{}).Error; err != nil {
		return err
	}
	if err := tx.Where("article_id = ?", articleID).Delete(&model.ArticleViewHistory{}).Error; err != nil {
		return err
	}
	return tx.Where("id = ?", articleID).Delete(&model.Article{}).Error
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
	var art model.Article
	if err := a.DB.First(&art, id).Error; err != nil || art.UserID != uid {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if err := a.DB.Transaction(func(tx *gorm.DB) error {
		return deleteArticleCascade(tx, id)
	}); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	purgeArticleOSSObjects(a.Cfg, a.OSS, a.Log, art)
	a.esDeleteArticle(id)
	resp.OK(c, gin.H{"ok": true})
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
	var art model.Article
	if err := a.DB.First(&art, id).Error; err != nil || art.UserID != uid {
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
	if a.OSS == nil {
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
	if err := a.OSS.UploadFile(key, tmp); err != nil {
		a.Log.Error("oss article cover upload", zap.Error(err), zap.Uint64("article_id", art.ID))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	url := a.Cfg.OSSObjectURL(key)
	if err := a.DB.Model(&art).Update("cover_url", url).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"cover_url": url})
}
