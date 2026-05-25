package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"minibili/internal/errcode"
	"minibili/internal/middleware"
	"minibili/internal/model"
	"minibili/internal/pkg/dailyreward"
	"minibili/internal/pkg/resp"
	"minibili/internal/pkg/usercoin"
)

// ToggleArticleFavorite toggles favorite on an article (图文收藏夹).
func (a *API) ToggleArticleFavorite(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	aid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || aid == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if _, ok := loadPublishedArticle(a, aid); !ok {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	var row model.ArticleFavorite
	res := a.DB.Where("user_id = ? AND article_id = ?", uid, aid).Limit(1).Find(&row)
	if res.Error != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if res.RowsAffected > 0 {
		if err := a.DB.Where("user_id = ? AND article_id = ?", uid, aid).
			Delete(&model.ArticleFavorite{}).Error; err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		_ = a.DB.Model(&model.Article{}).Where("id = ?", aid).
			UpdateColumn("fav_count", gorm.Expr("GREATEST(fav_count - ?, 0)", 1)).Error
		var art model.Article
		_ = a.DB.First(&art, aid).Error
		resp.OK(c, gin.H{"favorited": false, "fav_count": art.FavCount})
		return
	}
	if err := a.DB.Create(&model.ArticleFavorite{UserID: uid, ArticleID: aid}).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	_ = a.DB.Model(&model.Article{}).Where("id = ?", aid).
		UpdateColumn("fav_count", gorm.Expr("fav_count + ?", 1)).Error
	var art model.Article
	_ = a.DB.First(&art, aid).Error
	resp.OK(c, gin.H{"favorited": true, "fav_count": art.FavCount})
}

type articleCoinJSON struct {
	Amount int `json:"amount"`
}

// PostArticleCoin tips 1 or 2 coins on an article.
func (a *API) PostArticleCoin(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	aid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || aid == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	art, ok := loadPublishedArticle(a, aid)
	if !ok {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if art.UserID == uid {
		resp.Err(c, http.StatusBadRequest, errcode.CodeCannotCoinSelf)
		return
	}
	var body articleCoinJSON
	_ = c.ShouldBindJSON(&body)
	amount := body.Amount
	if amount != 1 && amount != 2 {
		amount = 1
	}
	var exist model.ArticleCoin
	res := a.DB.Where("user_id = ? AND article_id = ?", uid, aid).Limit(1).Find(&exist)
	if res.Error != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	coinBefore := dailyreward.CoinProgress(a.DB, uid)
	var viewer model.User
	var spentAmount int
	var myCoinAmount int

	if res.RowsAffected > 0 {
		if exist.Amount >= 2 {
			resp.Err(c, http.StatusBadRequest, errcode.CodeAlreadyCoined)
			return
		}
		spentAmount = 1
		myCoinAmount = 2
		if err := a.DB.Transaction(func(tx *gorm.DB) error {
			if err := usercoin.SpendOnArticleCoin(tx, uid, art.UserID, aid, spentAmount); err != nil {
				return err
			}
			if err := tx.Model(&exist).Update("amount", 2).Error; err != nil {
				return err
			}
			return tx.Model(&model.Article{}).Where("id = ?", aid).
				UpdateColumn("coin_count", gorm.Expr("coin_count + ?", 1)).Error
		}); err != nil {
			if errors.Is(err, usercoin.ErrInsufficientCoins) {
				resp.Err(c, http.StatusBadRequest, errcode.CodeInsufficientCoins)
				return
			}
			a.Log.Error("post article coin add", zap.Error(err))
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
	} else {
		if amount != 1 && amount != 2 {
			amount = 1
		}
		spentAmount = amount
		myCoinAmount = amount
		if err := a.DB.Transaction(func(tx *gorm.DB) error {
			if err := usercoin.SpendOnArticleCoin(tx, uid, art.UserID, aid, spentAmount); err != nil {
				return err
			}
			row := model.ArticleCoin{UserID: uid, ArticleID: aid, Amount: amount}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
			return tx.Model(&model.Article{}).Where("id = ?", aid).
				UpdateColumn("coin_count", gorm.Expr("coin_count + ?", amount)).Error
		}); err != nil {
			if errors.Is(err, usercoin.ErrInsufficientCoins) {
				resp.Err(c, http.StatusBadRequest, errcode.CodeInsufficientCoins)
				return
			}
			a.Log.Error("post article coin", zap.Error(err))
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
	}

	coinAfter := dailyreward.CoinProgress(a.DB, uid)
	_ = dailyreward.GrantCoinExp(a.DB, uid, coinBefore, coinAfter)
	_ = a.DB.First(&art, aid).Error
	_ = a.DB.First(&viewer, uid).Error
	resp.OK(c, gin.H{
		"coined":                  true,
		"coin_count":              art.CoinCount,
		"amount":                  spentAmount,
		"my_coin_amount":          myCoinAmount,
		"coined_by_me":            true,
		"coin_balance":            usercoin.BalanceFloat(viewer.CoinBalanceTenths),
		"daily_coin_exp_progress": coinAfter,
		"daily_coin_exp_max":      dailyreward.ExpCoinMax,
	})
}

// ListMyArticleFavorites returns the current user's 图文收藏夹.
func (a *API) ListMyArticleFavorites(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	a.listArticleFavoritesForUser(c, uid)
}

// ListUserArticleFavorites returns a user's public article favorites.
func (a *API) ListUserArticleFavorites(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil || userID == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var u model.User
	if err := a.DB.First(&u, userID).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	viewer, viewerOK := middleware.UserID(c)
	if !spaceViewerCanSee(userID, viewerOK, viewer, u.PrivacyPublicFavorites) {
		resp.OK(c, gin.H{"items": []gin.H{}, "next_cursor": "", "total": 0})
		return
	}
	a.listArticleFavoritesForUser(c, userID)
}

func (a *API) listArticleFavoritesForUser(c *gin.Context, uid uint64) {
	viewerUID, viewerOK := middleware.UserID(c)
	isOwnerView := viewerOK && viewerUID == uid
	limit := 20
	if s := c.Query("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 50 {
			limit = n
		}
	}
	curID, _ := strconv.ParseUint(c.Query("cursor"), 10, 64)
	q := a.DB.Model(&model.ArticleFavorite{}).Where("user_id = ?", uid)
	if curID > 0 {
		q = q.Where("id < ?", curID)
	}
	var favs []model.ArticleFavorite
	if err := q.Order("id DESC").Limit(limit + 1).Find(&favs).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	hasMore := len(favs) > limit
	if hasMore {
		favs = favs[:limit]
	}
	ids := make([]uint64, 0, len(favs))
	for _, f := range favs {
		ids = append(ids, f.ArticleID)
	}
	arts := map[uint64]model.Article{}
	if len(ids) > 0 {
		var rows []model.Article
		qArt := a.DB.Where("id IN ?", ids)
		if !isOwnerView {
			qArt = qArt.Where("status = ?", articleStatusPublished)
		}
		_ = qArt.Find(&rows).Error
		for i := range rows {
			arts[rows[i].ID] = rows[i]
		}
	}
	uids := make([]uint64, 0)
	for _, art := range arts {
		uids = append(uids, art.UserID)
	}
	names := map[uint64]string{}
	if len(uids) > 0 {
		var users []model.User
		_ = a.DB.Where("id IN ?", uids).Find(&users).Error
		for i := range users {
			usr := &users[i]
			n := model.DisplayUsername(usr)
			if usr.Nickname != "" && !model.IsUserAnonymized(usr) {
				n = usr.Nickname
			}
			names[usr.ID] = n
		}
	}
	items := make([]gin.H, 0, len(favs))
	for _, f := range favs {
		art, ok := arts[f.ArticleID]
		if !ok {
			if isOwnerView {
				items = append(items, gin.H{
					"id":            f.ArticleID,
					"title":         "专栏已不可用",
					"cover_url":     "",
					"status":        "",
					"view_count":    0,
					"comment_count": 0,
					"coin_count":    0,
					"fav_count":     0,
					"forward_count": 0,
					"published_at":  "",
					"created_at":    "",
					"author_name":   "",
					"favorited_at":  f.CreatedAt.Format("2006-01-02 15:04:05"),
					"unavailable":   true,
				})
			}
			continue
		}
		row := articleListItem(art, names[art.UserID], articleEngagement{FavoritedByMe: true})
		row["favorited_at"] = f.CreatedAt.Format("2006-01-02 15:04:05")
		items = append(items, row)
	}
	next := ""
	if hasMore && len(favs) > 0 {
		next = strconv.FormatUint(favs[len(favs)-1].ID, 10)
	}
	var total int64
	if isOwnerView {
		_ = a.DB.Table("article_favorites").
			Joins("INNER JOIN articles ON articles.id = article_favorites.article_id").
			Where("article_favorites.user_id = ?", uid).
			Count(&total).Error
	} else {
		_ = a.DB.Table("article_favorites").
			Joins("INNER JOIN articles ON articles.id = article_favorites.article_id AND articles.status = ?", articleStatusPublished).
			Where("article_favorites.user_id = ?", uid).
			Count(&total).Error
	}
	resp.OK(c, gin.H{"items": items, "next_cursor": next, "total": total})
}
