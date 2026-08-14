package handler

import (
	"cakecake/internal/errcode"
	"cakecake/internal/middleware"
	"cakecake/internal/model/user"
	"cakecake/internal/model/video"
	"cakecake/internal/pkg/dailyreward"
	"cakecake/internal/pkg/resp"
	vsvc "cakecake/internal/service/video"
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func uploaderAvatarForAPI(u *user.User) string {
	return avatarURLForAPI(u)
}

// uploaderNameForAPI is the UP display name on video cards (nickname if set, else username).
func uploaderNameForAPI(u *user.User) string {
	if u == nil {
		return ""
	}
	if nick := strings.TrimSpace(u.Nickname); nick != "" && !user.IsUserAnonymized(u) {
		return nick
	}
	return user.DisplayUsername(u)
}

// ListPublishedVideos is the home feed (F10, AC-4).
// Query: limit, cursor, zone_parent, sort=hot|time, days=1|3|7|30, arc_type=0|1 (1=recent uploads only).
// ListPublishedVideos godoc
// @Summary      List published videos
// @Description  Get paginated list of published videos
// @Tags        Videos
// @Produce     json
// @Param       page query int false "Page number" default(1)
// @Param       page_size query int false "Page size" default(20)
// @Param       zone query string false "Video zone slug"
// @Param       sort query string false "Sort field (created_at,play_count)"
// @Success     200 {object} map[string]interface{}
// @Router      /videos [get]
func (a *API) ListPublishedVideos(c *gin.Context) {
	limit := parseLimit(c, 20, 100)
	sortKey := strings.TrimSpace(c.DefaultQuery("sort", "hot"))
	zoneParent := ""
	if zp := normalizeVideoZone(strings.TrimSpace(c.Query("zone_parent"))); zp != "" {
		if p, _ := splitVideoZone(zp); p != "" {
			zoneParent = p
		}
	}
	days := 0
	if s := strings.TrimSpace(c.Query("days")); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			switch n {
			case 1, 3, 7, 30:
				days = n
			}
		}
	}
	arcType := 0
	if s := strings.TrimSpace(c.Query("arc_type")); s != "" {
		if n, err := strconv.Atoi(s); err == nil && (n == 0 || n == 1) {
			arcType = n
		}
	}
	recentOnly := days > 0 && arcType == 1
	res, err := a.VideoSvc.ListPublishedVideos(c.Request.Context(), vsvc.VideoListOpts{
		Limit:      limit,
		SortKey:    sortKey,
		ZoneParent: zoneParent,
		Days:       days,
		RecentOnly: recentOnly,
		Cursor:     c.Query("cursor"),
	})
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	items := make([]videoCardDTO, 0, len(res.Videos))
	for _, v := range res.Videos {
		pc, _ := a.Play.Display(c.Request.Context(), &v)
		items = append(items, videoCard(v, res.UploaderNames[v.UserID], pc, videoEngagement{}))
	}
	resp.OK(c, zoneVideoListResponse{
		Items:          items,
		NextCursor:     res.NextCursor,
		ZoneVideoCount: res.ZoneVideoCount,
		HasMore:        res.HasMore,
	})
}

// GetVideo returns detail for playback page (F3, F4).
// GetVideo godoc
// @Summary      Get video detail
// @Description  Get detailed video info by ID
// @Tags        Videos
// @Produce     json
// @Param       id path int true "Video ID"
// @Success     200 {object} map[string]interface{}
// @Router      /videos/{id} [get]
func (a *API) GetVideo(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	v, err := a.VideoSvc.GetVideoByID(c.Request.Context(), id)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	var viewer uint64
	if uid, ok := middleware.UserID(c); ok {
		viewer = uid
	}
	if v.Status != video.StatusPublished && v.UserID != viewer {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if v.Status == video.StatusPublished {
		_ = a.Play.Incr(c.Request.Context(), v.ID)
	}
	pc, _ := a.Play.Display(c.Request.Context(), v)
	var u user.User
	uPub, _ := a.UserSvc.GetUserPublic(c.Request.Context(), v.UserID)
	if uPub != nil {
		u = user.User{ID: uPub.ID, Username: uPub.Username, AvatarURL: uPub.AvatarURL, Nickname: uPub.Nickname, Sign: uPub.Sign}
	}
	watching := 0
	if a.Hub != nil {
		watching = a.Hub.RoomSize(id)
	}
	eng := a.getVideoEngagementFlags(c.Request.Context(), viewer, v.ID)
	detail := videoDetail(*v, u, pc, watching, eng)
	if v.Status == videoStatusDraft && viewer == v.UserID {
		draftHasSource := strings.TrimSpace(v.DraftRawKey) != ""
		detail.DraftHasSource = &draftHasSource
	}
	_, followerCnt := a.getFollowCounts(c.Request.Context(), v.UserID)
	detail.UploaderFollowerCount = followerCnt
	detail.UploaderPublishedCount = a.getUploaderPublishedCount(c.Request.Context(), v.UserID)
	if viewer > 0 && v.UserID != viewer {
		detail.FollowedByMe = a.isFollowing(c.Request.Context(), viewer, v.UserID)
		prog := 0
		if a.DailyRewardSvc != nil {
			prog = a.DailyRewardSvc.CoinProgress(viewer)
		}
		max := dailyreward.ExpCoinMax
		detail.DailyCoinExpProgress = &prog
		detail.DailyCoinExpMax = &max
	} else {
		detail.FollowedByMe = false
	}
	resp.OK(c, detail)
}

type videoEngagement struct {
	LikedByMe     bool
	FavoritedByMe bool
	CoinedByMe    bool
	MyCoinAmount  int // 0, 1, or 2 coins given by viewer on this video
	InWatchLater  bool
}

func (a *API) getVideoEngagementFlags(ctx context.Context, viewer, videoID uint64) videoEngagement {
	var e videoEngagement
	if viewer == 0 || videoID == 0 {
		return e
	}
	liked := a.EngagementSvc.BatchVideoLikes(ctx, viewer, []uint64{videoID})
	e.LikedByMe = liked[videoID]
	fav := a.FavoriteSvc.BatchFavorited(ctx, viewer, []uint64{videoID})
	e.FavoritedByMe = fav[videoID]
	coinMap := a.EngagementSvc.BatchCoinedByUser(ctx, viewer, []uint64{videoID})
	if amt, ok := coinMap[videoID]; ok && amt > 0 {
		e.CoinedByMe = true
		e.MyCoinAmount = amt
		if e.MyCoinAmount < 0 {
			e.MyCoinAmount = 0
		}
		if e.MyCoinAmount > 2 {
			e.MyCoinAmount = 2
		}
	}
	wl := a.EngagementSvc.BatchWatchLater(ctx, viewer, []uint64{videoID})
	e.InWatchLater = wl[videoID]
	return e
}

type videoCardDTO struct {
	ID              uint64  `json:"id"`
	UserID          uint64  `json:"user_id"`
	Title           string  `json:"title"`
	Description     string  `json:"description"`
	CoverURL        string  `json:"cover_url"`
	PlayCount       uint64  `json:"play_count"`
	DanmakuCount    uint64  `json:"danmaku_count"`
	CommentCount    uint64  `json:"comment_count"`
	LikeCount       uint64  `json:"like_count"`
	FavCount        uint64  `json:"fav_count"`
	CoinCount       uint64  `json:"coin_count"`
	LikedByMe       bool    `json:"liked_by_me"`
	FavoritedByMe   bool    `json:"favorited_by_me"`
	CoinedByMe      bool    `json:"coined_by_me"`
	InWatchLater    bool    `json:"in_watch_later"`
	Duration        float64 `json:"duration"`
	Uploader        string  `json:"uploader"`
	CreatedAt       string  `json:"created_at"`
	CommentsClosed  bool    `json:"comments_closed"`
	CommentsCurated bool    `json:"comments_curated"`
	DanmakuClosed   bool    `json:"danmaku_closed"`
	videoZoneFields
}

type videoDetailDTO struct {
	ID                     uint64   `json:"id"`
	UserID                 uint64   `json:"user_id"`
	Title                  string   `json:"title"`
	Description            string   `json:"description"`
	PlayCount              uint64   `json:"play_count"`
	DanmakuCount           uint64   `json:"danmaku_count"`
	CommentCount           uint64   `json:"comment_count"`
	LikeCount              uint64   `json:"like_count"`
	FavCount               uint64   `json:"fav_count"`
	CoinCount              uint64   `json:"coin_count"`
	LikedByMe              bool     `json:"liked_by_me"`
	FavoritedByMe          bool     `json:"favorited_by_me"`
	CoinedByMe             bool     `json:"coined_by_me"`
	MyCoinAmount           int      `json:"my_coin_amount"`
	InWatchLater           bool     `json:"in_watch_later"`
	WatchingCount          int      `json:"watching_count"`
	Duration               float64  `json:"duration"`
	Uploader               string   `json:"uploader"`
	UploaderSign           string   `json:"uploader_sign"`
	UploaderAvatarURL      string   `json:"uploader_avatar_url"`
	CreatedAt              string   `json:"created_at"`
	VideoURL               string   `json:"video_url"`
	CoverURL               string   `json:"cover_url"`
	Status                 string   `json:"status"`
	FailReason             string   `json:"fail_reason"`
	Tags                   []string `json:"tags"`
	CommentsClosed         bool     `json:"comments_closed"`
	CommentsCurated        bool     `json:"comments_curated"`
	DanmakuClosed          bool     `json:"danmaku_closed"`
	DraftHasSource         *bool    `json:"draft_has_source,omitempty"`
	UploaderFollowerCount  int64    `json:"uploader_follower_count"`
	UploaderPublishedCount int64    `json:"uploader_published_count"`
	FollowedByMe           bool     `json:"followed_by_me"`
	DailyCoinExpProgress   *int     `json:"daily_coin_exp_progress,omitempty"`
	DailyCoinExpMax        *int     `json:"daily_coin_exp_max,omitempty"`
	videoZoneFields
}

type zoneVideoListResponse struct {
	Items          []videoCardDTO `json:"items"`
	NextCursor     string         `json:"next_cursor"`
	ZoneVideoCount int64          `json:"zone_video_count"`
	HasMore        bool           `json:"has_more"`
}

func videoCard(v video.Video, up string, play uint64, eng videoEngagement) videoCardDTO {
	m := videoCardDTO{
		ID:              v.ID,
		UserID:          v.UserID,
		Title:           v.Title,
		Description:     v.Description,
		CoverURL:        v.CoverURL,
		PlayCount:       play,
		DanmakuCount:    v.DanmakuCount,
		CommentCount:    v.CommentCount,
		LikeCount:       v.LikeCount,
		FavCount:        v.FavCount,
		CoinCount:       v.CoinCount,
		LikedByMe:       eng.LikedByMe,
		FavoritedByMe:   eng.FavoritedByMe,
		CoinedByMe:      eng.CoinedByMe,
		InWatchLater:    eng.InWatchLater,
		Duration:        v.DurationSec,
		Uploader:        up,
		CreatedAt:       v.CreatedAt.Format("2006-01-02 15:04:05"),
		CommentsClosed:  v.CommentsClosed,
		CommentsCurated: v.CommentsCurated,
		DanmakuClosed:   v.DanmakuClosed,
	}
	appendVideoZoneFields(&m.videoZoneFields, v.Zone)
	return m
}

func videoDetail(v video.Video, u user.User, play uint64, watching int, eng videoEngagement) videoDetailDTO {
	m := videoDetailDTO{
		ID:                v.ID,
		UserID:            v.UserID,
		Title:             v.Title,
		Description:       v.Description,
		PlayCount:         play,
		DanmakuCount:      v.DanmakuCount,
		CommentCount:      v.CommentCount,
		LikeCount:         v.LikeCount,
		FavCount:          v.FavCount,
		CoinCount:         v.CoinCount,
		LikedByMe:         eng.LikedByMe,
		FavoritedByMe:     eng.FavoritedByMe,
		CoinedByMe:        eng.CoinedByMe,
		MyCoinAmount:      eng.MyCoinAmount,
		InWatchLater:      eng.InWatchLater,
		WatchingCount:     watching,
		Duration:          v.DurationSec,
		Uploader:          user.DisplayUsername(&u),
		UploaderSign:      strings.TrimSpace(u.Sign),
		UploaderAvatarURL: uploaderAvatarForAPI(&u),
		CreatedAt:         v.CreatedAt.Format("2006-01-02 15:04:05"),
		VideoURL:          v.VideoURL,
		CoverURL:          v.CoverURL,
		Status:            v.Status,
		FailReason:        vsvc.HumanizeFailReason(v.FailReason),
		Tags:              videoTagsForResponse(v.TagsJSON),
		CommentsClosed:    v.CommentsClosed,
		CommentsCurated:   v.CommentsCurated,
		DanmakuClosed:     v.DanmakuClosed,
	}
	appendVideoZoneFields(&m.videoZoneFields, v.Zone)
	return m
}
