package handler

import (
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"minibili/internal/errcode"
	"minibili/internal/middleware"
	"minibili/internal/model"
	"minibili/internal/pkg/resp"
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
	if page < 1 {
		page = 1
	}
	pageSize := queryIntDefault(c.Query("page_size"), 10)
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 50 {
		pageSize = 50
	}
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

	base := a.DB.Model(&model.Comment{}).
		Joins("INNER JOIN videos ON videos.id = comments.video_id AND videos.user_id = ?", uid)
	if pending {
		switch pendingStatus {
		case "ignored":
			// 已忽略：未公开且已标记忽略（不要求稿件仍开启精选）
			base = base.Where("comments.curated_ignored = ?", true).
				Where("comments.approved = ?", false)
		default:
			base = base.Where("videos.comments_curated = ?", true).
				Where("comments.approved = ?", false)
			switch pendingStatus {
			case "all":
				// 全部待精选：未处理 + 已忽略
			default: // unprocessed
				base = base.Where("comments.curated_ignored = ?", false)
			}
		}
		switch pendingScope {
		case "root":
			base = base.Where("comments.parent_id = ?", 0)
		case "reply":
			base = base.Where("comments.parent_id > ?", 0)
		}
	} else {
		base = base.Where("comments.approved = ?", true)
	}
	if filterVideoID > 0 {
		var owned model.Video
		if err := a.DB.Where("id = ? AND user_id = ?", filterVideoID, uid).First(&owned).Error; err != nil {
			resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
			return
		}
		base = base.Where("comments.video_id = ?", filterVideoID)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		base = base.Where("comments.content LIKE ? OR videos.title LIKE ?", like, like)
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if total > creatorCommentsMaxTotal {
		total = creatorCommentsMaxTotal
	}

	order := "comments.created_at DESC, comments.id DESC"
	switch sortKey {
	case "earliest":
		order = "comments.created_at ASC, comments.id ASC"
	case "likes":
		order = "comments.like_count DESC, comments.id DESC"
	case "replies":
		order = "(SELECT COUNT(*) FROM comments AS r WHERE r.parent_id = comments.id) DESC, comments.id DESC"
	}

	var list []model.Comment
	offset := (page - 1) * pageSize
	if err := base.Order(order).Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}

	videoIDs := make([]uint64, 0, len(list))
	userIDs := make([]uint64, 0, len(list))
	parentIDs := make([]uint64, 0)
	commentIDs := make([]uint64, len(list))
	for _, cm := range list {
		commentIDs = append(commentIDs, cm.ID)
		videoIDs = append(videoIDs, cm.VideoID)
		userIDs = append(userIDs, cm.UserID)
		if cm.ParentID > 0 {
			parentIDs = append(parentIDs, cm.ParentID)
		}
	}

	videos := map[uint64]model.Video{}
	if len(videoIDs) > 0 {
		var vlist []model.Video
		_ = a.DB.Where("id IN ?", videoIDs).Find(&vlist).Error
		for i := range vlist {
			videos[vlist[i].ID] = vlist[i]
		}
	}

	names := map[uint64]string{}
	avatars := map[uint64]string{}
	if len(userIDs) > 0 {
		var users []model.User
		_ = a.DB.Where("id IN ?", userIDs).Find(&users).Error
		for i := range users {
			usr := &users[i]
			names[usr.ID] = model.DisplayUsername(usr)
			avatars[usr.ID] = uploaderAvatarForAPI(usr)
		}
	}

	parents := map[uint64]model.Comment{}
	if len(parentIDs) > 0 {
		var plist []model.Comment
		_ = a.DB.Where("id IN ?", parentIDs).Find(&plist).Error
		for i := range plist {
			parents[plist[i].ID] = plist[i]
			userIDs = append(userIDs, plist[i].UserID)
		}
		var users []model.User
		_ = a.DB.Where("id IN ?", userIDs).Find(&users).Error
		for i := range users {
			usr := &users[i]
			names[usr.ID] = model.DisplayUsername(usr)
			avatars[usr.ID] = uploaderAvatarForAPI(usr)
		}
	}

	replyCounts := batchCommentReplyCounts(a.DB, commentIDs)

	likedByViewer := map[uint64]bool{}
	if viewerID, ok := middleware.UserID(c); ok && viewerID > 0 && len(commentIDs) > 0 {
		var likes []model.CommentLike
		_ = a.DB.Where("user_id = ? AND comment_id IN ?", viewerID, commentIDs).Find(&likes).Error
		for _, lk := range likes {
			likedByViewer[lk.CommentID] = true
		}
	}

	items := make([]gin.H, 0, len(list))
	for _, cm := range list {
		v := videos[cm.VideoID]
		item := gin.H{
			"id":          cm.ID,
			"video_id":    cm.VideoID,
			"user_id":     cm.UserID,
			"username":    names[cm.UserID],
			"avatar_url":  avatars[cm.UserID],
			"parent_id":   cm.ParentID,
			"content":     cm.Content,
			"like_count":   cm.LikeCount,
			"liked_by_me":  likedByViewer[cm.ID],
			"reply_count":  replyCounts[cm.ID],
			"created_at":      cm.CreatedAt.Format("2006-01-02 15:04:05"),
			"approved":        cm.Approved,
			"curated_ignored": cm.CuratedIgnored,
			"video": gin.H{
				"id":        v.ID,
				"title":     v.Title,
				"cover_url": v.CoverURL,
			},
		}
		if cm.ParentID > 0 {
			if p, ok := parents[cm.ParentID]; ok {
				item["parent"] = gin.H{
					"id":       p.ID,
					"user_id":  p.UserID,
					"username": names[p.UserID],
					"content":  previewCommentContent(p.Content, 80),
				}
			}
		}
		items = append(items, item)
	}

	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(pageSize)))
	}
	resp.OK(c, gin.H{
		"items":       items,
		"page":        page,
		"page_size":   pageSize,
		"total":       total,
		"total_pages": totalPages,
	})
}

func batchCommentReplyCounts(db *gorm.DB, ids []uint64) map[uint64]uint64 {
	out := make(map[uint64]uint64, len(ids))
	if len(ids) == 0 {
		return out
	}
	type row struct {
		ParentID uint64
		C        int64
	}
	var rows []row
	_ = db.Model(&model.Comment{}).
		Select("parent_id, COUNT(*) AS c").
		Where("parent_id IN ?", ids).
		Group("parent_id").
		Scan(&rows).Error
	for _, r := range rows {
		if r.C > 0 {
			out[r.ParentID] = uint64(r.C)
		}
	}
	return out
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
	if page < 1 {
		page = 1
	}
	pageSize := queryIntDefault(c.Query("page_size"), 10)
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 50 {
		pageSize = 50
	}
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

	base := a.DB.Model(&model.ArticleComment{}).
		Joins("INNER JOIN articles ON articles.id = article_comments.article_id AND articles.user_id = ? AND articles.status = ?", uid, articleStatusPublished)
	if pending {
		switch pendingStatus {
		case "ignored":
			base = base.Where("article_comments.curated_ignored = ?", true).
				Where("article_comments.approved = ?", false)
		default:
			base = base.Where("articles.comments_curated = ?", true).
				Where("article_comments.approved = ?", false)
			switch pendingStatus {
			case "all":
			default: // unprocessed
				base = base.Where("article_comments.curated_ignored = ?", false)
			}
		}
		switch pendingScope {
		case "root":
			base = base.Where("article_comments.parent_id = ?", 0)
		case "reply":
			base = base.Where("article_comments.parent_id > ?", 0)
		}
	} else {
		base = base.Where("article_comments.approved = ?", true)
	}
	if filterArticleID > 0 {
		var owned model.Article
		if err := a.DB.Where("id = ? AND user_id = ?", filterArticleID, uid).First(&owned).Error; err != nil {
			resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
			return
		}
		base = base.Where("article_comments.article_id = ?", filterArticleID)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		base = base.Where("article_comments.content LIKE ? OR articles.title LIKE ?", like, like)
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if total > creatorCommentsMaxTotal {
		total = creatorCommentsMaxTotal
	}

	order := "article_comments.created_at DESC, article_comments.id DESC"
	switch sortKey {
	case "earliest":
		order = "article_comments.created_at ASC, article_comments.id ASC"
	case "likes":
		order = "article_comments.like_count DESC, article_comments.id DESC"
	case "replies":
		order = "(SELECT COUNT(*) FROM article_comments AS r WHERE r.parent_id = article_comments.id) DESC, article_comments.id DESC"
	}

	var list []model.ArticleComment
	offset := (page - 1) * pageSize
	if err := base.Order(order).Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}

	articleIDs := make([]uint64, 0, len(list))
	userIDs := make([]uint64, 0, len(list))
	parentIDs := make([]uint64, 0)
	commentIDs := make([]uint64, len(list))
	for _, cm := range list {
		commentIDs = append(commentIDs, cm.ID)
		articleIDs = append(articleIDs, cm.ArticleID)
		userIDs = append(userIDs, cm.UserID)
		if cm.ParentID > 0 {
			parentIDs = append(parentIDs, cm.ParentID)
		}
	}

	articles := map[uint64]model.Article{}
	if len(articleIDs) > 0 {
		var alist []model.Article
		_ = a.DB.Where("id IN ?", articleIDs).Find(&alist).Error
		for i := range alist {
			articles[alist[i].ID] = alist[i]
		}
	}

	names := map[uint64]string{}
	avatars := map[uint64]string{}
	if len(userIDs) > 0 {
		var users []model.User
		_ = a.DB.Where("id IN ?", userIDs).Find(&users).Error
		for i := range users {
			usr := &users[i]
			names[usr.ID] = model.DisplayUsername(usr)
			avatars[usr.ID] = uploaderAvatarForAPI(usr)
		}
	}

	parents := map[uint64]model.ArticleComment{}
	if len(parentIDs) > 0 {
		var plist []model.ArticleComment
		_ = a.DB.Where("id IN ?", parentIDs).Find(&plist).Error
		for i := range plist {
			parents[plist[i].ID] = plist[i]
			userIDs = append(userIDs, plist[i].UserID)
		}
		var users []model.User
		_ = a.DB.Where("id IN ?", userIDs).Find(&users).Error
		for i := range users {
			usr := &users[i]
			names[usr.ID] = model.DisplayUsername(usr)
			avatars[usr.ID] = uploaderAvatarForAPI(usr)
		}
	}

	replyCounts := batchArticleCommentReplyCounts(a.DB, commentIDs)

	likedByViewer := map[uint64]bool{}
	if viewerID, ok := middleware.UserID(c); ok && viewerID > 0 && len(commentIDs) > 0 {
		var likes []model.ArticleCommentLike
		_ = a.DB.Where("user_id = ? AND comment_id IN ?", viewerID, commentIDs).Find(&likes).Error
		for _, lk := range likes {
			likedByViewer[lk.CommentID] = true
		}
	}

	items := make([]gin.H, 0, len(list))
	for _, cm := range list {
		art := articles[cm.ArticleID]
		item := gin.H{
			"id":              cm.ID,
			"article_id":      cm.ArticleID,
			"user_id":         cm.UserID,
			"username":        names[cm.UserID],
			"avatar_url":      avatars[cm.UserID],
			"parent_id":       cm.ParentID,
			"content":         cm.Content,
			"like_count":      cm.LikeCount,
			"liked_by_me":     likedByViewer[cm.ID],
			"reply_count":     replyCounts[cm.ID],
			"created_at":      cm.CreatedAt.Format("2006-01-02 15:04:05"),
			"approved":        cm.Approved,
			"curated_ignored": cm.CuratedIgnored,
			"article": gin.H{
				"id":        art.ID,
				"title":     art.Title,
				"cover_url": art.CoverURL,
			},
		}
		if cm.ParentID > 0 {
			if p, ok := parents[cm.ParentID]; ok {
				item["parent"] = gin.H{
					"id":       p.ID,
					"user_id":  p.UserID,
					"username": names[p.UserID],
					"content":  previewCommentContent(p.Content, 80),
				}
			}
		}
		items = append(items, item)
	}

	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(pageSize)))
	}
	resp.OK(c, gin.H{
		"items":       items,
		"page":        page,
		"page_size":   pageSize,
		"total":       total,
		"total_pages": totalPages,
	})
}

func dynamicDisplayTitle(d *model.UserDynamic) string {
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

func dynamicCoverURL(d *model.UserDynamic) string {
	if d == nil {
		return ""
	}
	imgs := parseDynamicImagesJSON(d.ImagesJSON)
	if len(imgs) > 0 {
		return imgs[0]
	}
	return ""
}

func batchDynamicCommentReplyCounts(db *gorm.DB, ids []uint64) map[uint64]uint64 {
	out := make(map[uint64]uint64, len(ids))
	if len(ids) == 0 {
		return out
	}
	type row struct {
		ParentID uint64
		C        int64
	}
	var rows []row
	_ = db.Model(&model.DynamicComment{}).
		Select("parent_id, COUNT(*) AS c").
		Where("parent_id IN ?", ids).
		Group("parent_id").
		Scan(&rows).Error
	for _, r := range rows {
		if r.C > 0 {
			out[r.ParentID] = uint64(r.C)
		}
	}
	return out
}

// listCreatorDynamicComments lists comments on the authenticated user's image/text dynamics.
func (a *API) listCreatorDynamicComments(c *gin.Context, uid uint64) {
	page := queryIntDefault(c.Query("page"), 1)
	if page < 1 {
		page = 1
	}
	pageSize := queryIntDefault(c.Query("page_size"), 10)
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 50 {
		pageSize = 50
	}
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

	base := a.DB.Model(&model.DynamicComment{}).
		Joins("INNER JOIN user_dynamics ON user_dynamics.id = dynamic_comments.dynamic_id AND user_dynamics.user_id = ?", uid)
	if pending {
		switch pendingStatus {
		case "ignored":
			base = base.Where("dynamic_comments.curated_ignored = ?", true).
				Where("dynamic_comments.approved = ?", false)
		default:
			base = base.Where("user_dynamics.comments_curated = ?", true).
				Where("dynamic_comments.approved = ?", false)
			switch pendingStatus {
			case "all":
			default: // unprocessed
				base = base.Where("dynamic_comments.curated_ignored = ?", false)
			}
		}
		switch pendingScope {
		case "root":
			base = base.Where("dynamic_comments.parent_id = ?", 0)
		case "reply":
			base = base.Where("dynamic_comments.parent_id > ?", 0)
		}
	} else {
		base = base.Where("dynamic_comments.approved = ?", true)
	}
	if filterDynamicID > 0 {
		var owned model.UserDynamic
		if err := a.DB.Where("id = ? AND user_id = ?", filterDynamicID, uid).First(&owned).Error; err != nil {
			resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
			return
		}
		base = base.Where("dynamic_comments.dynamic_id = ?", filterDynamicID)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		base = base.Where(
			"dynamic_comments.content LIKE ? OR user_dynamics.title LIKE ? OR user_dynamics.content LIKE ?",
			like, like, like,
		)
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if total > creatorCommentsMaxTotal {
		total = creatorCommentsMaxTotal
	}

	order := "dynamic_comments.created_at DESC, dynamic_comments.id DESC"
	switch sortKey {
	case "earliest":
		order = "dynamic_comments.created_at ASC, dynamic_comments.id ASC"
	case "likes":
		order = "dynamic_comments.like_count DESC, dynamic_comments.id DESC"
	case "replies":
		order = "(SELECT COUNT(*) FROM dynamic_comments AS r WHERE r.parent_id = dynamic_comments.id) DESC, dynamic_comments.id DESC"
	}

	var list []model.DynamicComment
	offset := (page - 1) * pageSize
	if err := base.Order(order).Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}

	dynamicIDs := make([]uint64, 0, len(list))
	userIDs := make([]uint64, 0, len(list))
	parentIDs := make([]uint64, 0)
	commentIDs := make([]uint64, len(list))
	for _, cm := range list {
		commentIDs = append(commentIDs, cm.ID)
		dynamicIDs = append(dynamicIDs, cm.DynamicID)
		userIDs = append(userIDs, cm.UserID)
		if cm.ParentID > 0 {
			parentIDs = append(parentIDs, cm.ParentID)
		}
	}

	dynamics := map[uint64]model.UserDynamic{}
	if len(dynamicIDs) > 0 {
		var dlist []model.UserDynamic
		_ = a.DB.Where("id IN ?", dynamicIDs).Find(&dlist).Error
		for i := range dlist {
			dynamics[dlist[i].ID] = dlist[i]
		}
	}

	names := map[uint64]string{}
	avatars := map[uint64]string{}
	if len(userIDs) > 0 {
		var users []model.User
		_ = a.DB.Where("id IN ?", userIDs).Find(&users).Error
		for i := range users {
			usr := &users[i]
			names[usr.ID] = model.DisplayUsername(usr)
			avatars[usr.ID] = uploaderAvatarForAPI(usr)
		}
	}

	parents := map[uint64]model.DynamicComment{}
	if len(parentIDs) > 0 {
		var plist []model.DynamicComment
		_ = a.DB.Where("id IN ?", parentIDs).Find(&plist).Error
		for i := range plist {
			parents[plist[i].ID] = plist[i]
			userIDs = append(userIDs, plist[i].UserID)
		}
		var users []model.User
		_ = a.DB.Where("id IN ?", userIDs).Find(&users).Error
		for i := range users {
			usr := &users[i]
			names[usr.ID] = model.DisplayUsername(usr)
			avatars[usr.ID] = uploaderAvatarForAPI(usr)
		}
	}

	replyCounts := batchDynamicCommentReplyCounts(a.DB, commentIDs)

	likedByViewer := map[uint64]bool{}
	if viewerID, ok := middleware.UserID(c); ok && viewerID > 0 && len(commentIDs) > 0 {
		var likes []model.DynamicCommentLike
		_ = a.DB.Where("user_id = ? AND comment_id IN ?", viewerID, commentIDs).Find(&likes).Error
		for _, lk := range likes {
			likedByViewer[lk.CommentID] = true
		}
	}

	items := make([]gin.H, 0, len(list))
	for _, cm := range list {
		dyn := dynamics[cm.DynamicID]
		item := gin.H{
			"id":              cm.ID,
			"dynamic_id":      cm.DynamicID,
			"user_id":         cm.UserID,
			"username":        names[cm.UserID],
			"avatar_url":      avatars[cm.UserID],
			"parent_id":       cm.ParentID,
			"content":         cm.Content,
			"like_count":      cm.LikeCount,
			"liked_by_me":     likedByViewer[cm.ID],
			"reply_count":     replyCounts[cm.ID],
			"created_at":      cm.CreatedAt.Format("2006-01-02 15:04:05"),
			"approved":        cm.Approved,
			"curated_ignored": cm.CuratedIgnored,
			"dynamic": gin.H{
				"id":        dyn.ID,
				"title":     dynamicDisplayTitle(&dyn),
				"cover_url": dynamicCoverURL(&dyn),
			},
		}
		if cm.ParentID > 0 {
			if p, ok := parents[cm.ParentID]; ok {
				item["parent"] = gin.H{
					"id":       p.ID,
					"user_id":  p.UserID,
					"username": names[p.UserID],
					"content":  previewCommentContent(p.Content, 80),
				}
			}
		}
		items = append(items, item)
	}

	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(pageSize)))
	}
	resp.OK(c, gin.H{
		"items":       items,
		"page":        page,
		"page_size":   pageSize,
		"total":       total,
		"total_pages": totalPages,
	})
}

func batchArticleCommentReplyCounts(db *gorm.DB, ids []uint64) map[uint64]uint64 {
	out := make(map[uint64]uint64, len(ids))
	if len(ids) == 0 {
		return out
	}
	type row struct {
		ParentID uint64
		C        int64
	}
	var rows []row
	_ = db.Model(&model.ArticleComment{}).
		Select("parent_id, COUNT(*) AS c").
		Where("parent_id IN ?", ids).
		Group("parent_id").
		Scan(&rows).Error
	for _, r := range rows {
		if r.C > 0 {
			out[r.ParentID] = uint64(r.C)
		}
	}
	return out
}
