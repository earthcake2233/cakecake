package handler

import (
	"minibili/internal/model/agent"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"minibili/internal/data"
	"minibili/internal/errcode"
	"minibili/internal/pkg/coverval"
	"minibili/internal/pkg/resp"
)

func (a *API) adminAgentMeta() gin.H {
	return gin.H{
		"deepseek_configured": a.Cfg != nil && strings.TrimSpace(a.Cfg.DeepSeekAPIKey) != "",
		"max_profiles":        data.MaxAgentProfilesLimit(),
	}
}

func adminAgentProfilePayload(p *agent.AgentProfile, globalPrompt string) gin.H {
	if p == nil {
		return gin.H{}
	}
	welcome := agent.ParseWelcomeMessages(p.WelcomeMessagesJSON)
	return gin.H{
		"id":                p.ID,
		"slug":              p.Slug,
		"bot_user_id":       p.BotUserID,
		"display_name":      p.DisplayName,
		"avatar_url":        p.AvatarURL,
		"sign":              p.Sign,
		"system_prompt":     p.SystemPrompt,
		"welcome_messages":  welcome,
		"sort_order":        p.SortOrder,
		"enabled":           p.Enabled,
		"updated_at":        p.UpdatedAt.Format("2006-01-02 15:04:05"),
		"global_system_prompt": globalPrompt,
	}
}

// AdminListAgentProfiles GET /api/v1/admin/agent-profiles
func (a *API) AdminListAgentProfiles(c *gin.Context) {
	list, err := a.Agent.ListAgentProfiles(c.Request.Context())
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	gp := a.Agent.GetGlobalSystemPrompt(c.Request.Context())
	items := make([]gin.H, 0, len(list))
	for i := range list {
		items = append(items, adminAgentProfilePayload(&list[i], gp))
	}
	out := a.adminAgentMeta()
	out["items"] = items
	resp.OK(c, out)
}

type adminAgentProfileWriteReq struct {
	Slug             string          `json:"slug"`
	DisplayName      string          `json:"display_name"`
	AvatarURL        string          `json:"avatar_url"`
	Sign             string          `json:"sign"`
	SystemPrompt     string          `json:"system_prompt"`
	WelcomeMessages  json.RawMessage `json:"welcome_messages"`
	SortOrder        *int            `json:"sort_order"`
	Enabled          *bool           `json:"enabled"`
}

func (a *API) validateAgentProfileWrite(req *adminAgentProfileWriteReq, isCreate bool) (slug, welcomeJSON string, code int) {
	if req == nil {
		return "", "", errcode.CodeParamError
	}
	displayName := strings.TrimSpace(req.DisplayName)
	systemPrompt := strings.TrimSpace(req.SystemPrompt)
	if utf8.RuneCountInString(displayName) < 1 || utf8.RuneCountInString(displayName) > 64 {
		return "", "", errcode.CodeParamError
	}
	if utf8.RuneCountInString(systemPrompt) < 10 || utf8.RuneCountInString(systemPrompt) > 12000 {
		return "", "", errcode.CodeParamError
	}
	if utf8.RuneCountInString(strings.TrimSpace(req.Sign)) > 500 {
		return "", "", errcode.CodeParamError
	}
	if utf8.RuneCountInString(strings.TrimSpace(req.AvatarURL)) > 1024 {
		return "", "", errcode.CodeParamError
	}
	welcomeList, err := data.UnmarshalWelcomeList(req.WelcomeMessages, nil)
	if err != nil || len(welcomeList) == 0 {
		return "", "", errcode.CodeParamError
	}
	for _, w := range welcomeList {
		if utf8.RuneCountInString(w) > 500 {
			return "", "", errcode.CodeParamError
		}
	}
	welcomeJSON = agent.EncodeWelcomeMessages(welcomeList)
	slug = strings.TrimSpace(req.Slug)
	if isCreate {
		slug, err = data.NormalizeAgentSlug(slug)
		if err != nil {
			return "", "", errcode.CodeParamError
		}
	} else if slug != "" {
		slug, err = data.NormalizeAgentSlug(slug)
		if err != nil {
			return "", "", errcode.CodeParamError
		}
	}
	return slug, welcomeJSON, 0
}

// AdminCreateAgentProfile POST /api/v1/admin/agent-profiles
func (a *API) AdminCreateAgentProfile(c *gin.Context) {
	ctx := c.Request.Context()
	cnt, _ := a.Agent.ProfileCount(ctx)
	if cnt >= int64(data.MaxAgentProfilesLimit()) {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var req adminAgentProfileWriteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	slug, welcomeJSON, code := a.validateAgentProfileWrite(&req, true)
	if code != 0 {
		resp.Err(c, http.StatusBadRequest, code)
		return
	}
	exists, _ := a.Agent.CheckAgentSlugExists(ctx, slug)
	if exists {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	displayName := strings.TrimSpace(req.DisplayName)
	sign := strings.TrimSpace(req.Sign)
	avatarURL := strings.TrimSpace(req.AvatarURL)
	botID, err := a.Agent.CreateAgentBotUser(ctx, slug, displayName, sign, avatarURL)
	if err != nil {
		a.Log.Error("create agent bot user", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	sortOrder := int(cnt)
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	p := agent.AgentProfile{
		Slug:                slug,
		BotUserID:           botID,
		DisplayName:         displayName,
		AvatarURL:           avatarURL,
		Sign:                sign,
		SystemPrompt:        strings.TrimSpace(req.SystemPrompt),
		WelcomeMessagesJSON: welcomeJSON,
		SortOrder:           sortOrder,
		Enabled:             enabled,
	}
	if err := a.Agent.CreateAgentProfile(ctx, &p); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, adminAgentProfilePayload(&p, a.Agent.GetGlobalSystemPrompt(ctx)))
}

// AdminUpdateAgentProfile PUT /api/v1/admin/agent-profiles/:id
func (a *API) AdminUpdateAgentProfile(c *gin.Context) {
	ctx := c.Request.Context()
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	p, err := a.Agent.GetAgentProfile(ctx, id)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	var req adminAgentProfileWriteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	newSlug, welcomeJSON, code := a.validateAgentProfileWrite(&req, false)
	if code != 0 {
		resp.Err(c, http.StatusBadRequest, code)
		return
	}
	updates := map[string]interface{}{}
	if v := strings.TrimSpace(req.DisplayName); v != "" {
		updates["display_name"] = v
	}
	if v := strings.TrimSpace(req.Sign); v != "" {
		updates["sign"] = v
	}
	if v := strings.TrimSpace(req.AvatarURL); v != "" {
		updates["avatar_url"] = v
	}
	if v := strings.TrimSpace(req.SystemPrompt); v != "" {
		updates["system_prompt"] = v
	}
	updates["welcome_messages_json"] = welcomeJSON
	if req.SortOrder != nil {
		updates["sort_order"] = *req.SortOrder
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if newSlug != "" && newSlug != p.Slug {
		if err := a.Agent.RenameAgentProfileSlug(ctx, p, newSlug); err != nil {
			resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
			return
		}
	}
	if len(updates) > 0 {
		if err := a.Agent.UpdateAgentProfile(ctx, id, updates); err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
	}
	p2, _ := a.Agent.GetAgentProfile(ctx, id)
	_ = a.Agent.SyncAgentProfile(ctx, p2)
	if a.Agent != nil {
		a.Agent.ReloadProfiles()
	}
	resp.OK(c, adminAgentProfilePayload(p2, a.Agent.GetGlobalSystemPrompt(ctx)))
}

// AdminDeleteAgentProfile DELETE /api/v1/admin/agent-profiles/:id
func (a *API) AdminDeleteAgentProfile(c *gin.Context) {
	ctx := c.Request.Context()
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	cnt, _ := a.Agent.CountActiveAgentProfiles(ctx)
	if cnt <= 1 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if err := a.Agent.DeleteAgentProfile(ctx, id); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if a.Agent != nil {
		a.Agent.ReloadProfiles()
	}
	resp.OK(c, gin.H{"deleted": true})
}

// AdminUploadAgentProfileAvatar POST /api/v1/admin/agent-profiles/:id/avatar
func (a *API) AdminUploadAgentProfileAvatar(c *gin.Context) {
	ctx := c.Request.Context()
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	p, err := a.Agent.GetAgentProfile(ctx, id)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if err := c.Request.ParseMultipartForm(12 << 20); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	fh, err := c.FormFile("image")
	if err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	oldAvatar := strings.TrimSpace(p.AvatarURL)
	url, code := a.uploadAgentProfileAvatarToOSS(fh, p.Slug)
	if code != 0 {
		resp.Err(c, http.StatusBadRequest, code)
		return
	}
	if err := a.Agent.UpdateAgentAvatar(ctx, id, url); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	p2, _ := a.Agent.GetAgentProfile(ctx, id)
	_ = a.Agent.SyncAgentProfile(ctx, p2)
	if agentAvatarURLChanged(oldAvatar, url) {
		purgeAgentAvatarOSS(a.Cfg, a.OSS, a.Log, oldAvatar)
	}
	gp := a.Agent.GetGlobalSystemPrompt(ctx)
	resp.OK(c, gin.H{"avatar_url": url, "profile": adminAgentProfilePayload(p2, gp)})
}

func (a *API) uploadAgentProfileAvatarToOSS(fh *multipart.FileHeader, slug string) (string, int) {
	if fh == nil {
		return "", errcode.CodeParamError
	}
	if code := coverval.ValidateCoverHeader(fh); code != 0 {
		return "", code
	}
	if a.OSS == nil {
		return "", errcode.CodeInternalError
	}
	key := fmt.Sprintf("agent/%s/avatar-%s.%s", slug, uuid.NewString(), bannerImageExt(fh))
	return a.uploadBannerImageToOSS(fh, key)
}

// Legacy singleton endpoints (compat): map to first profile.

func (a *API) AdminGetAgentSettings(c *gin.Context) {
	ctx := c.Request.Context()
	list, err := a.Agent.ListAgentProfiles(ctx)
	if err != nil || len(list) == 0 {
		if err := a.Agent.EnsureAgentProfiles(ctx); err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		list, _ = a.Agent.ListAgentProfiles(ctx)
	}
	if len(list) == 0 {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	p := list[0]
	welcome := agent.ParseWelcomeMessages(p.WelcomeMessagesJSON)
	welcomeOne := ""
	if len(welcome) > 0 {
		welcomeOne = welcome[0]
	}
	resp.OK(c, gin.H{
		"display_name":        p.DisplayName,
		"avatar_url":          p.AvatarURL,
		"sign":                p.Sign,
		"system_prompt":       p.SystemPrompt,
		"welcome_message":     welcomeOne,
		"assistant_enabled":   p.Enabled,
		"bot_user_id":         p.BotUserID,
		"updated_at":          p.UpdatedAt.Format("2006-01-02 15:04:05"),
		"deepseek_configured": a.Cfg != nil && strings.TrimSpace(a.Cfg.DeepSeekAPIKey) != "",
	})
}

func (a *API) AdminPutAgentSettings(c *gin.Context) {
	ctx := c.Request.Context()
	list, _ := a.Agent.ListAgentProfiles(ctx)
	if len(list) == 0 {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	c.Params = append(c.Params, gin.Param{Key: "id", Value: strconv.FormatUint(list[0].ID, 10)})
	var req adminAgentSettingsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	welcomeRaw, _ := json.Marshal([]string{strings.TrimSpace(req.WelcomeMessage)})
	write := adminAgentProfileWriteReq{
		DisplayName:     req.DisplayName,
		AvatarURL:       req.AvatarURL,
		Sign:            req.Sign,
		SystemPrompt:    req.SystemPrompt,
		WelcomeMessages: welcomeRaw,
	}
	if req.AssistantEnabled != nil {
		write.Enabled = req.AssistantEnabled
	}
	_, welcomeJSON, code := a.validateAgentProfileWrite(&write, false)
	if code != 0 {
		resp.Err(c, http.StatusBadRequest, code)
		return
	}
	p := list[0]
	updates := map[string]interface{}{
		"display_name":          strings.TrimSpace(req.DisplayName),
		"avatar_url":            strings.TrimSpace(req.AvatarURL),
		"sign":                  strings.TrimSpace(req.Sign),
		"system_prompt":         strings.TrimSpace(req.SystemPrompt),
		"welcome_messages_json": welcomeJSON,
	}
	if req.AssistantEnabled != nil {
		updates["enabled"] = *req.AssistantEnabled
	}
	if err := a.Agent.UpdateAgentProfile(ctx, p.ID, updates); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	p2, _ := a.Agent.GetAgentProfile(ctx, p.ID)
	_ = a.Agent.SyncAgentProfile(ctx, p2)
	a.AdminGetAgentSettings(c)
}

type adminAgentSettingsReq struct {
	DisplayName      string `json:"display_name"`
	AvatarURL        string `json:"avatar_url"`
	Sign             string `json:"sign"`
	SystemPrompt     string `json:"system_prompt"`
	WelcomeMessage   string `json:"welcome_message"`
	AssistantEnabled *bool  `json:"assistant_enabled"`
}

func (a *API) AdminUploadAgentAvatar(c *gin.Context) {
	list, _ := a.Agent.ListAgentProfiles(c.Request.Context())
	if len(list) == 0 {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	c.Params = append(c.Params, gin.Param{Key: "id", Value: strconv.FormatUint(list[0].ID, 10)})
	a.AdminUploadAgentProfileAvatar(c)
}
