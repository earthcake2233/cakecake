package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"minibili/internal/data"
	"minibili/internal/errcode"
	"minibili/internal/model"
	"minibili/internal/pkg/dailyreward"
	"minibili/internal/pkg/resp"
	"minibili/internal/pkg/username"
)

type registerReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type refreshReq struct {
	RefreshToken string `json:"refresh_token"`
}

type tokenPairResp struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Register creates a new user (F1).
func (a *API) Register(c *gin.Context) {
	var req registerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if !username.Valid(req.Username) || len(req.Password) < 8 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	lowName := strings.ToLower(req.Username)
	if strings.EqualFold(req.Username, strings.TrimSpace(a.Cfg.AgentBotUsername)) ||
		lowName == "minibili_ai" || strings.HasPrefix(lowName, "ai_") {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	u := model.User{Username: req.Username, PasswordHash: string(hash)}
	err = a.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&u).Error; err != nil {
			return err
		}
		cid := model.FormatCakeID(u.ID)
		return tx.Model(&u).Update("cake_id", cid).Error
	})
	if err != nil {
		low := strings.ToLower(err.Error())
		if strings.Contains(low, "duplicate") || strings.Contains(low, "unique") {
			resp.Err(c, http.StatusBadRequest, errcode.CodeUsernameExists)
			return
		}
		a.Log.Error("register insert", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	_ = a.DB.First(&u, u.ID)
	a.esIndexUser(u.ID)
	resp.JSON(c, http.StatusCreated, errcode.CodeSuccess, gin.H{
		"user_id":  u.ID,
		"username": u.Username,
		"cake_id":  u.CakeID,
	})
}

// Login returns JWT pair (F1, Skill S-009).
func (a *API) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var u model.User
	if err := a.DB.Where("username = ?", strings.TrimSpace(req.Username)).First(&u).Error; err != nil {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeInvalidLogin)
		return
	}
	_ = maybeFinalizeAccountDeletion(a, u.ID)
	if err := a.DB.First(&u, u.ID).Error; err != nil {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeInvalidLogin)
		return
	}
	if model.IsUserAnonymized(&u) {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeInvalidLogin)
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)) != nil {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeInvalidLogin)
		return
	}
	access, refresh, _, err := a.JWT.IssuePair(u.ID)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	_ = dailyreward.MarkLogin(a.DB, u.ID)
	a.Log.Info("user login success", zap.String("username", u.Username))
	resp.OK(c, tokenPairResp{AccessToken: access, RefreshToken: refresh})
}

// Refresh rotates refresh token (Skill S-009).
func (a *API) Refresh(c *gin.Context) {
	var req refreshReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	uid, tokenID, err := a.JWT.ParseRefresh(strings.TrimSpace(req.RefreshToken))
	if err != nil {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	_ = maybeFinalizeAccountDeletion(a, uid)
	var u model.User
	if err := a.DB.First(&u, uid).Error; err == nil && model.IsUserAnonymized(&u) {
		resp.Err(c, http.StatusForbidden, errcode.CodeAccountClosed)
		return
	}
	ctx := context.Background()
	if a.Redis.Exists(ctx, data.RefreshInvalidKey(tokenID)).Val() == 1 {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	_ = a.Redis.Set(ctx, data.RefreshInvalidKey(tokenID), "1", data.RefreshInvalidTTL).Err()
	access, refresh, _, err := a.JWT.IssuePair(uid)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, tokenPairResp{AccessToken: access, RefreshToken: refresh})
}

