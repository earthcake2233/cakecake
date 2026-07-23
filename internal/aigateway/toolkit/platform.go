package toolkit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"minibili/internal/model"
	"minibili/internal/pkg/sensitive"
	"minibili/internal/search"
)

// PlatformExecutor implements Executor using the platforms DB and search.
type PlatformExecutor struct {
	DB   *gorm.DB
	ES   *search.Client
	Sens *sensitive.Filter
}

func (p *PlatformExecutor) Execute(ctx context.Context, toolName string, args json.RawMessage) (string, error) {
	if p.Sens != nil {
		raw := string(args)
		if err := p.Sens.Check(raw); err != nil {
			return "", fmt.Errorf("sensitive content in arguments")
		}
	}
	switch toolName {
	case ToolSearchVideos:
		return p.searchVideos(ctx, args)
	case ToolGetVideoDetail:
		return p.getVideoDetail(ctx, args)
	case ToolGetTrending:
		return p.getTrending(ctx, args)
	case ToolGetVideoComments:
		return p.getVideoComments(ctx, args)
	case ToolGetVideoDanmaku:
		return p.getVideoDanmaku(ctx, args)
	default:
		return "", fmt.Errorf("unknown tool: %s", toolName)
	}
}

type searchVideosArgs struct {
	Keyword  string `json:"keyword"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}

func (p *PlatformExecutor) searchVideos(ctx context.Context, raw json.RawMessage) (string, error) {
	var args searchVideosArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	args.Keyword = strings.TrimSpace(args.Keyword)
	if args.Keyword == "" {
		return `[]`, nil
	}
	if args.Page < 1 {
		args.Page = 1
	}
	if args.PageSize < 1 || args.PageSize > 20 {
		args.PageSize = 10
	}

	if p.ES != nil && p.ES.Enabled() {
		res, err := p.ES.SearchAll(ctx, search.SearchParams{
			Keyword:  args.Keyword,
			Page:     args.Page,
			PageSize: args.PageSize,
		})
		if err != nil {
			return "", fmt.Errorf("search failed: %w", err)
		}
		type videoItem struct {
			ID       uint64 `json:"id"`
			Title    string `json:"title"`
			Author   string `json:"author"`
			Plays    uint64 `json:"plays"`
			Duration string `json:"duration"`
			Cover    string `json:"cover"`
		}
		items := make([]videoItem, 0, len(res.Result.Video))
		for _, v := range res.Result.Video {
			author := strings.TrimSpace(v.Author)
			if author == "" {
				author = "unknown"
			}
			items = append(items, videoItem{
				ID:       v.Aid,
				Title:    v.Title,
				Author:   author,
				Plays:    v.Play,
				Duration: v.Duration,
				Cover:    v.Pic,
			})
		}
		b, _ := json.Marshal(map[string]interface{}{
			"total": res.NumResults,
			"items": items,
		})
		return string(b), nil
	}
	// Fallback: simple DB search
	var videos []model.Video
	if err := p.DB.WithContext(ctx).
		Where("status = ? AND title LIKE ?", "published", "%"+args.Keyword+"%").
		Order("play_count DESC").
		Limit(args.PageSize).
		Offset((args.Page - 1) * args.PageSize).
		Find(&videos).Error; err != nil {
		return "", fmt.Errorf("db search failed: %w", err)
	}
	// Batch query uploaders
	userIDs := make([]uint64, 0, len(videos))
	for _, v := range videos {
		userIDs = append(userIDs, v.UserID)
	}
	userMap := make(map[uint64]string)
	if len(userIDs) > 0 {
		var users []model.User
		p.DB.WithContext(ctx).Where("id IN ?", userIDs).Find(&users)
		for _, u := range users {
			name := u.Username
			if n := strings.TrimSpace(u.Nickname); n != "" && !model.IsUserAnonymized(&u) {
				name = n
			}
			userMap[u.ID] = name
		}
	}
	type item struct {
		ID           uint64 `json:"id"`
		Title        string `json:"title"`
		UploaderName string `json:"uploader_name"`
		PlayCount    uint64 `json:"play_count"`
		Duration     float64 `json:"duration_sec"`
		CoverURL     string `json:"cover_url"`
	}
	items := make([]item, 0, len(videos))
	for _, v := range videos {
		uploaderName, _ := userMap[v.UserID]
		if uploaderName == "" {
			uploaderName = "unknown"
		}
		items = append(items, item{
			ID: v.ID, Title: v.Title, UploaderName: uploaderName,
			PlayCount: v.PlayCount, Duration: v.DurationSec, CoverURL: v.CoverURL,
		})
	}
	b, _ := json.Marshal(map[string]interface{}{"items": items})
	return string(b), nil
}

type videoDetailArgs struct {
	VideoID uint64 `json:"video_id"`
}

func (p *PlatformExecutor) getVideoDetail(ctx context.Context, raw json.RawMessage) (string, error) {
	var args videoDetailArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if args.VideoID == 0 {
		return "", fmt.Errorf("video_id is required")
	}
	var v model.Video
	if err := p.DB.WithContext(ctx).First(&v, args.VideoID).Error; err != nil {
		return fmt.Sprintf(`{"error": "video not found", "video_id": %d}`, args.VideoID), nil
	}
	var u model.User
	uploaderName := "unknown"
	if err := p.DB.WithContext(ctx).First(&u, v.UserID).Error; err == nil {
		if n := strings.TrimSpace(u.Nickname); n != "" && !model.IsUserAnonymized(&u) {
			uploaderName = n
		} else {
			uploaderName = u.Username
		}
	}
	var tags []string
	if v.TagsJSON != "" {
		json.Unmarshal([]byte(v.TagsJSON), &tags)
	}
	b, _ := json.Marshal(map[string]interface{}{
		"items": []map[string]interface{}{
			{
				"id":            v.ID,
				"title":         v.Title,
				"description":   truncateStr(v.Description, 200),
				"uploader":      uploaderName,
				"cover_url":     v.CoverURL,
				"duration_sec":  v.DurationSec,
				"play_count":    v.PlayCount,
				"like_count":    v.LikeCount,
				"comment_count": v.CommentCount,
				"danmaku_count": v.DanmakuCount,
				"tags":          tags,
				"zone":          v.Zone,
				"created_at":    v.CreatedAt.Format("2006-01-02 15:04:05"),
			},
		},
	})
	return string(b), nil
}

type trendingArgs struct {
	Limit int `json:"limit"`
}

func (p *PlatformExecutor) getTrending(ctx context.Context, raw json.RawMessage) (string, error) {
	var args trendingArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if args.Limit < 1 || args.Limit > 20 {
		args.Limit = 10
	}
	var videos []model.Video
	if err := p.DB.WithContext(ctx).
		Where("status = ?", "published").
		Order("play_count DESC").
		Limit(args.Limit).
		Find(&videos).Error; err != nil {
		return "", fmt.Errorf("trending query failed: %w", err)
	}
	// Batch query uploaders
	userIDs := make([]uint64, 0, len(videos))
	for _, v := range videos {
		userIDs = append(userIDs, v.UserID)
	}
	userMap := make(map[uint64]string)
	if len(userIDs) > 0 {
		var users []model.User
		p.DB.WithContext(ctx).Where("id IN ?", userIDs).Find(&users)
		for _, u := range users {
			name := u.Username
			if n := strings.TrimSpace(u.Nickname); n != "" && !model.IsUserAnonymized(&u) {
				name = n
			}
			userMap[u.ID] = name
		}
	}
	type item struct {
		Rank         int    `json:"rank"`
		Title        string `json:"title"`
		VideoID      uint64 `json:"video_id"`
		UploaderName string `json:"uploader_name"`
		PlayCount    uint64 `json:"play_count"`
		Duration     float64 `json:"duration_sec"`
		CoverURL     string `json:"cover_url"`
	}
	items := make([]item, 0, len(videos))
	for i, v := range videos {
		uploaderName, _ := userMap[v.UserID]
		if uploaderName == "" {
			uploaderName = "unknown"
		}
		items = append(items, item{
			Rank: i + 1, Title: v.Title, VideoID: v.ID,
			UploaderName: uploaderName, PlayCount: v.PlayCount,
			Duration: v.DurationSec, CoverURL: v.CoverURL,
		})
	}
	b, _ := json.Marshal(map[string]interface{}{"items": items})
	return string(b), nil
}

type commentsArgs struct {
	VideoID  uint64 `json:"video_id"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}

func (p *PlatformExecutor) getVideoComments(ctx context.Context, raw json.RawMessage) (string, error) {
	var args commentsArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if args.VideoID == 0 {
		return "", fmt.Errorf("video_id is required")
	}
	if args.Page < 1 {
		args.Page = 1
	}
	if args.PageSize < 1 || args.PageSize > 20 {
		args.PageSize = 10
	}
	var comments []model.Comment
	if err := p.DB.WithContext(ctx).
		Where(`video_id = ?`, args.VideoID).
		Order("id ASC").
		Limit(args.PageSize).
		Offset((args.Page - 1) * args.PageSize).
		Find(&comments).Error; err != nil {
		return "", fmt.Errorf("comments query failed: %w", err)
	}
	// Batch query commenters
	userIDs := make([]uint64, 0, len(comments))
	for _, c := range comments {
		userIDs = append(userIDs, c.UserID)
	}
	userMap := make(map[uint64]*model.User)
	if len(userIDs) > 0 {
		var users []model.User
		p.DB.WithContext(ctx).Where("id IN ?", userIDs).Find(&users)
		for i := range users {
			userMap[users[i].ID] = &users[i]
		}
	}
	type item struct {
		ID         uint64 `json:"id"`
		Content    string `json:"content"`
		LikeCount  uint64 `json:"like_count"`
		UserName   string `json:"user_name"`
		UserAvatar string `json:"user_avatar"`
	}
	items := make([]item, 0, len(comments))
	for _, c := range comments {
		userName := "匿名"
		userAvatar := ""
		if u, ok := userMap[c.UserID]; ok {
			userName = u.Username
			if n := strings.TrimSpace(u.Nickname); n != "" && !model.IsUserAnonymized(u) {
				userName = n
			}
			userAvatar = u.AvatarURL
		}
		items = append(items, item{
			ID: c.ID, Content: truncateStr(c.Content, 100),
			LikeCount: c.LikeCount, UserName: userName, UserAvatar: userAvatar,
		})
	}
	b, _ := json.Marshal(map[string]interface{}{"items": items})
	return string(b), nil
}

type danmakuArgs struct {
	VideoID uint64 `json:"video_id"`
	Limit   int    `json:"limit"`
}

func (p *PlatformExecutor) getVideoDanmaku(ctx context.Context, raw json.RawMessage) (string, error) {
	var args danmakuArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if args.VideoID == 0 {
		return "", fmt.Errorf("video_id is required")
	}
	if args.Limit < 1 || args.Limit > 50 {
		args.Limit = 20
	}
	var danmakus []model.Danmaku
	if err := p.DB.WithContext(ctx).
		Where(`video_id = ?`, args.VideoID).
		Order("id DESC").
		Limit(args.Limit).
		Find(&danmakus).Error; err != nil {
		return "", fmt.Errorf("danmaku query failed: %w", err)
	}
	// Batch query danmaku users
	userIDs := make([]uint64, 0, len(danmakus))
	for _, d := range danmakus {
		userIDs = append(userIDs, d.UserID)
	}
	userMap := make(map[uint64]string)
	if len(userIDs) > 0 {
		var users []model.User
		p.DB.WithContext(ctx).Where("id IN ?", userIDs).Find(&users)
		for _, u := range users {
			name := u.Username
			if n := strings.TrimSpace(u.Nickname); n != "" && !model.IsUserAnonymized(&u) {
				name = n
			}
			userMap[u.ID] = name
		}
	}
	type item struct {
		Content   string  `json:"content"`
		VideoTime float64 `json:"video_time"`
		Type      string  `json:"type"`
		Color     string  `json:"color"`
		UserName  string  `json:"user_name"`
	}
	items := make([]item, 0, len(danmakus))
	for _, d := range danmakus {
		userName, _ := userMap[d.UserID]
		if userName == "" {
			userName = "匿名"
		}
		items = append(items, item{
			Content: d.Content, VideoTime: d.VideoTime,
			Type: d.Type, Color: d.Color, UserName: userName,
		})
	}
	b, _ := json.Marshal(map[string]interface{}{"items": items})
	return string(b), nil
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
