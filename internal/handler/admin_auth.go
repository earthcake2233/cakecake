package handler

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"cakecake/internal/errcode"
	"cakecake/internal/middleware"
	"cakecake/internal/model/admin"
	"cakecake/internal/pkg/resp"
)

type adminLoginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type adminTokenPairResp struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// AdminLogin POST /api/v1/admin/auth/login
func (a *API) AdminLogin(c *gin.Context) {
	var req adminLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	adm, err := a.AuthSvc.FindAdminByUsername(c.Request.Context(), strings.TrimSpace(req.Username))
	if err != nil {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeInvalidLogin)
		return
	}
	if adm.Status != admin.StatusActive {
		resp.Err(c, http.StatusForbidden, errcode.CodeAdminDisabled)
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(adm.PasswordHash), []byte(req.Password)) != nil {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeInvalidLogin)
		return
	}
	access, refresh, _, err := a.JWT.IssueAdminPair(adm.ID)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	now := time.Now()
	_ = a.AuthSvc.UpdateAdminLoginTime(c.Request.Context(), adm.ID, now)
	a.Log.Info("admin login", zap.String("username", adm.Username), zap.Uint64("admin_id", adm.ID))
	resp.OK(c, adminTokenPairResp{AccessToken: access, RefreshToken: refresh})
}

// AdminRefresh POST /api/v1/admin/auth/refresh
func (a *API) AdminRefresh(c *gin.Context) {
	var req refreshReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	aid, tokenID, err := a.JWT.ParseAdminRefresh(strings.TrimSpace(req.RefreshToken))
	if err != nil {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	adm, err := a.AuthSvc.GetAdminByID(c.Request.Context(), aid)
	if err != nil || adm.Status != admin.StatusActive {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	ctx := context.Background()
	if a.AuthSvc.AdminRefreshTokenInvalid(ctx, tokenID) {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	_ = a.AuthSvc.MarkAdminRefreshTokenInvalid(ctx, tokenID)
	access, refresh, _, err := a.JWT.IssueAdminPair(adm.ID)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, adminTokenPairResp{AccessToken: access, RefreshToken: refresh})
}

// AdminMe GET /api/v1/admin/me
func (a *API) AdminMe(c *gin.Context) {
	type adminMeResponse struct {
		ID          uint64 `json:"id"`
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
	}
	aid, ok := adminIDFromCtx(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	adm, err := a.AuthSvc.GetAdminByID(c.Request.Context(), aid)
	if err != nil {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	resp.OK(c, adminMeResponse{
		ID:          adm.ID,
		Username:    adm.Username,
		DisplayName: adm.DisplayName,
	})
}

func adminIDFromCtx(c *gin.Context) (uint64, bool) {
	return middleware.AdminID(c)
}
