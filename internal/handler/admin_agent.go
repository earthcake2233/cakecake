package handler

import (
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
	"minibili/internal/model"
	"minibili/internal/pkg/coverval"
	"minibili/internal/pkg/resp"
)

func (a *API) adminAgentMeta() gin.H {
	return gin.H{
		"deepseek_configured": a.Cfg != nil && strings.TrimSpace(a.Cfg.DeepSeekAPIKey) != "",
		"max_profiles":        data.MaxAgentProfilesLimit(),
	}
}

func adminAgentProfilePayload(p *model.AgentProfile) gin.H {
	if p == nil {
		return gin.H{}
	}
	welcome := model.ParseWelcomeMessages(p.WelcomeMessagesJSON)
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
	}
}

// AdminListAgentProfiles GET /api/v1/admin/agent-profiles
func (a *API) AdminListAgentProfiles(c *gin.Context) {
	list, err := data.ListAgentProfiles(a.DB)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	items := make([]gin.H, 0, len(list))
	for i := range list {
		items = append(items, adminAgentProfilePayload(&list[i]))
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
	welcomeJSON = model.EncodeWelcomeMessages(welcomeList)
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
	cnt, _ := data.ProfileCount(a.DB)
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
	var exists int64
	_ = a.DB.Model(&model.AgentProfile{}).Where("slug = ?", slug).Count(&exists).Error
	if exists > 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	displayName := strings.TrimSpace(req.DisplayName)
	sign := strings.TrimSpace(req.Sign)
	avatarURL := strings.TrimSpace(req.AvatarURL)
	botID, err := data.CreateAgentBotUser(a.DB, slug, displayName, sign, avatarURL)
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
	p := model.AgentProfile{
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
	if err := a.DB.Create(&p).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, adminAgentProfilePayload(&p))
}

// AdminUpdateAgentProfile PUT /api/v1/admin/agent-profiles/:id
func (a *API) AdminUpdateAgentProfile(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var req adminAgentProfileWriteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	slug, welcomeJSON, code := a.validateAgentProfileWrite(&req, false)
	if code != 0 {
		resp.Err(c, http.StatusBadRequest, code)
		return
	}
	p, err := data.GetAgentProfile(a.DB, id)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if slug != "" && slug != p.Slug {
		if err := data.RenameAgentProfileSlug(a.DB, p, slug); err != nil {
			resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
			return
		}
		_ = a.DB.First(p, id).Error
	}
	oldAvatar := strings.TrimSpace(p.AvatarURL)
	newAvatar := strings.TrimSpace(req.AvatarURL)
	updates := map[string]interface{}{
		"display_name":          strings.TrimSpace(req.DisplayName),
		"avatar_url":            newAvatar,
		"sign":                  strings.TrimSpace(req.Sign),
		"system_prompt":         strings.TrimSpace(req.SystemPrompt),
		"welcome_messages_json": welcomeJSON,
	}
	if req.SortOrder != nil {
		updates["sort_order"] = *req.SortOrder
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if err := a.DB.Model(p).Updates(updates).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	_ = a.DB.First(p, id).Error
	if agentAvatarURLChanged(oldAvatar, newAvatar) {
		purgeAgentAvatarOSS(a.Cfg, a.OSS, a.Log, oldAvatar)
	}
	_ = data.SyncAgentProfile(a.DB, p)
	if a.Agent != nil {
		a.Agent.ReloadProfiles()
	}
	resp.OK(c, adminAgentProfilePayload(p))
}

// AdminDeleteAgentProfile DELETE /api/v1/admin/agent-profiles/:id — soft disable.
func (a *API) AdminDeleteAgentProfile(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var cnt int64
	_ = a.DB.Model(&model.AgentProfile{}).Where("enabled = ?", true).Count(&cnt).Error
	p, err := data.GetAgentProfile(a.DB, id)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if cnt <= 1 && p.Enabled {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if err := a.DB.Model(p).Update("enabled", false).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"disabled": true, "id": id})
}

// AdminUploadAgentProfileAvatar POST /api/v1/admin/agent-profiles/:id/avatar
func (a *API) AdminUploadAgentProfileAvatar(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	p, err := data.GetAgentProfile(a.DB, id)
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
	if err := a.DB.Model(p).Update("avatar_url", url).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	_ = a.DB.First(p, id).Error
	if agentAvatarURLChanged(oldAvatar, url) {
		purgeAgentAvatarOSS(a.Cfg, a.OSS, a.Log, oldAvatar)
	}
	_ = data.SyncAgentProfile(a.DB, p)
	resp.OK(c, gin.H{"avatar_url": url, "profile": adminAgentProfilePayload(p)})
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
	list, err := data.ListAgentProfiles(a.DB)
	if err != nil || len(list) == 0 {
		if err := data.EnsureAgentProfiles(a.DB, a.Cfg, a.Log); err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		list, _ = data.ListAgentProfiles(a.DB)
	}
	if len(list) == 0 {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	p := list[0]
	welcome := model.ParseWelcomeMessages(p.WelcomeMessagesJSON)
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
	list, _ := data.ListAgentProfiles(a.DB)
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
	_ = a.DB.Model(&p).Updates(updates).Error
	_ = a.DB.First(&p, p.ID).Error
	_ = data.SyncAgentProfile(a.DB, &p)
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
	list, _ := data.ListAgentProfiles(a.DB)
	if len(list) == 0 {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	c.Params = append(c.Params, gin.Param{Key: "id", Value: strconv.FormatUint(list[0].ID, 10)})
	a.AdminUploadAgentProfileAvatar(c)
}
