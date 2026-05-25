package handler

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"minibili/internal/errcode"
	"minibili/internal/middleware"
	"minibili/internal/model"
	"minibili/internal/pkg/coverval"
	"minibili/internal/pkg/dailyreward"
	"minibili/internal/pkg/resp"
	"minibili/internal/pkg/username"
	"minibili/internal/pkg/usercoin"
	"minibili/internal/pkg/userlevel"
)

type updateMeUsernameReq struct {
	Username string `json:"username"`
}

type updateMeProfileReq struct {
	Nickname string `json:"nickname"`
	Sign     string `json:"sign"`
	Gender   string `json:"gender"`
	Birthday string `json:"birthday"`
}

type updateMeAnnouncementReq struct {
	Announcement string `json:"announcement"`
}

type updateMePasswordReq struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

func ensureUserCakeID(db *gorm.DB, u *model.User) {
	if strings.TrimSpace(u.CakeID) != "" {
		return
	}
	cid := model.FormatCakeID(u.ID)
	_ = db.Model(u).Update("cake_id", cid).Error
	u.CakeID = cid
}

func normalizeGender(g string) string {
	switch strings.TrimSpace(g) {
	case "male", "female", "secret":
		return strings.TrimSpace(g)
	default:
		return "secret"
	}
}

func validProfileNickname(s string) bool {
	n := utf8.RuneCountInString(s)
	return n <= 30
}

func validProfileSign(s string) bool {
	return utf8.RuneCountInString(s) <= 500
}

func validSpaceAnnouncement(s string) bool {
	return utf8.RuneCountInString(s) <= 150
}

func validProfileBirthday(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return true
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return false
	}
	y := t.Year()
	return y >= 1900 && y <= 2100
}

// creatorUpInclusiveDays counts from first successful publish anchor to now (第 1 天起算).
func creatorUpInclusiveDays(first *time.Time) int {
	if first == nil || first.IsZero() {
		return 0
	}
	now := time.Now()
	if !now.After(*first) {
		return 1
	}
	d := int(now.Sub(*first).Hours() / 24)
	if d < 0 {
		return 1
	}
	return d + 1
}

// GetMe returns current user profile (F0).
func (a *API) GetMe(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	_ = maybeFinalizeAccountDeletion(a, uid)
	var u model.User
	if err := a.DB.First(&u, uid).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if model.IsUserAnonymized(&u) {
		resp.Err(c, http.StatusForbidden, errcode.CodeAccountClosed)
		return
	}
	ensureUserCakeID(a.DB, &u)
	_ = dailyreward.MarkLogin(a.DB, uid)
	_ = a.DB.First(&u, uid).Error
	avatar := avatarURLForAPI(&u)
	g := normalizeGender(u.Gender)
	out := gin.H{
		"user_id":      u.ID,
		"username":     u.Username,
		"cake_id":      strings.TrimSpace(u.CakeID),
		"nickname":     u.Nickname,
		"sign":         u.Sign,
		"announcement": strings.TrimSpace(u.SpaceAnnouncement),
		"gender":       g,
		"birthday":     strings.TrimSpace(u.Birthday),
		"avatar_url":   avatar,
		"created_at":   u.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	pending := u.DeletionRequestedAt != nil && u.DeletionEffectiveAt != nil &&
		time.Now().Before(*u.DeletionEffectiveAt)
	out["pending_deletion"] = pending
	if u.DeletionEffectiveAt != nil {
		out["deletion_effective_at"] = u.DeletionEffectiveAt.Format(time.RFC3339)
	} else {
		out["deletion_effective_at"] = nil
	}
	if u.FirstPublishedAt != nil && !u.FirstPublishedAt.IsZero() {
		out["first_published_at"] = u.FirstPublishedAt.UTC().Format(time.RFC3339)
		out["creator_up_days"] = creatorUpInclusiveDays(u.FirstPublishedAt)
	} else {
		out["first_published_at"] = nil
		out["creator_up_days"] = 0
	}
	priv := spacePrivacyFromUser(&u)
	out["space_privacy"] = priv
	out["level_info"] = userlevel.FromExperience(u.Experience)
	out["coin_balance"] = usercoin.BalanceFloat(u.CoinBalanceTenths)
	resp.OK(c, out)
}

// UpdateMeAnnouncement sets the personal-space bulletin (≤150 runes).
func (a *API) UpdateMeAnnouncement(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	var req updateMeAnnouncementReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	ann := strings.TrimSpace(req.Announcement)
	if !validSpaceAnnouncement(ann) {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var u model.User
	if err := a.DB.First(&u, uid).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if model.IsUserAnonymized(&u) {
		resp.Err(c, http.StatusForbidden, errcode.CodeAccountClosed)
		return
	}
	if err := a.DB.Model(&u).Update("space_announcement", ann).Error; err != nil {
		a.Log.Error("update announcement", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	_ = a.DB.First(&u, uid)
	outAnn := strings.TrimSpace(u.SpaceAnnouncement)
	resp.OK(c, gin.H{"user_id": u.ID, "announcement": outAnn})
}

// UpdateMeProfile persists nickname, signature, gender, birthday (Rule: personal center).
func (a *API) UpdateMeProfile(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	var req updateMeProfileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	nick := strings.TrimSpace(req.Nickname)
	sign := strings.TrimSpace(req.Sign)
	if !validProfileNickname(nick) {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if !validProfileSign(sign) {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	g := normalizeGender(req.Gender)
	bday := strings.TrimSpace(req.Birthday)
	if !validProfileBirthday(bday) {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var u model.User
	if err := a.DB.First(&u, uid).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	ensureUserCakeID(a.DB, &u)
	oldNick := strings.TrimSpace(u.Nickname)
	nickChanged := nick != oldNick
	updates := map[string]interface{}{
		"nickname": nick,
		"sign":     sign,
		"gender":   g,
		"birthday": bday,
	}
	if nickChanged {
		err := a.DB.Transaction(func(tx *gorm.DB) error {
			if err := usercoin.SpendWholeCoins(tx, uid, usercoin.NicknameChangeCostCoins); err != nil {
				return err
			}
			return tx.Model(&u).Updates(updates).Error
		})
		if err != nil {
			if errors.Is(err, usercoin.ErrInsufficientCoins) {
				resp.Err(c, http.StatusBadRequest, errcode.CodeInsufficientCoins)
				return
			}
			a.Log.Error("update profile", zap.Error(err))
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
	} else if err := a.DB.Model(&u).Updates(updates).Error; err != nil {
		a.Log.Error("update profile", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	_ = a.DB.First(&u, uid)
	a.esIndexUser(uid)
	avatar := avatarURLForAPI(&u)
	resp.OK(c, gin.H{
		"user_id":      u.ID,
		"username":     u.Username,
		"cake_id":      strings.TrimSpace(u.CakeID),
		"nickname":     u.Nickname,
		"sign":         u.Sign,
		"announcement": strings.TrimSpace(u.SpaceAnnouncement),
		"gender":       normalizeGender(u.Gender),
		"birthday":     strings.TrimSpace(u.Birthday),
		"avatar_url":   avatar,
		"created_at":   u.CreatedAt.Format("2006-01-02 15:04:05"),
		"coin_balance": usercoin.BalanceFloat(u.CoinBalanceTenths),
		"level_info":   userlevel.FromExperience(u.Experience),
	})
}

// UpdateMeUsername updates username with uniqueness check (F0).
func (a *API) UpdateMeUsername(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	var req updateMeUsernameReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	name := strings.TrimSpace(req.Username)
	if !username.Valid(name) {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var other model.User
	if err := a.DB.Where("username = ? AND id <> ?", name, uid).First(&other).Error; err == nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeUsernameExists)
		return
	}
	var u model.User
	if err := a.DB.First(&u, uid).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if u.Username == name {
		resp.OK(c, gin.H{"user_id": u.ID, "username": u.Username})
		return
	}
	if err := a.DB.Model(&u).Update("username", name).Error; err != nil {
		low := strings.ToLower(err.Error())
		if strings.Contains(low, "duplicate") || strings.Contains(low, "unique") {
			resp.Err(c, http.StatusBadRequest, errcode.CodeUsernameExists)
			return
		}
		a.Log.Error("update username", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	u.Username = name
	a.esIndexUser(uid)
	resp.OK(c, gin.H{"user_id": u.ID, "username": u.Username})
}

// UpdateMePassword changes password after verifying old password (F0, R-AUTH-2).
func (a *API) UpdateMePassword(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	var req updateMePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if len(req.NewPassword) < 8 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var u model.User
	if err := a.DB.First(&u, uid).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.OldPassword)) != nil {
		resp.Err(c, http.StatusForbidden, errcode.CodePasswordMismatch)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if err := a.DB.Model(&u).Update("password_hash", string(hash)).Error; err != nil {
		a.Log.Error("update password", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// UpdateMeAvatar uploads avatar to OSS at avatars/{user_id}.{ext} (F0, NF-7, R-BIZ-8).
func (a *API) UpdateMeAvatar(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	if err := c.Request.ParseMultipartForm(6 << 20); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
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
	if a.OSS == nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if err := os.MkdirAll(a.Cfg.TempUploadDir, 0o755); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	tmp := filepath.Join(a.Cfg.TempUploadDir, uuid.NewString()+filepath.Ext(fh.Filename))
	if err := saveUploadedFile(fh, tmp); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	defer os.Remove(tmp)
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(fh.Filename)), ".")
	if ext == "jpeg" {
		ext = "jpg"
	}
	key := fmt.Sprintf("avatars/%d.%s", uid, ext)
	if err := a.OSS.UploadFile(key, tmp); err != nil {
		a.Log.Error("oss avatar upload", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	url := a.Cfg.OSSObjectURL(key)
	now := time.Now()
	if err := a.DB.Model(&model.User{}).Where("id = ?", uid).Updates(map[string]interface{}{
		"avatar_url": url,
		"updated_at": now,
	}).Error; err != nil {
		a.Log.Error("avatar url save", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	var u model.User
	if err := a.DB.First(&u, uid).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"avatar_url": avatarURLForAPI(&u)})
}
