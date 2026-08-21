package handler

import (
	"fmt"
	"net/http"
	"path/filepath"

	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"cakecake/internal/errcode"
	"cakecake/internal/middleware"
	"cakecake/internal/pkg/coverval"
	"cakecake/internal/pkg/resp"
	"cakecake/internal/pkg/usercoin"
	"cakecake/internal/pkg/userlevel"
)

type meResponse struct {
	UserID       uint64 `json:"user_id"`
	Username     string `json:"username"`
	CakeID       string `json:"cake_id"`
	Nickname     string `json:"nickname"`
	Sign         string `json:"sign"`
	Announcement string `json:"announcement"`
	Gender       string `json:"gender"`
	Birthday     string `json:"birthday"`
	AvatarURL    string `json:"avatar_url"`
	CreatedAt    string `json:"created_at"`
	// PendingDeletion is true during the account-deletion cooling-off period.
	PendingDeletion     bool    `json:"pending_deletion,omitempty"`
	DeletionEffectiveAt *string `json:"deletion_effective_at,omitempty"`

	// Extra fields only present for normal (non-anonymized) users.
	SpacePrivacy *spacePrivacyPayload `json:"space_privacy,omitempty"`
	LevelInfo    *userlevel.Info      `json:"level_info,omitempty"`
	CoinBalance  *float64             `json:"coin_balance,omitempty"`
}

type updateMeProfileResponse struct {
	Nickname string `json:"nickname"`
	Sign     string `json:"sign"`
	Gender   string `json:"gender"`
	Birthday string `json:"birthday"`
	CakeID   string `json:"cake_id"`
}

type announcementResponse struct {
	Announcement string `json:"announcement"`
}

// GetMe returns the caller's own profile.
func (a *API) GetMe(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	_ = maybeFinalizeAccountDeletion(c.Request.Context(), a, uid)
	profile, err := a.UserSvc.GetMe(c.Request.Context(), uid)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if a.DailyRewardSvc != nil {
		_ = a.DailyRewardSvc.MarkLogin(uid)
	}
	g := normalizeGender(profile.Gender)
	out := meResponse{
		UserID:       profile.ID,
		Username:     profile.Username,
		CakeID:       strings.TrimSpace(profile.CakeID),
		Nickname:     profile.Nickname,
		Sign:         profile.Sign,
		Announcement: strings.TrimSpace(profile.Announcement),
		Gender:       g,
		Birthday:     strings.TrimSpace(profile.Birthday),
		AvatarURL:    profile.AvatarURL,
		CreatedAt:    profile.CreatedAt,
	}
	// Add extra fields expected by integration tests
	userModel, _ := a.UserSvc.GetUserByID(c.Request.Context(), uid)
	if userModel != nil {
		sp := spacePrivacyFromUser(userModel)
		li := userlevel.FromExperience(userModel.Experience)
		cb := usercoin.BalanceFloat(userModel.CoinBalanceTenths)
		out.SpacePrivacy = &sp
		out.LevelInfo = &li
		out.CoinBalance = &cb
		if userModel.DeletionRequestedAt != nil && userModel.DeletionEffectiveAt != nil &&
			time.Now().Before(*userModel.DeletionEffectiveAt) {
			eff := userModel.DeletionEffectiveAt.Format(time.RFC3339)
			out.PendingDeletion = true
			out.DeletionEffectiveAt = &eff
		}
	}
	resp.OK(c, out)
}

// UpdateMeProfile updates the caller's profile fields.
func (a *API) UpdateMeProfile(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	var body struct {
		Nickname string `json:"nickname"`
		Sign     string `json:"sign"`
		Gender   string `json:"gender"`
		Birthday string `json:"birthday"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	updates := make(map[string]interface{})
	if body.Nickname != "" {
		updates["nickname"] = body.Nickname
	}
	if body.Sign != "" {
		updates["sign"] = body.Sign
	}
	if body.Gender != "" {
		updates["gender"] = body.Gender
	}
	if body.Birthday != "" {
		updates["birthday"] = body.Birthday
	}
	if err := a.UserSvc.UpdateProfile(c.Request.Context(), uid, updates); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	a.esIndexUser(uid) // nickname/sign 都在 ES 用户文档里，改后异步重建索引
	profile, err := a.UserSvc.GetMe(c.Request.Context(), uid)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	g := normalizeGender(profile.Gender)
	resp.OK(c, updateMeProfileResponse{
		Nickname: profile.Nickname,
		Sign:     profile.Sign,
		Gender:   g,
		Birthday: strings.TrimSpace(profile.Birthday),
		CakeID:   strings.TrimSpace(profile.CakeID),
	})
}

// UpdateMeUsername renames the caller's account.
func (a *API) UpdateMeUsername(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	var body struct {
		Username string `json:"username"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || len(strings.TrimSpace(body.Username)) < 3 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if err := a.UserSvc.UpdateUsername(c.Request.Context(), uid, strings.TrimSpace(body.Username)); err != nil {
		resp.Err(c, http.StatusConflict, errcode.CodeUsernameExists)
		return
	}
	a.esIndexUser(uid) // username 在 ES 用户文档里，改名后异步重建索引
	resp.OK(c, okResponse{OK: true})
}

// UpdateMePassword changes the caller's password.
func (a *API) UpdateMePassword(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	var body struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if len(body.NewPassword) < 8 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}

	hashStr, err := a.UserSvc.GetPasswordHash(c.Request.Context(), uid)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hashStr), []byte(body.OldPassword)); err != nil {
		resp.Err(c, http.StatusForbidden, errcode.CodePasswordMismatch)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(body.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if err := a.UserSvc.UpdatePassword(c.Request.Context(), uid, string(hash)); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	// 改密后使该用户所有旧 Refresh Token 失效。
	if err := a.AuthSvc.BumpRefreshEpoch(c.Request.Context(), uid); err != nil {
		a.Log.Error("bump refresh epoch after password change", zap.Uint64("user_id", uid), zap.Error(err))
	}
	resp.OK(c, okResponse{OK: true})
}

// UpdateMeAvatar uploads/replaces the caller's avatar.
func (a *API) UpdateMeAvatar(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	fh, err := c.FormFile("avatar")
	if err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if code := coverval.ValidateAvatarHeader(fh); code != 0 {
		resp.Err(c, http.StatusBadRequest, code)
		return
	}
	ext := strings.TrimPrefix(filepath.Ext(fh.Filename), ".")
	if ext == "" {
		ext = "png"
	}
	objectKey := fmt.Sprintf("avatars/%d.%s", uid, ext)
	f, err := fh.Open()
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	defer f.Close()
	if err := a.StorageSvc.UploadReader(objectKey, f); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if err := a.UserSvc.UpdateAvatar(c.Request.Context(), uid, objectKey); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, imageURLResponse{ImageURL: objectKey})
}

// UpdateMeAnnouncement updates the caller's space announcement.
func (a *API) UpdateMeAnnouncement(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	var body struct {
		Announcement string `json:"announcement"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if len([]rune(body.Announcement)) > 150 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if err := a.UserSvc.UpdateAnnouncement(c.Request.Context(), uid, body.Announcement); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, announcementResponse{Announcement: body.Announcement})
}

func validProfileBirthday(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return true
	}
	if len(s) != 10 || s[4] != '-' || s[7] != '-' {
		return false
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return false
	}
	y := t.Year()
	return y >= 1900 && y <= 2100
}

func normalizeGender(input string) string {
	s := strings.TrimSpace(input)
	switch s {
	case "male", "female":
		return s
	default:
		return "secret"
	}
}

func validProfileNickname(s string) bool {
	return len([]rune(s)) <= 30
}

func validProfileSign(s string) bool {
	return len([]rune(s)) <= 500
}

func validSpaceAnnouncement(s string) bool {
	return len([]rune(s)) <= 150
}

func creatorUpInclusiveDays(first *time.Time) int {
	if first == nil || first.IsZero() {
		return 0
	}
	now := time.Now()
	if first.After(now) {
		return 1
	}
	days := int(now.Sub(*first).Hours() / 24)
	return days + 1
}
