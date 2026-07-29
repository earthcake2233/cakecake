package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"minibili/internal/errcode"
	"minibili/internal/pkg/resp"
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
	result, svcErr := a.AuthSvc.Register(c.Request.Context(), req.Username, req.Password)
	if svcErr != nil {
		code := errCodeFromSvc(svcErr)
		if code == errcode.CodeParamError || code == 40006 {
			resp.Err(c, http.StatusBadRequest, code)
			return
		}
		a.Log.Error("register", zap.Error(svcErr))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	a.esIndexUser(result.UserID)
	resp.JSON(c, http.StatusCreated, errcode.CodeSuccess, gin.H{
		"user_id":  result.UserID,
		"username": result.Username,
		"cake_id":  result.CakeID,
	})
}

// Login returns JWT pair (F1, Skill S-009).
func (a *API) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	brief, lookErr := a.AuthSvc.LookupUser(c.Request.Context(), req.Username)
	if lookErr != nil {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeInvalidLogin)
		return
	}
	_ = maybeFinalizeAccountDeletion(a, brief.ID)
	result, svcErr := a.AuthSvc.Authenticate(c.Request.Context(), brief.ID, req.Password)
	if svcErr != nil {
		code := errCodeFromSvc(svcErr)
		httpStatus := http.StatusInternalServerError
		if code == 40100 || code == errcode.CodeUnauthorized {
			httpStatus = http.StatusUnauthorized
		}
		resp.Err(c, httpStatus, code)
		return
	}
	resp.OK(c, tokenPairResp{AccessToken: result.AccessToken, RefreshToken: result.RefreshToken})
}

// Refresh rotates refresh token (Skill S-009).
func (a *API) Refresh(c *gin.Context) {
	var req refreshReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	uid, _, parseErr := a.JWT.ParseRefresh(strings.TrimSpace(req.RefreshToken))
	if parseErr == nil {
		_ = maybeFinalizeAccountDeletion(a, uid)
	}
	result, svcErr := a.AuthSvc.Refresh(c.Request.Context(), req.RefreshToken)
	if svcErr != nil {
		code := errCodeFromSvc(svcErr)
		httpStatus := http.StatusInternalServerError
		switch code {
		case errcode.CodeUnauthorized:
			httpStatus = http.StatusUnauthorized
		case 40302:
			httpStatus = http.StatusForbidden
		}
		resp.Err(c, httpStatus, code)
		return
	}
	resp.OK(c, tokenPairResp{AccessToken: result.AccessToken, RefreshToken: result.RefreshToken})
}
