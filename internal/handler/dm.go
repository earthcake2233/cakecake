package handler

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"minibili/internal/errcode"
	"minibili/internal/middleware"
	"minibili/internal/model"
	"minibili/internal/pkg/resp"
)

const dmMaxContentRunes = 500

type dmCreateConversationReq struct {
	PeerID uint64 `json:"peer_id"`
}

type dmPostMessageReq struct {
	Content string `json:"content"`
}

type dmSettingsReq struct {
	Pinned *bool `json:"pinned"`
	Muted  *bool `json:"muted"`
}

func dmPairIDs(a, b uint64) (low, high uint64) {
	if a < b {
		return a, b
	}
	return b, a
}

func dmPeerID(conv *model.DmConversation, self uint64) uint64 {
	if conv.UserLow == self {
		return conv.UserHigh
	}
	return conv.UserLow
}

func dmPinnedAtAfter(a, b *time.Time) bool {
	if a == nil && b == nil {
		return false
	}
	if a == nil {
		return false
	}
	if b == nil {
		return true
	}
	return a.After(*b)
}

func dmTrimPreview(s string) string {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) <= 80 {
		return s
	}
	r := []rune(s)
	return string(r[:80]) + "…"
}

func (a *API) dmUnreadTotal(uid uint64) int64 {
	var cnt int64
	_ = a.DB.Model(&model.DmParticipant{}).
		Where("user_id = ? AND unread_count > 0", uid).
		Count(&cnt).Error
	return cnt
}

func (a *API) dmUserBrief(db *gorm.DB, uid uint64) (name, avatar string) {
	var u model.User
	if err := db.First(&u, uid).Error; err != nil {
		return "用户", ""
	}
	name = model.DisplayUsername(&u)
	if nick := strings.TrimSpace(u.Nickname); nick != "" {
		name = nick
	}
	return name, uploaderAvatarForAPI(&u)
}

func (a *API) dmFormatMessage(m *model.DmMessage, senderName, senderAvatar string) gin.H {
	role := m.Role
	if role == "" {
		role = "user"
	}
	return gin.H{
		"id":              m.ID,
		"conversation_id": m.ConversationID,
		"sender_id":       m.SenderID,
		"sender_name":     senderName,
		"sender_avatar":   senderAvatar,
		"content":         m.Content,
		"role":            role,
		"created_at":      m.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

func (a *API) dmFormatConversation(conv *model.DmConversation, self uint64, part *model.DmParticipant) gin.H {
	peer := dmPeerID(conv, self)
	name, avatar := a.dmUserBrief(a.DB, peer)
	unread := uint32(0)
	pinned := false
	muted := false
	if part != nil {
		unread = part.UnreadCount
		pinned = part.Pinned
		muted = part.Muted
	}
	kind := conv.Kind
	if kind == "" {
		kind = model.DmKindHuman
	}
	return gin.H{
		"id":              conv.ID,
		"peer_id":         peer,
		"peer_name":       name,
		"peer_avatar":     avatar,
		"last_preview":    conv.LastPreview,
		"last_message_at": conv.LastMessageAt.Format("2006-01-02 15:04:05"),
		"unread_count":    unread,
		"pinned":          pinned,
		"muted":           muted,
		"kind":             kind,
		"is_agent":         a.dmIsAgentConv(conv),
		"agent_profile_id": conv.AgentProfileID,
	}
}

func (a *API) dmEnsureParticipant(tx *gorm.DB, convID, uid uint64) {
	var p model.DmParticipant
	if err := tx.Where("conversation_id = ? AND user_id = ?", convID, uid).First(&p).Error; err == nil {
		return
	}
	_ = tx.Create(&model.DmParticipant{
		ConversationID: convID,
		UserID:         uid,
		UnreadCount:    0,
	}).Error
}

func (a *API) dmPushEvent(userID uint64, payload gin.H) {
	if a.ChatHub == nil || userID == 0 {
		return
	}
	a.ChatHub.PushJSON(userID, payload)
}

// ListDmConversations returns recent 1:1 threads for the current user.
func (a *API) ListDmConversations(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	a.ensureAgentConversationFor(uid)
	var convs []model.DmConversation
	if err := a.DB.Where("user_low = ? OR user_high = ?", uid, uid).
		Order("last_message_at DESC").
		Limit(100).
		Find(&convs).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	partMap := map[uint64]*model.DmParticipant{}
	if len(convs) > 0 {
		ids := make([]uint64, len(convs))
		for i := range convs {
			ids[i] = convs[i].ID
		}
		var parts []model.DmParticipant
		_ = a.DB.Where("user_id = ? AND conversation_id IN ?", uid, ids).Find(&parts).Error
		for i := range parts {
			p := parts[i]
			partMap[p.ConversationID] = &p
		}
	}
	sort.Slice(convs, func(i, j int) bool {
		pi, pj := partMap[convs[i].ID], partMap[convs[j].ID]
		pinI := pi != nil && pi.Pinned
		pinJ := pj != nil && pj.Pinned
		if pinI != pinJ {
			return pinI
		}
		if pinI && pinJ {
			return dmPinnedAtAfter(pi.PinnedAt, pj.PinnedAt)
		}
		return convs[i].LastMessageAt.After(convs[j].LastMessageAt)
	})
	items := make([]gin.H, 0, len(convs))
	for i := range convs {
		conv := &convs[i]
		part := partMap[conv.ID]
		if part != nil && part.HiddenAt != nil {
			continue
		}
		items = append(items, a.dmFormatConversation(conv, uid, part))
	}
	resp.OK(c, gin.H{"items": items})
}

// CreateDmConversation finds or creates a thread with peer_id.
func (a *API) CreateDmConversation(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	var req dmCreateConversationReq
	if err := c.ShouldBindJSON(&req); err != nil || req.PeerID == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if req.PeerID == uid {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if a.Agent != nil && a.Agent.IsBotUser(req.PeerID) {
		a.ensureAgentConversationFor(uid)
		low, high := dmPairIDs(uid, req.PeerID)
		var conv model.DmConversation
		if err := a.DB.Where("user_low = ? AND user_high = ?", low, high).First(&conv).Error; err != nil {
			resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
			return
		}
		var part model.DmParticipant
		_ = a.DB.Where("conversation_id = ? AND user_id = ?", conv.ID, uid).First(&part).Error
		resp.OK(c, a.dmFormatConversation(&conv, uid, &part))
		return
	}
	var peer model.User
	if err := a.DB.First(&peer, req.PeerID).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if model.IsUserAnonymized(&peer) {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if dmUsersBlocked(a.DB, uid, req.PeerID) {
		resp.Err(c, http.StatusForbidden, errcode.CodeUserBlocked)
		return
	}
	low, high := dmPairIDs(uid, req.PeerID)
	var conv model.DmConversation
	err := a.DB.Where("user_low = ? AND user_high = ?", low, high).First(&conv).Error
	if err == gorm.ErrRecordNotFound {
		now := time.Now()
		conv = model.DmConversation{
			UserLow:       low,
			UserHigh:      high,
			Kind:          model.DmKindHuman,
			LastMessageAt: now,
			LastPreview:   "",
		}
		if err := a.DB.Create(&conv).Error; err != nil {
			a.Log.Error("create dm conversation", zap.Error(err))
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
	}
	tx := a.DB.Begin()
	a.dmEnsureParticipant(tx, conv.ID, uid)
	a.dmEnsureParticipant(tx, conv.ID, req.PeerID)
	_ = tx.Model(&model.DmParticipant{}).
		Where("conversation_id = ? AND user_id = ?", conv.ID, uid).
		Update("hidden_at", nil).Error
	_ = tx.Commit().Error

	var part model.DmParticipant
	_ = a.DB.Where("conversation_id = ? AND user_id = ?", conv.ID, uid).First(&part).Error
	resp.OK(c, a.dmFormatConversation(&conv, uid, &part))
}

// DeleteDmConversation hides the thread for the current user (does not delete peer's copy).
func (a *API) DeleteDmConversation(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	convID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || convID == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var conv model.DmConversation
	if err := a.DB.First(&conv, convID).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if uid != conv.UserLow && uid != conv.UserHigh {
		resp.Err(c, http.StatusForbidden, errcode.CodeForbidden)
		return
	}
	if a.dmIsAgentConv(&conv) {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	tx := a.DB.Begin()
	a.dmEnsureParticipant(tx, convID, uid)
	now := time.Now()
	if err := tx.Model(&model.DmParticipant{}).
		Where("conversation_id = ? AND user_id = ?", convID, uid).
		Updates(map[string]interface{}{
			"hidden_at":     &now,
			"unread_count":  0,
			"pinned":        false,
			"pinned_at":     nil,
		}).Error; err != nil {
		tx.Rollback()
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if err := tx.Commit().Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"deleted": true, "conversation_id": convID})
}

// ResetDmAgentConversation POST /api/v1/dm/conversations/:id/reset — clear AI chat history and restart.
func (a *API) ResetDmAgentConversation(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	convID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || convID == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var conv model.DmConversation
	if err := a.DB.First(&conv, convID).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if uid != conv.UserLow && uid != conv.UserHigh {
		resp.Err(c, http.StatusForbidden, errcode.CodeForbidden)
		return
	}
	if !a.dmIsAgentConv(&conv) {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if a.Agent == nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	msg, err := a.Agent.ResetConversation(c.Request.Context(), &conv, uid)
	if err != nil {
		a.Log.Error("reset agent conversation", zap.Uint64("conv", convID), zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	senderName, senderAvatar := a.dmUserBrief(a.DB, msg.SenderID)
	out := a.dmFormatMessage(msg, senderName, senderAvatar)
	var part model.DmParticipant
	_ = a.DB.Where("conversation_id = ? AND user_id = ?", conv.ID, uid).First(&part).Error
	resp.OK(c, gin.H{
		"conversation":    a.dmFormatConversation(&conv, uid, &part),
		"welcome_message": out,
	})
}

// PatchDmConversationSettings updates pin / mute for the current user's participant row.
func (a *API) PatchDmConversationSettings(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	convID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || convID == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var req dmSettingsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if req.Pinned == nil && req.Muted == nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var conv model.DmConversation
	if err := a.DB.First(&conv, convID).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if uid != conv.UserLow && uid != conv.UserHigh {
		resp.Err(c, http.StatusForbidden, errcode.CodeForbidden)
		return
	}
	tx := a.DB.Begin()
	a.dmEnsureParticipant(tx, convID, uid)
	var part model.DmParticipant
	if err := tx.Where("conversation_id = ? AND user_id = ?", convID, uid).First(&part).Error; err != nil {
		tx.Rollback()
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	updates := map[string]interface{}{}
	if req.Pinned != nil {
		if *req.Pinned {
			// 同时仅允许一个置顶会话：新置顶时取消其余会话置顶
			if err := tx.Model(&model.DmParticipant{}).
				Where("user_id = ? AND conversation_id != ?", uid, convID).
				Updates(map[string]interface{}{
					"pinned":    false,
					"pinned_at": nil,
				}).Error; err != nil {
				tx.Rollback()
				resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
				return
			}
		}
		updates["pinned"] = *req.Pinned
		if *req.Pinned {
			now := time.Now()
			updates["pinned_at"] = &now
		} else {
			updates["pinned_at"] = nil
		}
	}
	if req.Muted != nil {
		updates["muted"] = *req.Muted
	}
	if err := tx.Model(&part).Updates(updates).Error; err != nil {
		tx.Rollback()
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if err := tx.Commit().Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	_ = a.DB.Where("conversation_id = ? AND user_id = ?", convID, uid).First(&part).Error
	resp.OK(c, a.dmFormatConversation(&conv, uid, &part))
}

// ListDmMessages lists messages in a conversation (ASC by id).
func (a *API) ListDmMessages(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	convID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || convID == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var conv model.DmConversation
	if err := a.DB.First(&conv, convID).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if uid != conv.UserLow && uid != conv.UserHigh {
		resp.Err(c, http.StatusForbidden, errcode.CodeForbidden)
		return
	}
	peer := dmPeerID(&conv, uid)
	if !a.dmIsAgentConv(&conv) && dmUsersBlocked(a.DB, uid, peer) {
		resp.Err(c, http.StatusForbidden, errcode.CodeUserBlocked)
		return
	}
	limit := 50
	if s := c.Query("limit"); s != "" {
		if n, e := strconv.Atoi(s); e == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	curID, _ := strconv.ParseUint(c.Query("cursor"), 10, 64)
	q := a.DB.Model(&model.DmMessage{}).Where("conversation_id = ?", convID)
	if curID > 0 {
		q = q.Where("id < ?", curID)
	}
	var list []model.DmMessage
	if err := q.Order("id DESC").Limit(limit + 1).Find(&list).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	hasMore := len(list) > limit
	if hasMore {
		list = list[:limit]
	}
	// Return chronological order for UI.
	for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
		list[i], list[j] = list[j], list[i]
	}
	senderCache := map[uint64]struct {
		name   string
		avatar string
	}{}
	items := make([]gin.H, 0, len(list))
	for i := range list {
		m := &list[i]
		sc, ok := senderCache[m.SenderID]
		if !ok {
			sc.name, sc.avatar = a.dmUserBrief(a.DB, m.SenderID)
			senderCache[m.SenderID] = sc
		}
		items = append(items, a.dmFormatMessage(m, sc.name, sc.avatar))
	}
	next := ""
	if hasMore && len(list) > 0 {
		next = strconv.FormatUint(list[0].ID, 10)
	}
	_ = a.DB.Model(&model.DmParticipant{}).
		Where("conversation_id = ? AND user_id = ?", convID, uid).
		Update("unread_count", 0).Error
	peerName, peerAvatar := a.dmUserBrief(a.DB, peer)
	resp.OK(c, gin.H{
		"items":       items,
		"next_cursor": next,
		"peer_id":     peer,
		"peer_name":   peerName,
		"peer_avatar": peerAvatar,
	})
}

// PostDmMessage sends a message and pushes to participants via WebSocket.
func (a *API) PostDmMessage(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	convID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || convID == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var req dmPostMessageReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	content := strings.TrimSpace(req.Content)
	if n := utf8.RuneCountInString(content); n < 1 || n > dmMaxContentRunes {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var conv model.DmConversation
	if err := a.DB.First(&conv, convID).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if uid != conv.UserLow && uid != conv.UserHigh {
		resp.Err(c, http.StatusForbidden, errcode.CodeForbidden)
		return
	}
	peer := dmPeerID(&conv, uid)
	isAgent := a.dmIsAgentConv(&conv)
	if !isAgent && dmUsersBlocked(a.DB, uid, peer) {
		resp.Err(c, http.StatusForbidden, errcode.CodeUserBlocked)
		return
	}
	now := time.Now()
	msg := model.DmMessage{
		ConversationID: convID,
		SenderID:       uid,
		Role:           "user",
		Content:        content,
		CreatedAt:      now,
	}
	tx := a.DB.Begin()
	if err := tx.Create(&msg).Error; err != nil {
		tx.Rollback()
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if err := tx.Model(&conv).Updates(map[string]interface{}{
		"last_message_at": now,
		"last_preview":    dmTrimPreview(content),
	}).Error; err != nil {
		tx.Rollback()
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	a.dmEnsureParticipant(tx, convID, uid)
	if !isAgent {
		a.dmEnsureParticipant(tx, convID, peer)
	}
	if !isAgent {
		if err := tx.Model(&model.DmParticipant{}).
			Where("conversation_id = ? AND user_id = ?", convID, peer).
			Updates(map[string]interface{}{
				"unread_count": gorm.Expr("unread_count + ?", 1),
				"hidden_at":    nil,
			}).Error; err != nil {
			tx.Rollback()
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
	}
	if err := tx.Commit().Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	_ = a.DB.First(&conv, convID).Error
	senderName, senderAvatar := a.dmUserBrief(a.DB, uid)
	out := a.dmFormatMessage(&msg, senderName, senderAvatar)
	var selfPart model.DmParticipant
	_ = a.DB.Where("conversation_id = ? AND user_id = ?", convID, uid).First(&selfPart).Error
	convPayload := a.dmFormatConversation(&conv, uid, &selfPart)
	var pp model.DmParticipant
	_ = a.DB.Where("conversation_id = ? AND user_id = ?", convID, peer).First(&pp).Error
	peerConv := a.dmFormatConversation(&conv, peer, &pp)

	event := gin.H{"type": "dm_message", "message": out}
	a.dmPushEvent(uid, event)
	if !isAgent && !pp.Muted {
		a.dmPushEvent(peer, event)
	}
	a.dmPushEvent(uid, gin.H{"type": "dm_conversation", "conversation": convPayload})
	if !isAgent && !pp.Muted {
		a.dmPushEvent(peer, gin.H{"type": "dm_conversation", "conversation": peerConv})
	}

	if isAgent {
		convID := conv.ID
		humanID := uid
		userContent := content
		go func() {
			var c model.DmConversation
			if err := a.DB.First(&c, convID).Error; err != nil {
				return
			}
			a.runAgentReply(humanID, &c, userContent)
		}()
	}

	resp.OK(c, out)
}
