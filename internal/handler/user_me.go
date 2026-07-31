package handler

import (
	"fmt"
	"net/http"
	"path/filepath"

	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"minibili/internal/errcode"
	"minibili/internal/middleware"
	"minibili/internal/pkg/coverval"
	"minibili/internal/pkg/dailyreward"
	"minibili/internal/pkg/resp"
	"minibili/internal/pkg/usercoin"
	"minibili/internal/pkg/userlevel"
)

func (a *API) GetMe(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	_ = maybeFinalizeAccountDeletion(a, uid)
	profile, err := a.UserSvc.GetMe(c.Request.Context(), uid)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	_ = dailyreward.MarkLogin(a.DB, uid)
	g := normalizeGender(profile.Gender)
	out := gin.H{
		"user_id":      profile.ID,
		"username":     profile.Username,
		"cake_id":      strings.TrimSpace(profile.CakeID),
		"nickname":     profile.Nickname,
		"sign":         profile.Sign,
		"announcement": strings.TrimSpace(profile.Announcement),
		"gender":       g,
		"birthday":     strings.TrimSpace(profile.Birthday),
		"avatar_url":   profile.AvatarURL,
		"created_at":   profile.CreatedAt,
	}
	// Add extra fields expected by integration tests
	userModel, _ := a.UserSvc.GetUserByID(c.Request.Context(), uid)
	if userModel != nil {
		out["space_privacy"] = spacePrivacyFromUser(userModel)
		out["level_info"] = userlevel.FromExperience(userModel.Experience)
		out["coin_balance"] = usercoin.BalanceFloat(userModel.CoinBalanceTenths)
	}
	resp.OK(c, out)
}

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
	profile, err := a.UserSvc.GetMe(c.Request.Context(), uid)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	g := normalizeGender(profile.Gender)
	resp.OK(c, gin.H{
		"nickname": profile.Nickname,
		"sign":     profile.Sign,
		"gender":   g,
		"birthday": strings.TrimSpace(profile.Birthday),
		"cake_id":  strings.TrimSpace(profile.CakeID),
	})
}

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
	resp.OK(c, gin.H{"ok": true})
}

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
	resp.OK(c, gin.H{"ok": true})
}

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
	if err := a.OSS.UploadReader(objectKey, f); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if err := a.UserSvc.UpdateAvatar(c.Request.Context(), uid, objectKey); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"avatar_url": objectKey})
}

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
	resp.OK(c, gin.H{"announcement": body.Announcement})
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
