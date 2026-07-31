package handler

import (
	"minibili/internal/model/article"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"minibili/internal/errcode"
	"minibili/internal/middleware"
	"minibili/internal/pkg/dailyreward"
	"minibili/internal/pkg/resp"
	"minibili/internal/pkg/usercoin"
)

// ToggleArticleFavorite toggles favorite on an article.
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
	favorited, favCount, svcErr := a.ArticleSvc.ToggleArticleFavorite(c.Request.Context(), uid, aid)
	if svcErr != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"favorited": favorited, "fav_count": favCount})
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

	result, svcErr := a.ArticleSvc.PostArticleCoin(c.Request.Context(), uid, aid, amount)
	if svcErr != nil {
		a.Log.Error("post article coin", zap.Error(svcErr))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if result.AlreadyCoined {
		resp.Err(c, http.StatusBadRequest, errcode.CodeAlreadyCoined)
		return
	}
	if result.Insufficient {
		resp.Err(c, http.StatusBadRequest, errcode.CodeInsufficientCoins)
		return
	}

	viewerModel, _ := a.UserSvc.GetUserByID(c.Request.Context(), uid)
	coinBalance := 0.0
	if viewerModel != nil {
		coinBalance = usercoin.BalanceFloat(viewerModel.CoinBalanceTenths)
	}

	resp.OK(c, gin.H{
		"coined":                  true,
		"coin_count":              result.CoinCount,
		"amount":                  result.Amount,
		"my_coin_amount":          result.MyCoinAmount,
		"coined_by_me":            true,
		"coin_balance":            coinBalance,
		"daily_coin_exp_progress": result.DailyProgress,
		"daily_coin_exp_max":      dailyreward.ExpCoinMax,
	})
}

func (a *API) ListMyArticleFavorites(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	a.listArticleFavoritesForUser(c, uid)
}

// ListUserArticleFavorites returns a user's public article favorites.
// ListUserArticleFavorites godoc
// @Summary      List user article favorites
// @Description  Get paginated article favorites for a user space
// @Tags         Articles
// @Produce      json
// @Param        userId path int true "User ID"
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Page size" default(20)
// @Success      200 {object} map[string]interface{}
// @Router       /space/{userId}/article-favorites [get]
func (a *API) ListUserArticleFavorites(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil || userID == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	u, err := a.UserSvc.GetUserByID(c.Request.Context(), userID)
	if err != nil {
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
	limit := parseLimit(c, 20, 50)
	curID, _ := strconv.ParseUint(c.Query("cursor"), 10, 64)
	favs, hasMore, err := a.ArticleSvc.ListFavoritedArticlesV2(c.Request.Context(), uid, curID, limit)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	ids := make([]uint64, 0, len(favs))
	for _, f := range favs {
		ids = append(ids, f.ArticleID)
	}
	artsMap := a.ArticleSvc.BatchFetchArticles(c.Request.Context(), ids, !isOwnerView)
	arts := make(map[uint64]article.Article, len(artsMap))
	for id, aPtr := range artsMap {
		arts[id] = *aPtr
	}
	uids := make([]uint64, 0)
	for _, art := range arts {
		uids = append(uids, art.UserID)
	}
	names := map[uint64]string{}
	if len(uids) > 0 {
		for _, uid2 := range uids {
			userPub, _ := a.UserSvc.GetUserPublic(c.Request.Context(), uid2)
			if userPub != nil {
				names[uid2] = userPub.Username
			}
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
		total, _ = a.ArticleSvc.CountFavoritedArticles(c.Request.Context(), uid, false)
	} else {
		total, _ = a.ArticleSvc.CountFavoritedArticles(c.Request.Context(), uid, true)
	}
	resp.OK(c, gin.H{"items": items, "next_cursor": next, "total": total})
}
