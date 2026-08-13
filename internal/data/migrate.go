package data

import (
	"cakecake/internal/model/admin"
	"cakecake/internal/model/agent"
	"cakecake/internal/model/article"
	"cakecake/internal/model/comment"
	"cakecake/internal/model/danmaku"
	"cakecake/internal/model/dm"
	"cakecake/internal/model/dynamic"
	"cakecake/internal/model/extra"
	"cakecake/internal/model/notification"
	"cakecake/internal/model/system"
	"cakecake/internal/model/user"
	"cakecake/internal/model/video"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"cakecake/internal/pkg/usercoin"
)

// RegisteredMigrations returns every schema/data migration in execution order.
// New migrations MUST be appended at the end with an incremented version number.
func RegisteredMigrations() []Migration {
	return []Migration{
		{1, "core_schema", "create all domain tables via GORM AutoMigrate", autoMigrateCoreModels},
		{2, "playback_comment_columns", "add missing playback and comment columns", ensurePlaybackAndCommentColumns},
		{3, "backfill_cake_ids", "generate CakeID for users missing one", backfillUserCakeIDs},
		{4, "backfill_first_published_at", "set first_published_at for existing UP", backfillUserFirstPublishedAt},
		{5, "backfill_video_comment_notifs", "insert missing video_comment_received notifications", backfillVideoCommentNotifications},
		{6, "backfill_reply_received_notifs", "insert missing reply_received notifications", backfillReplyReceivedNotifications},
		{7, "backfill_favorite_folders", "create default favorite folder for existing users", backfillFavoriteFolders},
		{8, "backfill_user_coin_balance", "set default coin_balance_tenths for users with 0", backfillUserCoinBalance},
		{9, "backfill_coin_ledger", "record coin ledger entry for initial balance", backfillCoinLedger},
		{10, "migrate_fav_unique_index", "replace legacy video_favorites unique index", migrateVideoFavoriteUniqueIndex},
		{11, "migrate_user_search_history", "dedupe and rebuild search history index", migrateUserSearchHistory},
		{12, "backfill_dm_participant_pins", "ensure dm_participant pins column", backfillDmParticipantPins},
		{13, "ensure_dm_hidden_at", "add hidden_at column to dm_participants", ensureDmParticipantHiddenAt},
		{14, "backfill_comment_approved", "mark existing comments as approved", backfillCommentApproved},
		{15, "resync_video_comment_counts", "recompute comment_count for curated videos", resyncCuratedVideoCommentCounts},
		{16, "backfill_article_comment_approved", "mark existing article comments as approved", backfillArticleCommentApproved},
		{17, "resync_article_comment_counts", "recompute comment_count for curated articles", resyncCuratedArticleCommentCounts},
		{18, "backfill_dynamic_comment_approved", "mark existing dynamic comments as approved", backfillDynamicCommentApproved},
		{19, "resync_dynamic_comment_counts", "recompute comment_count for curated dynamics", resyncCuratedDynamicCommentCounts},
		{20, "dm_message_content_text", "enlarge dm_messages.content to TEXT for long AI replies", migrateDmMessageContentText},
		{21, "dm_message_suggestions", "add suggestions column for AI follow-up chips", migrateDmMessageSuggestions},
		{22, "agent_feedback_table", "create agent_feedbacks table", migrateAgentFeedbackTable},
		{23, "transcode_dead_letters", "create transcode_dead_letters audit table", migrateTranscodeDeadLetters},
		{24, "transcode_dead_letter_archive", "add archived_at for retention archiving", migrateTranscodeDeadLetterArchive},
		{25, "direct_upload_claims", "create direct_upload_claims idempotency table", migrateDirectUploadClaims},
	}
}

// migrateDirectUploadClaims creates the claim table that makes direct-upload
// submits idempotent (one raw_key -> one video).
func migrateDirectUploadClaims(db *gorm.DB, lg *zap.Logger) error {
	if err := db.AutoMigrate(&video.DirectUploadClaim{}); err != nil {
		return err
	}
	if lg != nil {
		lg.Info("created direct_upload_claims table")
	}
	return nil
}

// migrateTranscodeDeadLetterArchive adds archived_at so retention soft-archives
// dead letters instead of deleting the audit trail.
func migrateTranscodeDeadLetterArchive(db *gorm.DB, lg *zap.Logger) error {
	if err := db.AutoMigrate(&video.TranscodeDeadLetter{}); err != nil {
		return err
	}
	if lg != nil {
		lg.Info("added transcode_dead_letters.archived_at")
	}
	return nil
}

// migrateTranscodeDeadLetters creates the transcode_dead_letters audit table
// (including lifecycle columns) for installations whose v1 core_schema
// migration predates the model. Existing fresh databases get the table from
// autoMigrateCoreModels, so this is a no-op for them.
func migrateTranscodeDeadLetters(db *gorm.DB, lg *zap.Logger) error {
	if err := db.AutoMigrate(&video.TranscodeDeadLetter{}); err != nil {
		return err
	}
	if lg != nil {
		lg.Info("created transcode_dead_letters table")
	}
	return nil
}

// migrateAgentFeedbackTable creates agent_feedbacks for existing installations.
func migrateAgentFeedbackTable(db *gorm.DB, lg *zap.Logger) error {
	if db.Dialector.Name() != "mysql" {
		return nil
	}
	if err := db.AutoMigrate(&agent.AgentFeedback{}); err != nil {
		return err
	}
	if lg != nil {
		lg.Info("created agent_feedbacks table")
	}
	return nil
}

// migrateDmMessageSuggestions adds dm_messages.suggestions (JSON array of
// model-generated follow-up questions). Stays in sync with the DmMessage model
// tag and migrations/00003_*.sql.
func migrateDmMessageSuggestions(db *gorm.DB, lg *zap.Logger) error {
	if db.Dialector.Name() != "mysql" {
		return nil
	}
	if dbColumnExists(db, "dm_messages", "suggestions") {
		return nil
	}
	if err := db.Exec("ALTER TABLE `dm_messages` ADD COLUMN `suggestions` TEXT NULL").Error; err != nil {
		return err
	}
	if lg != nil {
		lg.Info("added dm_messages.suggestions")
	}
	return nil
}

// migrateDmMessageContentText enlarges dm_messages.content from VARCHAR(500) to
// TEXT so long AI assistant replies are not truncated. It must stay in sync
// with the DmMessage model tag (type:text) and migrations/00002_*.sql.
func migrateDmMessageContentText(db *gorm.DB, lg *zap.Logger) error {
	if db.Dialector.Name() != "mysql" {
		// Fresh sqlite test DBs already create the column as TEXT via AutoMigrate.
		return nil
	}
	if err := db.Exec("ALTER TABLE `dm_messages` MODIFY COLUMN `content` TEXT NOT NULL").Error; err != nil {
		return err
	}
	if lg != nil {
		lg.Info("dm_messages.content enlarged to TEXT")
	}
	return nil
}

// autoMigrateCoreModels runs GORM AutoMigrate on every domain model table.
func autoMigrateCoreModels(db *gorm.DB, lg *zap.Logger) error {
	return db.AutoMigrate(
		&user.User{},
		&video.Video{},
		&danmaku.Danmaku{},
		&danmaku.DanmakuLike{},
		&comment.Comment{},
		&comment.CommentLike{},
		&comment.CommentDislike{},
		&video.VideoLike{},
		&video.TranscodeDeadLetter{},
		&video.FavoriteFolder{},
		&video.VideoFavorite{},
		&video.VideoCoin{},
		&video.WatchLater{},
		&user.UserFollow{},
		&user.UserBlock{},
		&user.UserFollowGroup{},
		&user.UserFollowGroupMember{},
		&notification.Notification{},
		&notification.LikeNotifMute{},
		&extra.UserDailyTask{},
		&user.CoinLedger{},
		&extra.VideoViewHistory{},
		&extra.ArticleViewHistory{},
		&dm.DmConversation{},
		&dm.DmParticipant{},
		&dm.DmMessage{},
		&system.SystemConfig{},
		&agent.AgentSettings{},
		&agent.AgentProfile{},
		&agent.AgentFeedback{},
		&article.Article{},
		&article.ArticleFavorite{},
		&article.ArticleCoin{},
		&comment.ArticleComment{},
		&comment.ArticleCommentLike{},
		&comment.ArticleCommentDislike{},
		&dynamic.UserDynamic{},
		&comment.UserDynamicLike{},
		&comment.DynamicComment{},
		&comment.DynamicCommentLike{},
		&comment.DynamicCommentDislike{},
		&admin.Admin{},
		&admin.HomeBanner{},
		&admin.HotSearchOp{},
		&admin.HotSearchDisplayLayout{},
	)
}

// AutoMigrateAll applies all registered migrations in order (Skill S-002).
// Each migration version is recorded in schema_versions so re-runs are safe.
func AutoMigrateAll(db *gorm.DB, lg *zap.Logger) error {
	return RunVersionedMigrations(db, lg, RegisteredMigrations())
}

// migrateVideoFavoriteUniqueIndex replaces legacy (user_id, video_id) unique index
// with (user_id, video_id, folder_id) so one video can exist in multiple folders.
func migrateVideoFavoriteUniqueIndex(db *gorm.DB, lg *zap.Logger) error {
	m := db.Migrator()
	if !m.HasTable(&video.VideoFavorite{}) {
		return nil
	}

	legacy := []string{"idx_video_fav_user_video"}
	for _, name := range legacy {
		if !m.HasIndex(&video.VideoFavorite{}, name) {
			continue
		}
		if err := m.DropIndex(&video.VideoFavorite{}, name); err != nil {
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

	if !m.HasIndex(&video.VideoFavorite{}, "idx_video_fav_user_video_folder") {
		if err := m.CreateIndex(&video.VideoFavorite{}, "idx_video_fav_user_video_folder"); err != nil {
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
	res := db.Model(&user.User{}).Where("coin_balance_tenths = 0").
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
	var users []user.User
	if err := db.Find(&users).Error; err != nil {
		return err
	}
	for _, u := range users {
		if strings.TrimSpace(u.CakeID) != "" {
			continue
		}
		cid := user.FormatCakeID(u.ID)
		if err := db.Model(&user.User{}).Where("id = ?", u.ID).Update("cake_id", cid).Error; err != nil {
			return err
		}
	}
	return nil
}

func backfillUserFirstPublishedAt(db *gorm.DB, lg *zap.Logger) error {
	var users []user.User
	if err := db.Find(&users).Error; err != nil {
		return err
	}
	for _, u := range users {
		if u.FirstPublishedAt != nil && !u.FirstPublishedAt.IsZero() {
			continue
		}
		var mt sql.NullTime
		row := db.Model(&video.Video{}).
			Where("user_id = ? AND status = ?", u.ID, video.StatusPublished).
			Select("MIN(created_at)").
			Row()
		if err := row.Scan(&mt); err != nil || !mt.Valid {
			continue
		}
		if err := db.Model(&user.User{}).Where("id = ?", u.ID).
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
	var roots []comment.Comment
	if err := db.Where("parent_id = ?", 0).Find(&roots).Error; err != nil {
		return err
	}
	var nInsert int
	for i := range roots {
		cm := &roots[i]
		var v video.Video
		if err := db.First(&v, cm.VideoID).Error; err != nil || v.Status != video.StatusPublished {
			continue
		}
		if v.UserID == 0 || cm.UserID == v.UserID {
			continue
		}
		var exist int64
		if err := db.Model(&notification.Notification{}).
			Where("type = ? AND related_id = ?", "video_comment_received", cm.ID).
			Count(&exist).Error; err != nil {
			return err
		}
		if exist > 0 {
			continue
		}
		var u user.User
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
			SenderUsername:  user.DisplayUsername(&u),
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
		n := notification.Notification{
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
	var replies []comment.Comment
	if err := db.Where("parent_id > ?", 0).Find(&replies).Error; err != nil {
		return err
	}
	var nInsert int
	for i := range replies {
		reply := &replies[i]
		var parent comment.Comment
		if err := db.First(&parent, reply.ParentID).Error; err != nil {
			continue
		}
		if parent.UserID == reply.UserID {
			continue
		}
		var exist int64
		if err := db.Model(&notification.Notification{}).
			Where("type = ? AND related_id = ?", "reply_received", reply.ID).
			Count(&exist).Error; err != nil {
			return err
		}
		if exist > 0 {
			continue
		}
		var u user.User
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
			SenderUsername:       user.DisplayUsername(&u),
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
		n := notification.Notification{
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
	if err := db.Model(&user.User{}).Pluck("id", &userIDs).Error; err != nil {
		return err
	}
	for _, uid := range userIDs {
		var cnt int64
		if err := db.Model(&video.FavoriteFolder{}).Where("user_id = ?", uid).Count(&cnt).Error; err != nil && lg != nil {
			lg.Warn("backfill favorite folders: count existing failed", zap.Uint64("user_id", uid), zap.Error(err))
		}
		if cnt > 0 {
			continue
		}
		f := video.FavoriteFolder{
			UserID:    uid,
			Title:     "默认收藏夹",
			IsPublic:  true,
			IsDefault: true,
		}
		if err := db.Create(&f).Error; err != nil {
			return err
		}
		if err := db.Model(&video.VideoFavorite{}).
			Where("user_id = ? AND folder_id = ?", uid, 0).
			Update("folder_id", f.ID).Error; err != nil && lg != nil {
			lg.Warn("backfill favorite folders: reassign legacy favorites failed", zap.Uint64("user_id", uid), zap.Uint64("folder_id", f.ID), zap.Error(err))
		}
		if lg != nil {
			lg.Info("backfill default favorite folder", zap.Uint64("user_id", uid))
		}
	}
	return nil
}

func backfillCoinLedger(db *gorm.DB, lg *zap.Logger) error {
	var n int64
	if err := db.Model(&user.CoinLedger{}).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	var coins []video.VideoCoin
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
		var v video.Video
		if err := db.Select("user_id").First(&v, c.VideoID).Error; err == nil && v.UserID > 0 {
			share := usercoin.CreatorShareTenths(c.Amount)
			if share > 0 {
				if err := usercoin.RecordLedgerAt(db, v.UserID, share, usercoin.ReasonVideoTipIncome, c.VideoID, at); err != nil {
					return err
				}
			}
		}
	}
	var tasks []extra.UserDailyTask
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
			return m.HasColumn(&video.Video{}, "CommentsClosed")
		case "comments_curated":
			return m.HasColumn(&video.Video{}, "CommentsCurated")
		case "danmaku_closed":
			return m.HasColumn(&video.Video{}, "DanmakuClosed")
		}
	case "comments":
		switch column {
		case "approved":
			return m.HasColumn(&comment.Comment{}, "Approved")
		case "curated_ignored":
			return m.HasColumn(&comment.Comment{}, "CuratedIgnored")
		}
	case "articles":
		switch column {
		case "comments_closed":
			return m.HasColumn(&article.Article{}, "CommentsClosed")
		case "comments_curated":
			return m.HasColumn(&article.Article{}, "CommentsCurated")
		}
	case "article_comments":
		switch column {
		case "approved":
			return m.HasColumn(&comment.ArticleComment{}, "Approved")
		case "curated_ignored":
			return m.HasColumn(&comment.ArticleComment{}, "CuratedIgnored")
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
	if m.HasTable(&video.Video{}) {
		for _, col := range []string{"CommentsClosed", "CommentsCurated", "DanmakuClosed"} {
			if !m.HasColumn(&video.Video{}, col) {
				if err := m.AddColumn(&video.Video{}, col); err != nil {
					return err
				}
			}
		}
	}
	if m.HasTable(&comment.Comment{}) {
		for _, col := range []string{"Approved", "CuratedIgnored"} {
			if !m.HasColumn(&comment.Comment{}, col) {
				if err := m.AddColumn(&comment.Comment{}, col); err != nil {
					return err
				}
			}
		}
	}
	if m.HasTable(&article.Article{}) {
		for _, col := range []string{"CommentsClosed", "CommentsCurated", "FailReason", "ReviewedAt", "ReviewedByAdminID"} {
			if !m.HasColumn(&article.Article{}, col) {
				if err := m.AddColumn(&article.Article{}, col); err != nil {
					return err
				}
			}
		}
	}
	if m.HasTable(&comment.ArticleComment{}) {
		for _, col := range []string{"Approved", "CuratedIgnored"} {
			if !m.HasColumn(&comment.ArticleComment{}, col) {
				if err := m.AddColumn(&comment.ArticleComment{}, col); err != nil {
					return err
				}
			}
		}
	}
	if m.HasTable(&dynamic.UserDynamic{}) {
		for _, col := range []string{"CommentsClosed", "CommentsCurated"} {
			if !m.HasColumn(&dynamic.UserDynamic{}, col) {
				if err := m.AddColumn(&dynamic.UserDynamic{}, col); err != nil {
					return err
				}
			}
		}
	}
	if m.HasTable(&comment.DynamicComment{}) {
		for _, col := range []string{"Approved", "CuratedIgnored"} {
			if !m.HasColumn(&comment.DynamicComment{}, col) {
				if err := m.AddColumn(&comment.DynamicComment{}, col); err != nil {
					return err
				}
			}
		}
	}
	if m.HasTable(&danmaku.Danmaku{}) && !m.HasColumn(&danmaku.Danmaku{}, "LikeCount") {
		if err := m.AddColumn(&danmaku.Danmaku{}, "LikeCount"); err != nil {
			return err
		}
	}
	if m.HasTable(&danmaku.Danmaku{}) && !m.HasColumn(&danmaku.Danmaku{}, "FontSize") {
		if err := m.AddColumn(&danmaku.Danmaku{}, "FontSize"); err != nil {
			return err
		}
		if err := db.Model(&danmaku.Danmaku{}).Where("font_size = '' OR font_size IS NULL").Update("font_size", "md").Error; err != nil && lg != nil {
			lg.Warn("backfill danmaku font_size failed", zap.Error(err))
		}
	}
	return nil
}

func resyncCuratedVideoCommentCounts(db *gorm.DB, lg *zap.Logger) error {
	if !dbColumnExists(db, "videos", "comments_curated") || !dbColumnExists(db, "comments", "approved") {
		return nil
	}
	var videos []video.Video
	if err := db.Where("comments_curated = ?", true).Find(&videos).Error; err != nil {
		return err
	}
	for i := range videos {
		v := &videos[i]
		var cnt int64
		if err := db.Model(&comment.Comment{}).
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
	var articles []article.Article
	if err := db.Where("comments_curated = ?", true).Find(&articles).Error; err != nil {
		return err
	}
	for i := range articles {
		art := &articles[i]
		var cnt int64
		if err := db.Model(&comment.ArticleComment{}).
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
	var dynamics []dynamic.UserDynamic
	if err := db.Where("comments_curated = ?", true).Find(&dynamics).Error; err != nil {
		return err
	}
	for i := range dynamics {
		dyn := &dynamics[i]
		var cnt int64
		if err := db.Model(&comment.DynamicComment{}).
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
