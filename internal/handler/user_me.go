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
	"minibili/internal/model"
	"minibili/internal/pkg/coverval"
	"minibili/internal/pkg/resp"
)

func (a *API) GetMe(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	profile, err := a.UserSvc.GetMe(c.Request.Context(), uid)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	resp.OK(c, profile)
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
	resp.OK(c, gin.H{"ok": true})
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

	var u struct{ PasswordHash string }
	if err := a.DB.Raw("SELECT password_hash FROM users WHERE id = ?", uid).Scan(&u).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(body.OldPassword)); err != nil {
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
	if err := a.DB.Model(&model.User{}).Where("id = ?", uid).Update("avatar_url", objectKey).Error; err != nil {
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
	resp.OK(c, gin.H{"ok": true})
}

// validProfileBirthday validates birthday format YYYY-MM-DD, year 1900-2100.
// Empty string is valid (optional field).
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
