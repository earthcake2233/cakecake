package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"minibili/internal/errcode"
	"minibili/internal/pkg/resp"
)

// knownConfigKeys defines the set of valid runtime config keys.
var knownConfigKeys = map[string]bool{
	"agent_enabled":        true,
	"agent_daily_quota":    true,
	"agent_max_history":    true,
	"agent_history_ttl":    true,
	"agent_request_timeout": true,
	"rate_limit_enabled":   true,
	"rate_limit_rate":      true,
	"rate_limit_burst":     true,
}

// AdminListSystemConfigs returns all runtime system configs from the cache.
func (a *API) AdminListSystemConfigs(c *gin.Context) {
	if a.RuntimeCfg == nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	// Read all keys from the known set using RuntimeConfig.Get with nil fallback
	result := make(map[string]string, len(knownConfigKeys))
	for key := range knownConfigKeys {
		result[key] = a.RuntimeCfg.Get(key, "")
	}
	resp.OK(c, result)
}

type adminUpdateSystemConfigReq struct {
	Configs map[string]string `json:"configs" binding:"required"`
}

// AdminUpdateSystemConfig updates one or more runtime system configs.
func (a *API) AdminUpdateSystemConfig(c *gin.Context) {
	if a.RuntimeCfg == nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	var req adminUpdateSystemConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if len(req.Configs) == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	for key := range req.Configs {
		if !knownConfigKeys[key] {
			resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
			return
		}
	}
	for key, value := range req.Configs {
		if err := a.RuntimeCfg.Set(c.Request.Context(), key, value); err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
	}
	a.AdminListSystemConfigs(c)
}
