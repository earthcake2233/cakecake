package handler

import (
	"cakecake/internal/model/notification"
	"cakecake/internal/model/user"
	"cakecake/internal/model/video"
	"context"
	crand "crypto/rand"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"cakecake/internal/errcode"
	"cakecake/internal/middleware"
	"cakecake/internal/pkg/resp"
)

type deletionPasswordReq struct {
	Password string `json:"password"`
}

func deletionCoolingDays() int {
	n, err := crand.Int(crand.Reader, big.NewInt(24))
	if err != nil {
		return 14
	}
	return 7 + int(n.Int64())
}

func deleteOwnedVideosAndRelated(tx *gorm.DB, uid uint64) ([]video.Video, error) {
	var videos []video.Video
	if err := tx.Where("user_id = ?", uid).Find(&videos).Error; err != nil {
		return nil, err
	}
	for _, v := range videos {
		if err := deleteVideoCascade(tx, v.ID); err != nil {
			return videos, err
		}
	}
	return videos, nil
}

func finalizeUserAnonymization(tx *gorm.DB, uid uint64) ([]video.Video, error) {
	if err := tx.Where("recipient_id = ?", uid).Delete(&notification.Notification{}).Error; err != nil {
		return nil, err
	}
	if err := tx.Where("recipient_id = ?", uid).Delete(&notification.LikeNotifMute{}).Error; err != nil {
		return nil, err
	}
	removedVideos, err := deleteOwnedVideosAndRelated(tx, uid)
	if err != nil {
		return removedVideos, err
	}
	raw := strings.ReplaceAll(uuid.NewString(), "-", "")
	anonU := "d" + raw[:31]
	impossible := anonU + uuid.NewString()
	hash, err := bcrypt.GenerateFromPassword([]byte(impossible), bcrypt.DefaultCost)
	if err != nil {
		return removedVideos, err
	}
	now := time.Now()
	upd := map[string]interface{}{
		"username":              anonU,
		"nickname":              "",
		"sign":                  "",
		"avatar_url":            "",
		"gender":                "secret",
		"birthday":              "",
		"password_hash":         string(hash),
		"deletion_requested_at": nil,
		"deletion_effective_at": nil,
		"anonymized_at":         now,
	}
	return removedVideos, tx.Model(&user.User{}).Where("id = ?", uid).Updates(upd).Error
}

// maybeFinalizeAccountDeletion runs final anonymization when the cooling period has ended.
func maybeFinalizeAccountDeletion(a *API, uid uint64) error {
	if a == nil {
		return nil
	}
	ctx := context.Background()
	u, err := a.UserSvc.GetUserByID(ctx, uid)
	if err != nil {
		return nil
	}
	if user.IsUserAnonymized(u) {
		return nil
	}
	if u.DeletionEffectiveAt == nil || time.Now().Before(*u.DeletionEffectiveAt) {
		return nil
	}
	var removedVideos []video.Video
	err = a.UserSvc.FinalizeDeletion(ctx, uid, func(tx *gorm.DB) error {
		var innerErr error
		removedVideos, innerErr = finalizeUserAnonymization(tx, uid)
		return innerErr
	})
	if err != nil {
		return err
	}
	for _, v := range removedVideos {
		purgeVideoOSSObjects(a.Cfg, a.OSS, a.Log, v)
		a.esDeleteVideo(v.ID)
	}
	return nil
}

// RequestAccountDeletion starts the cooling-off period (password required).
func (a *API) RequestAccountDeletion(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	_ = maybeFinalizeAccountDeletion(a, uid)
	var req deletionPasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if strings.TrimSpace(req.Password) == "" {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	u, err := a.UserSvc.GetUserByID(c.Request.Context(), uid)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if user.IsUserAnonymized(u) {
		resp.Err(c, http.StatusForbidden, errcode.CodeAccountClosed)
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)) != nil {
		resp.Err(c, http.StatusForbidden, errcode.CodePasswordMismatch)
		return
	}
	if u.DeletionRequestedAt != nil && u.DeletionEffectiveAt != nil && time.Now().Before(*u.DeletionEffectiveAt) {
		resp.OK(c, gin.H{
			"ok":                    true,
			"pending":               true,
			"deletion_effective_at": u.DeletionEffectiveAt.Format(time.RFC3339),
		})
		return
	}
	days := deletionCoolingDays()
	eff := time.Now().AddDate(0, 0, days)
	now := time.Now()
	if err := a.UserSvc.RequestDeletion(c.Request.Context(), uid, now, eff); err != nil {
		a.Log.Error("request account deletion", zap.Error(err), zap.Uint64("user_id", uid))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{
		"ok":                    true,
		"pending":               true,
		"deletion_effective_at": eff.Format(time.RFC3339),
		"cooling_days":          days,
	})
}

// RevokeAccountDeletion cancels a pending deletion during the cooling-off period.
func (a *API) RevokeAccountDeletion(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	_ = maybeFinalizeAccountDeletion(a, uid)
	u, err := a.UserSvc.GetUserByID(c.Request.Context(), uid)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if user.IsUserAnonymized(u) {
		resp.Err(c, http.StatusBadRequest, errcode.CodeDeletionRevokeExpired)
		return
	}
	if u.DeletionRequestedAt == nil || u.DeletionEffectiveAt == nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if !time.Now().Before(*u.DeletionEffectiveAt) {
		resp.Err(c, http.StatusBadRequest, errcode.CodeDeletionRevokeExpired)
		return
	}
	if err := a.UserSvc.RevokeDeletion(c.Request.Context(), uid); err != nil {
		a.Log.Error("revoke account deletion", zap.Error(err), zap.Uint64("user_id", uid))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}
