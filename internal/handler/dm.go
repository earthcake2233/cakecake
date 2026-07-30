package handler

import (
	"minibili/internal/model/dm"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"minibili/internal/errcode"
	"minibili/internal/middleware"

	"context"
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

func dmPeerID(conv *dm.DmConversation, self uint64) uint64 {
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
	return a.DmSvc.UnreadTotal(context.Background(), uid)
}

func (a *API) dmUserBrief(ctx context.Context, uid uint64) (name, avatar string) {
	name, avatar, err := a.UserSvc.GetUserBrief(ctx, uid)
	if err != nil {
		return "用户", ""
	}
	return name, avatar
}

func (a *API) dmFormatMessage(m *dm.DmMessage, senderName, senderAvatar string) gin.H {
	role := m.Role
	if role == "" {
		role = "user"
	}
	return gin.H{
		"id":                m.ID,
		"conversation_id":   m.ConversationID,
		"sender_id":         m.SenderID,
		"sender_name":       senderName,
		"sender_avatar":     senderAvatar,
		"content":           m.Content,
		"role":              role,
		"tool_activities":   m.ToolActivities,
		"tool_result_data":  m.ToolResultData,
		"created_at":        m.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

func (a *API) dmFormatConversation(conv *dm.DmConversation, self uint64, part *dm.DmParticipant) gin.H {
	peer := dmPeerID(conv, self)
	name, avatar := a.dmUserBrief(context.Background(), peer)
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
		kind = dm.DmKindHuman
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

func (a *API) dmPushEvent(userID uint64, payload gin.H) {
	if a.ChatHub == nil || userID == 0 {
		return
	}
	a.ChatHub.PushJSON(userID, payload)
}

// ListDmConversations returns recent 1:1 threads for the current user.
// ListDmConversations godoc
// @Summary      List DM conversations
// @Description  Get all direct message conversations for the current user
// @Tags         DM
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Router       /dm/conversations [get]
func (a *API) ListDmConversations(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	a.ensureAgentConversationFor(uid)
	convs, parts, err := a.DmSvc.ListConversations(context.Background(), uid)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	partMap := map[uint64]*dm.DmParticipant{}
	for i := range parts {
		p := parts[i]
		partMap[p.ConversationID] = &p
	}
	sort.Slice(convs, func(i, j int) bool {
		pi, pj := partMap[convs[i].ID], partMap[convs[j].ID]
		pinI := pi != nil && pi.Pinned
		pinJ := pj != nil && pj.Pinned
		if pinI != pinJ { return pinI }
		if pinI && pinJ { return dmPinnedAtAfter(pi.PinnedAt, pj.PinnedAt) }
		return convs[i].LastMessageAt.After(convs[j].LastMessageAt)
	})
	items := make([]gin.H, 0, len(convs))
	for i := range convs {
		conv := &convs[i]
		part := partMap[conv.ID]
		if part != nil && part.HiddenAt != nil { continue }
		items = append(items, a.dmFormatConversation(conv, uid, part))
	}
	resp.OK(c, gin.H{"items": items})
}

// CreateDmConversation finds or creates a thread with peer_id.
// CreateDmConversation godoc
// @Summary      Create a DM conversation
// @Description  Start a new direct message conversation with another user
// @Tags         DM
// @Produce      json
// @Param        body body object{participant_id=int} true "Other participant user ID"
// @Success      200 {object} map[string]interface{}
// @Router       /dm/conversations [post]
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
		conv, err := a.DmSvc.FindConversationByUserIDs(context.Background(), uid, req.PeerID)
		if err != nil {
			resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
			return
		}
		part, _ := a.DmSvc.GetParticipant(context.Background(), conv.ID, uid)
		resp.OK(c, a.dmFormatConversation(conv, uid, part))
		return
	}
	// Verify peer exists and is not anonymized
	if _, _, err := a.UserSvc.GetUserBrief(context.Background(), req.PeerID); err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	blocked, err := a.FollowSvc.UsersBlocked(context.Background(), uid, req.PeerID)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if blocked {
		resp.Err(c, http.StatusForbidden, errcode.CodeUserBlocked)
		return
	}
	conv, part, err := a.DmSvc.GetOrCreateConversation(context.Background(), uid, req.PeerID)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, a.dmFormatConversation(conv, uid, part))
}

// DeleteDmConversation hides the thread for the current user (does not delete peer's copy).
// DeleteDmConversation godoc
// @Summary      Delete a DM conversation
// @Description  Remove a direct message conversation
// @Tags         DM
// @Produce      json
// @Param        id path int true "Conversation ID"
// @Success      200 {object} map[string]interface{}
// @Router       /dm/conversations/{id} [delete]
func (a *API) DeleteDmConversation(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	convID, err := strconv.ParseUint(c.Param("convId"), 10, 64)
	if err != nil || convID == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if err := a.DmSvc.DeleteConversation(context.Background(), convID, uid); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// ResetDmAgentConversation POST /api/v1/dm/conversations/:id/reset — clear AI chat history and restart.
// ResetDmAgentConversation godoc
// @Summary      Reset AI agent conversation
// @Description  Clear the conversation history for an AI agent DM
// @Tags         DM
// @Produce      json
// @Param        id path int true "Conversation ID"
// @Success      200 {object} map[string]interface{}
// @Router       /dm/conversations/{id}/reset [post]
func (a *API) ResetDmAgentConversation(c *gin.Context) {
	_, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	convID, err := strconv.ParseUint(c.Param("convId"), 10, 64)
	if err != nil || convID == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if err := a.DmSvc.ResetConversationForAgent(context.Background(), convID); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// PatchDmConversationSettings updates pin / mute for the current user's participant row.
// PatchDmConversationSettings godoc
// @Summary      Update DM conversation settings
// @Description  Update settings (e.g. agent config) for a conversation
// @Tags         DM
// @Produce      json
// @Param        id path int true "Conversation ID"
// @Param        body body object{} true "Settings"
// @Success      200 {object} map[string]interface{}
// @Router       /dm/conversations/{id}/settings [patch]
func (a *API) PatchDmConversationSettings(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	convID, err := strconv.ParseUint(c.Param("convId"), 10, 64)
	if err != nil || convID == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var body dmPatchSettingsJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	updates := make(map[string]interface{})
	if body.Pinned != nil { updates["pinned"] = *body.Pinned }
	if body.Hidden != nil { updates["hidden_at"] = time.Now() }
	if body.Unhidden != nil { updates["hidden_at"] = nil }
	if err := a.DmSvc.UpdateConversationSettings(context.Background(), convID, uid, updates); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// ListDmMessages lists messages in a conversation (ASC by id).
// ListDmMessages godoc
// @Summary      List messages in a conversation
// @Description  Get paginated messages for a DM conversation
// @Tags         DM
// @Produce      json
// @Param        id path int true "Conversation ID"
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Page size" default(50)
// @Success      200 {object} map[string]interface{}
// @Router       /dm/conversations/{id}/messages [get]
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
	conv, err := a.DmSvc.GetConversationByID(context.Background(), convID)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if uid != conv.UserLow && uid != conv.UserHigh {
		resp.Err(c, http.StatusForbidden, errcode.CodeForbidden)
		return
	}
	peer := dmPeerID(conv, uid)
	if !a.dmIsAgentConv(conv) {
		blocked, err := a.FollowSvc.UsersBlocked(context.Background(), uid, peer)
		if err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		if blocked {
			resp.Err(c, http.StatusForbidden, errcode.CodeUserBlocked)
			return
		}
	}
	limit := 50
	if s := c.Query("limit"); s != "" {
		if n, e := strconv.Atoi(s); e == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	curID, _ := strconv.ParseUint(c.Query("cursor"), 10, 64)
	list, err := a.DmSvc.ListMessages(context.Background(), convID, curID, limit+1)
	if err != nil {
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
			sc.name, sc.avatar, _ = a.UserSvc.GetUserBrief(context.Background(), m.SenderID)
			senderCache[m.SenderID] = sc
		}
		items = append(items, a.dmFormatMessage(m, sc.name, sc.avatar))
	}
	next := ""
	if hasMore && len(list) > 0 {
		next = strconv.FormatUint(list[0].ID, 10)
	}
	_ = a.DmSvc.MarkConversationRead(context.Background(), convID, uid)
	peerName, peerAvatar := a.dmUserBrief(context.Background(), peer)
	resp.OK(c, gin.H{
		"items":       items,
		"next_cursor": next,
		"peer_id":     peer,
		"peer_name":   peerName,
		"peer_avatar": peerAvatar,
	})
}

// PostDmMessage sends a message and pushes to participants via WebSocket.
// PostDmMessage godoc
// @Summary      Send a message in a conversation
// @Description  Post a new message to a DM conversation
// @Tags         DM
// @Produce      json
// @Param        id path int true "Conversation ID"
// @Param        body body object{content=string} true "Message content"
// @Success      200 {object} map[string]interface{}
// @Router       /dm/conversations/{id}/messages [post]
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
	conv, err := a.DmSvc.GetConversationByID(context.Background(), convID)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if uid != conv.UserLow && uid != conv.UserHigh {
		resp.Err(c, http.StatusForbidden, errcode.CodeForbidden)
		return
	}
	peer := dmPeerID(conv, uid)
	isAgent := a.dmIsAgentConv(conv)
	if !isAgent {
		blocked, err := a.FollowSvc.UsersBlocked(context.Background(), uid, peer)
		if err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		if blocked {
			resp.Err(c, http.StatusForbidden, errcode.CodeUserBlocked)
			return
		}
	}
	result, err := a.DmSvc.PostMessage(context.Background(), convID, uid, peer, content, dmTrimPreview(content), isAgent)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	senderName, senderAvatar := a.dmUserBrief(context.Background(), uid)
	out := a.dmFormatMessage(result.Message, senderName, senderAvatar)
	convPayload := a.dmFormatConversation(result.Conversation, uid, result.SelfPart)
	var peerConv gin.H
	if !isAgent && result.PeerPart != nil {
		peerConv = a.dmFormatConversation(result.Conversation, peer, result.PeerPart)
	}
	event := gin.H{"type": "dm_message", "message": out}
	a.dmPushEvent(uid, event)
	if !isAgent && result.PeerPart != nil && !result.PeerPart.Muted {
		a.dmPushEvent(peer, event)
	}
	a.dmPushEvent(uid, gin.H{"type": "dm_conversation", "conversation": convPayload})
	if !isAgent && result.PeerPart != nil && !result.PeerPart.Muted {
		a.dmPushEvent(peer, gin.H{"type": "dm_conversation", "conversation": peerConv})
	}
	if isAgent {
		convCopy := result.Conversation
		userContent := content
		go func() {
			a.runAgentReply(uid, convCopy, userContent)
		}()
	}
	resp.OK(c, out)
}

type dmPatchSettingsJSON struct {
	Pinned   *bool `json:"pinned,omitempty"`
	Hidden   *bool `json:"hidden,omitempty"`
	Unhidden *bool `json:"unhidden,omitempty"`
}
