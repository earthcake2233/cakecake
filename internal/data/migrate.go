package data

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"minibili/internal/model"
	"minibili/internal/pkg/usercoin"
)

// AutoMigrateAll applies schema for all domain models (Skill S-002).
func AutoMigrateAll(db *gorm.DB, lg *zap.Logger) error {
	if err := db.AutoMigrate(
		&model.User{},
		&model.Video{},
		&model.Danmaku{},
		&model.DanmakuLike{},
		&model.Comment{},
		&model.CommentLike{},
		&model.CommentDislike{},
		&model.VideoLike{},
		&model.FavoriteFolder{},
		&model.VideoFavorite{},
		&model.VideoCoin{},
		&model.WatchLater{},
		&model.UserFollow{},
		&model.UserBlock{},
		&model.UserFollowGroup{},
		&model.UserFollowGroupMember{},
		&model.Notification{},
		&model.LikeNotifMute{},
		&model.UserDailyTask{},
		&model.CoinLedger{},
		&model.VideoViewHistory{},
		&model.ArticleViewHistory{},
		&model.DmConversation{},
		&model.DmParticipant{},
		&model.DmMessage{},
		&model.SystemConfig{},
		&model.AgentSettings{},
		&model.AgentProfile{},
		&model.Article{},
		&model.ArticleFavorite{},
		&model.ArticleCoin{},
		&model.ArticleComment{},
		&model.ArticleCommentLike{},
		&model.ArticleCommentDislike{},
		&model.UserDynamic{},
		&model.UserDynamicLike{},
		&model.DynamicComment{},
		&model.DynamicCommentLike{},
		&model.DynamicCommentDislike{},
		&model.Admin{},
		&model.HomeBanner{},
		&model.HotSearchOp{},
		&model.HotSearchDisplayLayout{},
	); err != nil {
		return err
	}
	if err := ensurePlaybackAndCommentColumns(db, lg); err != nil {
		return err
	}
	if err := backfillUserCakeIDs(db, lg); err != nil {
		return err
	}
	if err := backfillUserFirstPublishedAt(db, lg); err != nil {
		return err
	}
	if err := backfillVideoCommentNotifications(db, lg); err != nil {
		return err
	}
	if err := backfillReplyReceivedNotifications(db, lg); err != nil {
		return err
	}
	if err := backfillFavoriteFolders(db, lg); err != nil {
		return err
	}
	if err := backfillUserCoinBalance(db, lg); err != nil {
		return err
	}
	if err := backfillCoinLedger(db, lg); err != nil {
		return err
	}
	if err := migrateVideoFavoriteUniqueIndex(db, lg); err != nil {
		return err
	}
	if err := migrateUserSearchHistory(db, lg); err != nil {
		return err
	}
	if err := backfillDmParticipantPins(db, lg); err != nil {
		return err
	}
	if err := ensureDmParticipantHiddenAt(db, lg); err != nil {
		return err
	}
	if err := backfillCommentApproved(db, lg); err != nil {
		return err
	}
	if err := resyncCuratedVideoCommentCounts(db, lg); err != nil {
		return err
	}
	if err := backfillArticleCommentApproved(db, lg); err != nil {
		return err
	}
	if err := resyncCuratedArticleCommentCounts(db, lg); err != nil {
		return err
	}
	if err := backfillDynamicCommentApproved(db, lg); err != nil {
		return err
	}
	if err := resyncCuratedDynamicCommentCounts(db, lg); err != nil {
		return err
	}
	if lg != nil {
		lg.Info("database AutoMigrate completed")
	}
	return nil
}

// migrateVideoFavoriteUniqueIndex replaces legacy (user_id, video_id) unique index
// with (user_id, video_id, folder_id) so one video can exist in multiple folders.
func migrateVideoFavoriteUniqueIndex(db *gorm.DB, lg *zap.Logger) error {
	m := db.Migrator()
	if !m.HasTable(&model.VideoFavorite{}) {
		return nil
	}

	legacy := []string{"idx_video_fav_user_video"}
	for _, name := range legacy {
		if !m.HasIndex(&model.VideoFavorite{}, name) {
			continue
		}
		if err := m.DropIndex(&model.VideoFavorite{}, name); err != nil {
			if lg != nil {
				lg.Warn("migrator drop legacy video_favorites index failed, trying SQL",
					zap.String("index", name), zap.Error(err))
			}
			if db.Dialector.Name() == "mysql" {
				if err := db.Exec("ALTER TABLE video_favorites DROP INDEX " + name).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		}
		if lg != nil {
			lg.Info("dropped legacy video_favorites index", zap.String("index", name))
		}
	}

	if !m.HasIndex(&model.VideoFavorite{}, "idx_video_fav_user_video_folder") {
		if err := m.CreateIndex(&model.VideoFavorite{}, "idx_video_fav_user_video_folder"); err != nil {
			return err
		}
		if lg != nil {
			lg.Info("created video_favorites index idx_video_fav_user_video_folder")
		}
	}
	return nil
}

func backfillUserCoinBalance(db *gorm.DB, lg *zap.Logger) error {
	// Existing rows created before coin_balance_tenths may be 0 until first login grant.
	res := db.Model(&model.User{}).Where("coin_balance_tenths = 0").
		Update("coin_balance_tenths", usercoin.DefaultCoinTenths)
	if res.Error != nil {
		return res.Error
	}
	if lg != nil && res.RowsAffected > 0 {
		lg.Info("backfill user coin_balance_tenths to default",
			zap.Int64("rows", res.RowsAffected),
			zap.Int64("default_tenths", usercoin.DefaultCoinTenths))
	}
	return nil
}

func backfillUserCakeIDs(db *gorm.DB, lg *zap.Logger) error {
	var users []model.User
	if err := db.Find(&users).Error; err != nil {
		return err
	}
	for _, u := range users {
		if strings.TrimSpace(u.CakeID) != "" {
			continue
		}
		cid := model.FormatCakeID(u.ID)
		if err := db.Model(&model.User{}).Where("id = ?", u.ID).Update("cake_id", cid).Error; err != nil {
			return err
		}
	}
	return nil
}

func backfillUserFirstPublishedAt(db *gorm.DB, lg *zap.Logger) error {
	var users []model.User
	if err := db.Find(&users).Error; err != nil {
		return err
	}
	for _, u := range users {
		if u.FirstPublishedAt != nil && !u.FirstPublishedAt.IsZero() {
			continue
		}
		var mt sql.NullTime
		row := db.Model(&model.Video{}).
			Where("user_id = ? AND status = ?", u.ID, "published").
			Select("MIN(created_at)").
			Row()
		if err := row.Scan(&mt); err != nil || !mt.Valid {
			continue
		}
		if err := db.Model(&model.User{}).Where("id = ?", u.ID).
			Update("first_published_at", mt.Time).Error; err != nil {
			return err
		}
		if lg != nil {
			lg.Info("backfill first_published_at", zap.Uint64("user_id", u.ID))
		}
	}
	return nil
}

// videoCommentNotifPayload mirrors handler.videoCommentNotifPayload JSON for formatNotification.
type videoCommentNotifPayload struct {
	SenderID        uint64 `json:"sender_id"`
	SenderUsername  string `json:"sender_username"`
	SenderAvatarURL string `json:"sender_avatar_url"`
	CommentID       uint64 `json:"comment_id"`
	CommentContent  string `json:"comment_content"`
	VideoID         uint64 `json:"video_id"`
	VideoTitle      string `json:"video_title"`
	CoverURL        string `json:"cover_url"`
}

// replyNotifPayload mirrors handler.replyNotifPayload JSON.
type replyNotifPayload struct {
	SenderID             uint64 `json:"sender_id"`
	SenderUsername       string `json:"sender_username"`
	SenderAvatarURL      string `json:"sender_avatar_url"`
	ReplyCommentID       uint64 `json:"reply_comment_id"`
	ReplyContent         string `json:"reply_content"`
	ParentCommentID      uint64 `json:"parent_comment_id"`
	ParentContentPreview string `json:"parent_content_preview"`
	VideoID              uint64 `json:"video_id"`
}

// backfillVideoCommentNotifications inserts missing video_comment_received rows for old top-level comments.
func backfillVideoCommentNotifications(db *gorm.DB, lg *zap.Logger) error {
	var roots []model.Comment
	if err := db.Where("parent_id = ?", 0).Find(&roots).Error; err != nil {
		return err
	}
	var nInsert int
	for i := range roots {
		cm := &roots[i]
		var v model.Video
		if err := db.First(&v, cm.VideoID).Error; err != nil || v.Status != "published" {
			continue
		}
		if v.UserID == 0 || cm.UserID == v.UserID {
			continue
		}
		var exist int64
		if err := db.Model(&model.Notification{}).
			Where("type = ? AND related_id = ?", "video_comment_received", cm.ID).
			Count(&exist).Error; err != nil {
			return err
		}
		if exist > 0 {
			continue
		}
		var u model.User
		if err := db.First(&u, cm.UserID).Error; err != nil {
			continue
		}
		title := strings.TrimSpace(v.Title)
		tr := []rune(title)
		if len(tr) > 80 {
			title = string(tr[:80])
		}
		pl := videoCommentNotifPayload{
			SenderID:        cm.UserID,
			SenderUsername:  model.DisplayUsername(&u),
			SenderAvatarURL: strings.TrimSpace(u.AvatarURL),
			CommentID:       cm.ID,
			CommentContent:  cm.Content,
			VideoID:         v.ID,
			VideoTitle:      title,
			CoverURL:        strings.TrimSpace(v.CoverURL),
		}
		pb, err := json.Marshal(pl)
		if err != nil {
			continue
		}
		prevShort := strings.TrimSpace(pl.CommentContent)
		sr := []rune(prevShort)
		if len(sr) > 32 {
			prevShort = string(sr[:32])
		}
		nm, _ := json.Marshal([]string{pl.SenderUsername})
		n := model.Notification{
			RecipientID:     v.UserID,
			Type:            "video_comment_received",
			RelatedID:       cm.ID,
			SenderNamesJSON: string(nm),
			TotalLikes:      0,
			CommentPreview:  prevShort,
			PayloadJSON:     string(pb),
			IsRead:          false,
		}
		if err := db.Create(&n).Error; err != nil {
			if lg != nil {
				lg.Warn("backfill video_comment_received failed", zap.Uint64("comment_id", cm.ID), zap.Error(err))
			}
			continue
		}
		nInsert++
	}
	if lg != nil && nInsert > 0 {
		lg.Info("backfill video_comment_received", zap.Int("inserted", nInsert))
	}
	return nil
}

// backfillReplyReceivedNotifications inserts missing reply_received for historical replies.
func backfillReplyReceivedNotifications(db *gorm.DB, lg *zap.Logger) error {
	var replies []model.Comment
	if err := db.Where("parent_id > ?", 0).Find(&replies).Error; err != nil {
		return err
	}
	var nInsert int
	for i := range replies {
		reply := &replies[i]
		var parent model.Comment
		if err := db.First(&parent, reply.ParentID).Error; err != nil {
			continue
		}
		if parent.UserID == reply.UserID {
			continue
		}
		var exist int64
		if err := db.Model(&model.Notification{}).
			Where("type = ? AND related_id = ?", "reply_received", reply.ID).
			Count(&exist).Error; err != nil {
			return err
		}
		if exist > 0 {
			continue
		}
		var u model.User
		if err := db.First(&u, reply.UserID).Error; err != nil {
			continue
		}
		preview := strings.TrimSpace(parent.Content)
		runes := []rune(preview)
		if len(runes) > 120 {
			preview = string(runes[:120])
		}
		pl := replyNotifPayload{
			SenderID:             reply.UserID,
			SenderUsername:       model.DisplayUsername(&u),
			SenderAvatarURL:      strings.TrimSpace(u.AvatarURL),
			ReplyCommentID:       reply.ID,
			ReplyContent:         reply.Content,
			ParentCommentID:      reply.ParentID,
			ParentContentPreview: preview,
			VideoID:              parent.VideoID,
		}
		pb, err := json.Marshal(pl)
		if err != nil {
			continue
		}
		prevShort := preview
		sr := []rune(prevShort)
		if len(sr) > 32 {
			prevShort = string(sr[:32])
		}
		nm, _ := json.Marshal([]string{pl.SenderUsername})
		n := model.Notification{
			RecipientID:     parent.UserID,
			Type:            "reply_received",
			RelatedID:       reply.ID,
			SenderNamesJSON: string(nm),
			TotalLikes:      0,
			CommentPreview:  prevShort,
			PayloadJSON:     string(pb),
			IsRead:          false,
		}
		if err := db.Create(&n).Error; err != nil {
			if lg != nil {
				lg.Warn("backfill reply_received failed", zap.Uint64("comment_id", reply.ID), zap.Error(err))
			}
			continue
		}
		nInsert++
	}
	if lg != nil && nInsert > 0 {
		lg.Info("backfill reply_received", zap.Int("inserted", nInsert))
	}
	return nil
}

func backfillFavoriteFolders(db *gorm.DB, lg *zap.Logger) error {
	var userIDs []uint64
	if err := db.Model(&model.User{}).Pluck("id", &userIDs).Error; err != nil {
		return err
	}
	for _, uid := range userIDs {
		var cnt int64
		_ = db.Model(&model.FavoriteFolder{}).Where("user_id = ?", uid).Count(&cnt).Error
		if cnt > 0 {
			continue
		}
		f := model.FavoriteFolder{
			UserID:    uid,
			Title:     "默认收藏夹",
			IsPublic:  true,
			IsDefault: true,
		}
		if err := db.Create(&f).Error; err != nil {
			return err
		}
		_ = db.Model(&model.VideoFavorite{}).
			Where("user_id = ? AND folder_id = ?", uid, 0).
			Update("folder_id", f.ID).Error
		if lg != nil {
			lg.Info("backfill default favorite folder", zap.Uint64("user_id", uid))
		}
	}
	return nil
}

func backfillCoinLedger(db *gorm.DB, lg *zap.Logger) error {
	var n int64
	if err := db.Model(&model.CoinLedger{}).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	var coins []model.VideoCoin
	if err := db.Order("created_at ASC").Find(&coins).Error; err != nil {
		return err
	}
	for i := range coins {
		c := &coins[i]
		at := c.CreatedAt
		if at.IsZero() {
			at = time.Now()
		}
		cost := usercoin.CostTenths(c.Amount)
		if err := usercoin.RecordLedgerAt(db, c.UserID, -cost, usercoin.ReasonVideoTip, c.VideoID, at); err != nil {
			return err
		}
		var v model.Video
		if err := db.Select("user_id").First(&v, c.VideoID).Error; err == nil && v.UserID > 0 {
			share := usercoin.CreatorShareTenths(c.Amount)
			if share > 0 {
				if err := usercoin.RecordLedgerAt(db, v.UserID, share, usercoin.ReasonVideoTipIncome, c.VideoID, at); err != nil {
					return err
				}
			}
		}
	}
	var tasks []model.UserDailyTask
	if err := db.Where("login_done = ?", true).Find(&tasks).Error; err != nil {
		return err
	}
	for i := range tasks {
		t := &tasks[i]
		at := t.UpdatedAt
		if at.IsZero() {
			at = t.CreatedAt
		}
		if at.IsZero() {
			at = time.Now()
		}
		if err := usercoin.RecordLedgerAt(db, t.UserID, usercoin.DailyLoginCoinTenths, usercoin.ReasonLoginReward, 0, at); err != nil {
			return err
		}
	}
	if lg != nil {
		lg.Info("backfill coin_ledger",
			zap.Int("video_coins", len(coins)),
			zap.Int("login_tasks", len(tasks)))
	}
	return nil
}

func isIgnorableAddColumnErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate column") ||
		strings.Contains(msg, "duplicate column name")
}

func dbColumnExists(db *gorm.DB, table, column string) bool {
	if db.Dialector.Name() == "mysql" {
		var n int64
		err := db.Raw(`
			SELECT COUNT(*) FROM information_schema.COLUMNS
			WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ?
		`, table, column).Scan(&n).Error
		return err == nil && n > 0
	}
	m := db.Migrator()
	switch table {
	case "videos":
		switch column {
		case "comments_closed":
			return m.HasColumn(&model.Video{}, "CommentsClosed")
		case "comments_curated":
			return m.HasColumn(&model.Video{}, "CommentsCurated")
		case "danmaku_closed":
			return m.HasColumn(&model.Video{}, "DanmakuClosed")
		}
	case "comments":
		switch column {
		case "approved":
			return m.HasColumn(&model.Comment{}, "Approved")
		case "curated_ignored":
			return m.HasColumn(&model.Comment{}, "CuratedIgnored")
		}
	case "articles":
		switch column {
		case "comments_closed":
			return m.HasColumn(&model.Article{}, "CommentsClosed")
		case "comments_curated":
			return m.HasColumn(&model.Article{}, "CommentsCurated")
		}
	case "article_comments":
		switch column {
		case "approved":
			return m.HasColumn(&model.ArticleComment{}, "Approved")
		case "curated_ignored":
			return m.HasColumn(&model.ArticleComment{}, "CuratedIgnored")
		}
	}
	return false
}

// ensurePlaybackAndCommentColumns adds columns that older deployments may lack.
func ensurePlaybackAndCommentColumns(db *gorm.DB, lg *zap.Logger) error {
	if db.Dialector.Name() == "mysql" {
		stmts := []struct {
			sql    string
			table  string
			column string
		}{
			{"ALTER TABLE videos ADD COLUMN comments_closed TINYINT(1) NOT NULL DEFAULT 0", "videos", "comments_closed"},
			{"ALTER TABLE videos ADD COLUMN comments_curated TINYINT(1) NOT NULL DEFAULT 0", "videos", "comments_curated"},
			{"ALTER TABLE videos ADD COLUMN danmaku_closed TINYINT(1) NOT NULL DEFAULT 0", "videos", "danmaku_closed"},
			{"ALTER TABLE comments ADD COLUMN approved TINYINT(1) NOT NULL DEFAULT 0", "comments", "approved"},
			{"ALTER TABLE comments ADD COLUMN curated_ignored TINYINT(1) NOT NULL DEFAULT 0", "comments", "curated_ignored"},
			{"ALTER TABLE articles ADD COLUMN comments_closed TINYINT(1) NOT NULL DEFAULT 0", "articles", "comments_closed"},
			{"ALTER TABLE articles ADD COLUMN comments_curated TINYINT(1) NOT NULL DEFAULT 0", "articles", "comments_curated"},
			{"ALTER TABLE article_comments ADD COLUMN approved TINYINT(1) NOT NULL DEFAULT 0", "article_comments", "approved"},
			{"ALTER TABLE article_comments ADD COLUMN curated_ignored TINYINT(1) NOT NULL DEFAULT 0", "article_comments", "curated_ignored"},
			{"ALTER TABLE user_dynamics ADD COLUMN comments_closed TINYINT(1) NOT NULL DEFAULT 0", "user_dynamics", "comments_closed"},
			{"ALTER TABLE user_dynamics ADD COLUMN comments_curated TINYINT(1) NOT NULL DEFAULT 0", "user_dynamics", "comments_curated"},
			{"ALTER TABLE dynamic_comments ADD COLUMN approved TINYINT(1) NOT NULL DEFAULT 0", "dynamic_comments", "approved"},
			{"ALTER TABLE dynamic_comments ADD COLUMN curated_ignored TINYINT(1) NOT NULL DEFAULT 0", "dynamic_comments", "curated_ignored"},
			{"ALTER TABLE danmakus ADD COLUMN like_count BIGINT UNSIGNED NOT NULL DEFAULT 0", "danmakus", "like_count"},
		}
		for _, it := range stmts {
			if dbColumnExists(db, it.table, it.column) {
				continue
			}
			if err := db.Exec(it.sql).Error; err != nil && !isIgnorableAddColumnErr(err) {
				return err
			}
			if lg != nil {
				lg.Info("added column", zap.String("table", it.table), zap.String("column", it.column))
			}
		}
		return nil
	}
	m := db.Migrator()
	if m.HasTable(&model.Video{}) {
		for _, col := range []string{"CommentsClosed", "CommentsCurated", "DanmakuClosed"} {
			if !m.HasColumn(&model.Video{}, col) {
				if err := m.AddColumn(&model.Video{}, col); err != nil {
					return err
				}
			}
		}
	}
	if m.HasTable(&model.Comment{}) {
		for _, col := range []string{"Approved", "CuratedIgnored"} {
			if !m.HasColumn(&model.Comment{}, col) {
				if err := m.AddColumn(&model.Comment{}, col); err != nil {
					return err
				}
			}
		}
	}
	if m.HasTable(&model.Article{}) {
		for _, col := range []string{"CommentsClosed", "CommentsCurated", "FailReason", "ReviewedAt", "ReviewedByAdminID"} {
			if !m.HasColumn(&model.Article{}, col) {
				if err := m.AddColumn(&model.Article{}, col); err != nil {
					return err
				}
			}
		}
	}
	if m.HasTable(&model.ArticleComment{}) {
		for _, col := range []string{"Approved", "CuratedIgnored"} {
			if !m.HasColumn(&model.ArticleComment{}, col) {
				if err := m.AddColumn(&model.ArticleComment{}, col); err != nil {
					return err
				}
			}
		}
	}
	if m.HasTable(&model.UserDynamic{}) {
		for _, col := range []string{"CommentsClosed", "CommentsCurated"} {
			if !m.HasColumn(&model.UserDynamic{}, col) {
				if err := m.AddColumn(&model.UserDynamic{}, col); err != nil {
					return err
				}
			}
		}
	}
	if m.HasTable(&model.DynamicComment{}) {
		for _, col := range []string{"Approved", "CuratedIgnored"} {
			if !m.HasColumn(&model.DynamicComment{}, col) {
				if err := m.AddColumn(&model.DynamicComment{}, col); err != nil {
					return err
				}
			}
		}
	}
	if m.HasTable(&model.Danmaku{}) && !m.HasColumn(&model.Danmaku{}, "LikeCount") {
		if err := m.AddColumn(&model.Danmaku{}, "LikeCount"); err != nil {
			return err
		}
	}
	if m.HasTable(&model.Danmaku{}) && !m.HasColumn(&model.Danmaku{}, "FontSize") {
		if err := m.AddColumn(&model.Danmaku{}, "FontSize"); err != nil {
			return err
		}
		_ = db.Model(&model.Danmaku{}).Where("font_size = '' OR font_size IS NULL").Update("font_size", "md").Error
	}
	return nil
}

func resyncCuratedVideoCommentCounts(db *gorm.DB, lg *zap.Logger) error {
	if !dbColumnExists(db, "videos", "comments_curated") || !dbColumnExists(db, "comments", "approved") {
		return nil
	}
	var videos []model.Video
	if err := db.Where("comments_curated = ?", true).Find(&videos).Error; err != nil {
		return err
	}
	for i := range videos {
		v := &videos[i]
		var cnt int64
		if err := db.Model(&model.Comment{}).
			Where("video_id = ? AND approved = ?", v.ID, true).
			Count(&cnt).Error; err != nil {
			return err
		}
		if err := db.Model(v).Update("comment_count", cnt).Error; err != nil {
			return err
		}
	}
	if lg != nil && len(videos) > 0 {
		lg.Info("resync curated video comment_count", zap.Int("videos", len(videos)))
	}
	return nil
}

func backfillCommentApproved(db *gorm.DB, lg *zap.Logger) error {
	if !dbColumnExists(db, "comments", "approved") {
		return nil
	}
	var res *gorm.DB
	if dbColumnExists(db, "videos", "comments_curated") {
		// Portable SQL (MySQL UPDATE+JOIN is invalid on SQLite).
		res = db.Exec(`
			UPDATE comments
			SET approved = 1
			WHERE approved = 0
			  AND video_id IN (SELECT id FROM videos WHERE comments_curated = 0)
		`)
	} else {
		res = db.Exec(`UPDATE comments SET approved = 1 WHERE approved = 0`)
	}
	if res.Error != nil {
		return res.Error
	}
	if lg != nil && res.RowsAffected > 0 {
		lg.Info("backfill comment approved (non-curated videos only)", zap.Int64("rows", res.RowsAffected))
	}
	return nil
}

func backfillArticleCommentApproved(db *gorm.DB, lg *zap.Logger) error {
	if !dbColumnExists(db, "article_comments", "approved") {
		return nil
	}
	var res *gorm.DB
	if dbColumnExists(db, "articles", "comments_curated") {
		res = db.Exec(`
			UPDATE article_comments
			SET approved = 1
			WHERE approved = 0
			  AND article_id IN (SELECT id FROM articles WHERE comments_curated = 0)
		`)
	} else {
		res = db.Exec(`UPDATE article_comments SET approved = 1 WHERE approved = 0`)
	}
	if res.Error != nil {
		return res.Error
	}
	if lg != nil && res.RowsAffected > 0 {
		lg.Info("backfill article comment approved (non-curated articles only)", zap.Int64("rows", res.RowsAffected))
	}
	return nil
}

func resyncCuratedArticleCommentCounts(db *gorm.DB, lg *zap.Logger) error {
	if !dbColumnExists(db, "articles", "comments_curated") || !dbColumnExists(db, "article_comments", "approved") {
		return nil
	}
	var articles []model.Article
	if err := db.Where("comments_curated = ?", true).Find(&articles).Error; err != nil {
		return err
	}
	for i := range articles {
		art := &articles[i]
		var cnt int64
		if err := db.Model(&model.ArticleComment{}).
			Where("article_id = ? AND approved = ?", art.ID, true).
			Count(&cnt).Error; err != nil {
			return err
		}
		if err := db.Model(art).Update("comment_count", cnt).Error; err != nil {
			return err
		}
	}
	if lg != nil && len(articles) > 0 {
		lg.Info("resync curated article comment_count", zap.Int("articles", len(articles)))
	}
	return nil
}

func backfillDynamicCommentApproved(db *gorm.DB, lg *zap.Logger) error {
	if !dbColumnExists(db, "dynamic_comments", "approved") {
		return nil
	}
	var res *gorm.DB
	if dbColumnExists(db, "user_dynamics", "comments_curated") {
		res = db.Exec(`
			UPDATE dynamic_comments
			SET approved = 1
			WHERE approved = 0
			  AND dynamic_id IN (SELECT id FROM user_dynamics WHERE comments_curated = 0)
		`)
	} else {
		res = db.Exec(`UPDATE dynamic_comments SET approved = 1 WHERE approved = 0`)
	}
	if res.Error != nil {
		return res.Error
	}
	if lg != nil && res.RowsAffected > 0 {
		lg.Info("backfill dynamic comment approved (non-curated dynamics only)", zap.Int64("rows", res.RowsAffected))
	}
	return nil
}

func resyncCuratedDynamicCommentCounts(db *gorm.DB, lg *zap.Logger) error {
	if !dbColumnExists(db, "user_dynamics", "comments_curated") || !dbColumnExists(db, "dynamic_comments", "approved") {
		return nil
	}
	var dynamics []model.UserDynamic
	if err := db.Where("comments_curated = ?", true).Find(&dynamics).Error; err != nil {
		return err
	}
	for i := range dynamics {
		dyn := &dynamics[i]
		var cnt int64
		if err := db.Model(&model.DynamicComment{}).
			Where("dynamic_id = ? AND approved = ?", dyn.ID, true).
			Count(&cnt).Error; err != nil {
			return err
		}
		if err := db.Model(dyn).Update("comment_count", cnt).Error; err != nil {
			return err
		}
	}
	if lg != nil && len(dynamics) > 0 {
		lg.Info("resync curated dynamic comment_count", zap.Int("dynamics", len(dynamics)))
	}
	return nil
}
